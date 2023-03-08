package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

const (
	hnApiBaseUri = "https://hacker-news.firebaseio.com/v0"
)

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
func newHiringStory(s []int) (uint64, error) {
	type hiringStory struct {
		Id    uint64 `json:"id"`
		Title string `json:"title"`
		Time  uint64 `json:"time"`
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
			hsId, err := CreateHiringStory(hs.Id, hs.Title, hs.Time)
			if err != nil {
				return 0, err
			}
			return hsId, nil
		}
	}

	return 0, fmt.Errorf("could not add new hiring story from Ids %v", s)
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

	hjStatus := HiringJobStatus(hj.Dead, hj.Deleted)
	_, err = CreateHiringJob(hj.Id, hsid, hj.Text, hj.Time, hjStatus)
	if err != nil {
		return 0, nil
	}

	return hjid, nil
}

// processJobPosts will attempt to fetch and process job items for a given hiring story
func processJobPosts(hsid uint64) {
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

	var savedIds = make(map[uint64]bool)
	rows, err := SelectHiringJobIds(int(hsid))
	if err != nil {
		panic(err)
	}
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

	hsId, err := GetLatestHiringStory()
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			log.Println("hiring story not found in db")
		} else {
			panic(err)
		}
	}

	idx := getIndex(userStoryIds, int(hsId))
	var hsid uint64
	if idx == -1 {
		log.Printf("expected story id %d not found in %v. will update...", hsId, userStoryIds)
		hsid, err = newHiringStory(userStoryIds)
		if err != nil {
			panic(err)
		}
	} else {
		hsid = uint64(userStoryIds[idx])
	}

	processJobPosts(hsid)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	hsId, err := GetLatestHiringStory()
	if err != nil {
		panic(err)
	}

	log.Printf("Found hiring story id %d", hsId)

	fmt.Fprint(w, "hello.")
}

func main() {
	// syncData()

	http.HandleFunc("/", indexHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
