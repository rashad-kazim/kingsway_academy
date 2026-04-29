package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`); err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		version := migrationVersion(name)
		if version == "" {
			return fmt.Errorf("invalid migration filename: %s", name)
		}

		applied, err := migrationApplied(ctx, pool, version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		upSQL, err := gooseUpSQL(string(body))
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, upSQL); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}

func migrationApplied(ctx context.Context, pool *pgxpool.Pool, version string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists)
	return exists, err
}

func migrationVersion(name string) string {
	version, _, ok := strings.Cut(name, "_")
	if !ok {
		return ""
	}

	return version
}

func gooseUpSQL(body string) (string, error) {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"

	upIndex := strings.Index(body, upMarker)
	if upIndex < 0 {
		return "", fmt.Errorf("missing %q marker", upMarker)
	}

	upSQL := body[upIndex+len(upMarker):]
	if downIndex := strings.Index(upSQL, downMarker); downIndex >= 0 {
		upSQL = upSQL[:downIndex]
	}
	upSQL = strings.TrimSpace(upSQL)
	if upSQL == "" {
		return "", fmt.Errorf("empty up migration")
	}

	return upSQL, nil
}
