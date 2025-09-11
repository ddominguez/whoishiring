package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestInitializeNewServer(t *testing.T) {
	t.Run("new_server_created", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &HNStore{db: db}
		story, job := setUpStoryWithJob(t, store)

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

func TestServer_seenHandler_request(t *testing.T) {
	t.Run("sets_job_as_seen", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &HNStore{db: db}
		story, job := setUpStoryWithJob(t, store)
		if job.Seen != 0 {
			t.Fatalf("expected seen value to be 0, got: %d", job.Seen)
		}

		s := &Server{
			store:    store,
			hnStory:  story,
			minJobId: job.HnId,
			maxJobId: job.HnId,
		}

		mux := s.GetMux()
		url := fmt.Sprintf("/api/seen/%d", job.HnId)
		req := httptest.NewRequest("GET", url, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status code %d, got: %d", http.StatusOK, rr.Code)
		}

		updatedJob := queryTestJobById(t, store, job.HnId)
		if updatedJob.HnId != job.HnId {
			t.Fatalf("expected job id %d, got: %d", job.HnId, updatedJob.HnId)
		}
		if updatedJob.Seen != 1 {
			t.Fatalf("expected seen value to be 1, got: %d", updatedJob.Seen)
		}
	})

	t.Run("invalid_path_param", func(t *testing.T) {
		s := &Server{}
		mux := s.GetMux()
		req := httptest.NewRequest("GET", "/api/seen/invalid", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got: %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("invalid_job_id", func(t *testing.T) {
		db := setupTestDB(t)
		defer db.Close()

		store := &HNStore{db: db}
		s := &Server{store: store}
		mux := s.GetMux()
		req := httptest.NewRequest("GET", "/api/seen/2", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status code %d, got: %d", http.StatusBadRequest, rr.Code)
		}
	})
}
