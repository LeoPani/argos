// Package ufop — analyzer scores a publication or portal-news item
// for intellectual-property potential and maps it to a UFOPOpportunity.
//
// Scoring model (0-10 scale):
//
//	Title keywords (high signal)  +2.0 each, capped at 4.0
//	Abstract keywords             +0.8 each, capped at 4.0
//	IPC category bonus            +0-2.0 based on category PI-richness
//	                              ─────────────────────────────────────
//	Total (before cap):                                          0-10.0
//
// Level thresholds:
//
//	>= 6.0 → high
//	>= 3.5 → medium
//	<  3.5 → low
package ufop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/domain"
)

// ipcSuggestions maps BERT category id to a human-readable IPC section
// label. The model was trained on 8 macro-categories (A..H).
var ipcSuggestions = [8]string{
	"A — Necessidades Humanas",
	"B — Operações e Transportes",
	"C — Química e Metalurgia",
	"D — Têxteis e Papel",
	"E — Construção Civil",
	"F — Engenharia Mecânica",
	"G — Física / Tecnologia da Informação",
	"H — Eletricidade e Eletrônica",
}

// ipcCategoryBonus is an extra PI-potential weight per BERT category.
// Chemistry (2), Physics/IT (6) and Electricity (7) score highest in INPI filings.
var ipcCategoryBonus = [8]float64{
	1.0, // A
	1.2, // B
	2.0, // C — Química
	0.5, // D
	0.8, // E
	1.5, // F
	2.0, // G — TI
	1.8, // H
}

// titleKeywords are single-word signals that in a *title* strongly suggest
// a patentable output.
var titleKeywords = []string{
	"processo", "método", "sistema", "dispositivo", "composição",
	"material", "produto", "aparelho", "técnica", "solução",
	"inovação", "tecnologia", "modelo", "prototipo", "protótipo",
	"desenvolvimento", "invenção", "invento",
}

// abstractKeywords are additional single-word signals in the body text.
var abstractKeywords = []string{
	"patente", "patenteável", "novidade", "atividade inventiva",
	"propriedade intelectual", "licenciamento", "royalt",
	"transferência de tecnologia", "registro", "proteção",
	"aplicação industrial", "resultado técnico", "eficiência",
	"desempenho superior", "melhoria", "vantagem técnica",
}

// AnalyzeInput holds the text to score.
type AnalyzeInput struct {
	Title    string
	Abstract string
	Authors  []string
	// Provided by callers who already know the publication FK.
	PublicationID *int64
	ExternalID    string
	Source        domain.UFOPSource
	URL           string
	PublishedAt   *time.Time
	Department    string
}

// Analyzer wraps the AI service to score opportunities.
type Analyzer struct {
	ai ai.AIService
}

// NewAnalyzer creates an Analyzer backed by the given AIService.
func NewAnalyzer(ai ai.AIService) *Analyzer {
	return &Analyzer{ai: ai}
}

// Analyze classifies the text, scores PI potential, and returns a
// ready-to-persist UFOPOpportunity. If the BERT service is down the
// function still returns a (lower-scored) result using keyword analysis
// alone and IPC category 0 with similarity 0.
func (a *Analyzer) Analyze(ctx context.Context, in AnalyzeInput) (*domain.UFOPOpportunity, error) {
	// ── 1. Keyword scoring ────────────────────────────────────────────────
	titleLower := strings.ToLower(in.Title)
	abstractLower := strings.ToLower(in.Abstract)

	var titleScore, abstractScore float64
	for _, kw := range titleKeywords {
		if strings.Contains(titleLower, kw) {
			titleScore += 2.0
		}
	}
	if titleScore > 4.0 {
		titleScore = 4.0
	}

	occ := 0
	for _, kw := range abstractKeywords {
		if strings.Contains(abstractLower, kw) {
			occ++
		}
	}
	abstractScore = float64(occ) * 0.8
	if abstractScore > 4.0 {
		abstractScore = 4.0
	}

	// ── 2. IPC classification (BERT) ──────────────────────────────────────
	text := in.Title
	if in.Abstract != "" {
		text = in.Title + ". " + in.Abstract
	}

	ipcCatID := 0
	classifyCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if catID, err := a.ai.ClassifyPatent(classifyCtx, text); err == nil {
		ipcCatID = catID
	}
	// On BERT failure: category defaults to 0 (A), bonus is 1.0.

	bonus := ipcCategoryBonus[ipcCatID]
	piScore := titleScore + abstractScore + bonus
	if piScore > 10.0 {
		piScore = 10.0
	}

	// ── 3. Similarity estimate ────────────────────────────────────────────
	// Rough heuristic: higher title keyword density → lower similarity
	// (presumably more niche / novel).  Clamped 10-75.
	similarityPct := 55 - int(titleScore*5)
	if similarityPct < 10 {
		similarityPct = 10
	}
	if similarityPct > 75 {
		similarityPct = 75
	}

	// ── 4. Level assignment ───────────────────────────────────────────────
	level := domain.UFOPLevelLow
	switch {
	case piScore >= 6.0:
		level = domain.UFOPLevelHigh
	case piScore >= 3.5:
		level = domain.UFOPLevelMedium
	}

	// ── 5. AI analysis template ───────────────────────────────────────────
	analysis := buildAnalysis(in.Title, ipcCatID, level, piScore, occ+int(titleScore/2))

	opp := &domain.UFOPOpportunity{
		Source:        in.Source,
		ExternalID:    in.ExternalID,
		Title:         in.Title,
		Authors:       in.Authors,
		Department:    in.Department,
		Abstract:      in.Abstract,
		URL:           in.URL,
		PublishedAt:   in.PublishedAt,
		IPCSuggestion: ipcSuggestions[ipcCatID],
		IPCCategory:   domain.IPCCategory(ipcCatID),
		Level:         level,
		SimilarityPct: similarityPct,
		PIScore:       piScore,
		AIAnalysis:    analysis,
		Status:        domain.UFOPStatusNew,
		PublicationID: in.PublicationID,
	}
	return opp, nil
}

// buildAnalysis generates a human-readable analysis summary in Portuguese.
func buildAnalysis(title string, catID int, level domain.UFOPOpportunityLevel, score float64, kwCount int) string {
	category := ipcSuggestions[catID]

	var potential, recommendation string
	switch level {
	case domain.UFOPLevelHigh:
		potential = "alto potencial de patenteabilidade"
		recommendation = "Recomenda-se iniciar imediatamente uma consulta de anterioridade " +
			"e avaliar o depósito de pedido de patente junto ao INPI. " +
			"O NIT-UFOP pode auxiliar na redação da reivindicação."
	case domain.UFOPLevelMedium:
		potential = "potencial moderado de patenteabilidade"
		recommendation = "Recomenda-se uma análise mais aprofundada de novidade e atividade " +
			"inventiva antes de decidir pelo depósito. " +
			"Considere uma reunião com o NIT-UFOP para avaliação."
	default:
		potential = "baixo potencial imediato de PI"
		recommendation = "A publicação pode ser relevante para monitoramento de tendências " +
			"e formação de base de conhecimento. " +
			"Mantenha em observação para futuras correlações."
	}

	titleSnip := title
	if len(titleSnip) > 60 {
		titleSnip = titleSnip[:60] + "…"
	}

	return fmt.Sprintf(
		"A publicação \"%s\" apresenta %s na categoria IPC %s "+
			"(PI Score: %.1f/10). "+
			"Foram identificados %d indicadores de PI no título e resumo. "+
			"%s",
		titleSnip, potential, category, score, kwCount, recommendation,
	)
}
