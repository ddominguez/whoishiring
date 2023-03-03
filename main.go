package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	hnApiBaseUri = "https://hacker-news.firebaseio.com/v0"
)

var db = sqlx.MustConnect("sqlite3", "whoishiring.db")

// syncData will fetch the latest who is hiring story
// insert new jobs from that story into our database.
func syncData() {
	fmt.Println("Updating database...")

	type hnUserResp struct {
		StoryIds []int `json:"submitted"`
	}

	resp, err := http.Get(hnApiBaseUri + "/user/whoishiring.json")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var userResp hnUserResp
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		panic(err)
	}

	// The story id we want should be in the first three items
	userStoryIds := userResp.StoryIds[0:3]

	type hiringStory struct {
		HnId  uint64 `db:"hn_id"`
		Title string
	}
	var hs hiringStory

	// Get the latest `who is hiring` story and check if the hn_id is in the user resp. 
	// If no results found, find the latest story from the user resp data.
	if err := db.Get(&hs, "select hn_id, title from hiring_story order by time desc limit 1"); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			log.Println("hiring story not found in db")
		} else {
			panic(err)
		}
	}
	log.Println(hs.HnId)
}

func main() {
	syncData()
}
