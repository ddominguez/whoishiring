package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const apiBaseURL = "https://hacker-news.firebaseio.com/v0"

// Story represents a Hacker News story.
type Story struct {
	Id    uint64   `json:"id"`
	Title string   `json:"title"`
	Time  uint64   `json:"time"`
	Kids  []uint64 `json:"kids"`
}

// Job represents a Hacker News job post.
type Job struct {
	Id      uint64 `json:"id"`
	Text    string `json:"text"`
	Time    uint64 `json:"time"`
	Dead    bool   `json:"dead"`
	Deleted bool   `json:"deleted"`
}

// Client is a client for the Hacker News API.
type Client struct {
	httpClient *http.Client
}

// NewClient create new Hacker News API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetStory fetches a Hacker News story by id
func (c *Client) GetStory(id uint64) (*Story, error) {
	url := fmt.Sprintf("%s/item/%d.json", apiBaseURL, id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get story %d: %w", id, err)
	}
	defer resp.Body.Close()

	var story Story
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return nil, fmt.Errorf("failed to decode story %d: %w", id, err)
	}

	return &story, nil
}

// GetJob fetches a Hacker News story by id
func (c *Client) GetJob(id uint64) (*Job, error) {
	url := fmt.Sprintf("%s/item/%d.json", apiBaseURL, id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get job %d: %w", id, err)
	}
	defer resp.Body.Close()

	var job Job
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode job %d: %w", id, err)
	}

	return &job, nil
}

// GetWhoIsHiringSubmissionIds fetches story IDs from user whoishiring
func (c *Client) GetWhoIsHiringSubmissionIds() ([]uint64, error) {
	url := fmt.Sprintf("%s/user/whoishiring.json", apiBaseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get whoishiring user: %w", err)
	}
	defer resp.Body.Close()

	var user struct {
		Submitted []uint64 `json:"submitted"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode whoishiring user: %w", err)
	}

	return user.Submitted, nil
}

func (c *Client) FindWhoIsHiringStory(storyIds []uint64) (*Story, error) {
	for _, id := range storyIds {
		story, err := c.GetStory(id)
		if err != nil {
			fmt.Printf("failed to get story %d: %v\n", id, err)
			continue
		}

		if strings.HasPrefix(story.Title, "Ask HN: Who is hiring?") {
			return story, nil
		}
	}

	return nil, fmt.Errorf("no 'Who is hiring?' story found in the provided IDs")
}
