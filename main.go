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

// getIndex will return the position of v in s
func getIndex[K comparable](s []K, v K) int {
	for i, sv := range s {
		if v == sv {
			return i
		}
	}
	return -1
}

// newHiringStory will attempt to insert a new hiring story to our db.
// Return the hacker news id.
func newHiringStory(s []int) (int, error) {
	type hiringStory struct {
		Id    int    `json:"id"`
		Title string `json:"title"`
		Time  int    `json:"time"`
	}
	hsSql := `INSERT INTO hiring_story (hn_id, title, time) VALUES (?, ?, ?)`
	for _, sv := range s {
		resp, err := http.Get(hnApiBaseUri + fmt.Sprintf("/item/%d.json", sv))
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		var hs hiringStory
		if err := json.NewDecoder(resp.Body).Decode(&hs); err != nil {
			panic(err)
		}

		if strings.HasPrefix(hs.Title, "Ask HN: Who is hiring?") {
			res := db.MustExec(hsSql, hs.Id, hs.Title, hs.Time)
			_, err := res.LastInsertId()
			if err != nil {
				panic(err)
			}
			return hs.Id, nil
		}
	}

	return -1, fmt.Errorf("could not add new hiring story from Ids %v", s)
}

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
		Id    int `db:"hn_id"`
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

	idx := getIndex(userStoryIds, hs.Id)
	var hnId int
	if idx == -1 {
		log.Printf("expected story id %d not found in %v. will update...", hs.Id, userStoryIds)
		hnId, err = newHiringStory(userStoryIds)
		if err != nil {
			panic(err)
		}
	} else {
		hnId = userStoryIds[idx]
	}

	log.Printf("using hiringStory id %d", hnId)
}

func main() {
	syncData()
}
