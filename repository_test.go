package main

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

// setupTestDB creates an in-memory sqlite3 database and applies available migrations.
func setupTestDB(t *testing.T) *sqlx.DB {
	migrationsPath := "./migrations"

	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	goose.SetLogger(goose.NopLogger())

	err = goose.SetDialect(db.DriverName())
	if err != nil {
		t.Fatalf("failed to set dialect: %v", err)
	}

	err = goose.Up(db.DB, migrationsPath)
	if err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

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
