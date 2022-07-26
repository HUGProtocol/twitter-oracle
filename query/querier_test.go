package query

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestQuerier_GetEventTwitterId(t *testing.T) {
	tokenStr := "AAAAAAAAAAAAAAAAAAAAAK%2FqeQEAAAAAx0whFAoIkeSsxfoXKdTN0SQn93w%3DI0snjdgGv0VEwbn5q3xLSiY7MAuU13jbxo3Q0RV7pbrtnv0sfB"
	querier := Querier{
		BeaverToken:            tokenStr,
		HUGTwitterName:         "HUG",
		AddEventTwitterHashtag: "NEWEVENT",
	}
	querier.newClient()
	_, err := querier.GetEventTwitterId(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPublicMetric(t *testing.T) {
	tokenStr := "AAAAAAAAAAAAAAAAAAAAAK%2FqeQEAAAAAx0whFAoIkeSsxfoXKdTN0SQn93w%3DI0snjdgGv0VEwbn5q3xLSiY7MAuU13jbxo3Q0RV7pbrtnv0sfB"
	querier := Querier{
		BeaverToken:            tokenStr,
		HUGTwitterName:         "HUG",
		AddEventTwitterHashtag: "NEWEVENT",
	}
	querier.newClient()
	_, err := querier.GetTweetPublicMetric(context.Background(), []string{"1496269563708665857"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseTime(t *testing.T) {
	ti, err := time.Parse(time.RFC3339, "2022-07-11T04:10:57.000Z")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(ti.String())
}
