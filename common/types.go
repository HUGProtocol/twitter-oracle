package common

import "time"

type EventTweetInfo struct {
	TweetId    string    `json:"tweet_id"`
	AuthorId   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	EventName  string    `json:"event_name"`
}

type QuoteInfo struct {
	TweetId     string                `json:"tweet_id"`
	AuthorId    string                `json:"author_id"`
	AuthorName  string                `json:"author_name"`
	Text        string                `json:"text"`
	CreatedAt   time.Time             `json:"created_at"`
	PublicMetic TweetPublicMetricInfo `json:"public_metic"`
}

type TweetPublicMetricInfo struct {
	RetweetCount int `json:"retweet_count"`
	ReplyCount   int `json:"reply_count"`
	LikeCount    int `json:"like_count"`
	QuoteCount   int `json:"quote_count"`
}

type ReplyInfo struct {
	AuthorId   string    `json:"author_id"`
	AuthorName string    `json:"author_name"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}
