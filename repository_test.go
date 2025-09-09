package main

import (
	"database/sql"
	"fmt"
	"testing"
	"time"
)

func TestHNStore_CreateStory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := &HNStore{db: db}

	expectedStory := &HnStory{
		HnId:  1,
		Title: "test story",
		Time:  uint64(time.Now().Unix()),
	}

	err := store.CreateStory(expectedStory)
	if err != nil {
		t.Fatalf("CreateStory() failed: %v", err)
	}

	var story HnStory
	err = store.db.Get(
		&story,
		"select hn_id, title, time from hiring_story where hn_id = ?",
		expectedStory.HnId,
	)
	if err != nil {
		t.Fatalf("failed to query story from database: %v", err)
	}
	if story != *expectedStory {
		t.Fatalf("expected story %+v, got %+v", *expectedStory, story)
	}
}

func TestHNStore_GetLatestStory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := &HNStore{db: db}

	expectedStory := &HnStory{
		HnId:  1,
		Title: "latest story",
		Time:  uint64(time.Now().Unix()),
	}

	err := store.CreateStory(expectedStory)
	if err != nil {
		t.Fatalf("CreateStory() failed: %v", err)
	}

	story, err := store.GetLatestStory()
	if err != nil {
		t.Fatalf("GetLatestStory() failed: %v", err)
	}
	if *story != *expectedStory {
		t.Fatalf("expected story %+v, got %+v", expectedStory, story)
	}
}

func TestHNStore_GetLatestStory_WithNoStories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store := &HNStore{db: db}

	story, err := store.GetLatestStory()
	if err == nil {
		t.Fatalf("expected error: %v", sql.ErrNoRows)
	}
	if story != nil {
		t.Errorf("expected no story, got %+v", story)
	}
}

func TestHnJob_TransformedText(t *testing.T) {
	nowUnix := time.Now().Unix()
	tests := []struct {
		name     string
		job      *HnJob
		expected string
	}{
		{
			name:     "text_simple",
			job:      &HnJob{HnId: 1, Time: uint64(nowUnix), Text: "Foo.\nBar:<p>Test"},
			expected: `<p class="my-2">Foo.</p><p class="my-2">Bar:</p><p class="my-2">Test</p><p class="my-2"><a href="https://news.ycombinator.com/item?id=%d">Posted: %v</a></p>`,
		},
		{
			name:     "text_single_word",
			job:      &HnJob{HnId: 1, Time: uint64(nowUnix), Text: "Foo."},
			expected: `<p class="my-2">Foo.</p><p class="my-2"><a href="https://news.ycombinator.com/item?id=%d">Posted: %v</a></p>`,
		},
		{
			name:     "text_empty",
			job:      &HnJob{HnId: 1, Time: uint64(nowUnix), Text: ""},
			expected: `<p class="my-2"><a href="https://news.ycombinator.com/item?id=%d">Posted: %v</a></p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.job.TransformedText()
			expected := fmt.Sprintf(tt.expected, tt.job.HnId, time.Unix(nowUnix, 0))
			if res != expected {
				t.Errorf("expected %s, got %s", expected, res)
			}
		})
	}
}
