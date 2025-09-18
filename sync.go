package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
)

type SyncProcess struct {
	store  *HNStore
	client *Client
}

// Run will fetch and save the latest WhoIsHiring story and jobs.
func (s *SyncProcess) Run() error {
	log.Println("starting data sync...")

	submissionIds, err := s.client.GetWhoIsHiringSubmissionIds()
	if err != nil {
		return err
	}

	// Hacker News will usually post 3 new stories at the beginning of the month:
	// `Who is hiring?`, `Freelancer? Seeking freelancer?`, and `Who wants to be hired?`
	// An assumption is being made that the latest `Who is hiring?` story is one
	// of the first 3 submission IDs.
	maxSubmissions := 3
	hiringStory, err := s.client.FindWhoIsHiringStory(submissionIds[0:maxSubmissions])
	if err != nil {
		return fmt.Errorf("failed to find who is hiring story: %w", err)
	}

	existingStory, err := s.store.GetLatestStory()
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	var hsHnId uint64
	if existingStory != nil && existingStory.HnId == hiringStory.Id {
		log.Printf("found existing hiring story: %d", existingStory.HnId)
		hsHnId = existingStory.HnId
	} else {
		err := s.store.CreateStory(&HnStory{
			HnId:  hiringStory.Id,
			Title: hiringStory.Title,
			Time:  hiringStory.Time,
		})
		if err != nil {
			return fmt.Errorf("failed to create hiring story: %w", err)
		}
		log.Println("new hiring story found and created")
		hsHnId = hiringStory.Id
	}

	return s.getNewJobs(hsHnId)
}

// getNewJobs will fetch and save new jobs for a given hiring story.
func (s *SyncProcess) getNewJobs(hnStoryId uint64) error {
	log.Printf("process jobs for hiring story id %d", hnStoryId)

	hs, err := s.client.GetStory(hnStoryId)
	if err != nil {
		return fmt.Errorf("failed to get story %d: %w", hnStoryId, err)
	}

	savedIds, err := s.store.GetJobIdsByStoryId(hnStoryId)
	if err != nil {
		return fmt.Errorf("failed to GetJobIdsByStoryId: %w", err)
	}

	var wg sync.WaitGroup

	// Save new job posts
	for _, jobId := range hs.Kids {
		if _, ok := savedIds[jobId]; ok {
			continue
		}

		// How is this possible, you ask??
		if jobId < 1 {
			log.Printf("Skipping hiring job id: %d\n", jobId)
			continue
		}

		wg.Add(1)
		go func(id uint64) {
			defer wg.Done()
			job, err := s.client.GetJob(id)
			if err != nil {
				log.Printf("failed to get job %d: %v", id, err)
				return
			}

			err = s.store.CreateJob(&HnJob{
				HnId:   job.Id,
				Text:   job.Text,
				Time:   job.Time,
				Status: job.StatusToDbValue(),
			}, hnStoryId)
			if err != nil {
				log.Printf("failed to create job %d: %v", id, err)
				return
			}

			log.Printf("added new hiring job %d", id)
		}(jobId)
	}

	wg.Wait()
	return nil
}

func NewSyncProcess(store *HNStore, client *Client) *SyncProcess {
	return &SyncProcess{
		store:  store,
		client: client,
	}
}
