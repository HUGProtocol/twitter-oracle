package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/g8rswimmer/go-twitter/v2"
	"net/http"
	"strings"
	"time"
	"twitter_oracle/common"
	"twitter_oracle/db"
)

var MAX_TIPS_LEN = 50

var EventFilter = "@ninox2022 #thought"

type Handler func(db *db.DBService, id string, conversation string, authorId string, authorName string, createTime time.Time, text string) error

//By Default add text as reply to conversation in db
func DefaultHandler(db *db.DBService, id string, conversation string, authorId string, authorName string, createTime time.Time, text string) error {
	//todo:add as reply in db
	return nil
}

type Subscriber struct {
	conversationHandler map[string]Handler
	BeaverToken         string
	client              *twitter.Client
	stream              *twitter.TweetStream
	db                  *db.DBService
	defaultHandler      Handler
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

func (s *Subscriber) AddDefaultHanler(handler Handler) {
	s.defaultHandler = handler
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

func (s *Subscriber) GetTweetById(ctx context.Context, id string) (*twitter.TweetRaw, error) {
	opts := twitter.TweetLookupOpts{
		Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
		TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID},
	}
	ids := []string{id}
	tweetResponse, err := s.client.TweetLookup(context.Background(), ids, opts)
	if err != nil {
		return nil, err
	}
	if tweetResponse.Raw == nil {
		return nil, errors.New("response tweet raw nil")
	}
	return tweetResponse.Raw, nil
}

func (s *Subscriber) reconnect(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute * 5)
	timeout := time.NewTimer(time.Minute * 60)
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
				fmt.Println("reconnect success")
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
	fmt.Println("start streaming")
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
				s.stream.Close()
				reconnectErr := s.reconnect(ctx)
				if reconnectErr != nil {
					return reconnectErr
				}
				fmt.Println("reconnected", time.Now())
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
		//handle certain conversation
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
				e := handler(s.db, tweet.ID, conversation, authorId, authorName, createTime, tweet.Text)
				if e != nil {
					//todo:handle handler error
					continue
				}
			}
		}

		if s.defaultHandler != nil {
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
			e := s.defaultHandler(s.db, tweet.ID, tweet.ConversationID, authorId, authorName, createTime, tweet.Text)
			if e != nil {
				//todo:handle handler error
				fmt.Println("default handle error", e, tweet)
				continue
			}
		}
	}
	return nil
}

func (s *Subscriber) LoadThoughtHandler(db *db.DBService, id string, conversation string, authorId string, authorName string, createTime time.Time, text string) error {
	fmt.Println("load thought", authorName, createTime)
	sourceUrl := ""
	tips := text
	if len(text) > MAX_TIPS_LEN {
		sepList := strings.Split(text, " ")
		summary := ""
		for _, sep := range sepList {
			summary = summary + sep
			if len(summary) > MAX_TIPS_LEN {
				summary = summary + " ..."
				break
			}
		}
	}
	if id == conversation {
		sourceUrl = fmt.Sprintf("https://twitter.com/%s/status/%s", authorName, conversation)
	} else {
		raw, err := s.GetTweetById(context.Background(), conversation)
		if err != nil {
			return err
		}
		userIdNameMap := make(map[string]string)
		for _, user := range raw.Includes.Users {
			if user == nil {
				continue
			}
			userIdNameMap[user.ID] = user.UserName
		}
		if len(raw.Tweets) == 0 {
			return errors.New("conversation tweet not found" + conversation)
		}
		conversationAuthor := userIdNameMap[raw.Tweets[0].AuthorID]
		sourceUrl = fmt.Sprintf("https://twitter.com/%s/status/%s", conversationAuthor, conversation)
	}
	return db.PutThought(authorName, text, sourceUrl, tips)
}

func (q *Subscriber) GetEventTwitterId(ctx context.Context, sinceId string) (*twitter.TweetRaw, error) {
	opts := twitter.TweetRecentSearchOpts{
		Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
		TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID},
		SinceID:     sinceId,
	}
	query := fmt.Sprintf(EventFilter)
	tweetResponse, err := q.client.TweetRecentSearch(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	if tweetResponse.Raw == nil {
		return nil, errors.New("response tweet raw nil")
	}
	return tweetResponse.Raw, nil
	//dictionaries := tweetResponse.Raw.TweetDictionaries()
	//latestTime, _ := time.Parse(time.RFC3339, "2022-07-11T05:37:44+00:00")
	//info := common.EventTweetInfo{TweetId: "0"}
	//for _, dic := range dictionaries {
	//	createTime, e := time.Parse(time.RFC3339, dic.Tweet.CreatedAt)
	//	if e != nil {
	//		continue
	//	}
	//	if latestTime.After(createTime) {
	//		continue
	//	}
	//	latestTime = createTime
	//
	//	info = common.EventTweetInfo{
	//		TweetId:    dic.Tweet.ID,
	//		AuthorId:   dic.Author.ID,
	//		AuthorName: dic.Author.Name,
	//		Text:       dic.Tweet.Text,
	//		CreatedAt:  createTime,
	//		EventName:  eventName,
	//	}
	//}
	//if info.TweetId == "0" {
	//	return info, EventTweetNotFoundError
	//}
	//return info, nil
}
