// Package service — MarketplaceService expõe patentes UFOP disponíveis
// para licenciamento, em formato apropriado para um portal público
// (sem dados sensíveis tipo CNPJ de licenciado, anotações internas, etc).
//
// Critério de "disponível":
//   - Patente concedida (status = 'classified')
//   - Sem contrato EXCLUSIVO ativo (non_exclusive contracts ainda permitem mais licenciados)
//   - Applicant = UFOP
//
// Inspiração: AUTM University Marketplace + EPO/Yet2.com tech offers.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// MarketplaceListing é uma patente UFOP disponível para licenciamento.
type MarketplaceListing struct {
	PatentID          int64    `json:"patent_id"`
	ApplicationNumber string   `json:"application_number"`
	Title             string   `json:"title"`
	Abstract          string   `json:"abstract"`
	Inventors         []string `json:"inventors"`
	FilingYear        int      `json:"filing_year"`
	IPCCategory       int      `json:"ipc_category"`
	IPCLetter         string   `json:"ipc_letter"`
	IPCName           string   `json:"ipc_name"`
	Status            string   `json:"status"`
	NonExclusiveSlots int      `json:"non_exclusive_slots_available"` // -1 se exclusiva impossivel, N se ainda há vagas
	ExistingLicensees int      `json:"existing_licensees"`
	SuggestedKind     string   `json:"suggested_license_kind"`        // exclusive | non_exclusive
}

type MarketplaceResponse struct {
	Items      []MarketplaceListing `json:"items"`
	Count      int                  `json:"count"`
	Categories map[string]int       `json:"by_ipc_category"`
}

// MarketplaceService é o read-model público.
type MarketplaceService struct{ db *sql.DB }

func NewMarketplaceService(db *sql.DB) *MarketplaceService {
	return &MarketplaceService{db: db}
}

// List returns available UFOP patents.
//
// Params:
//   ipcLetter   — filtra por seção IPC (A..H ou "")
//   search      — ILIKE título/abstract
//   limit       — max resultados (default 50, cap 200)
func (s *MarketplaceService) List(ctx context.Context, ipcLetter, search string, limit int) (*MarketplaceResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Build WHERE clause
	conditions := []string{
		"p.applicant ILIKE '%Ouro Preto%'",
		"p.status = 'classified'",
	}
	args := []any{}
	idx := 1

	if ipcLetter != "" && len(ipcLetter) > 0 {
		cat := ipcLetterToCategory(strings.ToUpper(ipcLetter[:1]))
		if cat >= 0 {
			conditions = append(conditions, fmt.Sprintf("p.ipc_category = $%d", idx))
			args = append(args, cat)
			idx++
		}
	}

	if search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(p.title ILIKE $%d OR p.abstract ILIKE $%d)", idx, idx,
		))
		args = append(args, "%"+search+"%")
		idx++
	}

	args = append(args, limit)

	q := `
		SELECT p.id, p.application_number, p.title, p.abstract,
		       p.inventors, p.filing_date, COALESCE(p.ipc_category, -1),
		       p.status,
		       COUNT(t.id) FILTER (WHERE t.status='active') AS active_count,
		       BOOL_OR(t.license_kind='exclusive' AND t.status='active') AS has_exclusive
		FROM patents p
		LEFT JOIN tt_contracts t ON t.patent_id = p.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		GROUP BY p.id
		ORDER BY p.filing_date DESC NULLS LAST
		LIMIT $` + fmt.Sprintf("%d", idx)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("marketplace list: %w", err)
	}
	defer rows.Close()

	resp := &MarketplaceResponse{
		Items:      []MarketplaceListing{},
		Categories: map[string]int{},
	}

	for rows.Next() {
		var (
			l            MarketplaceListing
			inventorsRaw sql.NullString  // pq.Array via stringified for simplicity
			filingDate   sql.NullTime
			activeCount  int
			hasExclusive sql.NullBool
		)
		if err := rows.Scan(
			&l.PatentID, &l.ApplicationNumber, &l.Title, &l.Abstract,
			&inventorsRaw, &filingDate, &l.IPCCategory, &l.Status,
			&activeCount, &hasExclusive,
		); err != nil {
			return nil, err
		}
		if filingDate.Valid {
			l.FilingYear = filingDate.Time.Year()
		}
		if l.IPCCategory >= 0 && l.IPCCategory < 8 {
			l.IPCLetter = ipcLetters[l.IPCCategory]
			l.IPCName   = ipcNames[l.IPCCategory]
		}
		l.Inventors = parseInventors(inventorsRaw)

		// Pricing model heuristic
		l.ExistingLicensees = activeCount
		if hasExclusive.Valid && hasExclusive.Bool {
			l.NonExclusiveSlots = 0
			l.SuggestedKind     = "indisponível (exclusiva ativa)"
		} else {
			// Soft cap: 5 non-exclusive licensees per patent
			l.NonExclusiveSlots = 5 - activeCount
			if l.NonExclusiveSlots < 0 {
				l.NonExclusiveSlots = 0
			}
			if activeCount == 0 {
				l.SuggestedKind = "exclusive ou non_exclusive"
			} else {
				l.SuggestedKind = "non_exclusive"
			}
		}

		resp.Items = append(resp.Items, l)
		if l.IPCLetter != "" {
			resp.Categories[l.IPCLetter]++
		}
	}
	resp.Count = len(resp.Items)
	return resp, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func ipcLetterToCategory(letter string) int {
	for i, l := range ipcLetters {
		if l == letter {
			return i
		}
	}
	return -1
}

// parseInventors handles pq array decoding via raw string.
// Format from Postgres: '{"João","Maria"}'.
func parseInventors(s sql.NullString) []string {
	if !s.Valid || s.String == "" || s.String == "{}" {
		return []string{}
	}
	raw := strings.TrimPrefix(s.String, "{")
	raw = strings.TrimSuffix(raw, "}")
	if raw == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
