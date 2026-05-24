// Package service — CalendarService agrega todos os deadlines relevantes
// pro NIT-UFOP num feed unificado: anuidades INPI, renovações de marca,
// milestones de contratos TT, prazos de disputas em andamento.
//
// Geração automática baseada nos dados já no banco — não precisa de
// data entry manual. Cada item tem `kind` para o frontend filtrar/colorir.
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// CalendarEvent é uma entrada no calendário.
type CalendarEvent struct {
	ID           string    `json:"id"`           // unique key (kind-ref-yr)
	Date         time.Time `json:"date"`
	Kind         string    `json:"kind"`         // annuity | renewal | milestone | dispute | filing
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	AmountBRL    float64   `json:"amount_brl,omitempty"`
	EntityType   string    `json:"entity_type"`  // patent | trademark | contract | dispute
	EntityID     int64     `json:"entity_id"`
	EntityRef    string    `json:"entity_ref"`   // app_number, process_number, etc
	Priority     string    `json:"priority"`     // critical | high | medium | low
	URL          string    `json:"url,omitempty"` // frontend route to entity
}

type CalendarResponse struct {
	From       time.Time       `json:"from"`
	To         time.Time       `json:"to"`
	Events     []CalendarEvent `json:"events"`
	Count      int             `json:"count"`
	ByKind     map[string]int  `json:"by_kind"`
}

// CalendarService.
type CalendarService struct{ db *sql.DB }

func NewCalendarService(db *sql.DB) *CalendarService { return &CalendarService{db: db} }

// Get returns all events in [from, to]. Defaults to next 365 days.
func (s *CalendarService) Get(ctx context.Context, from, to time.Time) (*CalendarResponse, error) {
	if from.IsZero() {
		from = time.Now()
	}
	if to.IsZero() {
		to = from.AddDate(1, 0, 0)
	}

	resp := &CalendarResponse{
		From: from, To: to,
		Events: []CalendarEvent{},
		ByKind: map[string]int{},
	}

	// 1) Patent annuities (INPI: anos 3-20, vencimento na data do depósito)
	patentEvents, _ := s.patentAnnuities(ctx, from, to)
	resp.Events = append(resp.Events, patentEvents...)

	// 2) Trademark renewals (granted_date + N*10y)
	tmEvents, _ := s.trademarkRenewals(ctx, from, to)
	resp.Events = append(resp.Events, tmEvents...)

	// 3) Contract milestones (from JSONB)
	milestones, _ := s.contractMilestones(ctx, from, to)
	resp.Events = append(resp.Events, milestones...)

	// 4) Dispute deadlines (open + 90d as default proxy)
	disputeEvents, _ := s.disputeDeadlines(ctx, from, to)
	resp.Events = append(resp.Events, disputeEvents...)

	// Sort cronologicamente
	sort.Slice(resp.Events, func(i, j int) bool {
		return resp.Events[i].Date.Before(resp.Events[j].Date)
	})

	for _, e := range resp.Events {
		resp.ByKind[e.Kind]++
	}
	resp.Count = len(resp.Events)
	return resp, nil
}

// ─── Patent annuities ────────────────────────────────────────────────────────

// INPI annuity schedule (MPE 2024 reference)
var annuityFees = map[int]float64{
	3: 310, 4: 310, 5: 310, 6: 310,
	7: 620, 8: 620, 9: 620, 10: 620,
	11: 930, 12: 930, 13: 930, 14: 930, 15: 930,
	16: 1240, 17: 1240, 18: 1240, 19: 1240, 20: 1240,
}

func (s *CalendarService) patentAnnuities(ctx context.Context, from, to time.Time) ([]CalendarEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, application_number, title, filing_date
		FROM patents
		WHERE filing_date IS NOT NULL
		  AND applicant ILIKE '%Ouro Preto%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CalendarEvent
	for rows.Next() {
		var (
			id       int64
			appNum   string
			title    string
			filing   sql.NullTime
		)
		if err := rows.Scan(&id, &appNum, &title, &filing); err != nil {
			continue
		}
		if !filing.Valid {
			continue
		}

		// Cada anuidade vence na data de aniversário do depósito
		for year := 3; year <= 20; year++ {
			due := filing.Time.AddDate(year, 0, 0)
			if due.Before(from) || due.After(to) {
				continue
			}
			fee := annuityFees[year]
			priority := "medium"
			days := int(time.Until(due).Hours() / 24)
			if days < 30 {
				priority = "critical"
			} else if days < 90 {
				priority = "high"
			}

			out = append(out, CalendarEvent{
				ID:          fmt.Sprintf("annuity-%d-%d", id, year),
				Date:        due,
				Kind:        "annuity",
				Title:       fmt.Sprintf("Anuidade %dº ano — %s", year, appNum),
				Description: fmt.Sprintf("Pagamento de anuidade INPI da patente. Tabela MPE 2024."),
				AmountBRL:   fee,
				EntityType:  "patent",
				EntityID:    id,
				EntityRef:   appNum,
				Priority:    priority,
				URL:         fmt.Sprintf("/patents/%d", id),
			})
		}
	}
	return out, nil
}

// ─── Trademark renewals ──────────────────────────────────────────────────────

func (s *CalendarService) trademarkRenewals(ctx context.Context, from, to time.Time) ([]CalendarEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, process_number, name, granted_date
		FROM trademarks
		WHERE granted_date IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CalendarEvent
	for rows.Next() {
		var (
			id      int64
			proc    string
			name    string
			granted sql.NullTime
		)
		if err := rows.Scan(&id, &proc, &name, &granted); err != nil {
			continue
		}
		if !granted.Valid {
			continue
		}

		// Renovação a cada 10 anos. Computa próximas 2 (cobre 20yr).
		for n := 1; n <= 3; n++ {
			due := granted.Time.AddDate(10*n, 0, 0)
			if due.Before(from) || due.After(to) {
				continue
			}
			out = append(out, CalendarEvent{
				ID:          fmt.Sprintf("renewal-%d-%d", id, n),
				Date:        due,
				Kind:        "renewal",
				Title:       fmt.Sprintf("Renovação marca %s — %dº ciclo", name, n),
				Description: "Renovação decenal junto ao INPI (Lei 9.279/96).",
				AmountBRL:   855,
				EntityType:  "trademark",
				EntityID:    id,
				EntityRef:   proc,
				Priority:    "high",
				URL:         fmt.Sprintf("/trademarks/%d", id),
			})
		}
	}
	return out, nil
}

// ─── Contract milestones (from JSONB) ────────────────────────────────────────

type rawMilestone struct {
	Label   string  `json:"label"`
	DueDate string  `json:"due_date"`
	FeeBRL  float64 `json:"fee_brl"`
	Done    bool    `json:"done"`
}

func (s *CalendarService) contractMilestones(ctx context.Context, from, to time.Time) ([]CalendarEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, contract_number, licensee, milestones
		FROM tt_contracts
		WHERE status IN ('active', 'negotiating')
		  AND milestones IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CalendarEvent
	for rows.Next() {
		var (
			id    int64
			num   string
			lic   string
			raw   sql.NullString
		)
		if err := rows.Scan(&id, &num, &lic, &raw); err != nil {
			continue
		}
		if !raw.Valid {
			continue
		}
		var ms []rawMilestone
		if err := json.Unmarshal([]byte(raw.String), &ms); err != nil {
			continue
		}
		for i, m := range ms {
			if m.Done || m.DueDate == "" {
				continue
			}
			due, err := time.Parse("2006-01-02", m.DueDate)
			if err != nil {
				continue
			}
			if due.Before(from) || due.After(to) {
				continue
			}
			out = append(out, CalendarEvent{
				ID:          fmt.Sprintf("milestone-%d-%d", id, i),
				Date:        due,
				Kind:        "milestone",
				Title:       fmt.Sprintf("Milestone TT: %s — %s", m.Label, num),
				Description: fmt.Sprintf("Marco contratual com %s.", lic),
				AmountBRL:   m.FeeBRL,
				EntityType:  "contract",
				EntityID:    id,
				EntityRef:   num,
				Priority:    "high",
				URL:         "/pool",
			})
		}
	}
	return out, nil
}

// ─── Dispute deadlines ───────────────────────────────────────────────────────

func (s *CalendarService) disputeDeadlines(ctx context.Context, from, to time.Time) ([]CalendarEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, case_number, title, opened_at
		FROM disputes
		WHERE status IN ('open', 'in_review', 'awaiting_info', 'escalated')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []CalendarEvent
	for rows.Next() {
		var (
			id       int64
			caseNum  string
			title    string
			opened   time.Time
		)
		if err := rows.Scan(&id, &caseNum, &title, &opened); err != nil {
			continue
		}
		// Default deadline: 90 dias após abertura (padrão arbitral INPI)
		due := opened.AddDate(0, 0, 90)
		if due.Before(from) || due.After(to) {
			continue
		}
		priority := "medium"
		days := int(time.Until(due).Hours() / 24)
		if days < 14 {
			priority = "critical"
		} else if days < 30 {
			priority = "high"
		}
		out = append(out, CalendarEvent{
			ID:          fmt.Sprintf("dispute-%d", id),
			Date:        due,
			Kind:        "dispute",
			Title:       fmt.Sprintf("Prazo arbitral — %s", caseNum),
			Description: title,
			EntityType:  "dispute",
			EntityID:    id,
			EntityRef:   caseNum,
			Priority:    priority,
			URL:         "/arbitragem",
		})
	}
	return out, nil
}
