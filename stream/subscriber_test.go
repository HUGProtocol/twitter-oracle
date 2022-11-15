package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/g8rswimmer/go-twitter/v2"
	"os"
	"testing"
	"time"
	"twitter_oracle/db"
)

var tokenStr = os.Getenv("TW_BEAVER")

func TestGetRule(t *testing.T) {
	sub := Subscriber{
		conversationHandler: make(map[string]Handler),
		BeaverToken:         tokenStr,
		client:              nil,
		stream:              nil,
	}
	sub.newClient()
	rules, err := sub.GetRules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(rules)
}

func TestDeleteRule(t *testing.T) {
	sub := Subscriber{
		conversationHandler: make(map[string]Handler),
		BeaverToken:         tokenStr,
		client:              nil,
		stream:              nil,
	}
	sub.newClient()
	err := sub.DeleteRules(context.Background(), []string{"1545290390944722945", "1546429906975727617"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAddRule(t *testing.T) {
	sub := Subscriber{
		conversationHandler: make(map[string]Handler),
		BeaverToken:         tokenStr,
		client:              nil,
		stream:              nil,
	}
	sub.newClient()
	rule := "#HugReply @metauce"
	tag := "replies"
	rules, err := sub.AddRule(context.Background(), rule, tag)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(rules)
}

func DummyHandler(db *db.DBService, id string, conversation string, authorId string, authorName string, createTime time.Time, text string) error {
	fmt.Printf("id:%v\nconversation:%v\nauthor id:%v\nauthor name:%v\ncreate time:%v\ntext:%v\n",
		id, conversation, authorId, authorName, createTime.String(), text)
	return nil
}

func TestStartStream(t *testing.T) {
	sub := Subscriber{
		conversationHandler: make(map[string]Handler),
		BeaverToken:         tokenStr,
		client:              nil,
		stream:              nil,
		defaultHandler:      DummyHandler,
	}
	sub.newClient()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	err := sub.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetTweetConversation(t *testing.T) {
	sub := Subscriber{
		conversationHandler: make(map[string]Handler),
		BeaverToken:         tokenStr,
		client:              nil,
		stream:              nil,
		defaultHandler:      DummyHandler,
	}
	sub.newClient()
	opts := twitter.TweetLookupOpts{
		Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
		TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID},
	}
	ids := []string{"1592228299337760768"}
	tweetResponse, err := sub.client.TweetLookup(context.Background(), ids, opts)
	if err != nil {
		panic(err)
	}
	dictionaries := tweetResponse.Raw.TweetDictionaries()

	enc, err := json.MarshalIndent(dictionaries, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(enc))

	tweet := tweetResponse.Raw.Tweets[0]
	DummyHandler(nil, tweet.ID, tweet.ConversationID, tweet.AuthorID, "mask", time.Now(), tweet.Text)
}
