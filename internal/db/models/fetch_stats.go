package models

import (
	"time"
)

// FetchStats represents statistics from GitLab project fetches
type FetchStats struct {
	ID            int64     `db:"id"`
	ProjectsCount int       `db:"projects_count"`
	BatchSize     int       `db:"batch_size"`
	Duration      float64   `db:"duration_seconds"`
	CreatedAt     time.Time `db:"created_at"`
}
