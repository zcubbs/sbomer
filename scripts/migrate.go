package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	mig "github.com/zcubbs/sbomer/migrations"
)

func main() {
	// Get database connection string from environment
	dbURL := os.Getenv("SBOMER_DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/sbomer?sslmode=disable"
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Create migrations table if it doesn't exist
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Unable to create migrations table: %v", err)
	}

	// Get list of applied migrations
	appliedMigrations := make(map[string]bool)
	rows, err := pool.Query(ctx, "SELECT name FROM migrations")
	if err != nil {
		log.Fatalf("Unable to query migrations: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			log.Fatalf("Unable to scan migration row: %v", err)
		}
		appliedMigrations[name] = true
	}

	// Read and sort migration files
	entries, err := mig.FS.ReadDir(".")
	if err != nil {
		log.Fatalf("Unable to read migrations directory: %v", err)
	}

	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}
	sort.Strings(migrations)

	// Group migrations by version
	migrationGroups := make(map[string][]string)
	for _, migration := range migrations {
		base := strings.TrimSuffix(migration, path.Ext(migration))
		if strings.HasSuffix(base, ".up") || strings.HasSuffix(base, ".down") {
			version := strings.TrimSuffix(base, ".up")
			version = strings.TrimSuffix(version, ".down")
			migrationGroups[version] = append(migrationGroups[version], migration)
		}
	}

	// Sort versions
	var versions []string
	for version := range migrationGroups {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	// Apply migrations in order
	for _, version := range versions {
		files := migrationGroups[version]
		for _, file := range files {
			if strings.HasSuffix(file, ".down.sql") {
				continue // Skip down migrations
			}

			if appliedMigrations[file] {
				log.Printf("Migration %s already applied, skipping", file)
				continue
			}

			log.Printf("Applying migration: %s", file)

			// Read migration file
			migration, err := mig.FS.ReadFile(file)
			if err != nil {
				log.Fatalf("Unable to read migration file %s: %v", file, err)
			}

			// Execute migration in a transaction
			tx, err := pool.Begin(ctx)
			if err != nil {
				log.Fatalf("Unable to start transaction: %v", err)
			}

			if _, err := tx.Exec(ctx, string(migration)); err != nil {
				tx.Rollback(ctx)
				log.Fatalf("Unable to execute migration %s: %v", file, err)
			}

			// Record migration
			if _, err := tx.Exec(ctx, "INSERT INTO migrations (name) VALUES ($1)", file); err != nil {
				tx.Rollback(ctx)
				log.Fatalf("Unable to record migration %s: %v", file, err)
			}

			if err := tx.Commit(ctx); err != nil {
				log.Fatalf("Unable to commit transaction: %v", err)
			}

			log.Printf("Successfully applied migration: %s", file)
		}
	}

	fmt.Println("All migrations completed successfully")
}
