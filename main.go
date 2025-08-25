package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// processJobPosts will fetch and process job items for a given hiring story
func processJobPosts(store HNRepository, client *Client, hnStoryId uint64) error {
	log.Printf("process jobs for hiring story id %d", hnStoryId)

	hs, err := client.GetStory(hnStoryId)
	if err != nil {
		return fmt.Errorf("failed to get story %d: %w", hnStoryId, err)
	}

	savedIds, err := store.GetJobIdsByStoryId(hnStoryId)

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

		job, err := client.GetJob(jobId)
		if err != nil {
			return fmt.Errorf("failed to get job %d: %v", jobId, err)
		}

		err = store.CreateJob(&HnJob{
			HnId:   job.Id,
			Text:   job.Text,
			Time:   job.Time,
			Status: job.StatusToDbValue(),
		}, hnStoryId)
		if err != nil {
			log.Printf("failed to create job %d: %v", jobId, err)
			continue
		}

		log.Printf("added new hiring job %d", jobId)
	}

	return nil
}

// syncData will fetch and save the latest WhoIsHiring story and jobs.
func syncData(store HNRepository) error {
	log.Println("starting data sync...")

	client := NewClient()
	submissionIds, err := client.GetWhoIsHiringSubmissionIds()

	// The story id we want should be in the first three items
	hiringStory, err := client.FindWhoIsHiringStory(submissionIds[0:3])
	if err != nil {
		return fmt.Errorf("failed to find who is hiring story: %w", err)
	}

	latestStory, err := store.GetLatestStory()
	if err != nil {
		return err
	}

	var hsHnId uint64
	if latestStory != nil && latestStory.HnId == hiringStory.Id {
		log.Println("existing hiring story found")
		hsHnId = latestStory.HnId
	} else {
		err := store.CreateStory(&HnStory{
			HnId:  hiringStory.Id,
			Title: hiringStory.Title,
			Time:  hiringStory.Time,
		})
		if err != nil {
			return fmt.Errorf("failed to create hiring story: %w", err)
		}
		log.Println("new hiring story found and created")
	}

	return processJobPosts(store, client, hsHnId)
}

func main() {
	sync := flag.Bool("sync", false, "Sync who is hiring data")
	serve := flag.Bool("serve", false, "Run server")
	flag.Parse()

	db, err := sqlx.Connect("sqlite3", "whoishiring.db")
	if err != nil {
		log.Fatal("failed to connect to database: %w", err)
	}
	defer db.Close()

	store := NewHNStore(db)

	if *sync {
		if err := syncData(store); err != nil {
			log.Fatal(err)
		}
	}

	if *serve {
		server, err := NewServer(store)
		if err != nil {
			log.Fatal("failed to create server:", err)
		}
		server.Run()
	}
}
