package main

import (
	"database/sql"
	"fmt"
	"log"
	"slices"
	"sync"
)

type SyncProcess struct {
	store  *HNStore
	client *Client
}

func NewSyncProcess(store *HNStore, client *Client) *SyncProcess {
	return &SyncProcess{
		store:  store,
		client: client,
	}
}

// Run will fetch and save the latest "Who is Hiring?" story and jobs.
func (s *SyncProcess) Run() error {
	log.Println("starting data sync...")

	storyID, err := s.getLatestStoryID()
	if err != nil {
		return err
	}

	if err := s.getNewJobs(storyID); err != nil {
		return err
	}

	return nil
}

// getLatestStoryID will return the latest "Who is Hiring?" story ID
func (s *SyncProcess) getLatestStoryID() (uint64, error) {
	submissionIds, err := s.client.GetWhoIsHiringSubmissionIds()
	if err != nil {
		return 0, err
	}

	// Hacker News will usually post 3 new stories at the beginning of the month:
	// "Who is hiring?", "Freelancer? Seeking freelancer?", and "Who wants to be hired?"
	// An assumption is being made that the latest "Who is hiring?" story is one
	// of the first 3 submission IDs.
	maxSubmissions := 3
	submissionsToSearch := submissionIds[0:maxSubmissions]

	existingStory, err := s.store.GetLatestStory()
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	// existingStory is still the latest story
	if existingStory != nil && slices.Contains(submissionsToSearch, existingStory.HnId) {
		log.Printf("found existing 'Who is Hiring?' story: %d", existingStory.HnId)
		return existingStory.HnId, nil
	}

	newStory, err := s.client.FindWhoIsHiringStory(submissionsToSearch)
	if err != nil {
		return 0, err
	}

	if err := s.store.CreateStory(&HnStory{
		HnId:  newStory.Id,
		Title: newStory.Title,
		Time:  newStory.Time,
	}); err != nil {
		return 0, err
	}

	log.Println("new 'Who is Hiring?' story found and created")
	return newStory.Id, nil
}

// getNewJobs will fetch and save new jobs for a given hiring story.
func (s *SyncProcess) getNewJobs(hnStoryId uint64) error {
	log.Printf("process jobs for 'Who is Hiring?' story id %d", hnStoryId)

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
