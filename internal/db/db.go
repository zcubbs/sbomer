package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zcubbs/sbomer/internal/models"
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
		INSERT INTO operations (project_id, operation, status, error_message)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.pool.Exec(ctx, query, projectID, operation, status, error)
	if err != nil {
		return fmt.Errorf("failed to log operation: %w", err)
	}
	return nil
}

// SaveSBOM saves or updates an SBOM for a project
func (db *DB) SaveSBOM(ctx context.Context, sbom *models.SBOM) error {
	query := `
		INSERT INTO sbom (
			project_uid,
			name,
			path,
			topics,
			sbom_data,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, CURRENT_TIMESTAMP
		)
		ON CONFLICT (project_uid) DO UPDATE SET
			name = EXCLUDED.name,
			path = EXCLUDED.path,
			topics = EXCLUDED.topics,
			sbom_data = EXCLUDED.sbom_data,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.pool.Exec(ctx, query,
		sbom.ProjectUID,
		sbom.Name,
		sbom.Path,
		sbom.Topics,
		sbom.SBOMData,
	)
	if err != nil {
		return fmt.Errorf("failed to save SBOM: %w", err)
	}

	return nil
}

// GetSBOM retrieves an SBOM by project UID
func (db *DB) GetSBOM(ctx context.Context, projectUID int) (*models.SBOM, error) {
	query := `
		SELECT
			project_uid,
			name,
			path,
			topics,
			sbom_data,
			created_at,
			updated_at
		FROM sbom
		WHERE project_uid = $1
	`

	sbom := &models.SBOM{}
	err := db.pool.QueryRow(ctx, query, projectUID).Scan(
		&sbom.ProjectUID,
		&sbom.Name,
		&sbom.Path,
		&sbom.Topics,
		&sbom.SBOMData,
		&sbom.CreatedAt,
		&sbom.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get SBOM: %w", err)
	}

	return sbom, nil
}
