package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/g8rswimmer/go-twitter/v2"
	"net/http"
	"time"
	"twitter_oracle/common"
	"twitter_oracle/db"
)

type Handler func(db *db.DBService, conversation string, authorId string, authorName string, createTime time.Time, text string) error

//By Default add text as reply to conversation in db
func DefaultHandler(db *db.DBService, conversation string, authorId string, authorName string, createTime time.Time, text string) error {
	//todo:add as reply in db
	return nil
}

type Subscriber struct {
	conversationHandler map[string]Handler
	BeaverToken         string
	client              *twitter.Client
	stream              *twitter.TweetStream
	db                  *db.DBService
}

func Init(db *db.DBService) (*Subscriber, error) {
	s := Subscriber{
		conversationHandler: make(map[string]Handler),
		BeaverToken:         common.BeaverToken,
		db:                  db,
	}
	convList, err := db.GetConversationList()
	if err != nil {
		return nil, err
	}
	for _, conv := range convList {
		s.conversationHandler[conv] = DefaultHandler
	}
	s.newClient()
	return &s, nil
}

func (s *Subscriber) newClient() {
	s.client = &twitter.Client{
		Authorizer: authorize{
			Token: s.BeaverToken,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}
}

func (s *Subscriber) AddConversation(conversationFilter string, handler Handler) {
	s.conversationHandler[conversationFilter] = handler
}

func (s *Subscriber) RemoveConversation(conversationFilter string) {
	if _, ok := s.conversationHandler[conversationFilter]; ok {
		delete(s.conversationHandler, conversationFilter)
	}
}

func (s *Subscriber) UpdateConversationHandler(conversationFilter string, handler Handler) {
	s.conversationHandler[conversationFilter] = handler
}

func (s *Subscriber) GetRules(ctx context.Context) (string, error) {
	searchStreamRules, err := s.client.TweetSearchStreamRules(ctx, []twitter.TweetSearchStreamRuleID{})
	if err != nil {
		return "", err
	}
	enc, err := json.MarshalIndent(searchStreamRules, "", "    ")
	if err != nil {
		return "", err
	}
	return string(enc), nil
}

func (s *Subscriber) AddRule(ctx context.Context, rule string, tag string) (string, error) {
	streamRule := twitter.TweetSearchStreamRule{
		Value: rule,
		Tag:   tag,
	}
	searchStreamRules, err := s.client.TweetSearchStreamAddRule(ctx, []twitter.TweetSearchStreamRule{streamRule}, false)
	if err != nil {
		return "", err
	}
	if len(searchStreamRules.Rules) == 0 {
		return "", errors.New("no rules in searchStreamRules.Rules")
	}
	return string(searchStreamRules.Rules[0].ID), nil
}

func (s *Subscriber) DeleteRules(ctx context.Context, ids []string) error {
	var ruleIDs []twitter.TweetSearchStreamRuleID
	for _, id := range ids {
		ruleIDs = append(ruleIDs, twitter.TweetSearchStreamRuleID(id))
	}
	_, err := s.client.TweetSearchStreamDeleteRuleByID(ctx, ruleIDs, true)
	return err
}

func (s *Subscriber) reconnect(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute * 2)
	timeout := time.NewTimer(time.Minute * 30)
	var err error
	for {
		select {
		case <-ticker.C:
			opts := twitter.TweetSearchStreamOpts{
				Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
				TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID},
			}
			s.stream, err = s.client.TweetSearchStream(ctx, opts)
			if err != nil {
				fmt.Printf("tweet sample callout error: %v\n", err)
				continue
			} else {
				return nil
			}
		case <-timeout.C:
			return errors.New("reconnect timeout")
		}
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	opts := twitter.TweetSearchStreamOpts{
		Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
		TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID},
	}
	var err error
	s.stream, err = s.client.TweetSearchStream(ctx, opts)
	if err != nil {
		return err
	}

	defer s.stream.Close()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case tm := <-s.stream.Tweets():
			tmb, err := json.Marshal(tm)
			if err != nil {
				fmt.Printf("error decoding tweet message %v", err)
			}
			fmt.Printf("tweet: %s\n\n", string(tmb))

			e := s.handleTweetMessage(tm)
			if e != nil {
				//todo:handle tweet message handle error
				fmt.Println(e)
			}

		case sm := <-s.stream.SystemMessages():
			smb, err := json.Marshal(sm)
			if err != nil {
				fmt.Printf("error decoding system message %v", err)
			}
			fmt.Printf("system: %s\n\n", string(smb))
		case strErr := <-s.stream.Err():
			fmt.Printf("error: %v\n\n", strErr)
		case <-ticker.C:
			if s.stream.Connection() == false {
				fmt.Printf("connection lost %v\n", time.Now())
				reconnectErr := s.reconnect(ctx)
				if reconnectErr != nil {
					return reconnectErr
				}
			}
		}
	}
}

func (s *Subscriber) handleTweetMessage(tweetMsg *twitter.TweetMessage) error {
	if tweetMsg == nil || tweetMsg.Raw == nil || tweetMsg.Raw.Tweets == nil || tweetMsg.Raw.Includes == nil || tweetMsg.Raw.Includes.Users == nil {
		return errors.New("tweet message response miss content")
	}
	userIdNameMap := make(map[string]string)
	for _, user := range tweetMsg.Raw.Includes.Users {
		if user == nil {
			continue
		}
		userIdNameMap[user.ID] = user.UserName
	}
	for _, tweet := range tweetMsg.Raw.Tweets {
		if tweet == nil {
			continue
		}
		for conversation, handler := range s.conversationHandler {
			if tweet.ConversationID == conversation {
				authorId := tweet.AuthorID
				authorName, ok := userIdNameMap[authorId]
				if !ok {
					//todo:handle stream message without author name
					fmt.Println("failed to get author name, author id:", authorId)
					continue
				}
				createTime, err := time.Parse(time.RFC3339, tweet.CreatedAt)
				if err != nil {
					createTime = time.Now()
				}
				e := handler(s.db, conversation, authorId, authorName, createTime, tweet.Text)
				if e != nil {
					//todo:handle handler error
					continue
				}
			}
		}
	}
	return nil
}
