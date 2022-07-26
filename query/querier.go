package query

import (
	"context"
	"fmt"
	twitter "github.com/g8rswimmer/go-twitter/v2"
	"net/http"
	"time"
	"twitter_oracle/common"
	"twitter_oracle/db"
	"twitter_oracle/log"
)

var QueryContextTimeout = time.Minute * 5

type Querier struct {
	BeaverToken            string
	HUGTwitterName         string
	AddEventTwitterHashtag string
	client                 *twitter.Client
	PollDur                time.Duration
	db                     *db.DBService
}

func Init(db *db.DBService, duration time.Duration) *Querier {
	q := Querier{
		BeaverToken:            common.BeaverToken,
		HUGTwitterName:         common.HugTwitterName,
		AddEventTwitterHashtag: common.AddEventTwitterHashtag,
		PollDur:                duration,
		db:                     db,
	}
	q.newClient()
	return &q
}

func (q *Querier) newClient() {
	q.client = &twitter.Client{
		Authorizer: authorize{
			Token: q.BeaverToken,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}
}

func (q *Querier) Start() {
	ticker := time.NewTicker(q.PollDur)
	start := make(chan struct{})
	start <- struct{}{}
	job := func() error {
		tweetIdList, err := q.db.GetEventTweetIdList()
		if err != nil {
			return err
		}
		for _, tweetId := range tweetIdList {
			//get public metric
			q.updatePublicMetric(tweetId)
			//poll quotes
			q.pollTweetQuotes(tweetId)
		}
		return nil
	}
	for {
		select {
		case <-start:
			err := job()
			if err != nil {
				panic(err)
			}
		case <-ticker.C:
			err := job()
			log.Error(err)
		}
	}
}

func (q *Querier) updatePublicMetric(tweetId string) {
	metricCtx, metricCancel := context.WithTimeout(context.Background(), QueryContextTimeout)
	defer metricCancel()
	metric, e := q.GetTweetPublicMetric(metricCtx, tweetId)
	if e != nil {
		log.Warn("GetTweetPublicMetric error", e, "tweetId", tweetId)
		return
	}
	metricCancel()
	e = q.db.PutEventPublicMetric(tweetId, metric)
	if e != nil {
		log.Warn("PutEventPublicMetric error", e, "tweetId", tweetId)
		return
	}
}

func (q *Querier) pollTweetQuotes(tweetId string) {
	lastQuoteId, e := q.db.GetLastQuoteId(tweetId)
	if e != nil {
		log.Warn("GetLastQuoteId error", e, "tweetId", tweetId)
		return
	}
	quoteCtx, quoteCancel := context.WithTimeout(context.Background(), QueryContextTimeout)
	defer quoteCancel()
	quotes, e := q.PollQuotes(quoteCtx, lastQuoteId, tweetId)
	if e != nil {
		log.Warn("PollQuotes error", e, "tweetId", tweetId, "last quote id", lastQuoteId)
		return
	}
	e = q.db.PutQuotes(tweetId, quotes)
	if e != nil {
		log.Warn("PutQuotes error", e, "tweetId", tweetId)
	}
}

func (q *Querier) GetEventTwitterId(ctx context.Context, eventName string, from string) (common.EventTweetInfo, error) {
	opts := twitter.TweetRecentSearchOpts{
		Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
		TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID},
	}
	query := fmt.Sprintf("#%v #%v from:%v @%v", eventName, q.AddEventTwitterHashtag, from, q.HUGTwitterName)
	tweetResponse, err := q.client.TweetRecentSearch(ctx, query, opts)
	if err != nil {
		return common.EventTweetInfo{}, err
	}

	dictionaries := tweetResponse.Raw.TweetDictionaries()
	latestTime, _ := time.Parse(time.RFC3339, "2022-07-11T05:37:44+00:00")
	info := common.EventTweetInfo{TweetId: "0"}
	for _, dic := range dictionaries {
		createTime, e := time.Parse(time.RFC3339, dic.Tweet.CreatedAt)
		if e != nil {
			continue
		}
		if latestTime.After(createTime) {
			continue
		}
		latestTime = createTime

		info = common.EventTweetInfo{
			TweetId:    dic.Tweet.ID,
			AuthorId:   dic.Author.ID,
			AuthorName: dic.Author.Name,
			Text:       dic.Tweet.Text,
			CreatedAt:  createTime,
			EventName:  eventName,
		}
	}
	if info.TweetId == "0" {
		return info, EventTweetNotFoundError
	}
	return info, nil
}

func (q *Querier) GetTweetPublicMetric(ctx context.Context, tweetId string) (common.TweetPublicMetricInfo, error) {
	opts := twitter.TweetLookupOpts{
		TweetFields: []twitter.TweetField{twitter.TweetFieldPublicMetrics},
	}
	tweetResponse, err := q.client.TweetLookup(ctx, []string{tweetId}, opts)
	if err != nil {
		return common.TweetPublicMetricInfo{}, err
	}

	dictionaries := tweetResponse.Raw.TweetDictionaries()
	info := common.TweetPublicMetricInfo{}

	for _, dic := range dictionaries {
		info = common.TweetPublicMetricInfo{
			RetweetCount: dic.Tweet.PublicMetrics.Retweets,
			ReplyCount:   dic.Tweet.PublicMetrics.Replies,
			LikeCount:    dic.Tweet.PublicMetrics.Likes,
			QuoteCount:   dic.Tweet.PublicMetrics.Quotes,
		}
		break
	}
	return info, nil
}

func (q *Querier) PollQuotes(ctx context.Context, lastQuoteId string, tweetId string) ([]common.QuoteInfo, error) {
	infoList := make([]common.QuoteInfo, 0)
	opts := twitter.QuoteTweetsLookupOpts{
		MaxResults:  10,
		Expansions:  []twitter.Expansion{twitter.ExpansionAuthorID},
		TweetFields: []twitter.TweetField{twitter.TweetFieldCreatedAt, twitter.TweetFieldConversationID, twitter.TweetFieldPublicMetrics},
	}

	for {
		tweetResponse, err := q.client.QuoteTweetsLookup(ctx, tweetId, opts)
		if err != nil {
			return infoList, err
		}
		dictionaries := tweetResponse.Raw.TweetDictionaries()

		for id, dic := range dictionaries {
			if id == lastQuoteId {
				return infoList, nil
			}
			createTime, e := time.Parse(time.RFC3339, dic.Tweet.CreatedAt)
			if e != nil {
				createTime = time.Now()
			}
			publicMetric := common.TweetPublicMetricInfo{
				RetweetCount: dic.Tweet.PublicMetrics.Retweets,
				ReplyCount:   dic.Tweet.PublicMetrics.Replies,
				LikeCount:    dic.Tweet.PublicMetrics.Likes,
				QuoteCount:   dic.Tweet.PublicMetrics.Quotes,
			}
			info := common.QuoteInfo{
				TweetId:     dic.Tweet.ID,
				AuthorId:    dic.Author.ID,
				AuthorName:  dic.Author.Name,
				Text:        dic.Tweet.Text,
				CreatedAt:   createTime,
				PublicMetic: publicMetric,
			}
			infoList = append(infoList, info)
		}
		if tweetResponse.Meta.NextToken == "" {
			return infoList, nil
		}
		opts.PaginationToken = tweetResponse.Meta.NextToken
	}
}
