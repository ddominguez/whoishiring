package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

const (
	hnApiBaseUri = "https://hacker-news.firebaseio.com/v0"
)

var firstLastIds *HiringJobId

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
			return 0, err
		}
		defer resp.Body.Close()

		var hs hiringStory
		if err := json.NewDecoder(resp.Body).Decode(&hs); err != nil {
			return 0, err
		}

		if strings.HasPrefix(hs.Title, "Ask HN: Who is hiring?") {
			hsHnId, err := CreateHiringStory(hs.Id, hs.Title, hs.Time)
			if err != nil {
				return 0, err
			}
			return hsHnId, nil
		}
	}

	return 0, fmt.Errorf("could not add new hiring story from Ids %v", s)
}

// newHiringJob will attempt to fetch a job item from hacker news
// and saves it to our database.
func newHiringJob(hsHnId, hjHnId uint64) (uint64, error) {
	resp, err := http.Get(hnApiBaseUri + fmt.Sprintf("/item/%d.json", hjHnId))
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
	_, err = CreateHiringJob(hj.Id, hsHnId, hj.Text, hj.Time, hjStatus)
	if err != nil {
		return 0, nil
	}

	return hjHnId, nil
}

// processJobPosts will attempt to fetch and process job items for a given hiring story
func processJobPosts(hsHnId uint64) error {
	log.Printf("process jobs for hiring story id %d", hsHnId)
	itemPath := fmt.Sprintf("/item/%d.json", hsHnId)
	resp, err := http.Get(hnApiBaseUri + itemPath)
	if err != nil {
		log.Printf("failed to request %s\n", itemPath)
		return err
	}
	defer resp.Body.Close()

	var hs struct {
		Kids []uint64 `json:"kids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hs); err != nil {
		log.Printf("failed to decode response for %s\n", itemPath)
		return err
	}

	var savedIds = make(map[uint64]bool)
	rows, err := SelectHiringJobIds(int(hsHnId))
	if err != nil {
		return err
	}
	for rows.Next() {
		var hnid uint64
		if err := rows.Scan(&hnid); err != nil {
			return err
		}
		savedIds[hnid] = true
	}

	// Save new job posts
	for _, v := range hs.Kids {
		if _, ok := savedIds[v]; ok {
			continue
		}
		if v < 1 {
			log.Printf("Skipping hiring job id: %d\n", v)
			continue
		}
		_, err := newHiringJob(uint64(hsHnId), v)
		if err != nil {
			return err
		}
		log.Printf("added new hiring job %d", v)
	}

	return nil
}

// syncData will fetch the latest who is hiring story
// insert new jobs from that story into our database.
func syncData() error {
	log.Println("starting data sync...")

	type hnUserResp struct {
		StoryIds []int `json:"submitted"`
	}

	resp, err := http.Get(hnApiBaseUri + "/user/whoishiring.json")
	if err != nil {
		log.Println("whoishiring.json request failed")
		return err
	}
	defer resp.Body.Close()

	var userResp hnUserResp
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		log.Println("failed to decode whoishiring.json response")
		return err
	}

	// The story id we want should be in the first three items
	userStoryIds := userResp.StoryIds[0:3]

	hs, err := GetLatestHiringStory()
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			log.Println("hiring story not found in db")
		} else {
			log.Println("failed to get latest hiring story")
			return err
		}
	}

	idx := getIndex(userStoryIds, int(hs.HnId))
	// hiring story hn_id
	var hsHnId uint64
	if idx == -1 {
		log.Printf("expected story id %d not found in %v. will update...", hs.HnId, userStoryIds)
		hsHnId, err = newHiringStory(userStoryIds)
		if err != nil {
			log.Println("failed to create new hiring story")
			return err
		}
	} else {
		hsHnId = uint64(userStoryIds[idx])
	}

	return processJobPosts(hsHnId)
}

// paramValue will return a parsed string as uint64 or a default value
func paramValue(v string, d uint64) uint64 {
	if v == "" {
		return d
	}

	converted, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return d
	}

	return converted
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	hs, err := GetLatestHiringStory()
	if err != nil {
		log.Println("failed to get latest story.", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if firstLastIds == nil {
		firstLastIds, err = GetMinMaxHiringJobIds(hs.HnId)
		if err != nil {
			log.Println("failed to get first and last ids", err)
		}
	}

	after := paramValue(r.URL.Query().Get("after"), 0)
	before := paramValue(r.URL.Query().Get("before"), 0)

	var hj *HiringJob
	hj, err = SelectCurrentHiringJob(hs.HnId, after, before)
	if err != nil {
		log.Println("failed to select hiring job.", err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	hj.Text = hj.transformedText()
	data := struct {
		Story HiringStory
		Job   HiringJob
		HjIds HiringJobId
	}{
		Story: *hs,
		Job:   *hj,
		HjIds: *firstLastIds,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFiles("templates/base.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("failed to execute to templates", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func seenHandler(w http.ResponseWriter, r *http.Request) {
	hnId, err := strconv.ParseUint(r.PathValue("hnId"), 10, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := SetHiringJobAsSeen(hnId); err != nil {
		log.Println(err)
	}
}

func main() {
	sync := flag.Bool("sync", false, "Sync who is hiring data")
	serve := flag.Bool("serve", false, "Run server")
	flag.Parse()

	if *sync {
		if err := syncData(); err != nil {
			log.Fatal(err)
		}
	}

	if *serve {
		http.HandleFunc("GET /api/seen/{hnId}", seenHandler)
		http.HandleFunc("GET /", indexHandler)
		fmt.Println("Listening on http://localhost:8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
}
