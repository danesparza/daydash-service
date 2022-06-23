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

// NewsFetchTask manages fetching & storing news items as they appear
// in the twitter news feed
func NewsFetchTask(ctx context.Context) {

	zlog.Infof("NewsFetchTask starting... ")

	for {
		select {
		case <-time.After(10 * time.Minute):
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

				//	Process the tweets that we found based on the story
				//	url that is associated with the tweet
				for _, tweet := range tweetsResponse {

					//	See if we have a record of the update already (based on the tweetid)
					//	If we find one, discard the tweet.  We don't need it.  Go to the next one.
					if story := GetStoryForUpdateID(cx, tweet.ID); story.URL != "" {
						continue
					}

					//	If we can't find an update, move to plan B:
					//	For each tweet
					//	- Get the full url for the short url
					//	- Get the image and smart crop it / base64 encode the data
					metaInfo, _ := GetStoryAndImageUrl(cx, tweet.Entities.Urls[0].URL)

					//	See if we have a matching story
					//	-- See if that url exists already as a story
					//  --> metaInfo.LongUrl
					currentStory := GetStoryForUrl(cx, metaInfo.LongUrl)

					//	If we don't have the story, create one
					if currentStory.URL == "" {
						//	Just use the current story and update all details
						currentStory = NewsStory{
							URL:      metaInfo.LongUrl,
							ShortURL: metaInfo.ShortUrl,
							Updates: []NewsStoryUpdate{{
								ID:        tweet.ID,
								Text:      tweet.Text, // Need to remove the url from the text
								Time:      int(tweet.CreatedAt.UTC().Unix()),
								Mediaurl:  metaInfo.ImageUrl,
								Mediadata: metaInfo.ImageData,
							}},
						}

						//	Add the story
						zlog.Infof("Adding story for tweetID: %v", tweet.ID)
						AddNewsStory(cx, currentStory)
					} else {
						//	If we already have the story, add an update
						currentStory.Updates = append(currentStory.Updates, NewsStoryUpdate{
							ID:        tweet.ID,
							Text:      tweet.Text,
							Time:      int(tweet.CreatedAt.UTC().Unix()),
							Mediaurl:  metaInfo.ImageUrl,
							Mediadata: metaInfo.ImageData,
						})

						//	Update the story
						zlog.Infof("Updating story with tweetID: %v", tweet.ID)
						UpdateNewsStory(cx, currentStory)
					}
				}

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
