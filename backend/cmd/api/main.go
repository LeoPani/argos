// argos-api is the Argos REST API server.
//
// Boot order:
//   1. Load config
//   2. Initialize structured logger
//   3. Open PostgreSQL pool + verify connectivity
//   4. Wire AI layer (BERT real + LLM stub) via the Composite
//   5. Construct repository + service
//   6. Build the HTTP router and start serving
//   7. Block on SIGINT/SIGTERM and shut down gracefully
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/ai/bert"
	"github.com/LeoPani/argos/backend/internal/ai/llm"
	"github.com/LeoPani/argos/backend/internal/config"
	"github.com/LeoPani/argos/backend/internal/platform/database"
	"github.com/LeoPani/argos/backend/internal/platform/logger"
	"github.com/LeoPani/argos/backend/internal/repository/postgres"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/internal/transport/httpapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	// --- Config ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// --- Logger ---
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})
	log.Info("argos-api starting", "addr", cfg.APIAddr)

	// Root context cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- Database ---
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

	// --- AI layer (BERT real + LLM stub, composed into one AIService) ---
	bertClient := bert.New(bert.Config{
		BaseURL:        cfg.AIBertURL,
		RequestTimeout: cfg.AIBertTimeout,
		MaxRetries:     cfg.AIBertMaxRetries,
		RetryBackoff:   500 * time.Millisecond,
	})
	llmClient := llm.New(llm.DefaultConfig())
	aiSvc := ai.NewComposite(bertClient, llmClient)
	log.Info("ai layer ready", "bert_url", cfg.AIBertURL)

	// --- Repository + Service ---
	patentRepo := postgres.NewPatentRepo(db)
	patentSvc := service.NewPatentService(patentRepo, aiSvc)
	log.Info("services wired")

	// --- Router ---
	handler := httpapi.NewRouter(httpapi.Deps{
		DB:            db,
		PatentService: patentSvc,
	})

	// --- HTTP server ---
	server := &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.APIReadTimeout,
		WriteTimeout:      cfg.APIWriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.APIAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server: %w", err)
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	// Graceful shutdown: up to 15s for in-flight requests.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("stopped cleanly")
	return nil
}
