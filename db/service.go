package db

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"twitter_oracle/common"
)

type DBService struct {
	DB *sql.DB
}

func Init(password string, url string) (*DBService, error) {
	dsn := "root:" + password + "@tcp(" + url + ")/" + "hug"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &DBService{
		DB: db,
	}, nil
}

func (db *DBService) GetConversationList() ([]string, error) {
	//todo:impl
	return nil, nil
}

func (db *DBService) GetEventTweetIdList() ([]string, error) {
	return nil, nil
}

func (db *DBService) PutEventPublicMetric(tweetId string, metric common.TweetPublicMetricInfo) error {
	return nil
}

func (db *DBService) PutQuotes(tweetId string, quoteList []common.QuoteInfo) error {
	return nil
}

func (db *DBService) GetLastQuoteId(tweetId string) (string, error) {
	return "", nil
}
