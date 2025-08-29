package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ApiStory represents a Hacker News story response item.
type ApiStory struct {
	Id    uint64   `json:"id"`
	Title string   `json:"title"`
	Time  uint64   `json:"time"`
	Kids  []uint64 `json:"kids"`
}

// ApiJob represents a Hacker News job response item.
type ApiJob struct {
	Id      uint64 `json:"id"`
	Text    string `json:"text"`
	Time    uint64 `json:"time"`
	Dead    bool   `json:"dead"`
	Deleted bool   `json:"deleted"`
}

// StatusToDbValue returns Hacker News status db value.
func (j *ApiJob) StatusToDbValue() uint8 {
	if j.Dead {
		return jobStatusDead
	}

	if j.Deleted {
		return jobStatusDeleted
	}

	return jobStatusOk
}

// Client is a client for the Hacker News API.
type Client struct {
	httpClient *http.Client
	baseUrl    string
}

// NewClient create new Hacker News API client.
func NewClient(baseUrl string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseUrl:    baseUrl,
	}
}

// GetStory fetches a Hacker News story by id.
func (c *Client) GetStory(id uint64) (*ApiStory, error) {
	url := fmt.Sprintf("%s/item/%d.json", c.baseUrl, id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get story %d: %w", id, err)
	}
	defer resp.Body.Close()

	var story ApiStory
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return nil, fmt.Errorf("failed to decode story %d: %w", id, err)
	}

	return &story, nil
}

// GetJob fetches a Hacker News story by id.
func (c *Client) GetJob(id uint64) (*ApiJob, error) {
	url := fmt.Sprintf("%s/item/%d.json", c.baseUrl, id)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get job %d: %w", id, err)
	}
	defer resp.Body.Close()

	var job ApiJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode job %d: %w", id, err)
	}

	return &job, nil
}

// GetWhoIsHiringSubmissionIds fetches story IDs from user whoishiring.
func (c *Client) GetWhoIsHiringSubmissionIds() ([]uint64, error) {
	url := fmt.Sprintf("%s/user/whoishiring.json", c.baseUrl)
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

// FindWhoIsHiringStory fetches the latest WhoIsHiring story.
func (c *Client) FindWhoIsHiringStory(storyIds []uint64) (*ApiStory, error) {
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
