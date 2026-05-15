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
	verify := flag.Bool("verify", false, "Verify saved jobs are still OK")
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
	baseUrl := "https://hacker-news.firebaseio.com/v0"

	if *sync {
		client := NewClient(baseUrl)
		sp := NewSyncProcess(store, client)
		if err := sp.Run(); err != nil {
			log.Fatal(err)
		}
	}

	if *verify {
		client := NewClient(baseUrl)
		v := NewVerifyProcess(store, client)
		if err := v.Run(); err != nil {
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
