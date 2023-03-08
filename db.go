package main

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	jobStatusOk      = 1
	jobStatusDead    = 2
	jobStatusDeleted = 3
)

var db = sqlx.MustConnect("sqlite3", "whoishiring.db")

func HiringJobStatus(dead bool, deleted bool) uint8 {
	if dead {
		return jobStatusDead
	}

	if deleted {
		return jobStatusDeleted
	}

	return jobStatusOk
}

func CreateHiringStory(hnId uint64, title string, time uint64) (uint64, error) {
	sql := `INSERT INTO hiring_story (hn_id, title, time) VALUES (?, ?, ?)`
	res := db.MustExec(sql, hnId, title, time)
	_, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return hnId, nil
}

func CreateHiringJob(hjId, hsId uint64, hjText string, hjTime uint64, hjStatus uint8) (uint64, error) {
	sql := `INSERT INTO hiring_job (hn_id, hiring_story_id, text, time, status) VALUES (?, ?, ?, ?, ?)`
	res := db.MustExec(sql, hjId, hsId, hjText, hjText, hjStatus)
	_, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return hjId, nil
}

func GetLatestHiringStory() (uint64, error) {
	type hiringStory struct {
		HnId  uint64 `db:"hn_id"`
		Title string
	}
	var hs hiringStory
	if err := db.Get(&hs, "SELECT hn_id, title FROM hiring_story ORDER BY time DESC LIMIT 1"); err != nil {
		return 0, err
	}

	return hs.HnId, nil
}

func SelectHiringJobIds(hsId int) (*sql.Rows, error) {
	sql := `select hn_id from hiring_job where hiring_story_id=?`
	rows, err := db.Query(sql, hsId)
	if err != nil {
		return nil, err
	}

	return rows, nil
}
