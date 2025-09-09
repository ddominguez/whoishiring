package main

import (
	"reflect"
	"testing"
	"time"
)

func TestInitializeNewServer(t *testing.T) {
	t.Run("new_server_created", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &HNStore{db: db}

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

		expectedServer := &Server{
			store:    store,
			hnStory:  story,
			minJobId: job.HnId,
			maxJobId: job.HnId,
		}

		server, err := InitializeNewServer(store)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !reflect.DeepEqual(*server, *expectedServer) {
			t.Fatalf("expected server %+v, got %+v", *expectedServer, *server)
		}
	})

	t.Run("no_stories_error", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &HNStore{db: db}
		server, err := InitializeNewServer(store)
		if err == nil {
			t.Fatal("expected an error, got nil")
		}

		expectedErrMsg := "GetLatestStory() returned zero rows."
		if err.Error() != expectedErrMsg {
			t.Fatalf("expected err %v, got %v", expectedErrMsg, err.Error())
		}

		if server != nil {
			t.Fatal("expected server to be nil on error")
		}
	})
}
