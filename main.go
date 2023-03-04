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
	hnApiBaseUri     = "https://hacker-news.firebaseio.com/v0"
	jobStatusOk      = 1
	jobStatusDead    = 2
	jobStatusDeleted = 3
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
			sql := `INSERT INTO hiring_story (hn_id, title, time) VALUES (?, ?, ?)`
			res := db.MustExec(sql, hs.Id, hs.Title, hs.Time)
			_, err := res.LastInsertId()
			if err != nil {
				panic(err)
			}
			return hs.Id, nil
		}
	}

	return -1, fmt.Errorf("could not add new hiring story from Ids %v", s)
}

func getStatus(dead bool, deleted bool) uint8 {
	if dead {
		return jobStatusDead
	}

	if deleted {
		return jobStatusDeleted
	}

	return jobStatusOk
}

// newHiringJob will attempt to fetch a job item from hacker news
// and saves it to our database.
func newHiringJob(hsid, hjid uint64) (uint64, error) {
	resp, err := http.Get(hnApiBaseUri + fmt.Sprintf("/item/%d.json", hjid))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var hj struct {
		Id      uint64 `json:"id"`
		Text    string `json:"text"`
		Time    uint64 `json:"time"`
		Dead    bool   `json:"dead"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hj); err != nil {
		return 0, err
	}

	sql := `INSERT INTO hiring_job (hn_id, hiring_story_id, text, time, status) VALUES (?, ?, ?, ?, ?)`
	status := getStatus(hj.Dead, hj.Deleted)
	res := db.MustExec(sql, hj.Id, hsid, hj.Text, hj.Time, status)
	_, err = res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return hjid, nil
}

// processJobPosts will attempt to fetch and process job items for a given hiring story
func processJobPosts(hsid int) {
	log.Printf("process jobs for hiring story id %d", hsid)
	resp, err := http.Get(hnApiBaseUri + fmt.Sprintf("/item/%d.json", hsid))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var hs struct {
		Kids []uint64 `json:"kids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hs); err != nil {
		panic(err)
	}

	sql := `select hn_id from hiring_job where hiring_story_id=?`
	rows, err := db.Query(sql, hsid)
	if err != nil {
		panic(err)
	}
	var savedIds = make(map[uint64]bool)
	for rows.Next() {
		var hnid uint64
		if err := rows.Scan(&hnid); err != nil {
			panic(err)
		}
		savedIds[hnid] = true
	}

	// Save new job posts
	for _, v := range hs.Kids {
		if _, ok := savedIds[v]; ok {
			continue
		}
		_, err := newHiringJob(uint64(hsid), v)
		if err != nil {
			log.Printf("failed to process hiring job id %d. %s", v, err)
			continue
		}
		log.Printf("added new hiring job %d", v)
	}
}

// syncData will fetch the latest who is hiring story
// insert new jobs from that story into our database.
func syncData() {
	log.Println("starting data sync...")

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
	var hsid int
	if idx == -1 {
		log.Printf("expected story id %d not found in %v. will update...", hs.Id, userStoryIds)
		hsid, err = newHiringStory(userStoryIds)
		if err != nil {
			panic(err)
		}
	} else {
		hsid = userStoryIds[idx]
	}

	processJobPosts(hsid)
}

func main() {
	syncData()
}
