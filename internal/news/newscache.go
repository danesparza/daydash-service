package news

import (
	"context"
	"fmt"
	"sort"

	"github.com/newrelic/go-agent/v3/integrations/nrmongo"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	collection *mongo.Collection
)

// NewsStory represents a single news document in the news collection
type NewsStory struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"` // MongoDB ID
	URL      string             `bson:"url"`           // Full url without the querystring
	ShortURL string             `bson:"short_url"`     // Shortned url (useful for QR code generation)
	Updates  []NewsStoryUpdate  `bson:"updates"`
	Entities []NewsStoryEntity  `bson:"entities"`
}

// NewsStoryUpdate represents a news story update for a news item
type NewsStoryUpdate struct {
	ID        string `bson:"id"`        // Twitter id of the news update
	Text      string `bson:"text"`      // Text of the twitter update
	Time      int    `bson:"time"`      // Time of the update
	Mediaurl  string `bson:"mediaurl"`  // Url of the image used
	Mediadata string `bson:"mediadata"` // Base64 encoded resized and centered image
}

type NewsStoryEntity struct {
	Type string `bson:"type"` // Entity type (Person, Organization, Place)
	Text string `bson:"text"` // Entity text
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
	storedItemResponse := NewsStory{}

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return retval
	}
	defer client.Disconnect(ctx)

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

// GetStoryForUpdateID returns the story associated with a given udpate id (tweetid)
func GetStoryForUpdateID(ctx context.Context, updateID string) NewsStory {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News GetStoryForTweetID")
	defer segment.End()

	retval := NewsStory{}

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return retval
	}
	defer client.Disconnect(ctx)

	collection = client.Database("dashboard").Collection("news")

	//	First, find the item that has the updateID
	filter := bson.D{{"updates.id", updateID}}
	item := collection.FindOne(ctx, filter)

	err = item.Decode(&retval)
	if err != nil {
		//	If we didn't get anything back, that's OK in this case.  Just get out
		if err == mongo.ErrNoDocuments {
			return retval
		}

		zlog.Errorw("problem decoding news item",
			"error", err,
		)
	}

	return retval
}

// GetStoryForUrl returns the story for a given story url
func GetStoryForUrl(ctx context.Context, storyUrl string) NewsStory {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News GetStoryForUrl")
	defer segment.End()

	retval := NewsStory{}

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return retval
	}
	defer client.Disconnect(ctx)

	collection = client.Database("dashboard").Collection("news")

	//	First, find the item that has the updateID
	filter := bson.D{{"url", storyUrl}}
	item := collection.FindOne(ctx, filter)

	err = item.Decode(&retval)
	if err != nil {
		//	If we didn't get anything back, that's OK in this case.  Just get out
		if err == mongo.ErrNoDocuments {
			return retval
		}

		//	Log everything else
		zlog.Errorw("problem decoding news item",
			"error", err,
		)
	}

	return retval
}

// AddNewsStory updates a story
func AddNewsStory(ctx context.Context, story NewsStory) error {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News AddNewsStory")
	defer segment.End()

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return err
	}
	defer client.Disconnect(ctx)

	collection = client.Database("dashboard").Collection("news")

	_, err = collection.InsertOne(context.TODO(), story)
	if err != nil {
		zlog.Errorw("problem inserting to MongoDB",
			"error", err,
		)
		return err
	}

	return nil
}

// UpdateNewsStory updates a story
func UpdateNewsStory(ctx context.Context, story NewsStory) error {

	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News UpdateNewsStory")
	defer segment.End()

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return err
	}
	defer client.Disconnect(ctx)

	collection = client.Database("dashboard").Collection("news")

	update := bson.M{
		"$set": story,
	}

	filter := bson.D{{"_id", story.ID}}
	_, err = collection.UpdateOne(context.TODO(), filter, update, nil)
	if err != nil {
		zlog.Errorw("problem updating MongoDB",
			"error", err,
		)
		return err
	}

	return nil
}

// GetRecentNewsStories gets recent stored news items in MongoDB.
func GetRecentNewsStories(ctx context.Context, numberOfStories int64) ([]NewsStory, error) {
	txn := newrelic.FromContext(ctx)
	segment := txn.StartSegment("News GetRecentNewsStories")
	defer segment.End()

	retval := []NewsStory{}

	//	Create a client & connect
	clientOptions := options.Client().ApplyURI(viper.GetString("news.mongodb")).SetMonitor(nrmongo.NewCommandMonitor(nil))
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		zlog.Errorw("problem connecting to MongoDB",
			"error", err,
		)
		return retval, err
	}
	defer client.Disconnect(ctx)

	collection = client.Database("dashboard").Collection("news")

	//	First, find all news items
	filter := bson.D{{}}
	opts := options.Find().SetSort(bson.D{{"updates.id", -1}}).SetLimit(numberOfStories)
	cur, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("problem finding items in dashboard.news: %v", err)
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var n NewsStory
		err := cur.Decode(&n)
		if err != nil {
			return nil, fmt.Errorf("problem decoding news item: %v", err)
		}

		retval = append(retval, n)
	}

	if err := cur.Err(); err != nil {
		return nil, fmt.Errorf("problem navigating through the list of items: %v", err)
	}

	if len(retval) == 0 {
		return nil, mongo.ErrNoDocuments
	}

	return retval, nil
}
