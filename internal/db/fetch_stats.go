package db

import (
	"context"
	"fmt"

	"github.com/zcubbs/sbomer/internal/db/models"
)

// SaveFetchStats saves fetch statistics to the database
func (db *DB) SaveFetchStats(ctx context.Context, stats *models.FetchStats) error {
	query := `
		INSERT INTO fetch_stats (
			projects_count,
			batch_size,
			duration_seconds,
			created_at
		) VALUES (
			$1, $2, $3, $4
		) RETURNING id`

	err := db.pool.QueryRow(ctx, query,
		stats.ProjectsCount,
		stats.BatchSize,
		stats.Duration,
		stats.CreatedAt,
	).Scan(&stats.ID)

	if err != nil {
		return fmt.Errorf("failed to save fetch stats: %w", err)
	}

	return nil
}
