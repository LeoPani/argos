// Package ufop — portal_scraper fetches news items from the UFOP web
// portal (ufop.br) and filters them by intellectual-property keywords.
//
// The scraper intentionally avoids fragile CSS-selector parsing:
// it walks the HTML token stream, collects <a> elements with href text
// containing /noticias/, and then scores each title against a keyword
// list.  No third-party HTML library is required beyond
// golang.org/x/net/html.
package ufop

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	portalBase     = "https://www.ufop.br"
	portalNewsPath = "/noticias"
	portalSource   = "portal"

	scraperTimeout  = 20 * time.Second
	scraperMaxItems = 50 // safety cap per run
)

// piKeywords are lowercase search terms that suggest IP relevance.
var piKeywords = []string{
	"patente", "invenção", "invento", "processo", "método", "tecnologia",
	"inovação", "transferência de tecnologia", "propriedade intelectual",
	"licenciamento", "royalt", "registro", "composição", "dispositivo",
	"sistema", "aparelho", "pesquisa", "desenvolvimento", "produto",
	"biotecnologia", "software", "algoritmo", "nanotecnologia",
}

// PortalNews is a news item scraped from the UFOP portal.
type PortalNews struct {
	Title    string
	URL      string
	Date     *time.Time
	Keywords []string // matched PI keywords (lowercase)
	Abstract string   // lead paragraph or snippet
}

// PortalScraper fetches and parses UFOP news pages.
type PortalScraper struct {
	client *http.Client
	log    *slog.Logger
}

// NewPortalScraper creates a portal scraper.
func NewPortalScraper(log *slog.Logger) *PortalScraper {
	return &PortalScraper{
		client: &http.Client{Timeout: scraperTimeout},
		log:    log,
	}
}

// ScrapeNews fetches the UFOP news listing and returns items that
// contain at least one PI keyword in title or snippet.
func (s *PortalScraper) ScrapeNews(ctx context.Context) ([]PortalNews, error) {
	listURL := portalBase + portalNewsPath

	links, err := s.scrapeNewsList(ctx, listURL)
	if err != nil {
		return nil, fmt.Errorf("portal scraper: scrape list: %w", err)
	}
	s.log.Info("portal scraper: found news links", "count", len(links))

	var results []PortalNews
	for i, link := range links {
		if i >= scraperMaxItems {
			break
		}
		if ctx.Err() != nil {
			return results, ctx.Err()
		}

		news, err := s.scrapeArticle(ctx, link)
		if err != nil {
			s.log.Warn("portal scraper: skip article", "url", link, "err", err)
			continue
		}
		if news == nil {
			continue // no PI keywords
		}
		results = append(results, *news)

		// Rate-limit — be a good citizen.
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}

	s.log.Info("portal scraper: done", "pi_matches", len(results))
	return results, nil
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// scrapeNewsList fetches the news listing page and returns hrefs.
func (s *PortalScraper) scrapeNewsList(ctx context.Context, listURL string) ([]string, error) {
	body, err := s.fetch(ctx, listURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var links []string
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				href := strings.TrimSpace(attr.Val)
				if href == "" || href == "#" {
					continue
				}
				resolved := resolveURL(portalBase, href)
				if resolved == "" || seen[resolved] {
					continue
				}
				if isNewsLink(resolved) {
					seen[resolved] = true
					links = append(links, resolved)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return links, nil
}

// scrapeArticle fetches a single article and returns PortalNews if PI
// keywords match, or nil if not relevant.
func (s *PortalScraper) scrapeArticle(ctx context.Context, articleURL string) (*PortalNews, error) {
	body, err := s.fetch(ctx, articleURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	doc, err := html.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	title, abstract, dateStr := extractArticleContent(doc)
	if title == "" {
		return nil, nil
	}

	combined := strings.ToLower(title + " " + abstract)
	matched := matchedKeywords(combined)
	if len(matched) == 0 {
		return nil, nil
	}

	var pubDate *time.Time
	for _, layout := range []string{"02/01/2006", "2006-01-02", "January 2, 2006"} {
		if t, err := time.Parse(layout, strings.TrimSpace(dateStr)); err == nil {
			pubDate = &t
			break
		}
	}

	return &PortalNews{
		Title:    title,
		URL:      articleURL,
		Date:     pubDate,
		Keywords: matched,
		Abstract: truncate(abstract, 800),
	}, nil
}

// fetch performs a GET request and returns the response body.
func (s *PortalScraper) fetch(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request %s: %w", u, err)
	}
	req.Header.Set("User-Agent", "Argos-IP-Intelligence/1.0 (+https://github.com/LeoPani/argos)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get %s: %w", u, err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("http %d for %s", resp.StatusCode, u)
	}
	return resp.Body, nil
}

// ─── HTML helpers ─────────────────────────────────────────────────────────────

// extractArticleContent returns title, first-paragraph text and date hint.
func extractArticleContent(doc *html.Node) (title, body, date string) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1", "h2":
				if title == "" {
					title = strings.TrimSpace(textContent(n))
				}
			case "time":
				if date == "" {
					for _, a := range n.Attr {
						if a.Key == "datetime" {
							date = a.Val
						}
					}
					if date == "" {
						date = strings.TrimSpace(textContent(n))
					}
				}
			case "p":
				if len(body) < 400 {
					txt := strings.TrimSpace(textContent(n))
					if len(txt) > 50 {
						if body != "" {
							body += " "
						}
						body += txt
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return title, body, date
}

// textContent extracts all visible text from a node subtree.
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

// isNewsLink returns true for hrefs that look like UFOP news article pages.
func isNewsLink(href string) bool {
	u, err := url.Parse(href)
	if err != nil {
		return false
	}
	path := u.Path
	return strings.Contains(path, "/noticia") ||
		strings.Contains(path, "/noticias/") ||
		strings.Contains(path, "/news/")
}

// resolveURL resolves href relative to base.
func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "/") {
		return base + href
	}
	return "" // relative non-root paths ignored
}

// matchedKeywords returns which piKeywords appear in the text.
func matchedKeywords(lower string) []string {
	var found []string
	for _, kw := range piKeywords {
		if strings.Contains(lower, kw) {
			found = append(found, kw)
		}
	}
	return found
}

// truncate limits a string to n bytes at a word boundary.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if idx := strings.LastIndex(cut, " "); idx > 0 {
		cut = cut[:idx]
	}
	return cut + "…"
}
