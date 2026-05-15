package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
)

type VerifyProcess struct {
	store  *HNStore
	client *Client
}

func NewVerifyProcess(store *HNStore, client *Client) *VerifyProcess {
	return &VerifyProcess{
		store:  store,
		client: client,
	}
}

func (v *VerifyProcess) Run() error {
	log.Println("starting verify process...")

	latestStory, err := v.store.GetLatestStory()
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("GetLatestStory() returned zero rows.")
		}
		return fmt.Errorf("failed to get latest hiring story: %w", err)
	}
	log.Printf("latest hiring story id: %d", latestStory.HnId)

	jobs, err := v.store.GetOkJobIdsByStoryId(latestStory.HnId)
	if err != nil {
		return fmt.Errorf("failed to get job ids with OK status: %w", err)
	}
	log.Printf("found %d jobs with OK status", len(jobs))

	var wg sync.WaitGroup

	for jobId := range jobs {
		wg.Go(func() {
			j, err := v.client.GetJob(jobId)
			if err != nil {
				log.Println(err)
				return
			}
			hnStatus := j.StatusToDbValue()
			if hnStatus != jobStatusOk {
				err := v.store.SetJobStatus(jobId, hnStatus)
				if err != nil {
					log.Println(err)
					return
				}
				log.Printf("job id %d is NOT OK, updated status to %d", jobId, hnStatus)
			}
		})
	}

	wg.Wait()

	return nil
}
