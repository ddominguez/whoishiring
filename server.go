package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"
)

type Server struct {
	store    HNRepository
	hnStory  *HnStory
	minJobId uint64
	maxJobId uint64
}

// NewServer creates a new Server.
func NewServer(store HNRepository) (*Server, error) {
	latestStory, err := store.GetLatestStory()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest hiring story: %w", err)
	}

	minJobId, maxJobId, err := store.GetMinMaxJobsIds(latestStory.HnId)
	if err != nil {
		return nil, fmt.Errorf("failed to get min/max hiring job IDs: %w", err)
	}

	return &Server{
		store:    store,
		hnStory:  latestStory,
		minJobId: minJobId,
		maxJobId: maxJobId,
	}, nil
}

// Run starts the web server.
func (s *Server) Run() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.indexHandler)
	mux.HandleFunc("POST /api/seen/{hnId}", s.seenHandler)
	fmt.Println("Listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
	before, _ := strconv.ParseUint(r.URL.Query().Get("before"), 10, 64)

	var hj *HnJob
	var err error
	if after == 0 && before > 0 {
		hj, err = s.store.GetPreviousJobById(s.hnStory.HnId, before)
	} else if after > 0 && before == 0 {
		hj, err = s.store.GetNextJobById(s.hnStory.HnId, after)
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
	hnId, err := strconv.ParseUint(r.PathValue("hnId"), 10, 0)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := s.store.SetJobAsSeen(hnId); err != nil {
		log.Println(err)
	}
}
