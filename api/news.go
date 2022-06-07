package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// NewsReport defines a news report
type NewsReport struct {
	Items   NewsItems `json:"items"`
	Version string    `json:"version"`
}

// NewsItem represents a single news item
type NewsItem struct {
	ID         string `json:"id"`
	CreateTime int64  `json:"createtime"`
	Text       string `json:"text"`
	MediaURL   string `json:"mediaurl"`
	MediaData  string `json:"mediadata"`
	StoryURL   string `json:"storyurl"`
}

type NewsItems []NewsItem

//	Let NewsItems know how to sort by implementing the sort interface (https://pkg.go.dev/sort#Interface):
func (n NewsItems) Len() int {
	return len(n)
}

func (n NewsItems) Less(i, j int) bool {
	return n[i].CreateTime > n[j].CreateTime
}

func (n NewsItems) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// TwitterTimelineResponse represents the response to a timeline request
type TwitterTimelineResponse struct {
	Tweets []Tweet `json:"data"`
	Meta   struct {
		OldestID    string `json:"oldest_id"`
		NewestID    string `json:"newest_id"`
		ResultCount int    `json:"result_count"`
		NextToken   string `json:"next_token"`
	} `json:"meta"`
}

// Tweet represents an individual tweet
type Tweet struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
	Entities  struct {
		Annotations []struct {
			Start          int     `json:"start"`
			End            int     `json:"end"`
			Probability    float64 `json:"probability"`
			Type           string  `json:"type"`
			NormalizedText string  `json:"normalized_text"`
		} `json:"annotations"`
		Urls []struct {
			Start       int    `json:"start"`
			End         int    `json:"end"`
			URL         string `json:"url"`
			ExpandedURL string `json:"expanded_url"`
			DisplayURL  string `json:"display_url"`
		} `json:"urls"`
	} `json:"entities"`
	ID string `json:"id"`
}

// NewsService is the interface for all services that can fetch news data
type NewsService interface {
	// GetNewsReport gets the news report
	GetNewsReport(ctx context.Context) (NewsReport, error)
}

// GetNewsReport gets breaking news from CNN
func (s Service) GetNewsReport(rw http.ResponseWriter, req *http.Request) {

	txn := newrelic.FromContext(req.Context())
	segment := txn.StartSegment("News GetNewsReport")
	defer segment.End()

	retval := NewsReport{}

	//	Get the api key:
	apikey := os.Getenv("TWITTER_V2_BEARER_TOKEN")
	if apikey == "" {
		zlog.Errorw(
			"{TWITTER_V2_BEARER_TOKEN} is blank but shouldn't be",
		)
		return
	}

	//	Create our request with the cnnbrk userid (you can get the userid by calling
	//	https://api.twitter.com/2/users/by/username/cnnbrk ):
	//	Fetch the url:
	clientRequest, err := http.NewRequestWithContext(req.Context(), http.MethodGet, "https://api.twitter.com/2/users/428333/tweets", nil)
	if err != nil {
		zlog.Errorw(
			"cannot create request",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	//	Set our query params
	q := clientRequest.URL.Query()
	q.Add("tweet.fields", "created_at,entities")
	clientRequest.URL.RawQuery = q.Encode()

	//	Set our headers
	clientRequest.Header.Set("Content-Type", "application/json; charset=UTF-8")
	clientRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apikey))
	clientRequest = newrelic.RequestWithTransactionContext(clientRequest, txn)

	//	Execute the request
	client := http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	clientResponse, err := client.Do(clientRequest)
	if err != nil {
		zlog.Errorw(
			"error when sending request to Twitter API server",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}
	defer clientResponse.Body.Close()

	//	Decode the response:
	twResponse := TwitterTimelineResponse{}
	err = json.NewDecoder(clientResponse.Body).Decode(&twResponse)
	if err != nil {
		zlog.Errorw(
			"problem decoding the response from the Twitter API server",
			"error", err,
		)
		txn.NoticeError(err)
		return
	}

	//	First ... Get a count of all tweets that have have urls associated with them:
	tweetsWithUrls := 0
	for _, tweet := range twResponse.Tweets {
		if len(tweet.Entities.Urls) > 0 {
			tweetsWithUrls++
		}
	}

	//	Then, for the tweets with media (photos), fetch the associated images and encode them
	c := make(chan NewsItem)
	timeout := time.After(25 * time.Second)

	for _, tweet := range twResponse.Tweets {

		//	Following example here ... https://go.dev/talks/2012/concurrency.slide#47
		go func(taskTxn *newrelic.Transaction, taskTweet Tweet) {

			mediaURL := ""
			storyURL := ""
			storyDisplayURL := ""
			storyText := taskTweet.Text
			mediaData := ""

			//	If we have an associated link, fetch it and get the image url associated (if one exists):
			if len(taskTweet.Entities.Urls) > 0 {
				storyURL = taskTweet.Entities.Urls[0].ExpandedURL
				storyDisplayURL = taskTweet.Entities.Urls[0].URL
				mediaURL, _ = GetTwitterImageUrlFromPage(taskTxn, storyURL)

				//	If the story text contains the display link, remove it:
				storyText = strings.Replace(storyText, storyDisplayURL, "", 1)
				storyText = strings.TrimSpace(storyText)

				//	If we have a mediaURL
				//	...fetch the image, encode it, add it to mediadata
				//	...add the story to the collection
				if mediaURL != "" {

					response, resizeErr := getResizedEncodedImage(taskTxn, mediaURL, 600, 300)
					if resizeErr != nil {
						zlog.Errorw(
							"problem getting the encoded mediaData image",
							"tweetID", taskTweet.ID,
							"mediaUrl", mediaURL,
						)
						txn.NoticeError(err)
					} else {
						mediaData = response
					}
				}

				c <- NewsItem{
					ID:         taskTweet.ID,
					CreateTime: taskTweet.CreatedAt.Unix(),
					Text:       storyText,
					MediaURL:   mediaURL,
					MediaData:  mediaData,
					StoryURL:   storyURL}
			}
		}(txn, tweet)

	}

	//	Gather all the responses...
loop:
	for i := 0; i < tweetsWithUrls; i++ {
		select {
		case result := <-c:
			retval.Items = append(retval.Items, result)
		case <-timeout:
			zlog.Errorw("timed out getting information about tweets")
			break loop
		}
	}

	//	Sort the responses
	sort.Sort(retval.Items)

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)
}

// GetTwitterImageUrlFromPage gets the twitter:image meta content tag contents for the given page url
// by fetching and parsing the page
func GetTwitterImageUrlFromPage(txn *newrelic.Transaction, page string) (string, error) {

	//	Set the initial value
	retval := ""

	//	Fetch the url:
	req, err := http.NewRequest(http.MethodGet, page, nil)
	if err != nil {
		zlog.Errorw(
			"cannot create request",
			"error", err,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("cannot create request: %v", err)
	}

	req = newrelic.RequestWithTransactionContext(req, txn)

	client := http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		zlog.Errorw(
			"error executing request to fetch url",
			"error", err,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("error executing request to fetch url: %v", err)
	}

	//	Read in the response
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		zlog.Errorw(
			"cannot read all of response body",
			"error", err,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("cannot read all of response body: %v", err)
	}

	// Find the meta item with the name 'twitter:image'
	doc.Find("meta[property='og:image']").Each(func(i int, s *goquery.Selection) {
		// For each item found, set the return value
		retval = s.AttrOr("content", "")
	})

	return retval, nil
}

func getResizedEncodedImage(txn *newrelic.Transaction, imageUrl string, width, height int) (string, error) {

	//	Our return value
	retval := ""

	type SubImager interface {
		SubImage(r image.Rectangle) image.Image
	}

	//	Fetch the url:
	req, err := http.NewRequest(http.MethodGet, imageUrl, nil)
	if err != nil {
		zlog.Errorw(
			"cannot create request to fetch image",
			"error", err,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("cannot create request to fetch image: %v", err)
	}

	req = newrelic.RequestWithTransactionContext(req, txn)

	client := http.Client{}
	client.Transport = newrelic.NewRoundTripper(client.Transport)
	response, err := client.Do(req)
	if response != nil {
		defer response.Body.Close()
	}
	if err != nil {
		zlog.Errorw(
			"error fetching image url",
			"url", imageUrl,
			"error", err,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("error fetching image url: %v", err)
	}

	if response.StatusCode != 200 {
		return retval, fmt.Errorf("expected http 200 status code but got %v instead", response.StatusCode)
	}

	//	Analyze the image and crop it
	img, _, err := image.Decode(response.Body)
	if err != nil {
		zlog.Errorw(
			"error reading source image",
			"url", imageUrl,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("error reading source image: %v", err)
	}

	resizer := nfnt.NewDefaultResizer()
	analyzer := smartcrop.NewAnalyzer(resizer)
	topCrop, err := analyzer.FindBestCrop(img, width, height)
	if err != nil {
		zlog.Errorw(
			"error finding best crop",
			"url", imageUrl,
		)
		txn.NoticeError(err)
		return retval, fmt.Errorf("error finding best crop: %v", err)
	}

	img = img.(SubImager).SubImage(topCrop)
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		img = resizer.Resize(img, uint(width), uint(height))
	}

	//	Encode as jpg image data
	buffer := new(bytes.Buffer)
	jpeg.Encode(buffer, img, nil)

	//	base64 encode the image data and set the return value
	retval = fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(buffer.Bytes()))

	return retval, nil
}
