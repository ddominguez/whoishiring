package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type HnStory struct {
	HnId  uint64 `db:"hn_id"`
	Title string `db:"title"`
	Time  uint64 `db:"time"`
}

type HnJob struct {
	HnId   uint64 `db:"hn_id"`
	Text   string `db:"text"`
	Time   uint64 `db:"time"`
	Seen   uint8  `db:"seen"`
	Saved  uint8  `db:"saved"`
	Status uint8  `db:"status"`
}

// TransformedText returns HnJob Text with updated html.
func (j *HnJob) TransformedText() string {
	var result string

	jobTxt := strings.TrimSpace(j.Text)
	postedLink := fmt.Sprintf(
		`<p class="my-2"><a href="https://news.ycombinator.com/item?id=%d">Posted: %s</a></p>`,
		j.HnId,
		time.Unix(int64(j.Time), 0),
	)

	if jobTxt == "" {
		return postedLink
	}

	for lineVal := range strings.SplitSeq(jobTxt, "\n") {
		for parVal := range strings.SplitSeq(lineVal, "<p>") {
			result = result + fmt.Sprintf(`<p class="my-2">%s</p>`, parVal)
		}
	}

	result = result + postedLink
	return result
}

type HNStore struct {
	db *sqlx.DB
}

// CreateStory inserts a new WhoIsHiring story into the db.
func (s *HNStore) CreateStory(story *HnStory) error {
	query := `INSERT INTO hiring_story (hn_id, title, time)
						VALUES (?, ?, ?)`

	_, err := s.db.Exec(query, story.HnId, story.Title, story.Time)
	if err != nil {
		return fmt.Errorf("failed to create hiring story: %w", err)
	}

	return nil
}

// CreateJob inserts a new WhoIsHiring job into the db.
func (s *HNStore) CreateJob(job *HnJob, hnStoryId uint64) error {
	query := `INSERT INTO hiring_job (hn_id, hiring_story_hn_id, text, time, status)
						VALUES (?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, job.HnId, hnStoryId, job.Text, job.Time, job.Status)
	if err != nil {
		return fmt.Errorf("failed to create hiring job: %w", err)
	}

	return nil
}

// GetJobIdsByStoryId retrieves jobs ids for a given story.
func (s *HNStore) GetJobIdsByStoryId(hnStoryId uint64) (map[uint64]bool, error) {
	query := `SELECT hn_id FROM hiring_job WHERE hiring_story_hn_id=?`

	rows, err := s.db.Query(query, hnStoryId)
	if err != nil {
		return nil, fmt.Errorf("failed to select hiring job IDs: %w", err)
	}
	defer rows.Close()

	ids := make(map[uint64]bool)
	for rows.Next() {
		var id uint64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan hiring job ID: %w", err)
		}
		ids[id] = true
	}

	return ids, nil
}

// GetLatestStory retrieves the latest hiring story from the database.
func (s *HNStore) GetLatestStory() (*HnStory, error) {
	var story HnStory
	query := "SELECT hn_id, title, time FROM hiring_story ORDER BY time DESC LIMIT 1"

	err := s.db.Get(&story, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get latest hiring story: %w", err)
	}

	return &story, nil
}

// GetMinMaxJobIDs retrieves the min and max job IDs for a hiring story.
func (s *HNStore) GetMinMaxJobIDs(hnStoryId uint64) (uint64, uint64, error) {
	var result struct {
		Min uint64 `db:"min"`
		Max uint64 `db:"max"`
	}

	query := `SELECT min(hn_id) as min, max(hn_id) as max
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=?`
	if err := s.db.Get(&result, query, hnStoryId, jobStatusOk); err != nil {
		return 0, 0, fmt.Errorf("failed to get min/max hiring job IDs: %w", err)
	}

	return result.Min, result.Max, nil
}

// GetFirstJob retrieves first WhoIsHiring job.
func (s *HNStore) GetFirstJob(hnStoryId uint64) (*HnJob, error) {
	var job HnJob

	query := `SELECT hn_id, seen, saved, text, time
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=?
            ORDER BY hn_id DESC
            Limit 1`
	if err := s.db.Get(&job, query, hnStoryId, jobStatusOk); err != nil {
		return nil, fmt.Errorf("failed to select first hiring job: %w", err)
	}

	return &job, nil
}

// GetNextJobById retrieves the next WhoIsHiring job.
func (s *HNStore) GetNextJobById(hnStoryId, hnJobId uint64) (*HnJob, error) {
	var job HnJob

	query := `SELECT hn_id, seen, saved, text, time
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=? and hn_id < ?
            ORDER BY hn_id DESC
            Limit 1`
	if err := s.db.Get(&job, query, hnStoryId, jobStatusOk, hnJobId); err != nil {
		return nil, fmt.Errorf("failed to select next hiring job: %w", err)
	}

	return &job, nil
}

// GetNextJobById retrieves the previous WhoIsHiring job.
func (s *HNStore) GetPreviousJobById(hnStoryId, hnJobId uint64) (*HnJob, error) {
	var job HnJob

	query := `SELECT hn_id, seen, saved, text, time
            FROM hiring_job
            WHERE hiring_story_hn_id=? and status=? and hn_id > ?
            ORDER BY hn_id ASC
            Limit 1`
	if err := s.db.Get(&job, query, hnStoryId, jobStatusOk, hnJobId); err != nil {
		return nil, fmt.Errorf("failed to select previous hiring job: %w", err)
	}

	return &job, nil
}

// SetJobAsSeen marks a job as seen.
func (s *HNStore) SetJobAsSeen(hnJobId uint64) error {
	res, err := s.db.Exec(`UPDATE hiring_job set seen=1 where hn_id=?`, hnJobId)
	if err != nil {
		return fmt.Errorf("failed to set hiring job as seen: %w", err)
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if affectedRows == 0 {
		return errors.New("zero rows updated")
	}

	return nil
}

// NewHNStore creates a new HNStore.
func NewHNStore(db *sqlx.DB) *HNStore {
	return &HNStore{db: db}
}
