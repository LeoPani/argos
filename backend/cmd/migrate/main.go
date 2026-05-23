// argos-migrate is a tiny SQL migration runner.
//
// It tracks applied migrations in a `schema_migrations` table and applies
// any *.up.sql files from ./migrations that haven't run yet, in lexical
// order (so 0001_*, 0002_*, ... is exactly what you want).
//
// Usage:
//
//	go run ./cmd/migrate           # apply pending migrations
//	go run ./cmd/migrate status    # show what's applied
//
// For Phase 1 this is plenty. When the project grows we'll swap to
// golang-migrate/migrate without changing the migration files themselves.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/config"
	"github.com/LeoPani/argos/backend/internal/platform/database"
	"github.com/LeoPani/argos/backend/internal/platform/logger"
)

const migrationsDir = "./migrations"

func main() {
	if err := run(); err != nil {
		slog.Error("migrate: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Minute,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		return applyPending(ctx, db)
	case "status":
		return printStatus(ctx, db)
	default:
		return fmt.Errorf("unknown command %q (use 'up' or 'status')", cmd)
	}
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	const q = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`
	_, err := db.ExecContext(ctx, q)
	return err
}

// listMigrationFiles returns sorted *.up.sql filenames in migrationsDir.
func listMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", migrationsDir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// versionFromFile strips ".up.sql" to use the base name as version key.
func versionFromFile(name string) string {
	return strings.TrimSuffix(name, ".up.sql")
}

func applyPending(ctx context.Context, db *sql.DB) error {
	files, err := listMigrationFiles()
	if err != nil {
		return err
	}

	applied, err := loadAppliedVersions(ctx, db)
	if err != nil {
		return err
	}

	any := false
	for _, file := range files {
		version := versionFromFile(file)
		if applied[version] {
			continue
		}
		any = true
		slog.Info("applying migration", "version", version)

		fullPath := filepath.Join(migrationsDir, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", fullPath, err)
		}

		if err := runInTx(ctx, db, string(content), version); err != nil {
			return fmt.Errorf("apply %s: %w", version, err)
		}
		slog.Info("applied", "version", version)
	}

	if !any {
		slog.Info("no pending migrations; database is up to date")
	}
	return nil
}

// runInTx executes the migration SQL and inserts the version row in a
// single transaction so either everything sticks or nothing does.
func runInTx(ctx context.Context, db *sql.DB, sqlText, version string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, sqlText); err != nil {
		return fmt.Errorf("exec sql: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("record version: %w", err)
	}
	return tx.Commit()
}

func loadAppliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query applied: %w", err)
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out[v] = true
	}
	return out, rows.Err()
}

func printStatus(ctx context.Context, db *sql.DB) error {
	files, err := listMigrationFiles()
	if err != nil {
		return err
	}
	applied, err := loadAppliedVersions(ctx, db)
	if err != nil {
		return err
	}

	fmt.Printf("%-40s %s\n", "VERSION", "STATUS")
	fmt.Printf("%-40s %s\n", strings.Repeat("-", 40), "------")
	for _, f := range files {
		v := versionFromFile(f)
		status := "pending"
		if applied[v] {
			status = "applied"
		}
		fmt.Printf("%-40s %s\n", v, status)
	}
	return nil
}
