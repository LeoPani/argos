// argos-reanalyze-ufop reprocessa todas as oportunidades UFOP existentes no
// banco pelo classificador atual (Groq se GROQ_API_KEY setado, senão
// heurística v2). Útil pra:
//
//   - Aplicar uma nova versão do classificador retroativamente
//   - Limpar falsos positivos (ex: Direito do Trabalho → IPC H)
//   - Migrar dataset pré-existente pros campos novos (is_patentable, etc)
//
// Preserva o `status` manual (reviewed/converted/dismissed) — apenas re-classifica.
//
// Usage:
//
//	go run ./cmd/reanalyze-ufop                  # todos
//	go run ./cmd/reanalyze-ufop --dry-run        # mostra mudanças sem persistir
//	go run ./cmd/reanalyze-ufop --limit 10       # primeiros 10 (smoke test)
//	go run ./cmd/reanalyze-ufop --only-direito   # só dept Direito (debug falso positivo)
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/ai/bert"
	"github.com/LeoPani/argos/backend/internal/ai/groqclassifier"
	"github.com/LeoPani/argos/backend/internal/ai/llm"
	"github.com/LeoPani/argos/backend/internal/config"
	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/platform/database"
	"github.com/LeoPani/argos/backend/internal/platform/logger"
	pg "github.com/LeoPani/argos/backend/internal/repository/postgres"
	"github.com/LeoPani/argos/backend/internal/worker/ufop"
)

func main() {
	if err := run(); err != nil {
		slog.Error("reanalyze-ufop: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dryRun     bool
		limit      int
		onlyDept   string
		onlyLevel  string
	)
	flag.BoolVar(&dryRun, "dry-run", false, "Mostra mudanças sem persistir")
	flag.IntVar(&limit, "limit", 0, "Limite de records (0 = todos)")
	flag.StringVar(&onlyDept, "only-dept", "", "Substring case-insensitive pra filtrar dept (ex: 'direito')")
	flag.StringVar(&onlyLevel, "only-level", "", "Filtra por opportunity_level atual ('high', 'medium', 'low')")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxLifetime: cfg.DBConnMaxLifetime,
		ConnMaxIdleTime: 5 * time.Minute,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer db.Close()

	// Wire AI layer igual a main API
	bertClient := bert.New(bert.Config{BaseURL: cfg.AIBertURL, RequestTimeout: cfg.AIBertTimeout, MaxRetries: cfg.AIBertMaxRetries, RetryBackoff: 500 * time.Millisecond})
	llmClient := llm.New(llm.DefaultConfig())
	aiSvc := ai.NewComposite(bertClient, llmClient)

	analyzer := ufop.NewAnalyzer(aiSvc).WithLogger(log)
	if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
		gc := groqclassifier.New(groqclassifier.Config{APIKey: groqKey, RequestTimeout: 20 * time.Second})
		if gc != nil {
			analyzer = analyzer.WithGroq(gc)
			log.Info("groq classifier enabled", "model", gc.Model())
		}
	} else {
		log.Warn("GROQ_API_KEY não setado — usando só heurística v2")
	}

	repo := pg.NewUFOPRepo(db)

	// Lista TUDO (status=" " para não filtrar)
	all, err := repo.List(ctx, domain.UFOPFilter{Limit: 1000})
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	log.Info("loaded opportunities", "count", len(all))

	processed, changed, errs := 0, 0, 0
	for _, opp := range all {
		// Aplicar filtros
		if onlyDept != "" && !strings.Contains(strings.ToLower(opp.Department), strings.ToLower(onlyDept)) {
			continue
		}
		if onlyLevel != "" && string(opp.Level) != onlyLevel {
			continue
		}
		if limit > 0 && processed >= limit {
			break
		}
		processed++

		// Reanalisar
		nopp, err := analyzer.Analyze(ctx, ufop.AnalyzeInput{
			Title:      opp.Title,
			Abstract:   opp.Abstract,
			Authors:    opp.Authors,
			ExternalID: opp.ExternalID,
			Source:     opp.Source,
			URL:        opp.URL,
			PublishedAt: opp.PublishedAt,
			Department: opp.Department,
		})
		if err != nil {
			errs++
			log.Error("analyze failed", "ext_id", opp.ExternalID, "err", err)
			continue
		}

		// Preserva status manual
		nopp.Status = opp.Status
		nopp.PublicationID = opp.PublicationID

		// Detectar mudança significativa
		if changedFields := diffOpp(&opp, nopp); changedFields != "" {
			changed++
			fmt.Printf("[%d] %s · %s\n  before: lvl=%s ipc=%s score=%.1f\n  after:  lvl=%s ipc=%s score=%.1f  patentable=%s\n  changes: %s\n  why: %s\n\n",
				opp.ID, truncate(opp.Title, 80), opp.Department,
				opp.Level, opp.IPCSuggestion, opp.PIScore,
				nopp.Level, nopp.IPCSuggestion, nopp.PIScore,
				formatPatentable(nopp.IsPatentable),
				changedFields, truncate(nopp.Rationale, 200))
		}

		if !dryRun {
			if err := repo.Upsert(ctx, nopp); err != nil {
				errs++
				log.Error("upsert failed", "id", opp.ID, "err", err)
			}
		}
	}

	verb := "would update"
	if !dryRun {
		verb = "updated"
	}
	fmt.Printf("\nProcessed: %d · Changed (%s): %d · Errors: %d\n", processed, verb, changed, errs)
	if dryRun {
		fmt.Println("DRY-RUN: nada foi persistido. Rode sem --dry-run para aplicar.")
	}
	return nil
}

func diffOpp(before, after *domain.UFOPOpportunity) string {
	var diffs []string
	if before.Level != after.Level {
		diffs = append(diffs, fmt.Sprintf("level=%s→%s", before.Level, after.Level))
	}
	if before.IPCCategory != after.IPCCategory {
		diffs = append(diffs, fmt.Sprintf("ipc=%d→%d", before.IPCCategory, after.IPCCategory))
	}
	if absDiff(before.PIScore, after.PIScore) > 0.5 {
		diffs = append(diffs, fmt.Sprintf("score=%.1f→%.1f", before.PIScore, after.PIScore))
	}
	if before.ClassifierVersion != after.ClassifierVersion {
		diffs = append(diffs, fmt.Sprintf("classifier=%s→%s",
			defaultStr(before.ClassifierVersion, "?"), after.ClassifierVersion))
	}
	if (before.IsPatentable == nil) != (after.IsPatentable == nil) ||
		(before.IsPatentable != nil && after.IsPatentable != nil && *before.IsPatentable != *after.IsPatentable) {
		diffs = append(diffs, fmt.Sprintf("patentable=%s→%s",
			formatPatentable(before.IsPatentable), formatPatentable(after.IsPatentable)))
	}
	return strings.Join(diffs, ", ")
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

func formatPatentable(p *bool) string {
	if p == nil {
		return "null"
	}
	if *p {
		return "true"
	}
	return "false"
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
