// Package database centralizes PostgreSQL connection setup. Higher layers
// receive a *sql.DB and never construct one themselves; this keeps the
// pool configuration (and lib/pq import) in a single, well-known place.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // postgres driver registration (side-effect import)
)

// Config holds every tunable for the connection pool. Values come from
// the application Config (environment variables) — never hard-code.
type Config struct {
	DSN             string        // postgres://user:pass@host:port/db?sslmode=...
	MaxOpenConns    int           // hard ceiling on concurrent connections
	MaxIdleConns    int           // pool of warm, ready-to-use connections
	ConnMaxLifetime time.Duration // recycle connections to avoid stale TCP
	ConnMaxIdleTime time.Duration // close idle conns to release DB resources
	PingTimeout     time.Duration // bounds the initial connectivity check
}

// DefaultConfig returns production-sensible defaults. Override per env.
func DefaultConfig(dsn string) Config {
	return Config{
		DSN:             dsn,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		PingTimeout:     5 * time.Second,
	}
}

// New opens a configured *sql.DB and verifies connectivity with a
// bounded ping. The caller owns the handle and MUST Close() on shutdown.
//
// Note: sql.Open does NOT establish a connection; it only validates
// arguments. The PingContext below is what actually proves we can talk
// to PostgreSQL — failing fast at boot is much friendlier than failing
// on the first user request.
func New(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("database: DSN is required")
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		// Close to avoid leaking the half-initialized pool.
		_ = db.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	return db, nil
}
