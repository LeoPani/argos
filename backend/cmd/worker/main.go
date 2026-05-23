// argos-worker is the Argos background worker.
//
// Responsibilities (planned, by phase):
//   Phase 2: download INPI RPI XML, stream-parse, classify, persist
//   Phase 2: pull Lens.org metadata for INPI patents
//   Phase 4: batch pending Proofs into Merkle roots and submit to chain
//   Phase 6: pull Web of Science records
//
// This file wires shared infrastructure (config, db, AI). The actual
// pipelines live in internal/worker/<source>/ and will be invoked here
// once their packages are implemented.
package main

import (
	"context"
	"fmt"
	"log/slog"
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
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker: fatal", "err", err)
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
	log.Info("argos-worker starting",
		"ai_url", cfg.AIBertURL,
		"concurrency", cfg.WorkerConcurrency,
	)

	// Root context cancelled on SIGINT/SIGTERM.
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

	// ---- AI layer: BERT (active) + LLM (stub) wired via Composite ----
	bertClient := bert.New(bert.Config{
		BaseURL:        cfg.AIBertURL,
		RequestTimeout: cfg.AIBertTimeout,
		MaxRetries:     cfg.AIBertMaxRetries,
		RetryBackoff:   500 * time.Millisecond,
	})
	llmClient := llm.New(llm.DefaultConfig())
	aiSvc := ai.NewComposite(bertClient, llmClient)
	log.Info("ai layer ready", "adapters", "bert+llm-stub")

	// The actual pipeline goes here in Phase 2:
	//
	//   pipe := inpipatents.NewPipeline(repo, aiSvc, inpipatents.Config{
	//       Concurrency:  cfg.WorkerConcurrency,
	//       DownloadDir:  cfg.WorkerINPIDownloadDir,
	//   })
	//   return pipe.Run(ctx)
	//
	// For now we use aiSvc with a smoke probe so the variable isn't
	// flagged as unused, and so we can verify connectivity to the
	// FastAPI service from inside the worker process at boot.
	probeClassifier(ctx, log, aiSvc)

	log.Info("worker ready; awaiting shutdown signal")

	// Block on shutdown signal. Phase 2 replaces this with a worker pool
	// driven by the pipeline above; the graceful shutdown shape stays
	// the same.
	<-ctx.Done()
	log.Info("shutdown signal received")

	// Drain window so future pipeline workers can finish in-flight items.
	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	log.Info("stopped cleanly")
	return nil
}

// probeClassifier tries one harmless classification call at startup to
// surface connectivity issues with the FastAPI service early. Failure
// is logged as a WARN, not an error: the worker can still process other
// future jobs (e.g. Lens.org ingestion) without the classifier.
func probeClassifier(ctx context.Context, log *slog.Logger, aiSvc ai.AIService) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const sample = "Sistema e método para classificação automatizada de patentes."
	cat, err := aiSvc.ClassifyPatent(probeCtx, sample)
	if err != nil {
		log.Warn("classifier probe failed (worker will run without classification)", "err", err)
		return
	}
	log.Info("classifier probe ok", "sample_category", cat)
}
