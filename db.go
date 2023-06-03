package main

import (
	"database/sql"
	"fmt"
	"strings"
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
	HnId  uint64 `db:"hn_id"`
	Text  string
	Time  uint64
	Seen  uint8
	Saved uint8
}

type HiringJobTime struct {
	Min uint64
	Max uint64
}

// transformedText will parse the job text and return
// a string with updated html and styles
func (hj HiringJob) transformedText() string {
	var s string
	var l []string
	st := strings.Split(hj.Text, "\n")
	for _, v := range st {
		sl := strings.Split(v, "<p>")
		for _, slv := range sl {
			l = append(l, slv)
		}
	}
	for _, v := range l {
		s = s + fmt.Sprintf(`<p class="my-2">%s</p>`, v)
	}
	tm := time.Unix(int64(hj.Time), 0)
	s = s + fmt.Sprintf(`<p class="my-2"><a href="https://news.ycombinator.com/item?id=%d">Posted: %s</a></p>`, hj.HnId, tm)
	return s
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

func CreateHiringJob(hjId, hsHnId uint64, hjText string, hjTime uint64, hjStatus uint8) (uint64, error) {
	sql := `INSERT INTO hiring_job (hn_id, hiring_story_hn_id, text, time, status) VALUES (?, ?, ?, ?, ?)`
	res := db.MustExec(sql, hjId, hsHnId, hjText, hjTime, hjStatus)
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

func SelectHiringJobIds(hsHnId int) (*sql.Rows, error) {
	sql := `SELECT hn_id FROM hiring_job WHERE hiring_story_hn_id=?`
	rows, err := db.Query(sql, hsHnId)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func SelectNextHiringJob(hsHnId, hnTime uint64) (*HiringJob, error) {
	var hj HiringJob
	sql := `SELECT hn_id, seen, saved, text, time
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=? and time < ?
            ORDER BY time Desc
            Limit 1`
	if hnTime == 0 {
		hnTime = uint64(time.Now().Unix())
	}
	if err := db.Get(&hj, sql, hsHnId, jobStatusOk, hnTime); err != nil {
		return &hj, err
	}

	return &hj, nil
}

func SelectPreviousHiringJob(hsHnId, hnTime uint64) (*HiringJob, error) {
	var hj HiringJob
	sql := `SELECT hn_id, seen, saved, text, time
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=? and time > ?
            ORDER BY time ASC
            Limit 1`
	if err := db.Get(&hj, sql, hsHnId, jobStatusOk, hnTime); err != nil {
		return &hj, err
	}

	return &hj, nil
}

func GetMinMaxHiringJobTime(hsHnId uint64) (*HiringJobTime, error) {
	var t HiringJobTime
	sql := `SELECT min(time) as min, max(time) as max
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=?`
	if err := db.Get(&t, sql, hsHnId, jobStatusOk); err != nil {
		return &t, err
	}

	return &t, nil
}

func SetHiringJobAsSeen(hnId uint64) error {
	_, err := db.Exec(`UPDATE hiring_job set seen=1 where hn_id=?`, hnId)
	if err != nil {
		return err
	}
	return nil
}
