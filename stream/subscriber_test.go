package stream

import (
	"context"
	"fmt"
	"testing"
	"time"
	"twitter_oracle/db"
)

var tokenStr = "AAAAAAAAAAAAAAAAAAAAAK%2FqeQEAAAAAx0whFAoIkeSsxfoXKdTN0SQn93w%3DI0snjdgGv0VEwbn5q3xLSiY7MAuU13jbxo3Q0RV7pbrtnv0sfB"

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

func DummyHandler(db *db.DBService, conversation string, authorId string, authorName string, createTime time.Time, text string) error {
	fmt.Printf("conversation:%v\nauthor id:%v\nauthor name:%v\ncreate time:%v\ntext:%v\n",
		conversation, authorId, authorName, createTime.String(), text)
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
