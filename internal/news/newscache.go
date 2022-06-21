package news

import (
	"context"
	"fmt"
	"sort"

	"github.com/danesparza/daydash-service/internal/telemetry"
	"github.com/newrelic/go-agent/v3/integrations/nrmongo"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	collection *mongo.Collection
)

// StoredNewsItem represents a single news document in the news collection
type StoredNewsItem struct {
	ID struct {
		Oid string `json:"$oid"` // Mongo document ID
	} `json:"_id"`
	URL      string `json:"url"`       // Full url without the querystring
	ShortURL string `json:"short_url"` // Shortned url (useful for QR code generation)
	Updates  []struct {
		ID        string `json:"id"`        // Twitter id of the news update
		Text      string `json:"text"`      // Text of the twitter update
		Time      int    `json:"time"`      // Time of the update
		Mediaurl  string `json:"mediaurl"`  // Url of the image used
		Mediadata string `json:"mediadata"` // Base64 encoded resized and centered image
	} `json:"updates"`
	Entities []struct {
		Type string `json:"type"` // Entity type (Person, Organization, Place)
		Text string `json:"text"` // Entity text
	} `json:"entities"`
}

// UpdateStoredNewsItems gets the latest news feed, matches on existing news story urls,
// and adds any news story updates that don't already exist
func UpdateStoredNewsItems() error {
	return nil
}

//  TweetIDIsStored returns 'true' if tweet is stored in the cache, false if it's not
func TweetIDIsStored(id string) bool {
	return false
}

// GetMostRecentTweetID returns the most recent tweetid (or blank, if there is a problem finding it)
func GetMostRecentTweetID(ctx context.Context) string {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News GetMostRecentTweetID")
	defer segment.End()

	retval := ""
	storedItemResponse := StoredNewsItem{}

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return retval
	}

	collection = client.Database("dashboard").Collection("news")

	//	First, find the item that has the most recent update(s)
	filter := bson.D{{}}
	opts := options.FindOne().SetSort(bson.D{{"updates.id", -1}})
	item := collection.FindOne(ctx, filter, opts)

	err = item.Decode(&storedItemResponse)
	if err != nil {
		zlog.Errorw("problem decoding news item",
			"error", err,
		)
		return retval
	}

	//	Then, find the most recent update and return that tweet id
	tweetIDs := []string{}
	for _, update := range storedItemResponse.Updates {
		tweetIDs = append(tweetIDs, update.ID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(tweetIDs)))

	retval = tweetIDs[0]
	return retval
}

// StoreNewsStoryUpdates takes a twitter feed and stores updates if the story/update doesn't already exist
func StoreNewsStoryUpdates(ctx context.Context, tweets []Tweet) error {

	//	For each passed tweet

	//	See if

	return nil
}

// GetAllStoredNewsItems gets all stored news items in MongoDB.
func GetAllStoredNewsItems(ctx context.Context) ([]StoredNewsItem, error) {
	newsItems := []StoredNewsItem{}

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("problem connecting to MongoDB: %v", err)
	}

	collection = client.Database("dashboard").Collection("news")

	txn := telemetry.NRApp.StartTransaction("GetAllStoredNewsItems")
	defer txn.End()
	ctx = newrelic.NewContext(ctx, txn)

	//	First, find all news items
	filter := bson.D{{}}
	opts := options.Find().SetSort(bson.D{{"updates.id", -1}})
	cur, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("problem finding items in dashboard.news: %v", err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var n StoredNewsItem
		err := cur.Decode(&n)
		if err != nil {
			return nil, fmt.Errorf("problem decoding news item: %v", err)
		}

		newsItems = append(newsItems, n)
	}

	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("problem navigating through the list of items: %v", err)
	}

	if len(newsItems) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return newsItems, nil
}
