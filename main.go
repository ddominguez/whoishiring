package main

import (
	"database/sql"
	"flag"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

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
		baseUrl := "https://hacker-news.firebaseio.com/v0"
		client := NewClient(baseUrl)
		sp := NewSyncProcess(store, client)
		if err := sp.Run(); err != nil {
			log.Fatal(err)
		}
	}

	if *serve {
		latestStory, err := store.GetLatestStory()
		if err != nil {
			if err == sql.ErrNoRows {
				log.Fatal("GetLatestStory() returned zero rows.")
			}
			log.Fatalf("failed to get latest hiring story: %v", err)
		}

		minJobId, maxJobId, err := store.GetMinMaxJobsIds(latestStory.HnId)
		if err != nil {
			log.Fatalf("GetMinMaxJobsIds(%d) failed: %v", latestStory.HnId, err)
		}
		server := NewServer(store, latestStory, minJobId, maxJobId)
		server.Run()
	}
}
