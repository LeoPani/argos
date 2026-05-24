// Package service — EnrichmentService orchestrates Lens.org citation
// fetching + persistence into patent_citations and patent_metrics.
//
// Persistence model:
//   1. Lens client returns PatentMetricsResult
//   2. Insert each citation as a row in patent_citations
//   3. Aggregate counts + Lanjouw-Schankerman components into patent_metrics
//   4. Also compute Hall-Jaffe-Trajtenberg originality/generality
package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
	"github.com/LeoPani/argos/backend/internal/worker/lens"
)

// EnrichmentService fetches and persists citation data.
type EnrichmentService struct {
	db         *sql.DB
	patentRepo repository.PatentRepository
	lens       *lens.PatentClient
}

func NewEnrichmentService(db *sql.DB, patents repository.PatentRepository, lensClient *lens.PatentClient) *EnrichmentService {
	return &EnrichmentService{db: db, patentRepo: patents, lens: lensClient}
}

// EnrichmentStats is the result of a batch run.
type EnrichmentStats struct {
	Processed       int     `json:"processed"`
	Errors          int     `json:"errors"`
	SourceUsed      string  `json:"source"` // "lens" | "mock"
	AverageFwdCites float64 `json:"avg_fwd_citations"`
}

// EnrichAll fetches citation data for every UFOP patent and persists it.
func (s *EnrichmentService) EnrichAll(ctx context.Context, limit int) (*EnrichmentStats, error) {
	// UFOP-scoped query — Search filter in PatentFilter doesn't cover applicant,
	// so we issue a direct SQL query here.
	patents, err := s.listUFOPPatents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list ufop patents: %w", err)
	}

	stats := &EnrichmentStats{SourceUsed: "mock"}
	totalFwd := 0

	for _, p := range patents {
		if ctx.Err() != nil {
			return stats, ctx.Err()
		}
		result, err := s.lens.Enrich(ctx, &p)
		if err != nil {
			stats.Errors++
			continue
		}
		if result.Source == "lens" {
			stats.SourceUsed = "lens"
		}
		if err := s.persistOne(ctx, &p, result); err != nil {
			stats.Errors++
			continue
		}
		totalFwd += len(result.ForwardCitations)
		stats.Processed++
	}

	if stats.Processed > 0 {
		stats.AverageFwdCites = float64(totalFwd) / float64(stats.Processed)
	}
	return stats, nil
}

// EnrichOne enriches a single patent (used for on-demand from the UI).
func (s *EnrichmentService) EnrichOne(ctx context.Context, patentID int64) (*lens.PatentMetricsResult, error) {
	p, err := s.patentRepo.GetByID(ctx, patentID)
	if err != nil {
		return nil, err
	}
	result, err := s.lens.Enrich(ctx, p)
	if err != nil {
		return nil, err
	}
	if err := s.persistOne(ctx, p, result); err != nil {
		return nil, err
	}
	return result, nil
}

// persistOne is the transactional write: clears prior citations for the
// patent, inserts the new ones, computes aggregates + HJT indices, writes
// patent_metrics row.
func (s *EnrichmentService) persistOne(ctx context.Context, p *domain.Patent, r *lens.PatentMetricsResult) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Replace strategy: delete + re-insert.
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM patent_citations WHERE source_patent_id = $1", p.ID); err != nil {
		return err
	}

	insert := `
		INSERT INTO patent_citations
		  (source_patent_id, citation_kind, cited_lens_id, cited_app_number,
		   cited_title, cited_year, cited_ipc_codes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	for _, c := range r.ForwardCitations {
		if _, err := tx.ExecContext(ctx, insert,
			p.ID, "forward", c.LensID, c.AppNumber, c.Title, c.Year, ipcArray(c.IPCCodes)); err != nil {
			return err
		}
	}
	for _, c := range r.BackwardCitations {
		if _, err := tx.ExecContext(ctx, insert,
			p.ID, "backward", c.LensID, c.AppNumber, c.Title, c.Year, ipcArray(c.IPCCodes)); err != nil {
			return err
		}
	}

	// HJT (2001) Originality (backward) and Generality (forward) indices.
	originality := hjtIndex(r.BackwardCitations)
	generality  := hjtIndex(r.ForwardCitations)

	// PCI Lanjouw-Schankerman (2004)
	pci := pciScore(len(r.ForwardCitations), r.FamilySize, r.ClaimsCount, len(r.BackwardCitations))

	upsert := `
		INSERT INTO patent_metrics
		  (patent_id, forward_citations, backward_citations, scientific_citations,
		   family_size, claims_count, pci_score, originality_index, generality_index,
		   computed_at, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10)
		ON CONFLICT (patent_id) DO UPDATE SET
		  forward_citations    = EXCLUDED.forward_citations,
		  backward_citations   = EXCLUDED.backward_citations,
		  scientific_citations = EXCLUDED.scientific_citations,
		  family_size          = EXCLUDED.family_size,
		  claims_count         = EXCLUDED.claims_count,
		  pci_score            = EXCLUDED.pci_score,
		  originality_index    = EXCLUDED.originality_index,
		  generality_index     = EXCLUDED.generality_index,
		  computed_at          = NOW(),
		  source               = EXCLUDED.source`

	if _, err := tx.ExecContext(ctx, upsert,
		p.ID, len(r.ForwardCitations), len(r.BackwardCitations), r.ScientificCites,
		r.FamilySize, r.ClaimsCount, pci, originality, generality, r.Source,
	); err != nil {
		return err
	}

	return tx.Commit()
}

// listUFOPPatents bypasses the repository's Search filter (which only
// covers title/abstract) and matches the applicant column directly.
func (s *EnrichmentService) listUFOPPatents(ctx context.Context, limit int) ([]domain.Patent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, application_number, title, applicant, ipc_category, filing_date
		FROM patents
		WHERE applicant ILIKE '%Ouro Preto%'
		ORDER BY id
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Patent
	for rows.Next() {
		var (
			p          domain.Patent
			ipcCat     sql.NullInt64
			filingDate sql.NullTime
		)
		if err := rows.Scan(&p.ID, &p.ApplicationNumber, &p.Title, &p.Applicant, &ipcCat, &filingDate); err != nil {
			return nil, err
		}
		if ipcCat.Valid {
			p.IPCCategory = domain.IPCCategory(ipcCat.Int64)
		}
		if filingDate.Valid {
			t := filingDate.Time
			p.FilingDate = &t
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ─── HJT computation ──────────────────────────────────────────────────────────

// hjtIndex implements 1 − Σⱼ (sⱼ)² over IPC class shares.
// When citations have multiple IPC codes per row, we count each row's
// primary (first) IPC class.
func hjtIndex(refs []lens.CitationRef) float64 {
	if len(refs) == 0 {
		return 0
	}
	counts := map[string]int{}
	total := 0
	for _, r := range refs {
		if len(r.IPCCodes) == 0 {
			continue
		}
		// Use just the section letter (A..H) — coarse-grained per HJT 2001
		section := r.IPCCodes[0]
		if len(section) > 0 {
			counts[string(section[0])]++
			total++
		}
	}
	if total == 0 {
		return 0
	}
	sumSq := 0.0
	for _, n := range counts {
		s := float64(n) / float64(total)
		sumSq += s * s
	}
	return math.Round((1-sumSq)*10000) / 10000 // round to 4 decimals
}

// pciScore — Lanjouw & Schankerman (2004) composite, min-max normalized.
func pciScore(fwd, family, claims, bwd int) float64 {
	nFwd    := math.Min(1.0, float64(fwd)/50)
	nFam    := math.Min(1.0, float64(family)/10)
	nClaims := math.Min(1.0, float64(claims)/50)
	nBwd    := math.Min(1.0, float64(bwd)/30)
	pci := (0.46*nFwd + 0.27*nFam + 0.16*nClaims + 0.11*nBwd) * 100
	return math.Round(pci*100) / 100
}

// ipcArray formats []string for Postgres TEXT[].
func ipcArray(codes []string) any {
	if len(codes) == 0 {
		return "{}"
	}
	quoted := make([]string, len(codes))
	for i, c := range codes {
		quoted[i] = `"` + c + `"`
	}
	out := "{"
	for i, q := range quoted {
		if i > 0 {
			out += ","
		}
		out += q
	}
	out += "}"
	return out
}
