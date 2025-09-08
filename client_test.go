package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestApiJob_StatusToDbValue(t *testing.T) {
	tests := []struct {
		name     string
		job      ApiJob
		expected uint8
	}{
		{
			name:     "status_ok",
			job:      ApiJob{Id: 1, Text: "test job", Dead: false, Deleted: false},
			expected: jobStatusOk,
		},
		{
			name:     "status_deleted",
			job:      ApiJob{Id: 1, Text: "deleted job", Dead: false, Deleted: true},
			expected: jobStatusDeleted,
		},
		{
			name:     "status_dead",
			job:      ApiJob{Id: 1, Text: "dead job", Dead: true, Deleted: false},
			expected: jobStatusDead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.job.StatusToDbValue()
			if res != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, res)
			}
		})
	}
}

func TestClientGetStory(t *testing.T) {
	t.Run("is successful", func(t *testing.T) {
		testID := uint64(1)
		expectedStory := ApiStory{
			Id:    testID,
			Title: "Test Story",
			Time:  1677721599,
			Kids:  []uint64{2, 3},
		}
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got: %s", r.Method)
				}

				expectedPath := fmt.Sprintf("/item/%d.json", testID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected request to %q, got: %q", expectedPath, r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(expectedStory)
			}),
		)
		defer server.Close()

		client := NewClient(server.URL)
		story, err := client.GetStory(testID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if story == nil {
			t.Fatal("expected a story, got nil")
		}
		if !reflect.DeepEqual(*story, expectedStory) {
			t.Fatalf("expected story %+v, got %+v", expectedStory, *story)
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}),
		)
		defer server.Close()

		client := NewClient(server.URL)
		story, err := client.GetStory(1)

		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if story != nil {
			t.Fatal("expected nil story on error, got non-nil")
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
		)
		defer server.Close()

		client := NewClient(server.URL)
		story, err := client.GetStory(1)

		if err == nil {
			t.Fatal("expected an error, got nil")
		}
		if story != nil {
			t.Fatal("expected nil story on error, got non-nil")
		}
	})
}
