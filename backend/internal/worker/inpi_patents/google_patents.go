// Package inpi_patents — GooglePatentsClient busca patentes UFOP no
// Google Patents (que indexa INPI + USPTO + EPO + WIPO).
//
// API descoberta: https://patents.google.com/xhr/query?url=assignee%3D...&page=N
// Free e sem auth, mas:
//   - Não-oficial: pode quebrar
//   - Rate limit não documentado — usar 1 req/s
//   - HTML tem tags <b> de highlight — limpamos
//
// Para UFOP: assignee="Universidade Federal de Ouro Preto" retorna 261
// patentes (verificado em 2026-05).
package inpipatents

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const googlePatentsEndpoint = "https://patents.google.com/xhr/query"

// GooglePatentsClient.
type GooglePatentsClient struct {
	client *http.Client
	log    *slog.Logger
}

func NewGooglePatentsClient(log *slog.Logger) *GooglePatentsClient {
	return &GooglePatentsClient{
		client: &http.Client{Timeout: 30 * time.Second},
		log:    log,
	}
}

// PatentResult é o que extraímos.
type PatentResult struct {
	GoogleID        string
	ApplicationNumber string // ex: "AU2018200535B2", "WO2014000077A1"
	Title           string
	Abstract        string
	Assignee        string
	FilingDate      string // "2018-01-23"
	PublicationDate string
	Country         string // "AU", "WO", "US", "BR"
}

// Search retorna patentes do assignee paginadas.
//
//	assignee — nome do depositante (será URL-encoded)
//	pages    — quantas páginas pegar (cada página tem ~10 results)
func (c *GooglePatentsClient) Search(ctx context.Context, assignee string, pages int) ([]PatentResult, error) {
	if pages <= 0 {
		pages = 1
	}
	if pages > 30 {
		pages = 30 // safety
	}

	var all []PatentResult

	for page := 0; page < pages; page++ {
		if ctx.Err() != nil {
			return all, ctx.Err()
		}

		results, err := c.searchPage(ctx, assignee, page)
		if err != nil {
			c.log.Warn("google patents: page failed", "page", page, "err", err)
			break
		}
		if len(results) == 0 {
			break // fim
		}
		all = append(all, results...)

		// Rate limit gentil
		select {
		case <-ctx.Done():
			return all, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	c.log.Info("google patents: search done", "assignee", assignee, "total", len(all))
	return all, nil
}

func (c *GooglePatentsClient) searchPage(ctx context.Context, assignee string, page int) ([]PatentResult, error) {
	// Constrói URL com filtro de assignee
	innerQuery := fmt.Sprintf(`assignee="%s"`, assignee)
	endpoint := fmt.Sprintf("%s?url=%s&page=%d&exp=",
		googlePatentsEndpoint, url.QueryEscape(innerQuery), page)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 Argos-IP-Intelligence/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	// Parse JSON
	var raw struct {
		Results struct {
			TotalNumResults int `json:"total_num_results"`
			Cluster []struct {
				Result []struct {
					ID     string `json:"id"`
					Patent struct {
						Title           string `json:"title"`
						Snippet         string `json:"snippet"`
						Assignee        string `json:"assignee"`
						FilingDate      string `json:"filing_date"`
						PublicationDate string `json:"publication_date"`
					} `json:"patent"`
				} `json:"result"`
			} `json:"cluster"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	var out []PatentResult
	for _, cluster := range raw.Results.Cluster {
		for _, r := range cluster.Result {
			out = append(out, PatentResult{
				GoogleID:          r.ID,
				ApplicationNumber: extractAppNumber(r.ID),
				Country:           extractCountry(r.ID),
				Title:             cleanHTML(r.Patent.Title),
				Abstract:          cleanHTML(r.Patent.Snippet),
				Assignee:          cleanHTML(r.Patent.Assignee),
				FilingDate:        r.Patent.FilingDate,
				PublicationDate:   r.Patent.PublicationDate,
			})
		}
	}
	return out, nil
}

// extractAppNumber — de "patent/AU2018200535B2/en" → "AU2018200535B2"
func extractAppNumber(googleID string) string {
	parts := strings.Split(googleID, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return googleID
}

// extractCountry — primeiros 2 chars da app number (AU, WO, US, BR, EP, etc)
func extractCountry(googleID string) string {
	app := extractAppNumber(googleID)
	if len(app) >= 2 {
		return app[:2]
	}
	return ""
}

var htmlTag = regexp.MustCompile(`<[^>]+>`)
var htmlEntity = regexp.MustCompile(`&[a-z]+;`)

func cleanHTML(s string) string {
	s = htmlTag.ReplaceAllString(s, "")
	s = htmlEntity.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "&hellip;": return "…"
		case "&amp;":    return "&"
		case "&lt;":     return "<"
		case "&gt;":     return ">"
		case "&quot;":   return `"`
		case "&nbsp;":   return " "
		}
		return ""
	})
	return strings.TrimSpace(s)
}
