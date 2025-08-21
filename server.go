package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"text/template"
)

type Server struct {
	currHiringStory *HiringStory
	firstLastIds    *HiringJobId
}

func NewServer() (*Server, error) {
	latestHiringStory, err := GetLatestHiringStory()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest hiring story: %w", err)
	}

	firstLastIds, err := GetMinMaxHiringJobIds(latestHiringStory.HnId)
	if err != nil {
		return nil, fmt.Errorf("failed to get min/max hiring job IDs: %w", err)
	}

	return &Server{
		currHiringStory: latestHiringStory,
		firstLastIds:    firstLastIds,
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

	hj, err := SelectCurrentHiringJob(s.currHiringStory.HnId, after, before)
	if err != nil {
		log.Println("failed to select hiring job:", err)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	hj.Text = hj.transformedText()

	data := struct {
		Story HiringStory
		Job   HiringJob
		HjIds HiringJobId
	}{
		Story: *s.currHiringStory,
		Job:   *hj,
		HjIds: *s.firstLastIds,
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

	if err := SetHiringJobAsSeen(hnId); err != nil {
		log.Println(err)
	}
}
