package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"text/template"
)

type Server struct {
	store    *HNStore
	hnStory  *HnStory
	minJobId uint64
	maxJobId uint64
}

// NewServer creates a new Server.
func NewServer(
	store *HNStore,
	latestStory *HnStory,
	minJobId, maxJobId uint64,
) *Server {
	return &Server{
		store:    store,
		hnStory:  latestStory,
		minJobId: minJobId,
		maxJobId: maxJobId,
	}
}

// InitializeNewServer creates a Server configured with the latest story and its job ID range.
func InitializeNewServer(store *HNStore) (*Server, error) {
	latestStory, err := store.GetLatestStory()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("GetLatestStory() returned zero rows.")
		}
		return nil, fmt.Errorf("failed to get latest hiring story: %w", err)
	}

	minJobId, maxJobId, err := store.GetMinMaxJobIDs(latestStory.HnId)
	if err != nil {
		return nil, fmt.Errorf("GetMinMaxJobsIds(%d) failed: %w", latestStory.HnId, err)
	}

	return NewServer(store, latestStory, minJobId, maxJobId), nil
}

// GetMux creates a new serve mux and registers its handler funcs.
func (s *Server) GetMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.indexHandler)
	mux.HandleFunc("GET /api/seen/{hnId}", s.seenHandler)
	return mux
}

// Run starts the web server.
func (s *Server) Run() {
	mux := s.GetMux()
	fmt.Println("Listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// parseUint64OrDefault parses stringVal as a uint64, returning defaultVal if
// stringVal is empty or invalid.
func (s *Server) parseUint64OrDefault(stringVal string, defaultVal uint64) uint64 {
	trimmed := strings.TrimSpace(stringVal)
	if trimmed == "" {
		return defaultVal
	}

	converted, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		log.Printf("failed to convert string %s to uint64\n", trimmed)
		return defaultVal
	}

	return converted
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	after := s.parseUint64OrDefault(r.URL.Query().Get("after"), 0)
	before := s.parseUint64OrDefault(r.URL.Query().Get("before"), 0)

	var hj *HnJob
	var err error
	if after == 0 && before > 0 {
		hj, err = s.store.GetJobBeforeID(s.hnStory.HnId, before)
	} else if after > 0 && before == 0 {
		hj, err = s.store.GetJobAfterID(s.hnStory.HnId, after)
	} else {
		hj, err = s.store.GetFirstJob(s.hnStory.HnId)
	}
	if err != nil {
		log.Println("failed to select hiring job:", err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	hj.Text = hj.TransformedText()
	data := struct {
		Story    *HnStory
		Job      *HnJob
		MinJobId uint64
		MaxJobId uint64
	}{
		Story:    s.hnStory,
		Job:      hj,
		MinJobId: s.minJobId,
		MaxJobId: s.maxJobId,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.ParseFiles("templates/base.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Println("failed to execute to templates", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) seenHandler(w http.ResponseWriter, r *http.Request) {
	pathValue := r.PathValue("hnId")
	hnId, err := strconv.ParseUint(pathValue, 10, 64)
	if err != nil {
		log.Printf("failed to convert path value:%q to uint64", pathValue)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err := s.store.SetJobAsSeen(hnId); err != nil {
		if errors.Is(err, ZeroRowsUpdated) {
			log.Printf("SetJobAsSeen(%d) did not updates any rows", hnId)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
}
