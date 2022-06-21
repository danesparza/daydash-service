package news

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
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/muesli/smartcrop"
	"github.com/muesli/smartcrop/nfnt"
	"github.com/newrelic/go-agent/v3/newrelic"
)

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

// FetchTweets gets the latest tweets (from the given tweetID)
func FetchTweets(ctx context.Context, sinceTweetID string) ([]Tweet, error) {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News FetchTweets")
	defer segment.End()

	retval := []Tweet{}

	//	Get the api key:
	apikey := os.Getenv("TWITTER_V2_BEARER_TOKEN")
	if apikey == "" {
		zlog.Errorw(
			"{TWITTER_V2_BEARER_TOKEN} is blank but shouldn't be",
		)
		return retval, fmt.Errorf("{TWITTER_V2_BEARER_TOKEN} is blank but shouldn't be")
	}

	//	Create our request with the cnnbrk userid (you can get the userid by calling
	//	https://api.twitter.com/2/users/by/username/cnnbrk ):
	//	Fetch the url:
	clientRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.twitter.com/2/users/428333/tweets", nil)
	if err != nil {
		zlog.Errorw(
			"cannot create request",
			"error", err,
		)
		txn.NoticeError(err)
		return retval, err
	}

	//	Set our query params
	q := clientRequest.URL.Query()
	q.Add("tweet.fields", "created_at,entities")
	if strings.TrimSpace(sinceTweetID) != "" { // If we were passed a since_id, use it
		q.Add("since_id", sinceTweetID)
	}
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
		return retval, fmt.Errorf("error when sending request to Twitter API server: %v", err)
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
		return retval, fmt.Errorf("problem decoding the response from the Twitter API server: %v", err)
	}

	//	Return what we have:
	retval = twResponse.Tweets
	return retval, nil

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
