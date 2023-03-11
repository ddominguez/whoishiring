package main

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	jobStatusOk      = 1
	jobStatusDead    = 2
	jobStatusDeleted = 3
)

var db = sqlx.MustConnect("sqlite3", "whoishiring.db")

type HiringStory struct {
	HnId  uint64 `db:"hn_id"`
	Title string
}

type HiringJob struct {
	HnId uint64 `db:"hn_id"`
	Text string
	Time uint64
}

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
	res := db.MustExec(sql, hjId, hsId, hjText, hjTime, hjStatus)
	_, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return hjId, nil
}

func GetLatestHiringStory() (*HiringStory, error) {
	var hs HiringStory
	if err := db.Get(&hs, "SELECT hn_id, title FROM hiring_story ORDER BY time DESC LIMIT 1"); err != nil {
		return &hs, err
	}

	return &hs, nil
}

func SelectHiringJobIds(hsId int) (*sql.Rows, error) {
	sql := `SELECT hn_id FROM hiring_job WHERE hiring_story_id=?`
	rows, err := db.Query(sql, hsId)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func SelectNextHiringJob(hsId uint64, hnTime uint64) (*HiringJob, error) {
	var hj HiringJob
	sql := `SELECT hn_id, text, time
            FROM hiring_job
            WHERE hiring_story_id=? and status=? and time < ?
            ORDER BY time Desc
            Limit 1`
	if hnTime == 0 {
		hnTime = uint64(time.Now().Unix())
	}
	if err := db.Get(&hj, sql, hsId, jobStatusOk, hnTime); err != nil {
		return &hj, err
	}

	return &hj, nil
}

func SelectPreviousHiringJob(hsId uint64, hnTime uint64) (*HiringJob, error) {
	var hj HiringJob
	sql := `SELECT hn_id, text, time
            FROM hiring_job
            WHERE hiring_story_id=? and status=? and time > ?
            ORDER BY time ASC
            Limit 1`
	if err := db.Get(&hj, sql, hsId, jobStatusOk, hnTime); err != nil {
		return &hj, err
	}

	return &hj, nil
}
