// Package logger sets up the application's structured logger.
// Uses log/slog from the Go standard library (1.21+), which gives us
// JSON output for production and pluggable handlers for testing.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds logger settings, populated from environment variables.
type Config struct {
	Level  string // "debug" | "info" | "warn" | "error"  (default: info)
	Format string // "json"  | "text"                      (default: json)
}

// New builds a configured *slog.Logger AND registers it as the default.
// Calling slog.Info(...) anywhere in the app picks up this logger.
func New(cfg Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(cfg.Level)}

	var handler slog.Handler
	if strings.EqualFold(cfg.Format, "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// parseLevel maps a string to a slog.Level. Unknown values fall back to info.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
