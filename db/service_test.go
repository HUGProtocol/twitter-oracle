package db

import "testing"

func TestPutThought(t *testing.T) {
	db, err := Init()
	if err != nil {
		t.Fatal(err)
	}
	author := "ninox2022"
	text := "this is a good day @ninox2022 #thought"
	conversationId := "1587629551169204224"
	err = db.PutThought(author, text, conversationId)
	if err != nil {
		t.Fatal(err)
	}
}
