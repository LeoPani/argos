// Package service — SearchService is a federated, read-model search
// that combines hits from patents, trademarks, disputes and tt_contracts
// into a single ranked list.
//
// Uses ILIKE for now (simple, fast on the volumes we have). When the
// dataset grows, swap for pg_trgm or a real FTS index.
package service

import (
	"context"
	"database/sql"
	"fmt"
)

// SearchHit is a single result row.
type SearchHit struct {
	Kind      string `json:"kind"`      // patent | trademark | dispute | contract
	ID        int64  `json:"id"`
	Reference string `json:"reference"` // application_number, process_number, etc.
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`  // applicant / owner / kind / status
	URL       string `json:"url"`       // suggested frontend path
}

// SearchResponse aggregates everything.
type SearchResponse struct {
	Query string      `json:"query"`
	Total int         `json:"total"`
	Hits  []SearchHit `json:"hits"`
}

// SearchService federates queries across entities.
type SearchService struct{ db *sql.DB }

func NewSearchService(db *sql.DB) *SearchService { return &SearchService{db: db} }

// Search runs the federated query. Limit is per-entity (so a "ufop"
// query may return up to 6*4 = 24 results).
func (s *SearchService) Search(ctx context.Context, q string, perEntity int) (*SearchResponse, error) {
	if perEntity <= 0 {
		perEntity = 6
	}
	if q == "" {
		return &SearchResponse{Query: q, Total: 0, Hits: []SearchHit{}}, nil
	}

	hits := make([]SearchHit, 0, perEntity*4)

	// Each entity uses a small dedicated query — keeps the SQL readable
	// and lets each table use its own indexed columns.
	for _, fn := range []func(context.Context, string, int) ([]SearchHit, error){
		s.searchPatents, s.searchTrademarks, s.searchDisputes, s.searchContracts,
	} {
		results, err := fn(ctx, q, perEntity)
		if err == nil {
			hits = append(hits, results...)
		}
	}

	return &SearchResponse{Query: q, Total: len(hits), Hits: hits}, nil
}

func (s *SearchService) searchPatents(ctx context.Context, q string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, application_number, title, applicant
		FROM patents
		WHERE title ILIKE $1 OR abstract ILIKE $1 OR applicant ILIKE $1 OR application_number ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var (
			id        int64
			appNum    string
			title     string
			applicant string
		)
		if err := rows.Scan(&id, &appNum, &title, &applicant); err != nil {
			return nil, err
		}
		hits = append(hits, SearchHit{
			Kind:      "patent",
			ID:        id,
			Reference: appNum,
			Title:     title,
			Subtitle:  applicant,
			URL:       fmt.Sprintf("/patents/%d", id),
		})
	}
	return hits, rows.Err()
}

func (s *SearchService) searchTrademarks(ctx context.Context, q string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, process_number, name, owner, status
		FROM trademarks
		WHERE name ILIKE $1 OR owner ILIKE $1 OR process_number ILIKE $1 OR normalized_name ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var (
			id      int64
			procNum string
			name    string
			owner   string
			status  string
		)
		if err := rows.Scan(&id, &procNum, &name, &owner, &status); err != nil {
			return nil, err
		}
		hits = append(hits, SearchHit{
			Kind:      "trademark",
			ID:        id,
			Reference: procNum,
			Title:     name,
			Subtitle:  fmt.Sprintf("%s · %s", owner, status),
			URL:       fmt.Sprintf("/trademarks/%d", id),
		})
	}
	return hits, rows.Err()
}

func (s *SearchService) searchDisputes(ctx context.Context, q string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, case_number, title, kind, status
		FROM disputes
		WHERE title ILIKE $1 OR summary ILIKE $1 OR case_number ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var (
			id      int64
			caseNum string
			title   string
			kind    string
			status  string
		)
		if err := rows.Scan(&id, &caseNum, &title, &kind, &status); err != nil {
			return nil, err
		}
		hits = append(hits, SearchHit{
			Kind:      "dispute",
			ID:        id,
			Reference: caseNum,
			Title:     title,
			Subtitle:  fmt.Sprintf("%s · %s", kind, status),
			URL:       "/arbitragem",
		})
	}
	return hits, rows.Err()
}

func (s *SearchService) searchContracts(ctx context.Context, q string, limit int) ([]SearchHit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, contract_number, licensee, license_kind, status
		FROM tt_contracts
		WHERE contract_number ILIKE $1 OR licensee ILIKE $1 OR notes ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2`, "%"+q+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []SearchHit
	for rows.Next() {
		var (
			id      int64
			num     string
			lessee  string
			kind    string
			status  string
		)
		if err := rows.Scan(&id, &num, &lessee, &kind, &status); err != nil {
			return nil, err
		}
		hits = append(hits, SearchHit{
			Kind:      "contract",
			ID:        id,
			Reference: num,
			Title:     "TT → " + lessee,
			Subtitle:  fmt.Sprintf("%s · %s", kind, status),
			URL:       "/pool",
		})
	}
	return hits, rows.Err()
}
