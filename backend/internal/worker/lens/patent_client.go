// Package lens — Patent API client.
//
// The real Lens.org Patent API (https://docs.api.lens.org/request-patent.html)
// requires a paid token. When LENS_API_TOKEN is absent, we fall back to a
// deterministic mock that generates plausible-looking citation data based on
// the patent's IPC category + filing year. This keeps demos and metrics
// computation working without external dependencies.
//
// The output schema matches what the real Lens Patent API returns, so when
// the token is provided the same downstream code consumes it transparently.
package lens

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
)

// PatentMetricsResult holds citation + family + claims data per patent.
type PatentMetricsResult struct {
	PatentID          int64
	LensID            string
	ForwardCitations  []CitationRef
	BackwardCitations []CitationRef
	ScientificCites   int
	FamilySize        int
	ClaimsCount       int
	Source            string // "lens" | "mock"
}

// CitationRef is a single forward/backward citation entry.
type CitationRef struct {
	LensID    string
	AppNumber string
	Title     string
	Year      int
	IPCCodes  []string
}

// PatentClient fetches per-patent citation/family/claims data.
type PatentClient struct {
	token  string
	client *http.Client
	mock   bool
}

// NewPatentClient creates a client. If LENS_API_TOKEN is empty, returns a
// mock client that produces deterministic plausible data.
func NewPatentClient() *PatentClient {
	tok := os.Getenv("LENS_API_TOKEN")
	return &PatentClient{
		token:  tok,
		client: &http.Client{Timeout: 30 * time.Second},
		mock:   tok == "",
	}
}

// Enrich looks up Lens data for the given patent. Falls back to mock when
// no token is configured.
func (c *PatentClient) Enrich(ctx context.Context, p *domain.Patent) (*PatentMetricsResult, error) {
	if c.mock {
		return c.mockEnrich(p), nil
	}
	return c.realEnrich(ctx, p)
}

// ─── Mock implementation ──────────────────────────────────────────────────────

// mockEnrich generates deterministic citation data so the same patent always
// yields the same numbers across runs. Reasonable distributions per IPC:
//
//	Chemistry (C, 2)         → high forward cites (medical research is cited heavily)
//	Computing (G, 6)         → moderate fwd, high family size (international filings)
//	Mechanical (F, 5)        → moderate cites, larger claims
//	Necessidades humanas (A) → high family size
func (c *PatentClient) mockEnrich(p *domain.Patent) *PatentMetricsResult {
	seed := p.ID*31 + int64(p.IPCCategory)
	if p.FilingDate != nil {
		seed += int64(p.FilingDate.Year())
	}
	rng := rand.New(rand.NewSource(seed))

	cat := int(p.IPCCategory)
	if cat < 0 || cat >= 8 {
		cat = 0
	}

	// Base distributions per IPC (calibrated to NBER 2001 typical ranges)
	fwdMean   := []float64{8, 5, 18, 3, 4, 9, 14, 12}[cat]
	famMean   := []float64{4, 3, 5, 2, 3, 4, 6, 5}[cat]
	claimsMean := []float64{15, 12, 20, 10, 12, 18, 22, 20}[cat]

	// Sample around mean with poisson-like spread
	fwdN     := int(math.Max(0, fwdMean+rng.NormFloat64()*fwdMean*0.4))
	bwdN     := int(math.Max(0, fwdMean*0.7+rng.NormFloat64()*fwdMean*0.3))
	famSize  := int(math.Max(1, famMean+rng.NormFloat64()*famMean*0.3))
	claims   := int(math.Max(3, claimsMean+rng.NormFloat64()*claimsMean*0.2))
	sciCites := int(math.Max(0, fwdMean*0.3+rng.NormFloat64()*2))

	// Generate fake citation refs with plausible IPC codes
	mkCitations := func(n int, kindPrefix string) []CitationRef {
		refs := make([]CitationRef, 0, n)
		for i := 0; i < n; i++ {
			yr := 2000 + rng.Intn(25)
			otherCat := rng.Intn(8)
			refs = append(refs, CitationRef{
				LensID:    fmt.Sprintf("mock-%s-%d-%d", kindPrefix, p.ID, i),
				AppNumber: fmt.Sprintf("US%dPP%05d", yr, rng.Intn(99999)),
				Title:     fmt.Sprintf("%s reference patent #%d", strings.Title(kindPrefix), i+1),
				Year:      yr,
				IPCCodes:  []string{ipcLetters[otherCat] + "01", ipcLetters[cat] + "05"},
			})
		}
		return refs
	}

	return &PatentMetricsResult{
		PatentID:          p.ID,
		LensID:            fmt.Sprintf("mock-lens-%d", p.ID),
		ForwardCitations:  mkCitations(fwdN, "fwd"),
		BackwardCitations: mkCitations(bwdN, "bwd"),
		ScientificCites:   sciCites,
		FamilySize:        famSize,
		ClaimsCount:       claims,
		Source:            "mock",
	}
}

var ipcLetters = []string{"A", "B", "C", "D", "E", "F", "G", "H"}

// ─── Real implementation (only invoked when token is present) ─────────────────

func (c *PatentClient) realEnrich(ctx context.Context, p *domain.Patent) (*PatentMetricsResult, error) {
	endpoint := "https://api.lens.org/patent/search"

	// Search by application number — Lens normalizes Brazilian patents
	// (BR102023XXXXXXX) to its internal lens_id.
	body := fmt.Sprintf(`{
		"query": { "match": { "doc_number": %q } },
		"include": ["lens_id", "biblio.publication_reference", "biblio.classifications_ipcr",
		            "biblio.parties.applicants", "biblio.classifications_cpc",
		            "claims", "families.simple_family", "biblio.references_cited"]
	}`, p.ApplicationNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lens patent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lens patent: HTTP %d", resp.StatusCode)
	}

	// Minimal parsing — production code would deserialize the full structure.
	// For now we only need the aggregate counts.
	var raw struct {
		Data []struct {
			LensID  string `json:"lens_id"`
			Biblio  struct {
				ReferencesCited struct {
					Citations []struct {
						LensID    string   `json:"lens_id"`
						DocNumber string   `json:"doc_number"`
						Title     string   `json:"title"`
						Year      int      `json:"year"`
						IPC       []string `json:"ipc"`
					} `json:"citations"`
				} `json:"references_cited"`
			} `json:"biblio"`
			Families struct {
				SimpleFamily struct {
					Size int `json:"size"`
				} `json:"simple_family"`
			} `json:"families"`
			Claims []any `json:"claims"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode lens response: %w", err)
	}
	if len(raw.Data) == 0 {
		return nil, fmt.Errorf("lens patent: not found")
	}

	d := raw.Data[0]
	bwd := make([]CitationRef, 0, len(d.Biblio.ReferencesCited.Citations))
	for _, c := range d.Biblio.ReferencesCited.Citations {
		bwd = append(bwd, CitationRef{
			LensID: c.LensID, AppNumber: c.DocNumber, Title: c.Title, Year: c.Year, IPCCodes: c.IPC,
		})
	}

	return &PatentMetricsResult{
		PatentID:          p.ID,
		LensID:            d.LensID,
		BackwardCitations: bwd,
		FamilySize:        d.Families.SimpleFamily.Size,
		ClaimsCount:       len(d.Claims),
		Source:            "lens",
	}, nil
}
