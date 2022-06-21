package news

import (
	"context"
	"time"

	"github.com/danesparza/daydash-service/internal/logger"
	"go.uber.org/zap"
)

var (
	rootlogger *zap.Logger
	zlog       *zap.SugaredLogger
)

// NewsFetchTask manages fetching & storing news items as they appear
// in the twitter news feed
func NewsFetchTask(ctx context.Context) {

	zlog.Infof("NewsFetchTask starting... ")

	for {
		select {
		case <-time.After(5 * time.Minute):
			//	As we get a request on a channel to fire a trigger...
			//	Create a goroutine
			go func(cx context.Context) {

				//	Get the latest tweet id

				//	Pass that 'sinceid' to FetchTweets

				//	Get the latest tweets

				//	Pass the latest tweets to a channel

				//	Create a worker channel for:
				//	- Reviewing latest tweets
				//	- Seeing if we have a record of a tweet already, first
				//	- If we don't have a record... (send this to a channel so we can do this?)
				//	-- Fetch the url for the story (capture the long url, minus the querystring)
				//	-- Process the main image for the story
				//	-- Save the new story and update information

			}(ctx) // Launch the goroutine
		case <-ctx.Done():
			zlog.Infof("NewsFetchTask stopping")
			return
		}
	}

}

func init() {
	rootlogger, _ = logger.NewProd()
	defer rootlogger.Sync() // flushes buffer, if any
	zlog = rootlogger.Sugar()
}
