package db

import (
	"os"
	"testing"
	"twitter_oracle/common"
)

func TestPutThought(t *testing.T) {
	common.BeaverToken = os.Getenv("TW_BEAVER")
	db, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	author := "ninox2022"
	text := "this is a good day @ninox2022 #thought"
	conversationId := "1587629551169204224"
	tips := "test tips"
	err = db.PutThought(author, text, conversationId, tips)
	if err != nil {
		t.Fatal(err)
	}
}
