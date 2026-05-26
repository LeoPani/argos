// Package service — INPIPublicationService expõe consultas sobre o dataset
// extraído das RPIs (revistas semanais do INPI). Ingestão feita pelo
// script Python ai-service/inpi_rpi_harvest.py.
package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/lib/pq"
)

type INPIPublication struct {
	ID            int64    `json:"id"`
	RPINumber     int      `json:"rpi_number"`
	RPISection    string   `json:"rpi_section"`
	ProcessNumber string   `json:"process_number"`
	DespachoCode  string   `json:"despacho_code"`
	Title         string   `json:"title"`
	Applicant     string   `json:"applicant"`
	IPCCodes      []string `json:"ipc_codes"`
	IsUFOP        bool     `json:"is_ufop"`
}

type INPIStats struct {
	TotalRecords  int            `json:"total_records"`
	UFOPRecords   int            `json:"ufop_records"`
	BySection     map[string]int `json:"by_section"`
	LatestRPI     int            `json:"latest_rpi"`
	OldestRPI     int            `json:"oldest_rpi"`
}

type INPIPublicationService struct{ db *sql.DB }

func NewINPIPublicationService(db *sql.DB) *INPIPublicationService {
	return &INPIPublicationService{db: db}
}

// Stats — visão geral pra UI / banner de honestidade.
func (s *INPIPublicationService) Stats(ctx context.Context) (*INPIStats, error) {
	stats := &INPIStats{BySection: map[string]int{}}

	row := s.db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*) AS total,
		  COUNT(*) FILTER (WHERE is_ufop)        AS ufop,
		  COALESCE(MAX(rpi_number), 0)           AS latest,
		  COALESCE(MIN(rpi_number), 0)           AS oldest
		FROM inpi_publications`)
	if err := row.Scan(&stats.TotalRecords, &stats.UFOPRecords, &stats.LatestRPI, &stats.OldestRPI); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT COALESCE(rpi_section, 'unknown'), COUNT(*)
		FROM inpi_publications
		GROUP BY rpi_section`)
	if err != nil {
		return stats, nil
	}
	defer rows.Close()
	for rows.Next() {
		var section string
		var n int
		if err := rows.Scan(&section, &n); err == nil {
			stats.BySection[section] = n
		}
	}
	return stats, nil
}

// TimelinePoint é um ponto no gráfico temporal de despachos INPI.
type TimelinePoint struct {
	Period string `json:"period"` // "2024-01" ou "RPI 2890"
	RPI    int    `json:"rpi"`    // número da RPI
	Total  int64  `json:"total"`  // total de despachos no período
	UFOP   int64  `json:"ufop"`   // despachos is_ufop no período
}

// Timeline agrupa despachos por RPI number para mostrar evolução temporal.
// Retorna os últimos `limit` RPIs em ordem ascendente (mais antigo → mais recente).
func (s *INPIPublicationService) Timeline(ctx context.Context, limit int) ([]TimelinePoint, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	// Subquery pega os N RPIs mais recentes, depois agrega de forma ascendente
	q := fmt.Sprintf(`
		SELECT rpi_number,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE is_ufop) AS ufop
		FROM inpi_publications
		WHERE rpi_number IN (
		  SELECT DISTINCT rpi_number
		  FROM inpi_publications
		  ORDER BY rpi_number DESC
		  LIMIT %d
		)
		GROUP BY rpi_number
		ORDER BY rpi_number ASC`, limit)

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return []TimelinePoint{}, nil // table may not exist
	}
	defer rows.Close()

	out := make([]TimelinePoint, 0, limit)
	for rows.Next() {
		var pt TimelinePoint
		if err := rows.Scan(&pt.RPI, &pt.Total, &pt.UFOP); err != nil {
			continue
		}
		pt.Period = fmt.Sprintf("RPI %d", pt.RPI)
		out = append(out, pt)
	}
	return out, nil
}

// ListUFOP — despachos UFOP recentes. Útil pra mostrar no dashboard.
func (s *INPIPublicationService) ListUFOP(ctx context.Context, limit int) ([]INPIPublication, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rpi_number, COALESCE(rpi_section,''), process_number,
		       COALESCE(despacho_code,''), COALESCE(title,''),
		       COALESCE(applicant,''), COALESCE(ipc_codes, '{}'), is_ufop
		FROM inpi_publications
		WHERE is_ufop = TRUE
		ORDER BY rpi_number DESC, id DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]INPIPublication, 0, limit)
	for rows.Next() {
		var p INPIPublication
		if err := rows.Scan(&p.ID, &p.RPINumber, &p.RPISection, &p.ProcessNumber,
			&p.DespachoCode, &p.Title, &p.Applicant, pq.Array(&p.IPCCodes), &p.IsUFOP); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
