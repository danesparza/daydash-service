package news_test

import (
	"context"
	"testing"

	"github.com/danesparza/daydash-service/internal/news"
)

func TestFetchTweets_DoesNotReturnError(t *testing.T) {
	ctx := context.TODO()

	tweets, err := news.FetchTweets(ctx, "")
	if err != nil {
		t.Errorf("Returned error and we didn't expect that: %v", err)
	}

	//	Spit out what we know:
	for _, tweet := range tweets {
		t.Logf("Tweet (id) text: (%v) %v\n", tweet.ID, tweet.Text)
	}
}
