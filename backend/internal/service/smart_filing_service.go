// Package service — SmartFilingService implementa o assistente acadêmico
// que ajuda o NIT-UFOP a avaliar um draft de invenção *antes* do depósito
// no INPI.
//
// Fluxo:
//   1. Inventor escreve título + abstract do projeto
//   2. BERT classifica o IPC (8 categorias A..H)
//   3. Buscamos patentes similares no portfolio (Search ILIKE)
//   4. Computamos um score de patenteabilidade combinando:
//      - distintividade do título (HJT-like, sobre palavras únicas)
//      - especificidade do abstract (>200 chars → mais defensável)
//      - novidade vs prior art interno (1 − max(similaridade))
//   5. Geramos um draft de claim estruturado (template)
//
// Referências metodológicas:
//
//   Lerner, J., & Seru, A. (2017). "The use and misuse of patent data."
//   Review of Financial Studies, 30(6), 2199-2244.
//   (justifica metricas alternativas quando citation data é escassa)
//
//   Bessen, J. (2008). "The value of U.S. patents by owner and patent
//   characteristics." Research Policy, 37(5), 932-945.
//   (caracteristicas de patente como proxy de qualidade)
//
//   Trajtenberg, M., Henderson, R., & Jaffe, A. (1997). "University versus
//   corporate patents: A window on the basicness of invention."
//   Economics of Innovation and New Technology, 5(1), 19-50.
//   (analise de patentes universitárias — relevante para UFOP)
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/ai/groqclassifier"
	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// SmartFilingService orchestrates the BERT call + prior art search.
type SmartFilingService struct {
	ai      ai.AIService
	patents repository.PatentRepository
	groq    *groqclassifier.Client // optional; nil → template claim
}

func NewSmartFilingService(ai ai.AIService, patents repository.PatentRepository) *SmartFilingService {
	return &SmartFilingService{ai: ai, patents: patents}
}

// WithGroq enables LLM-generated claim drafting.
func (s *SmartFilingService) WithGroq(gc *groqclassifier.Client) *SmartFilingService {
	s.groq = gc
	return s
}

// FilingInput is the inventor's draft.
type FilingInput struct {
	Title    string `json:"title"`
	Abstract string `json:"abstract"`
	Field    string `json:"field,omitempty"` // self-described area (free text)
}

// FilingSuggestion is the structured output from the assistant.
type FilingSuggestion struct {
	IPCCategory    int               `json:"ipc_category"`        // 0..7 ou -1
	IPCLetter      string            `json:"ipc_letter"`          // A..H
	IPCName        string            `json:"ipc_name"`            // "Química e Metalurgia"
	IPCConfidence  string            `json:"ipc_confidence"`      // "high" | "low" (se BERT respondeu)
	Distinctiveness float64          `json:"distinctiveness"`     // 0-100
	Specificity    float64           `json:"specificity"`         // 0-100 (abstract richness)
	NoveltyScore   float64           `json:"novelty_score"`       // 0-100 (1 - max similarity)
	OverallScore   float64           `json:"overall_score"`       // composite 0-100
	Recommendation string            `json:"recommendation"`      // proceed | refine | not_recommended
	FilingPriorArtHits   []FilingPriorArtHit     `json:"prior_art_hits"`      // top 5 similares
	SuggestedClaim string            `json:"suggested_claim"`     // template generated
	NextSteps      []string          `json:"next_steps"`
	Methodology    string            `json:"methodology"`         // "Lerner_Seru_2017"
}

// FilingPriorArtHit is one similar patent found in the internal portfolio.
type FilingPriorArtHit struct {
	PatentID          int64   `json:"patent_id"`
	ApplicationNumber string  `json:"application_number"`
	Title             string  `json:"title"`
	Applicant         string  `json:"applicant"`
	IPCCategory       int     `json:"ipc_category"`
	SimilarityPct     int     `json:"similarity_pct"`
	Status            string  `json:"status"`
}

// Analyze runs the full pipeline.
func (s *SmartFilingService) Analyze(ctx context.Context, in FilingInput) (*FilingSuggestion, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, fmt.Errorf("title required")
	}

	out := &FilingSuggestion{
		Methodology: "Lerner_Seru_2017",
	}

	// ── 1. BERT classification ──────────────────────────────────────────
	text := in.Title
	if in.Abstract != "" {
		text = in.Title + ". " + in.Abstract
	}
	catID, err := s.ai.ClassifyPatent(ctx, text)
	if err == nil && catID >= 0 && catID < 8 {
		out.IPCCategory = catID
		out.IPCLetter = ipcLetters[catID]
		out.IPCName = ipcNames[catID]
		out.IPCConfidence = "high"
	} else {
		out.IPCCategory = -1
		out.IPCConfidence = "low"
		out.IPCLetter = ""
		out.IPCName = "BERT indisponível — categoria não inferida"
	}

	// ── 2. Internal prior art search ────────────────────────────────────
	// Use abstract first (richer), fallback to title
	searchTerm := in.Abstract
	if len(searchTerm) < 30 {
		searchTerm = in.Title
	}

	// Heuristic: extract 3 most informative words for search
	keywords := topKeywords(searchTerm, 3)
	var hits []FilingPriorArtHit
	for _, kw := range keywords {
		patents, err := s.patents.List(ctx, domain.PatentFilter{Search: kw, Limit: 5})
		if err != nil {
			continue
		}
		for _, p := range patents {
			sim := titleAbstractSimilarity(in.Title+" "+in.Abstract, p.Title+" "+p.Abstract)
			hits = append(hits, FilingPriorArtHit{
				PatentID:          p.ID,
				ApplicationNumber: p.ApplicationNumber,
				Title:             p.Title,
				Applicant:         p.Applicant,
				IPCCategory:       int(p.IPCCategory),
				SimilarityPct:     int(sim * 100),
				Status:            string(p.Status),
			})
		}
	}
	hits = dedupAndTopN(hits, 5)
	out.FilingPriorArtHits = hits

	// ── 3. Score components ────────────────────────────────────────────
	out.Distinctiveness = math.Min(100, distinctiveness(in.Title)*10)
	out.Specificity     = math.Min(100, float64(len(in.Abstract))/4)

	maxSim := 0.0
	for _, h := range hits {
		if float64(h.SimilarityPct) > maxSim {
			maxSim = float64(h.SimilarityPct)
		}
	}
	out.NoveltyScore = math.Max(0, 100-maxSim)

	// Composite: weights inspired by Bessen (2008) value components
	out.OverallScore = mround1(
		0.30*out.Distinctiveness + 0.30*out.Specificity + 0.40*out.NoveltyScore,
	)

	// ── 4. Recommendation ──────────────────────────────────────────────
	var nextSteps []string
	switch {
	case out.OverallScore >= 70:
		out.Recommendation = "proceed"
		nextSteps = []string{
			"Encaminhar ao NIT-UFOP para depósito provisório (PI) ou modelo de utilidade (MU)",
			"Solicitar revisão técnica de claims com agente de PI antes do depósito",
			"Definir estratégia: depósito Brasil-only ou família internacional via PCT",
		}
	case out.OverallScore >= 45:
		out.Recommendation = "refine"
		nextSteps = []string{
			"Refinar o abstract para destacar diferenciais sobre prior art interno",
			"Considerar limitar reivindicações a aspectos mais inovadores",
			"Re-analisar após enriquecer descrição (alvo: score > 70)",
		}
	default:
		out.Recommendation = "not_recommended"
		nextSteps = []string{
			"Score baixo de patenteabilidade — alta sobreposição com prior art",
			"Avaliar publicação acadêmica como alternativa (preserva autoria)",
			"Re-conceber o projeto buscando ângulo realmente novo",
		}
	}
	out.NextSteps = nextSteps

	// ── 5. Generate claim (Groq LLM if available, else template) ────────
	if s.groq != nil {
		if claim, err := s.generateClaimWithGroq(ctx, in, out); err == nil && claim != "" {
			out.SuggestedClaim = claim
		} else {
			out.SuggestedClaim = generateClaim(in, out)
		}
	} else {
		out.SuggestedClaim = generateClaim(in, out)
	}

	return out, nil
}

// generateClaimWithGroq uses llama-3.3-70b to draft a real independent claim
// following Brazilian INPI standards (Lei 9.279/96 + Diretrizes INPI 2023).
func (s *SmartFilingService) generateClaimWithGroq(ctx context.Context, in FilingInput, sug *FilingSuggestion) (string, error) {
	ipcNote := ""
	if sug.IPCLetter != "" {
		ipcNote = fmt.Sprintf("Categoria IPC sugerida: %s (%s).", sug.IPCLetter, sug.IPCName)
	}

	priorArtNote := ""
	if len(sug.FilingPriorArtHits) > 0 {
		titles := make([]string, 0, len(sug.FilingPriorArtHits))
		for _, h := range sug.FilingPriorArtHits {
			titles = append(titles, h.Title)
		}
		priorArtNote = fmt.Sprintf("Prior art interno encontrado (evitar sobreposição): %s.", strings.Join(titles, "; "))
	}

	// Art. 10 exclusion check — detect excluded subject matter in title/abstract
	art10Alert := detectArt10Exclusions(in.Title + " " + in.Abstract)

	userMsg := fmt.Sprintf(`Elabore uma reivindicação independente para depósito no INPI conforme Lei 9.279/96.

TÍTULO: %s
ABSTRACT: %s
CAMPO: %s
%s
%s
%s

Instruções:
1. A reivindicação deve começar com o título da invenção (gerundio ou substantivo).
2. Use a estrutura: "[TÍTULO], caracterizado(a) por: a) [elemento 1]; b) [elemento 2]; ...".
3. Seja técnico e específico — extraia os diferenciais reais do abstract.
4. Adicione 2 reivindicações dependentes (claim 2 e claim 3).
5. Ao final, inclua uma seção "ALERTAS ART. 10 LPI" se houver exclusões potenciais.
6. Responda em JSON: {"claim": "...", "dependent_claims": ["...", "..."], "art10_alerts": ["..."]}`,
		in.Title, in.Abstract, in.Field, ipcNote, priorArtNote, art10Alert)

	type groqMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	payload := map[string]any{
		"model":       "llama-3.3-70b-versatile",
		"max_tokens":  1200,
		"temperature": 0.4,
		"messages": []groqMsg{
			{Role: "system", Content: "Você é um agente de Propriedade Intelectual especializado em reivindicações de patente brasileiras (INPI). Responda sempre em JSON válido."},
			{Role: "user", Content: userMsg},
		},
	}

	result, err := s.groq.RawChat(ctx, payload)
	if err != nil {
		return "", err
	}

	// Parse JSON response
	type claimResp struct {
		Claim           string   `json:"claim"`
		DependentClaims []string `json:"dependent_claims"`
		Art10Alerts     []string `json:"art10_alerts"`
	}
	var cr claimResp
	if err := json.Unmarshal([]byte(result), &cr); err != nil {
		// If JSON parse fails, return raw text
		return result, nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "REIVINDICAÇÃO 1 (independente)\n\n%s\n\n", cr.Claim)
	for i, dep := range cr.DependentClaims {
		fmt.Fprintf(&sb, "REIVINDICAÇÃO %d (dependente)\n\n%s\n\n", i+2, dep)
	}
	if len(cr.Art10Alerts) > 0 {
		sb.WriteString("⚠️ ALERTAS ART. 10 LPI (matéria excluída de patenteabilidade)\n")
		for _, a := range cr.Art10Alerts {
			fmt.Fprintf(&sb, "• %s\n", a)
		}
		sb.WriteString("\nConsulte um agente de PI antes de depositar.\n")
	}
	return sb.String(), nil
}

// detectArt10Exclusions checks for keywords associated with Art. 10 LPI exclusions
// (software per se, business methods, mental acts, printed matter, etc.)
func detectArt10Exclusions(text string) string {
	text = strings.ToLower(text)
	type exclusion struct{ keywords []string; desc string }
	exclusions := []exclusion{
		{[]string{"software", "programa de computador", "aplicativo", "app"}, "Art. 10, V: programas de computador per se são excluídos. Descreva o efeito técnico do hardware/sistema."},
		{[]string{"método de negócio", "modelo de negócio", "método comercial", "plano comercial"}, "Art. 10, III: métodos comerciais/de negócio são excluídos. Foque no processo técnico."},
		{[]string{"método matemático", "algoritmo", "cálculo"}, "Art. 10, I: métodos matemáticos são excluídos. Patentear apenas a aplicação técnica."},
		{[]string{"regras de jogo", "método de ensino", "método pedagógico"}, "Art. 10, IV: regras/métodos pedagógicos são excluídos."},
		{[]string{"obra literária", "musical", "artística", "cinematográfica"}, "Art. 10, VII: obras intelectuais/artísticas são protegidas por direito autoral, não patente."},
	}

	var found []string
	for _, ex := range exclusions {
		for _, kw := range ex.keywords {
			if strings.Contains(text, kw) {
				found = append(found, ex.desc)
				break
			}
		}
	}
	if len(found) == 0 {
		return ""
	}
	return "ALERTA ART. 10 LPI: " + strings.Join(found, " | ")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// distinctiveness — variedade de palavras únicas (HJT-light em texto livre).
func distinctiveness(s string) float64 {
	words := strings.Fields(strings.ToLower(s))
	if len(words) == 0 {
		return 0
	}
	seen := map[string]bool{}
	for _, w := range words {
		seen[w] = true
	}
	return float64(len(seen))
}

// titleAbstractSimilarity — Jaccard de bigrams (estável + leve).
func titleAbstractSimilarity(a, b string) float64 {
	bgA := bigrams(a)
	bgB := bigrams(b)
	if len(bgA) == 0 || len(bgB) == 0 {
		return 0
	}
	common := 0
	for k := range bgA {
		if bgB[k] {
			common++
		}
	}
	union := len(bgA) + len(bgB) - common
	if union == 0 {
		return 0
	}
	return float64(common) / float64(union)
}

func bigrams(s string) map[string]bool {
	words := tokenizeFiling(s)
	out := map[string]bool{}
	for i := 0; i+1 < len(words); i++ {
		out[words[i]+" "+words[i+1]] = true
	}
	return out
}

// stopWords — palavras comuns em PT-BR técnico que não diferenciam.
var stopWords = map[string]bool{
	"de": true, "do": true, "da": true, "dos": true, "das": true,
	"e": true, "o": true, "a": true, "os": true, "as": true,
	"um": true, "uma": true, "uns": true, "umas": true,
	"para": true, "com": true, "em": true, "no": true, "na": true, "nos": true, "nas": true,
	"por": true, "se": true, "que": true, "ou": true,
	"sistema": true, "metodo": true, "método": true, "processo": true,
	"dispositivo": true, "aparelho": true, "composicao": true, "composição": true,
}

func tokenizeFiling(s string) []string {
	clean := strings.ToLower(s)
	for _, ch := range ",.;:!?\n\r\t()[]{}\"'" {
		clean = strings.ReplaceAll(clean, string(ch), " ")
	}
	raw := strings.Fields(clean)
	out := raw[:0]
	for _, w := range raw {
		if len(w) <= 2 || stopWords[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

// topKeywords — pega as N palavras informativas mais longas.
func topKeywords(s string, n int) []string {
	tokens := tokenizeFiling(s)
	// dedup + score = comprimento (proxy de info)
	seen := map[string]int{}
	for _, t := range tokens {
		if len(t) > seen[t] {
			seen[t] = len(t)
		}
	}
	type kv struct{ k string; v int }
	pairs := make([]kv, 0, len(seen))
	for k, v := range seen {
		pairs = append(pairs, kv{k, v})
	}
	// sort by length desc (bubble — small set)
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].v > pairs[i].v {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	out := make([]string, 0, n)
	for i := 0; i < len(pairs) && i < n; i++ {
		out = append(out, pairs[i].k)
	}
	return out
}

func dedupAndTopN(hits []FilingPriorArtHit, n int) []FilingPriorArtHit {
	seen := map[int64]bool{}
	uniq := hits[:0]
	for _, h := range hits {
		if !seen[h.PatentID] {
			seen[h.PatentID] = true
			uniq = append(uniq, h)
		}
	}
	// sort by similarity desc
	for i := 0; i < len(uniq); i++ {
		for j := i + 1; j < len(uniq); j++ {
			if uniq[j].SimilarityPct > uniq[i].SimilarityPct {
				uniq[i], uniq[j] = uniq[j], uniq[i]
			}
		}
	}
	if len(uniq) > n {
		uniq = uniq[:n]
	}
	return uniq
}

// generateClaim — produz um template de reivindicação estruturada que o
// NIT-UFOP pode usar como ponto de partida. Inspirado no formato brasileiro
// padrão (Lei 9.279/96 + Diretrizes INPI).
func generateClaim(in FilingInput, sug *FilingSuggestion) string {
	ipc := "(IPC a definir)"
	if sug.IPCLetter != "" {
		ipc = fmt.Sprintf("(IPC sugerido: %s — %s)", sug.IPCLetter, sug.IPCName)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "REIVINDICAÇÃO INDEPENDENTE %s\n\n", ipc)
	fmt.Fprintf(&sb, "1. %s, caracterizado por:\n\n", strings.TrimSuffix(in.Title, "."))

	// Extrai 3-5 elementos-chave do abstract pra usar como features
	keywords := topKeywords(in.Abstract, 5)
	for i, kw := range keywords {
		fmt.Fprintf(&sb, "   %s. compreender %s [especificar];\n", romanNumeral(i+1), kw)
	}

	sb.WriteString("\n")
	fmt.Fprintf(&sb, "REIVINDICAÇÕES DEPENDENTES (sugeridas)\n\n")
	fmt.Fprintf(&sb, "2. %s, conforme reivindicação 1, ainda caracterizado por:\n",
		strings.TrimSuffix(in.Title, "."))
	fmt.Fprintf(&sb, "   - [aplicação alternativa]\n")
	fmt.Fprintf(&sb, "   - [variação técnica]\n\n")
	fmt.Fprintf(&sb, "NOTA: este template é apenas um esqueleto. Revisão por agente de PI\n")
	fmt.Fprintf(&sb, "é obrigatória antes do depósito no INPI.\n")

	return sb.String()
}

func romanNumeral(n int) string {
	r := []string{"i", "ii", "iii", "iv", "v", "vi", "vii", "viii"}
	if n >= 1 && n <= len(r) {
		return r[n-1]
	}
	return fmt.Sprintf("%d", n)
}
