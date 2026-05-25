// argos-harvest-ufop colhe publicações reais do Repositório Institucional UFOP
// via OAI-PMH e roda o analyzer para detectar oportunidades de PI.
//
// Foco inicial:
//   - Direito (DEDIR/EDTM/PPG Direito)
//   - Engenharia de Minas (DEMIN/Escola de Minas)
//
// Usage:
//
//	go run ./cmd/harvest-ufop                 # ambos departamentos (limite 50)
//	go run ./cmd/harvest-ufop direito 100     # só Direito, até 100 records
//	go run ./cmd/harvest-ufop minas 100       # só Eng Minas
//
// O analyzer já é tolerante a BERT offline — sem AI roda em heurística pura.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/ai/bert"
	"github.com/LeoPani/argos/backend/internal/ai/llm"
	"github.com/LeoPani/argos/backend/internal/config"
	"github.com/LeoPani/argos/backend/internal/platform/database"
	"github.com/LeoPani/argos/backend/internal/platform/logger"
	pg "github.com/LeoPani/argos/backend/internal/repository/postgres"
	"github.com/LeoPani/argos/backend/internal/service"
	"github.com/LeoPani/argos/backend/internal/worker/ufop"
)

func main() {
	if err := run(); err != nil {
		slog.Error("harvest-ufop: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DSN:             cfg.DatabaseURL,
		MaxOpenConns:    4,
		MaxIdleConns:    2,
		ConnMaxLifetime: 10 * time.Minute,
		PingTimeout:     5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer db.Close()

	// AI tolerante: se BERT offline, classificação fica em -1 (analyzer já trata).
	bertClient := bert.New(bert.Config{
		BaseURL:        cfg.AIBertURL,
		RequestTimeout: cfg.AIBertTimeout,
		MaxRetries:     1,
		RetryBackoff:   200 * time.Millisecond,
	})
	llmClient := llm.New(llm.DefaultConfig())
	aiSvc := ai.NewComposite(bertClient, llmClient)

	pubRepo  := pg.NewPublicationRepo(db)
	ufopRepo := pg.NewUFOPRepo(db)

	oaiClient   := ufop.NewOAIClient(log)
	portalScrap := ufop.NewPortalScraper(log)
	analyzer    := ufop.NewAnalyzer(aiSvc)
	svc := service.NewUFOPService(ufopRepo, pubRepo, oaiClient, portalScrap, analyzer, log)

	// Parse args
	target := "all"
	limit := 50
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil {
			limit = n
		}
	}

	sets := []struct{ name, spec string }{}
	switch target {
	case "direito":
		sets = append(sets,
			struct{ name, spec string }{"DEDIR — Departamento de Direito (graduação)", ufop.UFOPSetDepDireito},
			struct{ name, spec string }{"EDTM — Escola de Direito, Turismo, Museologia", ufop.UFOPSetEscolaDireito},
			struct{ name, spec string }{"PPG Direito — pós-graduação stricto sensu",     ufop.UFOPSetPPGDireito},
		)
	case "minas":
		sets = append(sets,
			struct{ name, spec string }{"DEMIN — Engenharia de Minas (graduação)",       ufop.UFOPSetDepEngMinas},
			struct{ name, spec string }{"EM — Escola de Minas (umbrella)",                ufop.UFOPSetEscolaMinas},
			struct{ name, spec string }{"PPGEM — PPG em Engenharia Mineral",              ufop.UFOPSetPPGEngMineral},
			struct{ name, spec string }{"DEGEO — Geologia (complementar)",                ufop.UFOPSetDepGeologia},
		)
	case "all", "":
		sets = append(sets,
			// Direito
			struct{ name, spec string }{"DEDIR — Direito (grad)",                  ufop.UFOPSetDepDireito},
			struct{ name, spec string }{"EDTM — Escola de Direito (umbrella)",     ufop.UFOPSetEscolaDireito},
			struct{ name, spec string }{"PPG Direito",                              ufop.UFOPSetPPGDireito},
			// Minas
			struct{ name, spec string }{"DEMIN — Eng. Minas (grad)",                ufop.UFOPSetDepEngMinas},
			struct{ name, spec string }{"EM — Escola de Minas (umbrella)",          ufop.UFOPSetEscolaMinas},
			struct{ name, spec string }{"PPGEM — PPG Engenharia Mineral",           ufop.UFOPSetPPGEngMineral},
			struct{ name, spec string }{"DEGEO — Geologia",                          ufop.UFOPSetDepGeologia},
		)
	default:
		return fmt.Errorf("target desconhecido: %q (use direito|minas|all)", target)
	}

	totalFetched := 0
	totalUpserted := 0
	for _, s := range sets {
		log.Info("=== Harvesting ===", "set", s.name, "spec", s.spec, "limit", limit)
		stats, err := svc.HarvestOAISet(ctx, "", s.spec, limit)
		if err != nil {
			log.Warn("harvest failed", "set", s.spec, "err", err)
			continue
		}
		log.Info("=== Done ===",
			"set", s.name,
			"fetched", stats.Fetched,
			"upserted", stats.Upserted,
			"errors", stats.Errors)
		totalFetched += stats.Fetched
		totalUpserted += stats.Upserted
	}

	log.Info("HARVEST COMPLETE",
		"sets", len(sets),
		"total_fetched", totalFetched,
		"total_upserted_opportunities", totalUpserted)
	return nil
}
