package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/danesparza/daydash-service/internal/news"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// NewsReport defines a news report
type NewsReport struct {
	Items NewsItems `json:"items"`
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

// NewsService is the interface for all services that can fetch news data
type NewsService interface {
	// GetNewsReport gets the news report
	GetNewsReport(ctx context.Context) (NewsReport, error)
}

// GetNewsReport godoc
// @Summary Gets breaking news from CNN
// @Description Gets breaking news from CNN.  Images are included inline as base64 encoded jpeg images
// @Tags dashboard
// @Accept  json
// @Produce  json
// @Success 200 {object} api.NewsReport
// @Failure 400 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /news [get]
func (s Service) GetNewsReport(rw http.ResponseWriter, req *http.Request) {

	txn := newrelic.FromContext(req.Context())
	segment := txn.StartSegment("News GetNewsReport")
	defer segment.End()

	retval := NewsReport{}

	//	Fetch the items from MongoDB
	//	Only get the latest update from each story
	newsItems, err := news.GetRecentNewsStories(req.Context(), 6)
	if err != nil {
		txn.NoticeError(err)
		sendErrorResponse(rw, err, http.StatusBadRequest)
	}

	//	Spit out what we know:
	for _, item := range newsItems {
		retval.Items = append(retval.Items, NewsItem{
			ID:         item.Updates[0].ID,
			Text:       item.Updates[0].Text,
			CreateTime: int64(item.Updates[0].Time),
			MediaURL:   item.Updates[0].Mediaurl,
			MediaData:  item.Updates[0].Mediadata,
			StoryURL:   item.ShortURL,
		})
	}

	//	Sort the responses (do I still need this?)
	// sort.Sort(retval.Items)

	//	Serialize to JSON & return the response:
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(rw).Encode(retval)
}
