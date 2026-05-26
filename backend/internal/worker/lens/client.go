// Package lens implements a client for the Lens.org Scholarly API.
//
// Lens.org is a free patent and scholarly database. Register at:
//   https://www.lens.org/lens/user/subscriptions#scholar
//
// Set LENS_API_TOKEN in your environment.
// Free tier: 200 requests/day, 50 results/request.
package lens

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
)

const (
	lensScholarURL = "https://api.lens.org/scholarly/search"
	lensPatentURL  = "https://api.lens.org/patent/search"
	maxPageSize    = 50
)

// Client is a Lens.org API client.
type Client struct {
	token      string
	httpClient *http.Client
	log        *slog.Logger
}

// NewClient creates a Lens.org client. token is your Bearer token.
func NewClient(token string, log *slog.Logger) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log,
	}
}

// ─── Scholar search ───────────────────────────────────────────────────────────

type scholarQuery struct {
	Query struct {
		Match struct {
			Title string `json:"title,omitempty"`
		} `json:"match,omitempty"`
		Bool *boolQuery `json:"bool,omitempty"`
	} `json:"query"`
	Size      int      `json:"size"`
	From      int      `json:"from"`
	Sort      []sortSpec `json:"sort,omitempty"`
	Include   []string `json:"include,omitempty"`
}

type boolQuery struct {
	Must []interface{} `json:"must,omitempty"`
}

type sortSpec map[string]map[string]string

type ScholarResult struct {
	Total int               `json:"total"`
	Data  []ScholarRecord   `json:"data"`
}

type ScholarRecord struct {
	LensID        string    `json:"lens_id"`
	Title         string    `json:"title"`
	Abstract      string    `json:"abstract"`
	Authors       []ScholarAuthor `json:"authors"`
	Affiliations  []ScholarAff   `json:"institutions"`
	DOI           string    `json:"doi"`
	PublishedDate string    `json:"date_published"`
	CitationCount int       `json:"scholarly_citations_count"`
	SourceType    string    `json:"publication_type"`
	Source        struct {
		Title string `json:"title"`
	} `json:"source"`
	Keywords []string `json:"keywords"`
	URLs     []string `json:"external_ids"`
}

type ScholarAuthor struct {
	Name string `json:"display_name"`
}

type ScholarAff struct {
	Name string `json:"name"`
}

// SearchScholar searches Lens.org for scholarly publications.
func (c *Client) SearchScholar(ctx context.Context, query string, size, from int) (*ScholarResult, error) {
	if c.token == "" {
		return nil, fmt.Errorf("lens: LENS_API_TOKEN not configured")
	}
	if size <= 0 || size > maxPageSize {
		size = maxPageSize
	}

	body := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  query,
				"fields": []string{"title", "abstract", "keywords"},
			},
		},
		"size":    size,
		"from":    from,
		"include": []string{"lens_id", "title", "abstract", "authors", "date_published",
			"doi", "scholarly_citations_count", "publication_type", "source", "keywords"},
	}

	return c.doScholarRequest(ctx, body)
}

// SearchByAffiliation searches for publications from a specific institution.
func (c *Client) SearchByAffiliation(ctx context.Context, institution string, size, from int) (*ScholarResult, error) {
	if c.token == "" {
		return nil, fmt.Errorf("lens: LENS_API_TOKEN not configured")
	}

	body := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"author.affiliation.name": institution,
			},
		},
		"size": size,
		"from": from,
		"include": []string{"lens_id", "title", "abstract", "authors", "date_published",
			"doi", "scholarly_citations_count", "publication_type", "source"},
	}

	return c.doScholarRequest(ctx, body)
}

func (c *Client) doScholarRequest(ctx context.Context, body interface{}) (*ScholarResult, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("lens marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, lensScholarURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("lens build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("lens request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("lens: invalid or missing API token")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("lens: rate limit exceeded — free tier allows 200 req/day")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lens: HTTP %d", resp.StatusCode)
	}

	var result ScholarResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("lens decode: %w", err)
	}

	c.log.Info("lens: search completed", "query", "...", "total", result.Total, "returned", len(result.Data))
	return &result, nil
}

// ─── Mapping to domain.Publication ───────────────────────────────────────────

// ToPublication converts a ScholarRecord to a domain.Publication.
func (r *ScholarRecord) ToPublication() *domain.Publication {
	authors := make([]string, 0, len(r.Authors))
	for _, a := range r.Authors {
		if a.Name != "" {
			authors = append(authors, a.Name)
		}
	}

	affiliations := make([]string, 0, len(r.Affiliations))
	for _, a := range r.Affiliations {
		if a.Name != "" {
			affiliations = append(affiliations, a.Name)
		}
	}

	var pubDate *time.Time
	for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
		if t, err := time.Parse(layout, strings.TrimSpace(r.PublishedDate)); err == nil {
			pubDate = &t
			break
		}
	}

	kind := domain.PublicationKindArticle
	switch strings.ToLower(r.SourceType) {
	case "conference_proceedings", "conference":
		kind = domain.PublicationKindConference
	case "book", "book_chapter":
		kind = domain.PublicationKindBook
	case "preprint":
		kind = domain.PublicationKindPreprint
	case "review":
		kind = domain.PublicationKindReview
	case "thesis", "dissertation":
		kind = domain.PublicationKindThesis
	}

	return &domain.Publication{
		Source:        domain.PublicationSourceLens,
		ExternalID:    r.LensID,
		DOI:           r.DOI,
		Title:         r.Title,
		Abstract:      r.Abstract,
		Authors:       authors,
		Affiliations:  affiliations,
		Kind:          kind,
		Journal:       r.Source.Title,
		PublishedDate: pubDate,
		CitationCount: r.CitationCount,
		Keywords:      r.Keywords,
	}
}
