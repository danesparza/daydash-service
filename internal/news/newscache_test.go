package news_test

import (
	"context"
	"log"
	"testing"

	"github.com/danesparza/daydash-service/internal/news"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

func init() {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("Couldn't find home directory - %v", err)
	}

	viper.AddConfigPath(home)              // adding home directory as first search path
	viper.AddConfigPath(".")               // also look in the working directory
	viper.SetConfigName("daydash-service") // name the config file (without extension)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	viper.ReadInConfig()
}

func TestNewsCache_GetAllNewsItems_DoesNotReturnError(t *testing.T) {
	ctx := context.Background()

	newsItems, err := news.GetAllStoredNewsItems(ctx)
	if err != nil {
		t.Errorf("Returned error and we didn't expect that: %v", err)
	}

	//	Spit out what we know:
	for _, item := range newsItems {
		t.Logf("News story: %v\n", item.Updates[0].Text)
	}
}

func TestNewsCache_GetMostRecentTweetID_GetsMostRecentTweet(t *testing.T) {
	ctx := context.TODO()

	tweetID := news.GetMostRecentTweetID(ctx)
	if tweetID == "" {
		t.Errorf("Returned an empty tweetid and we didn't expect that")
	}

	t.Logf("Most recent tweetID: %v\n", tweetID)
}
