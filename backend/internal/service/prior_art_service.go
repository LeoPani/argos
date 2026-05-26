package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// PriorArtHit is a single search result with similarity score.
type PriorArtHit struct {
	Kind           string  // "patent" | "trademark" | "publication"
	ID             int64
	Number         string
	Title          string
	Owner          string
	FilingDate     string
	SimilarityPct  int // 0-100
}

// PriorArtResult aggregates hits and a risk score.
type PriorArtResult struct {
	Query     string
	RiskScore float64 // 0-10
	Hits      []PriorArtHit
}

// PriorArtService searches for prior art across patents, trademarks, and publications.
type PriorArtService struct {
	patents    repository.PatentRepository
	trademarks repository.TrademarkRepository
	pubs       repository.PublicationRepository
}

func NewPriorArtService(
	patents repository.PatentRepository,
	trademarks repository.TrademarkRepository,
	pubs repository.PublicationRepository,
) *PriorArtService {
	return &PriorArtService{patents: patents, trademarks: trademarks, pubs: pubs}
}

// Search performs full-text prior art search across all IP types.
func (s *PriorArtService) Search(ctx context.Context, query, kind string) (*PriorArtResult, error) {
	result := &PriorArtResult{Query: query}
	words := strings.Fields(query)
	if len(words) == 0 {
		return result, nil
	}

	// Search patents — try full query first, then fall back to individual keywords
	// so "processo nopol catalisador" still finds "NOPOL'S ACETYLATION PROCESS".
	if kind == "patent" || kind == "both" {
		seen := map[int64]bool{}
		searchTerms := uniqueTerms(query)

		for _, term := range searchTerms {
			f := domain.PatentFilter{Search: term, Limit: 10}
			f.Normalize()
			patents, err := s.patents.List(ctx, f)
			if err != nil {
				return nil, fmt.Errorf("prior art patent search: %w", err)
			}
			for _, p := range patents {
				if seen[p.ID] {
					continue
				}
				seen[p.ID] = true
				sim := estimateSimilarity(query, p.Title+" "+p.Abstract)
				result.Hits = append(result.Hits, PriorArtHit{
					Kind: "patent", ID: p.ID,
					Number: p.ApplicationNumber, Title: p.Title,
					Owner: p.Applicant, SimilarityPct: sim,
				})
			}
		}
	}

	// Search trademarks
	if kind == "trademark" || kind == "both" {
		f := domain.TrademarkFilter{Search: query, Limit: 10}
		f.Normalize()
		marks, err := s.trademarks.List(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("prior art trademark search: %w", err)
		}
		for _, m := range marks {
			sim := estimateSimilarity(query, m.Name)
			result.Hits = append(result.Hits, PriorArtHit{
				Kind: "trademark", ID: m.ID,
				Number: m.ProcessNumber, Title: m.Name,
				Owner: m.Owner, SimilarityPct: sim,
			})
		}
	}

	// Calculate risk score: max similarity / 10, clamped 0-10
	var maxSim int
	for _, h := range result.Hits {
		if h.SimilarityPct > maxSim {
			maxSim = h.SimilarityPct
		}
	}
	result.RiskScore = float64(maxSim) / 10.0

	// Sort hits by similarity descending
	sortHitsBySimiliarity(result.Hits)
	return result, nil
}

// estimateSimilarity computes a naive word-overlap similarity (0-100).
func estimateSimilarity(query, target string) int {
	qWords := tokenize(query)
	tWords := tokenize(target)
	if len(qWords) == 0 || len(tWords) == 0 {
		return 0
	}

	tSet := make(map[string]struct{}, len(tWords))
	for _, w := range tWords {
		tSet[w] = struct{}{}
	}

	overlap := 0
	for _, w := range qWords {
		if _, ok := tSet[w]; ok {
			overlap++
		}
	}

	// Jaccard-ish: overlap / union
	union := len(qWords) + len(tWords) - overlap
	if union == 0 {
		return 0
	}
	sim := (overlap * 100) / union
	if sim > 100 {
		return 100
	}
	return sim
}

// uniqueTerms returns the full query string plus any individual word with 4+ chars
// that's not a Portuguese stop word. This ensures multi-word queries still hit
// the DB via keyword fallback.
var priorArtStopWords = map[string]bool{
	"para": true, "com": true, "que": true, "uma": true, "dos": true,
	"das": true, "por": true, "como": true, "mais": true, "seu": true,
	"sua": true, "não": true, "este": true, "esta": true, "esse": true,
	"essa": true, "the": true, "and": true, "for": true, "with": true,
	"that": true, "from": true, "this": true,
}

func uniqueTerms(query string) []string {
	terms := []string{query} // always try full query first
	seen := map[string]bool{query: true}
	for _, w := range strings.Fields(strings.ToLower(query)) {
		w = strings.Trim(w, ".,;:!?\"'()-")
		if len(w) >= 4 && !priorArtStopWords[w] && !seen[w] {
			terms = append(terms, w)
			seen[w] = true
		}
	}
	return terms
}

func tokenize(s string) []string {
	s = strings.ToLower(s)
	var words []string
	for _, w := range strings.Fields(s) {
		w = strings.Trim(w, ".,;:!?\"'()-")
		if len(w) > 2 {
			words = append(words, w)
		}
	}
	return words
}

func sortHitsBySimiliarity(hits []PriorArtHit) {
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].SimilarityPct > hits[j-1].SimilarityPct; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
}
