// Package config loads runtime configuration from environment variables.
// Everything the app needs to know to start up (DB DSN, ports, AI URLs,
// timeouts) is centralized here and passed downstream as a typed struct.
//
// Loading priority:
//  1. Real environment variables (os.Getenv)
//  2. Hard-coded fallback defaults (suitable for local dev)
//
// Production deployments populate env vars from secrets managers, .env
// files loaded by docker-compose, or systemd unit Environment= directives.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config aggregates every runtime knob the application reads.
// See backend/.env.example for the exhaustive list with comments.
type Config struct {
	// --- Database ---
	DatabaseURL       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration

	// --- HTTP server ---
	APIAddr         string
	APIReadTimeout  time.Duration
	APIWriteTimeout time.Duration

	// --- AI: BERT classifier (Phase 1) ---
	AIBertURL        string
	AIBertTimeout    time.Duration
	AIBertMaxRetries int

	// --- Worker (Phase 1+) ---
	WorkerConcurrency     int
	WorkerINPIDownloadDir string

	// --- Logging ---
	LogLevel  string
	LogFormat string
}

// Load reads environment variables and returns a populated Config.
// Returns an error only for malformed values (non-numeric where a number
// is expected, unparseable duration, etc.); missing variables fall back
// to safe defaults so local dev works with zero setup.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://argos:argos_dev@localhost:5432/argos?sslmode=disable"),
		APIAddr:               getEnv("API_ADDR", ":8080"),
		AIBertURL:             getEnv("AI_BERT_URL", "http://localhost:8000"),
		WorkerINPIDownloadDir: getEnv("WORKER_INPI_DOWNLOAD_DIR", "/tmp/argos/rpi"),
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		LogFormat:             getEnv("LOG_FORMAT", "json"),
	}

	var err error

	if cfg.DBMaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", 25); err != nil {
		return nil, err
	}
	if cfg.DBMaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", 10); err != nil {
		return nil, err
	}
	if cfg.DBConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute); err != nil {
		return nil, err
	}

	if cfg.APIReadTimeout, err = getEnvDuration("API_READ_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if cfg.APIWriteTimeout, err = getEnvDuration("API_WRITE_TIMEOUT", 30*time.Second); err != nil {
		return nil, err
	}

	if cfg.AIBertTimeout, err = getEnvDuration("AI_BERT_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if cfg.AIBertMaxRetries, err = getEnvInt("AI_BERT_MAX_RETRIES", 2); err != nil {
		return nil, err
	}

	if cfg.WorkerConcurrency, err = getEnvInt("WORKER_CONCURRENCY", 8); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnv returns the env var value or fallback if missing/empty.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// getEnvInt parses the env var as an int. Empty -> fallback.
func getEnvInt(key string, fallback int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid int for %s=%q: %w", key, raw, err)
	}
	return n, nil
}

// getEnvDuration parses the env var as a time.Duration (e.g. "30s", "5m").
func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("config: invalid duration for %s=%q: %w", key, raw, err)
	}
	return d, nil
}
