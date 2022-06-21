package news

import (
	"context"
	"time"

	"github.com/danesparza/daydash-service/internal/logger"
	"github.com/danesparza/daydash-service/internal/telemetry"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
)

var (
	rootlogger *zap.Logger
	zlog       *zap.SugaredLogger
)

// BackgroundNewsProcessor encapsulates background processing operations for the news
type BackgroundNewsProcessor struct {

	// ProcessTweet signals a tweet to be processed
	ProcessTweet chan Tweet
}

// NewsFetchTask manages fetching & passing tweets to the tweet processor as they appear
// in the twitter news feed
func (bnp BackgroundNewsProcessor) NewsFetchTask(ctx context.Context) {

	zlog.Infof("NewsFetchTask starting... ")

	for {
		select {
		case <-time.After(10 * time.Second):
			//	As we get a request on a channel to fire a trigger...
			//	Create a goroutine
			go func(cx context.Context) {
				//	Start a background transaction
				txn := telemetry.NRApp.StartTransaction("NewsFetchTask")
				defer txn.End()

				//	Add it to the context
				cx = newrelic.NewContext(cx, txn)

				//	Get the latest tweet id
				tweetsSinceID := GetMostRecentTweetID(cx)

				//	Pass that 'sinceid' to FetchTweets
				tweetsResponse, err := FetchTweets(cx, tweetsSinceID)
				if err != nil {
					//	Notice the error
					txn.NoticeError(err)
					return
				}

				//	Pass the latest tweets to a channel
				for _, tweet := range tweetsResponse {
					bnp.ProcessTweet <- tweet
				}

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

// ProcessTweets manages processing & storing news items
func (bnp BackgroundNewsProcessor) ProcessTweets(ctx context.Context) {

	zlog.Infof("ProcessTweets starting... ")

	for {
		select {
		case tweetRequest := <-bnp.ProcessTweet:
			//	As we get a request on a channel to fire a trigger...
			//	Create a goroutine
			go func(cx context.Context) {
				//	Start a background transaction
				txn := telemetry.NRApp.StartTransaction("ProcessTweets")
				defer txn.End()

				//	Add it to the context
				cx = newrelic.NewContext(cx, txn)

				zlog.Infof("Process tweet (ID): (%v) - %v", tweetRequest.ID, tweetRequest.Text)

				//	This worker should
				//	- See if we have a record of the tweet already, first
				//	- If we don't have a record...
				//	-- Fetch the url for the story (capture the long url, minus the querystring)
				//	-- Process the main image for the story
				//	-- Save the new story and update information

			}(ctx) // Launch the goroutine
		case <-ctx.Done():
			zlog.Infof("ProcessTweets stopping")
			return
		}
	}

}

func init() {
	rootlogger, _ = logger.NewProd()
	defer rootlogger.Sync() // flushes buffer, if any
	zlog = rootlogger.Sugar()
}
