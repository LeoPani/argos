// argos-api is the Argos REST API server.
//
// Responsibilities at this stage:
//   - Load env-based config
//   - Initialize structured logger
//   - Open the PostgreSQL pool and verify connectivity
//   - Expose GET /health that reports DB liveness
//   - Shut down gracefully on SIGINT/SIGTERM
//
// REST resource handlers (patents, trademarks, disputes...) will be
// mounted as the project advances. The wiring style chosen here scales
// cleanly: we add packages, never touch this file's structure.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LeoPani/argos/backend/internal/config"
	"github.com/LeoPani/argos/backend/internal/platform/database"
	"github.com/LeoPani/argos/backend/internal/platform/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// ---- Config ----
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ---- Logger ----
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})
	log.Info("argos-api starting", "addr", cfg.APIAddr)

	// Root context cancelled on SIGINT/SIGTERM — propagates everywhere.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- Database ----
	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: 5 * time.Minute,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer db.Close()
	log.Info("database connected")

	// ---- Router ----
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(db))

	// ---- HTTP server with sensible timeouts ----
	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.APIReadTimeout,
		WriteTimeout:      cfg.APIWriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	// Run server in a goroutine so we can listen for shutdown signal.
	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.APIAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Block until either the server crashes or a signal arrives.
	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server: %w", err)
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	// Graceful shutdown: give in-flight requests up to 15s to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("stopped cleanly")
	return nil
}

// healthHandler returns 200 OK only if the DB pool answers within 2s.
// In load balancers / k8s liveness probes this is what determines whether
// the pod is considered healthy.
func healthHandler(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// pinger is a tiny interface satisfied by *sql.DB. Letting the handler
// depend on the interface (not the concrete type) makes it trivial to
// unit-test with a fake.
type pinger interface {
	PingContext(ctx context.Context) error
}
