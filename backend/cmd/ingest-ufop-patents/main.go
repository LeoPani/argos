// argos-ingest-ufop-patents pesca patentes UFOP REAIS do Google Patents
// (que indexa INPI + USPTO + EPO + WIPO).
//
// Usage:
//
//	go run ./cmd/ingest-ufop-patents          # 5 páginas (~50 patentes)
//	go run ./cmd/ingest-ufop-patents 27       # 27 páginas (todas 261)
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/config"
	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/platform/database"
	"github.com/LeoPani/argos/backend/internal/platform/logger"
	pg "github.com/LeoPani/argos/backend/internal/repository/postgres"
	inpipatents "github.com/LeoPani/argos/backend/internal/worker/inpi_patents"
)

func main() {
	if err := run(); err != nil {
		slog.Error("ingest-ufop-patents: fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(logger.Config{Level: cfg.LogLevel, Format: cfg.LogFormat})

	pages := 5
	if len(os.Args) > 1 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil {
			pages = n
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := database.New(ctx, database.Config{
		DSN: cfg.DatabaseURL, MaxOpenConns: 4, MaxIdleConns: 2,
		ConnMaxLifetime: 5 * time.Minute, PingTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	patentRepo := pg.NewPatentRepo(db)
	client := inpipatents.NewGooglePatentsClient(log)

	log.Info("=== Pescando patentes UFOP via Google Patents ===", "pages", pages)
	results, err := client.Search(ctx, "Universidade Federal de Ouro Preto", pages)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	inserted, duplicates, errs := 0, 0, 0
	for _, r := range results {
		p := convertToDomainPatent(&r)
		if p == nil {
			continue
		}
		if err := patentRepo.Insert(ctx, p); err != nil {
			if errors.Is(err, domain.ErrDuplicate) {
				duplicates++
				continue
			}
			log.Warn("insert failed", "app", r.ApplicationNumber, "err", err)
			errs++
			continue
		}
		inserted++
	}

	log.Info("INGEST COMPLETE",
		"fetched", len(results),
		"inserted", inserted,
		"duplicates", duplicates,
		"errors", errs)
	return nil
}

// convertToDomainPatent — adapta o resultado do Google Patents pro domain.Patent.
func convertToDomainPatent(r *inpipatents.PatentResult) *domain.Patent {
	if r.ApplicationNumber == "" || r.Title == "" {
		return nil
	}

	var filing, pub *time.Time
	if r.FilingDate != "" {
		if t, err := time.Parse("2006-01-02", r.FilingDate); err == nil {
			filing = &t
		}
	}
	if r.PublicationDate != "" {
		if t, err := time.Parse("2006-01-02", r.PublicationDate); err == nil {
			pub = &t
		}
	}

	// IPC heurístico baseado no título (BERT offline). Não-classificado
	// fica como -1, frontend mostra "—".
	ipcCat := guessIPC(r.Title + " " + r.Abstract)

	return &domain.Patent{
		ApplicationNumber: r.ApplicationNumber,
		Title:             r.Title,
		Abstract:          r.Abstract,
		Applicant:         "Universidade Federal de Ouro Preto",
		Inventors:         []string{}, // Google Patents API básica não retorna
		FilingDate:        filing,
		PublicationDate:   pub,
		IPCCategory:       domain.IPCCategory(ipcCat),
		IPCCode:           "",      // Google retorna em outro endpoint
		RPIIssue:          r.Country, // country como proxy
		Status:            "classified",
	}
}

// guessIPC — heurística leve por keyword do título/abstract.
// Suficiente pra agrupar no dashboard sem BERT.
func guessIPC(text string) int {
	t := strings.ToLower(text)
	if containsAny(t, "pharmac", "vaccine", "drug", "treatment", "antibiotic",
		"farmac", "vacina", "medicament", "saúde") {
		return 0 // A
	}
	if containsAny(t, "separation", "flotation", "leach", "concentrat",
		"flotação", "lixiviação", "moagem", "concentr") {
		return 1 // B
	}
	if containsAny(t, "chemical", "metal", "alloy", "ore", "composition",
		"químic", "metalurg", "minério", "composição") {
		return 2 // C
	}
	if containsAny(t, "construction", "concrete", "structure",
		"construç", "concreto", "estrutur") {
		return 4 // E
	}
	if containsAny(t, "computer", "neural", "machine learning", "algorithm",
		"sensor", "software", "redes neurais", "deep learning") {
		return 6 // G
	}
	if containsAny(t, "electric", "battery", "circuit", "energy",
		"elétric", "bateria", "energia") {
		return 7 // H
	}
	return 2 // C — default razoável pra UFOP (Eng. Minas dominante)
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
