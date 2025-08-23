package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
)

// processJobPosts will attempt to fetch and process job items for a given hiring story
func processJobPosts(client *Client, hsHnId uint64) error {
	log.Printf("process jobs for hiring story id %d", hsHnId)

	hs, err := client.GetStory(hsHnId)
	if err != nil {
		return fmt.Errorf("failed to get story %d: %w", hsHnId, err)
	}

	var savedIds = make(map[uint64]bool)
	rows, err := SelectHiringJobIds(int(hsHnId))
	if err != nil {
		return err
	}
	for rows.Next() {
		var hnid uint64
		if err := rows.Scan(&hnid); err != nil {
			return err
		}
		savedIds[hnid] = true
	}

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

		hj, err := client.GetJob(jobId)
		if err != nil {
			return fmt.Errorf("failed to get job %d: %v", jobId, err)
		}

		hjStatus := HiringJobStatus(hj.Dead, hj.Deleted)
		_, err = CreateHiringJob(hj.Id, hsHnId, hj.Text, hj.Time, hjStatus)
		if err != nil {
			log.Printf("failed to create job %d: %v", jobId, err)
			continue
		}

		log.Printf("added new hiring job %d", jobId)
	}

	return nil
}

// syncData will fetch and save the latest WhoIsHiring story and jobs.
func syncData() error {
	log.Println("starting data sync...")

	client := NewClient()
	submissionIds, err := client.GetWhoIsHiringSubmissionIds()

	// The story id we want should be in the first three items
	hiringStory, err := client.FindWhoIsHiringStory(submissionIds[0:3])
	if err != nil {
		return fmt.Errorf("failed to find who is hiring story: %w", err)
	}

	latestStory, err := GetLatestHiringStory()
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			log.Println("hiring story not found in db")
		} else {
			log.Println("failed to get latest hiring story")
			return err
		}
	}

	var hsHnId uint64
	if latestStory != nil && latestStory.HnId == hiringStory.Id {
		log.Println("existing hiring story found")
		hsHnId = latestStory.HnId
	} else {
		hsHnId, err = CreateHiringStory(hiringStory.Id, hiringStory.Title, hiringStory.Time)
		if err != nil {
			return fmt.Errorf("failed to create hiring story: %w", err)
		}
		log.Println("new hiring story found and created")
	}

	return processJobPosts(client, hsHnId)
}

func main() {
	sync := flag.Bool("sync", false, "Sync who is hiring data")
	serve := flag.Bool("serve", false, "Run server")
	flag.Parse()

	if *sync {
		if err := syncData(); err != nil {
			log.Fatal(err)
		}
	}

	if *serve {
		server, err := NewServer()
		if err != nil {
			log.Fatal("failed to create server:", err)
		}
		server.Run()
	}
}
