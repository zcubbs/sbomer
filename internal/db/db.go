package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connStr string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

func (db *DB) LogOperation(ctx context.Context, projectID int, operation string, status string, error string) error {
	query := `
		INSERT INTO operations (project_id, operation, status, error)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.pool.Exec(ctx, query, projectID, operation, status, error)
	if err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}
	return nil
}
