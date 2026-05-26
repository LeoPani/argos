// Package service — StatsService aggregates counts and breakdowns
// across all entities for the dashboard BI panel.
//
// Unlike entity-specific services, this is intentionally a "read model":
// it queries the database directly with aggregation SQL rather than
// going through the per-entity repositories. This avoids polluting the
// domain repositories with reporting concerns.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ─── Response types ───────────────────────────────────────────────────────────

// StatsCounts is the top-row KPI grid.
type StatsCounts struct {
	Patents            int64 `json:"patents"`
	PatentsClassified  int64 `json:"patents_classified"`
	Trademarks         int64 `json:"trademarks"`
	TrademarksActive   int64 `json:"trademarks_active"`
	Disputes           int64 `json:"disputes"`
	DisputesOpen       int64 `json:"disputes_open"`
	UFOPOpportunities  int64 `json:"ufop_opportunities"`
	UFOPHigh           int64 `json:"ufop_high"`
	INPIPublications   int64 `json:"inpi_publications"`  // despachos from RPI harvest
	LatestRPI          int64 `json:"latest_rpi"`         // most recent RPI number ingested
	IPTimestamps       int64 `json:"ip_timestamps"`      // proof-of-existence records
}

// IPCSlice is one entry in the IPC distribution chart.
type IPCSlice struct {
	Category int    `json:"category"` // 0..7
	Letter   string `json:"letter"`   // "A".."H"
	Name     string `json:"name"`     // "Química"
	Count    int64  `json:"count"`
	Pct      int    `json:"pct"`
}

// StatusSlice is a generic slice for status pie/bar charts.
type StatusSlice struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
	Pct    int    `json:"pct"`
}

// ActivityItem is a single row in the recent-activity feed.
type ActivityItem struct {
	Kind      string    `json:"kind"`       // "patent" | "trademark" | "dispute" | "ufop"
	ID        int64     `json:"id"`
	Reference string    `json:"reference"`  // application number / process number / case number
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// StatsResponse is the full payload returned by GET /api/v1/stats.
type StatsResponse struct {
	Counts            StatsCounts    `json:"counts"`
	IPCDistribution   []IPCSlice     `json:"ipc_distribution"`
	PatentStatuses    []StatusSlice  `json:"patent_statuses"`
	TrademarkStatuses []StatusSlice  `json:"trademark_statuses"`
	RecentActivity    []ActivityItem `json:"recent_activity"`
	GeneratedAt       time.Time      `json:"generated_at"`
}

// ─── Service ──────────────────────────────────────────────────────────────────

// StatsService computes dashboard aggregations directly from the database.
type StatsService struct{ db *sql.DB }

// NewStatsService creates the service.
func NewStatsService(db *sql.DB) *StatsService { return &StatsService{db: db} }

// Get assembles the full dashboard payload.
func (s *StatsService) Get(ctx context.Context) (*StatsResponse, error) {
	counts, err := s.counts(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats counts: %w", err)
	}

	ipcDist, err := s.ipcDistribution(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats ipc: %w", err)
	}

	patentStatuses, err := s.statusBreakdown(ctx, "patents", "status")
	if err != nil {
		return nil, fmt.Errorf("stats patent status: %w", err)
	}

	tmStatuses, err := s.statusBreakdown(ctx, "trademarks", "status")
	if err != nil {
		return nil, fmt.Errorf("stats trademark status: %w", err)
	}

	activity, err := s.recentActivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("stats activity: %w", err)
	}

	return &StatsResponse{
		Counts:            counts,
		IPCDistribution:   ipcDist,
		PatentStatuses:    patentStatuses,
		TrademarkStatuses: tmStatuses,
		RecentActivity:    activity,
		GeneratedAt:       time.Now(),
	}, nil
}

// ─── Internals ────────────────────────────────────────────────────────────────

// counts returns top-line counts across all entities. Missing tables are
// treated as zero — keeps the dashboard alive when a migration hasn't run.
func (s *StatsService) counts(ctx context.Context) (StatsCounts, error) {
	var c StatsCounts

	queries := []struct {
		sql string
		dst *int64
	}{
		{`SELECT COUNT(*) FROM patents`, &c.Patents},
		{`SELECT COUNT(*) FROM patents WHERE status = 'classified'`, &c.PatentsClassified},
		{`SELECT COUNT(*) FROM trademarks`, &c.Trademarks},
		{`SELECT COUNT(*) FROM trademarks WHERE status = 'granted'`, &c.TrademarksActive},
		{`SELECT COUNT(*) FROM disputes`, &c.Disputes},
		{`SELECT COUNT(*) FROM disputes WHERE status IN ('open','in_analysis','mediation','urgent')`, &c.DisputesOpen},
		// Conta APENAS patenteáveis (Art. 8 LPI) — exclui rejeitadas por Art. 10
		// (Direito, Letras, etc). is_patentable IS NULL é tratado como
		// patenteável (legado antes da migration 0014).
		{`SELECT COUNT(*) FROM ufop_opportunities WHERE COALESCE(is_patentable, true)`, &c.UFOPOpportunities},
		{`SELECT COUNT(*) FROM ufop_opportunities WHERE opportunity_level = 'high' AND COALESCE(is_patentable, true)`, &c.UFOPHigh},
		{`SELECT COUNT(*) FROM inpi_publications`, &c.INPIPublications},
		{`SELECT COALESCE(MAX(rpi_number), 0) FROM inpi_publications`, &c.LatestRPI},
		{`SELECT COUNT(*) FROM ip_timestamps`, &c.IPTimestamps},
	}

	for _, q := range queries {
		if err := s.db.QueryRowContext(ctx, q.sql).Scan(q.dst); err != nil {
			// Table may not exist — treat as 0 rather than failing the whole call.
			*q.dst = 0
		}
	}
	return c, nil
}

// ipcDistribution groups patents by IPC category (0..7).
func (s *StatsService) ipcDistribution(ctx context.Context) ([]IPCSlice, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ipc_category, COUNT(*)
		FROM patents
		WHERE ipc_category IS NOT NULL AND ipc_category BETWEEN 0 AND 7
		GROUP BY ipc_category
		ORDER BY ipc_category`)
	if err != nil {
		return []IPCSlice{}, nil // table missing → empty
	}
	defer rows.Close()

	var slices []IPCSlice
	var total int64
	for rows.Next() {
		var s IPCSlice
		if err := rows.Scan(&s.Category, &s.Count); err != nil {
			return nil, err
		}
		s.Letter = ipcLetters[s.Category]
		s.Name = ipcNames[s.Category]
		slices = append(slices, s)
		total += s.Count
	}
	if total > 0 {
		for i := range slices {
			slices[i].Pct = int(float64(slices[i].Count) / float64(total) * 100)
		}
	}
	if slices == nil {
		slices = []IPCSlice{}
	}
	return slices, nil
}

// statusBreakdown returns count and percentage per status for a given table.
func (s *StatsService) statusBreakdown(ctx context.Context, table, col string) ([]StatusSlice, error) {
	// table/col are trusted (compile-time literals in callers)
	q := fmt.Sprintf("SELECT %s, COUNT(*) FROM %s GROUP BY %s ORDER BY COUNT(*) DESC", col, table, col)
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return []StatusSlice{}, nil
	}
	defer rows.Close()

	var slices []StatusSlice
	var total int64
	for rows.Next() {
		var s StatusSlice
		if err := rows.Scan(&s.Status, &s.Count); err != nil {
			return nil, err
		}
		slices = append(slices, s)
		total += s.Count
	}
	if total > 0 {
		for i := range slices {
			slices[i].Pct = int(float64(slices[i].Count) / float64(total) * 100)
		}
	}
	if slices == nil {
		slices = []StatusSlice{}
	}
	return slices, nil
}

// recentActivity unions the most recent rows across patents, trademarks,
// disputes and ufop_opportunities, sorted by created_at desc.
func (s *StatsService) recentActivity(ctx context.Context) ([]ActivityItem, error) {
	const q = `
		(SELECT 'patent'    AS kind, id, application_number AS ref, title, status::text, created_at
		 FROM patents       ORDER BY created_at DESC LIMIT 10)
		UNION ALL
		(SELECT 'trademark' AS kind, id, process_number,     name,  status::text, created_at
		 FROM trademarks    ORDER BY created_at DESC LIMIT 10)
		UNION ALL
		(SELECT 'dispute'   AS kind, id, case_number,        title, status::text, created_at
		 FROM disputes      ORDER BY created_at DESC LIMIT 10)
		UNION ALL
		(SELECT 'ufop'      AS kind, id, external_id,        title, status::text, created_at
		 FROM ufop_opportunities
		 WHERE COALESCE(is_patentable, true)
		 ORDER BY created_at DESC LIMIT 10)
		ORDER BY created_at DESC LIMIT 15`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return []ActivityItem{}, nil
	}
	defer rows.Close()

	var items []ActivityItem
	for rows.Next() {
		var it ActivityItem
		if err := rows.Scan(&it.Kind, &it.ID, &it.Reference, &it.Title, &it.Status, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if items == nil {
		items = []ActivityItem{}
	}
	return items, nil
}

// ─── IPC labels ───────────────────────────────────────────────────────────────

var ipcLetters = [8]string{"A", "B", "C", "D", "E", "F", "G", "H"}

var ipcNames = [8]string{
	"Necessidades Humanas",
	"Operações e Transportes",
	"Química e Metalurgia",
	"Têxteis e Papel",
	"Construção Civil",
	"Engenharia Mecânica",
	"Física / TI",
	"Eletricidade",
}
