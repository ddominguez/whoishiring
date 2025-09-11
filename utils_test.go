package main

import (
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

// setUpStoryWithJob adds a story and job to the DB for testing.
func setUpStoryWithJob(t *testing.T, store *HNStore) (*HnStory, *HnJob) {
	story := &HnStory{
		HnId:  1,
		Title: "test story",
		Time:  uint64(time.Now().Unix()),
	}
	if err := store.CreateStory(story); err != nil {
		t.Fatalf("CreateStory() failed: %v", err)
	}

	job := &HnJob{
		HnId:   1,
		Text:   "test job",
		Time:   uint64(time.Now().Unix()),
		Status: jobStatusOk,
	}
	if err := store.CreateJob(job, story.HnId); err != nil {
		t.Fatalf("CreateJob() failed: %v", err)
	}

	return story, job
}

func queryTestJobById(t *testing.T, store *HNStore, jobId uint64) *HnJob {
	var job HnJob

	query := `SELECT hn_id, seen, saved, status, text, time
            FROM hiring_job
            WHERE hn_id=?`
	if err := store.db.Get(&job, query, jobId); err != nil {
		t.Fatalf("failed to query test job: %v", err)
	}

	return &job
}
