// Package ufop implements scrapers for UFOP academic sources.
//
// OAI-PMH endpoint: https://repositorio.ufop.br/oai/request
// Protocol: OAI-PMH 2.0 (Open Archives Initiative Protocol for Metadata Harvesting)
// Format:   oai_dc (Dublin Core) and oai_etdms (theses)
package ufop

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
)

const (
	// DSpace 7 oficial endpoint (path /server/oai/request).
	// O legado /oai/request redireciona para a SPA Angular e retorna HTML, não XML.
	oaiEndpoint = "https://repositorio.ufop.br/server/oai/request"
	oaiSource   = domain.PublicationSource("ufop_oai")
)

// Sets oficiais do DSpace UFOP (descobertos via ListSets em 2026).
// Documentação acadêmica: discovered by querying /server/oai/request?verb=ListSets.
const (
	// — Direito (graduação + pós) —
	UFOPSetDepDireito    = "com_123456789_656"   // DEDIR  — Departamento de Direito (graduação/TCCs)
	UFOPSetEscolaDireito = "com_123456789_653"   // EDTM   — Escola de Direito, Turismo e Museologia
	UFOPSetPPGDireito    = "com_123456789_10890" // PPG-Direito — pós-graduação stricto sensu

	// — Engenharia de Minas + Escola de Minas (graduação + pós) —
	UFOPSetDepEngMinas   = "com_123456789_510"   // DEMIN  — Departamento de Engenharia de Minas (graduação)
	UFOPSetEscolaMinas   = "com_123456789_6"     // EM     — Escola de Minas (umbrella histórica)
	UFOPSetPPGEngMineral = "com_123456789_576"   // PPGEM  — PPG em Engenharia Mineral
	UFOPSetDepGeologia   = "com_123456789_8"     // DEGEO  — Geologia (complementar à mineração)
)

// OAIClient fetches records from UFOP's OAI-PMH repository.
type OAIClient struct {
	client *http.Client
	log    *slog.Logger
}

// NewOAIClient creates an OAI-PMH client.
func NewOAIClient(log *slog.Logger) *OAIClient {
	return &OAIClient{
		client: &http.Client{Timeout: 30 * time.Second},
		log:    log,
	}
}

// ─── OAI-PMH XML structs ──────────────────────────────────────────────────────

type oaiResponse struct {
	XMLName     xml.Name       `xml:"OAI-PMH"`
	Error       oaiError       `xml:"error"`
	ListRecords oaiListRecords `xml:"ListRecords"`
	// ResumptionToken já está dentro de ListRecords (nested struct), não duplicar.
}

type oaiError struct {
	Code    string `xml:"code,attr"`
	Message string `xml:",chardata"`
}

type oaiListRecords struct {
	Records         []oaiRecord `xml:"record"`
	ResumptionToken string      `xml:"resumptionToken"`
}

type oaiRecord struct {
	Header   oaiHeader   `xml:"header"`
	Metadata oaiMetadata `xml:"metadata"`
}

type oaiHeader struct {
	Identifier string `xml:"identifier"`
	Datestamp  string `xml:"datestamp"`
	Status     string `xml:"status,attr"`
}

type oaiMetadata struct {
	DC oaiDC `xml:"dc"`
}

type oaiDC struct {
	Titles       []string `xml:"title"`
	Creators     []string `xml:"creator"`
	Subjects     []string `xml:"subject"`
	Descriptions []string `xml:"description"`
	Publishers   []string `xml:"publisher"`
	Dates        []string `xml:"date"`
	Types        []string `xml:"type"`
	Identifiers  []string `xml:"identifier"`
}

// HarvestResult is returned by Harvest.
type HarvestResult struct {
	Publications []*domain.Publication
	Total        int
}

// Harvest fetches records from the UFOP repository.
//   from        — filter by datestamp >= from (RFC 3339 date, ex: "2023-01-01"). Empty = all.
//   set         — DSpace community/collection spec (UFOPSetDepDireito, etc). Empty = all.
//   maxRecords  — cap. The N MOST RECENT records (by publication date) are returned.
//
// OAI-PMH não tem ordenação explícita, então coletamos *3× maxRecords*
// e ordenamos por PublishedDate desc localmente. Isso garante "mais novos"
// sem pegar o repositório inteiro.
func (c *OAIClient) Harvest(ctx context.Context, from, set string, maxRecords int) (*HarvestResult, error) {
	result := &HarvestResult{}

	// Coletamos um buffer maior pra poder ordenar por data desc depois.
	collectBuffer := maxRecords * 3
	if collectBuffer == 0 {
		collectBuffer = 0 // sem limite quando caller pede tudo
	}

	params := url.Values{
		"verb":           {"ListRecords"},
		"metadataPrefix": {"oai_dc"},
	}
	if from != "" {
		params.Set("from", from)
	}
	if set != "" {
		params.Set("set", set)
	}

	resumptionToken := ""
	for {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		var reqParams url.Values
		if resumptionToken != "" {
			reqParams = url.Values{
				"verb":            {"ListRecords"},
				"resumptionToken": {resumptionToken},
			}
		} else {
			reqParams = params
		}

		oaiResp, err := c.request(ctx, reqParams)
		if err != nil {
			return result, err
		}
		if oaiResp.Error.Code != "" {
			if oaiResp.Error.Code == "noRecordsMatch" {
				break
			}
			return result, fmt.Errorf("oai error %s: %s", oaiResp.Error.Code, oaiResp.Error.Message)
		}

		for _, rec := range oaiResp.ListRecords.Records {
			if rec.Header.Status == "deleted" {
				continue
			}
			pub := mapOAIToDomain(rec)
			if pub != nil {
				result.Publications = append(result.Publications, pub)
				result.Total++
			}
			if collectBuffer > 0 && result.Total >= collectBuffer {
				// Coletou buffer suficiente, vai ordenar + truncar abaixo
				goto sortAndTrim
			}
		}

		resumptionToken = oaiResp.ListRecords.ResumptionToken
		if resumptionToken == "" {
			break
		}
		c.log.Info("ufop oai: resuming", "token", resumptionToken[:min(20, len(resumptionToken))], "so_far", result.Total)

		// Respectful rate limiting.
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

sortAndTrim:
	// Ordena por PublishedDate desc (mais novos primeiro).
	// Records sem PublishedDate vão pro fim.
	sort.SliceStable(result.Publications, func(i, j int) bool {
		a, b := result.Publications[i].PublishedDate, result.Publications[j].PublishedDate
		if a == nil && b == nil {
			return false
		}
		if a == nil {
			return false
		}
		if b == nil {
			return true
		}
		return a.After(*b)
	})

	// Trunca para maxRecords após ordenar
	if maxRecords > 0 && len(result.Publications) > maxRecords {
		result.Publications = result.Publications[:maxRecords]
		result.Total = maxRecords
	}

	c.log.Info("ufop oai: harvest complete (sorted desc)", "total", result.Total, "set", set)
	return result, nil
}

func (c *OAIClient) request(ctx context.Context, params url.Values) (*oaiResponse, error) {
	reqURL := oaiEndpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build oai request: %w", err)
	}
	req.Header.Set("User-Agent", "Argos-IP-Intelligence/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("oai HTTP %d: %s", resp.StatusCode, body)
	}

	var oaiResp oaiResponse
	if err := xml.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("decode oai response: %w", err)
	}
	return &oaiResp, nil
}

// ─── Mapping ─────────────────────────────────────────────────────────────────

func mapOAIToDomain(rec oaiRecord) *domain.Publication {
	dc := rec.Metadata.DC

	title := ""
	for _, t := range dc.Titles {
		if t = strings.TrimSpace(t); t != "" {
			title = t
			break
		}
	}
	if title == "" {
		return nil
	}

	abstract := ""
	for _, d := range dc.Descriptions {
		if len(d) > len(abstract) {
			abstract = strings.TrimSpace(d)
		}
	}

	authors := make([]string, 0, len(dc.Creators))
	for _, cr := range dc.Creators {
		if cr = strings.TrimSpace(cr); cr != "" {
			authors = append(authors, cr)
		}
	}

	keywords := make([]string, 0, len(dc.Subjects))
	for _, s := range dc.Subjects {
		if s = strings.TrimSpace(s); s != "" {
			keywords = append(keywords, s)
		}
	}

	var pubDate *time.Time
	for _, d := range dc.Dates {
		for _, layout := range []string{"2006-01-02", "2006-01", "2006"} {
			if t, err := time.Parse(layout, strings.TrimSpace(d)); err == nil {
				pubDate = &t
				break
			}
		}
		if pubDate != nil {
			break
		}
	}

	kind := domain.PublicationKindOther
	for _, t := range dc.Types {
		tl := strings.ToLower(t)
		if strings.Contains(tl, "disserta") || strings.Contains(tl, "tese") || strings.Contains(tl, "thesis") {
			kind = domain.PublicationKindThesis
			break
		}
		if strings.Contains(tl, "article") || strings.Contains(tl, "artigo") {
			kind = domain.PublicationKindArticle
			break
		}
	}

	// DOI preferred, mas se ausente usamos o handle do repositório UFOP
	// (URL clicável que o usuário verá no site).
	doi := ""
	for _, id := range dc.Identifiers {
		if strings.HasPrefix(id, "http://dx.doi.org/") || strings.HasPrefix(id, "https://doi.org/") {
			doi = id
			break
		}
	}
	// Fallback: pega o primeiro handle do repositorio.ufop.br
	if doi == "" {
		for _, id := range dc.Identifiers {
			if strings.Contains(id, "repositorio.ufop.br/handle/") ||
				strings.Contains(id, "repositorio.ufop.br/jspui/handle/") {
				doi = id
				break
			}
		}
	}

	return &domain.Publication{
		Source:        domain.PublicationSourceManual, // using "manual" until ufop_oai is in constraint
		ExternalID:    rec.Header.Identifier,
		DOI:           doi,
		Title:         title,
		Abstract:      abstract,
		Authors:       authors,
		Affiliations:  []string{"Universidade Federal de Ouro Preto"},
		Kind:          kind,
		Journal:       "RI-UFOP",
		PublishedDate: pubDate,
		Keywords:      keywords,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
