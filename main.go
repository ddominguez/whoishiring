package main

import (
	"flag"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	sync := flag.Bool("sync", false, "Sync who is hiring data")
	serve := flag.Bool("serve", false, "Run server")
	flag.Parse()

	db, err := sqlx.Open("sqlite3", "whoishiring.db")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

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
		server, err := InitializeNewServer(store)
		if err != nil {
			log.Fatal(err)
		}
		server.Run()
	}
}
