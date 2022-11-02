package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"time"
	"twitter_oracle/common"
)

type DBService struct {
	pool *pgxpool.Pool
}

func Init() (*DBService, error) {

	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}
	fmt.Println(os.Getenv("DATABASE_URL"))
	return &DBService{
		pool: dbpool,
	}, nil
}

func (db *DBService) PutThought(author, text, conversationId string) error {
	//get user
	getUserSql := "select address from users where twitter=$1"
	address := ""
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := db.pool.QueryRow(ctx, getUserSql, author).Scan(&address)
	if err != nil {
		return err
	}
	//insert thought
	putThoughtSql := "insert into thoughts(content, address, source_url, submit_state) values ($1, $2, $3, $4)"
	//https://twitter.com/ninox2022/status/1587630498012332032
	source_url := fmt.Sprintf("https://twitter.com/%s/status/%s", author, conversationId)
	_, err = db.pool.Exec(ctx, putThoughtSql, text, address, source_url, "save")
	return err
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
