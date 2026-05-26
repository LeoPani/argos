// Package service — ArbitrationAI is the heuristic comparator that
// produces structured verdicts for arbitration disputes.
//
// Method v1 (heuristic_v1) — fully local, no external API calls:
//
//	Trademark conflicts:
//	  • Priority (first-to-file): older filing date wins
//	  • Distinctiveness: longer names + Levenshtein distance to existing
//	    marks (we have ~30 trademarks in the bank to compare against)
//	  • Nice class overlap: identical classes → strong signal
//	  • Phonetic similarity (Soundex PT-BR): catches "VEGABRAS" vs "VEGA NATURAL"
//	  • Active status: granted > filed > expired
//
//	Patent conflicts:
//	  • Filing date priority
//	  • IPC category match
//	  • Abstract length / specificity (proxy for breadth of claim)
//	  • Status (classified > pending)
//
// Each subject ends with a score 0..100. The winner is the highest.
// Confidence is (winner - runnerUp) clamped to 0..100.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/LeoPani/argos/backend/internal/ai/groqclassifier"
	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository"
)

// ArbitrationStorage abstracts the few repo methods we need so the
// service is decoupled from a specific postgres implementation.
type ArbitrationStorage interface {
	ListSubjects(ctx context.Context, disputeID int64) ([]domain.DisputeSubject, error)
	SaveVerdict(ctx context.Context, v *domain.ArbitrationVerdict) error
	AddSubject(ctx context.Context, s *domain.DisputeSubject) error
	DeleteSubject(ctx context.Context, id int64) error
	LatestVerdict(ctx context.Context, disputeID int64) (*domain.ArbitrationVerdict, error)
}

// ArbitrationAI compares dispute subjects heuristically, with optional LLM enrichment.
type ArbitrationAI struct {
	storage    ArbitrationStorage
	patents    repository.PatentRepository
	trademarks repository.TrademarkRepository
	disputes   repository.DisputeRepository
	groq       *groqclassifier.Client // optional — nil = heuristic only
}

// NewArbitrationAI wires up the analyzer.
func NewArbitrationAI(
	storage ArbitrationStorage,
	patents repository.PatentRepository,
	trademarks repository.TrademarkRepository,
	disputes repository.DisputeRepository,
) *ArbitrationAI {
	return &ArbitrationAI{storage: storage, patents: patents, trademarks: trademarks, disputes: disputes}
}

// WithGroq enables LLM-powered comparison (optional, gracefully degrades if nil).
func (a *ArbitrationAI) WithGroq(gc *groqclassifier.Client) *ArbitrationAI {
	a.groq = gc
	return a
}

// ─── Quick PI comparison (no dispute required) ────────────────────────────────

// PIComparisonRequest contains two patent IDs to compare.
type PIComparisonRequest struct {
	PatentAID int64 `json:"patent_a_id"`
	PatentBID int64 `json:"patent_b_id"`
}

// PIComparisonResult is the output of a quick direct comparison.
type PIComparisonResult struct {
	PatentA             *domain.Patent                       `json:"patent_a"`
	PatentB             *domain.Patent                       `json:"patent_b"`
	Method              string                               `json:"method"` // "llm_groq" | "heuristic"
	SimilarityScore     float64                              `json:"similarity_score"`
	ConflictAreas       []string                             `json:"conflict_areas"`
	DifferentiatingClaims []string                           `json:"differentiating_claims"`
	Recommendation      string                               `json:"recommendation"`
	Narrative           string                               `json:"narrative"`
	PatentAStrengths    []string                             `json:"patent_a_strengths"`
	PatentBStrengths    []string                             `json:"patent_b_strengths"`
	PriorityWinner      string                               `json:"priority_winner"` // "A" | "B" | "equal"
}

// ComparePatents performs a direct comparison between two patents.
// If Groq is available it calls the LLM; otherwise falls back to local heuristic.
func (a *ArbitrationAI) ComparePatents(ctx context.Context, idA, idB int64) (*PIComparisonResult, error) {
	pA, err := a.patents.GetByID(ctx, idA)
	if err != nil {
		return nil, fmt.Errorf("patent A #%d: %w", idA, err)
	}
	pB, err := a.patents.GetByID(ctx, idB)
	if err != nil {
		return nil, fmt.Errorf("patent B #%d: %w", idB, err)
	}

	result := &PIComparisonResult{
		PatentA: pA,
		PatentB: pB,
	}

	// Priority (first-to-file).
	result.PriorityWinner = comparePriority(pA, pB)

	// Try LLM first.
	if a.groq != nil {
		filingA, filingB := "", ""
		if pA.FilingDate != nil {
			filingA = pA.FilingDate.Format("02/01/2006")
		}
		if pB.FilingDate != nil {
			filingB = pB.FilingDate.Format("02/01/2006")
		}
		llmRes, err := a.groq.ComparePatents(ctx,
			pA.Title, pA.Abstract, fmt.Sprintf("%d", pA.IPCCategory), filingA,
			pB.Title, pB.Abstract, fmt.Sprintf("%d", pB.IPCCategory), filingB,
		)
		if err == nil {
			result.Method                = "llm_groq"
			result.SimilarityScore       = llmRes.SimilarityScore
			result.ConflictAreas         = llmRes.ConflictAreas
			result.DifferentiatingClaims = llmRes.DifferentiatingClaims
			result.Recommendation        = llmRes.Recommendation
			result.Narrative             = llmRes.Narrative
			result.PatentAStrengths      = llmRes.PatentAStrengths
			result.PatentBStrengths      = llmRes.PatentBStrengths
			return result, nil
		}
		// LLM failed — fall through to heuristic.
	}

	// Heuristic fallback.
	result.Method = "heuristic"
	result.SimilarityScore, result.ConflictAreas, result.DifferentiatingClaims,
		result.Recommendation, result.Narrative,
		result.PatentAStrengths, result.PatentBStrengths =
		heuristicPatentCompare(pA, pB)

	return result, nil
}

// comparePriority returns "A", "B", or "equal" based on filing dates.
func comparePriority(a, b *domain.Patent) string {
	if a.FilingDate == nil && b.FilingDate == nil {
		return "equal"
	}
	if a.FilingDate == nil {
		return "B"
	}
	if b.FilingDate == nil {
		return "A"
	}
	switch {
	case a.FilingDate.Before(*b.FilingDate):
		return "A"
	case b.FilingDate.Before(*a.FilingDate):
		return "B"
	default:
		return "equal"
	}
}

// heuristicPatentCompare produces a local comparison when Groq is unavailable.
func heuristicPatentCompare(a, b *domain.Patent) (
	score float64, conflicts, diffs []string, rec, narrative string,
	aStr, bStr []string,
) {
	// IPC overlap.
	ipcSame := a.IPCCategory == b.IPCCategory && a.IPCCategory.IsValid()

	// Abstract similarity via trigrams.
	score = trigramSimilarity(a.Abstract, b.Abstract)
	if ipcSame {
		score = math.Min(1.0, score+0.15) // boost for same IPC
	}

	if ipcSame {
		conflicts = append(conflicts, fmt.Sprintf("Mesma categoria IPC: %d", a.IPCCategory))
	}

	// Title words overlap.
	titleSim := trigramSimilarity(a.Title, b.Title)
	if titleSim > 0.5 {
		conflicts = append(conflicts, fmt.Sprintf("Títulos com alta similaridade (%.0f%%)", titleSim*100))
	}

	// Differentiators.
	if !ipcSame {
		diffs = append(diffs, fmt.Sprintf("Categorias IPC distintas (%d vs %d)", a.IPCCategory, b.IPCCategory))
	}
	la, lb := len(a.Abstract), len(b.Abstract)
	if la > 0 && lb > 0 {
		ratio := float64(la) / float64(lb)
		if ratio < 0.5 {
			diffs = append(diffs, "PI-B tem reivindicação muito mais ampla (abstract maior)")
		} else if ratio > 2.0 {
			diffs = append(diffs, "PI-A tem reivindicação muito mais ampla (abstract maior)")
		}
	}

	switch {
	case score >= 0.65:
		rec = "possivel_infracao"
		narrative = fmt.Sprintf(
			"Alta similaridade técnica (%.0f%%) detectada entre as duas PIs na mesma categoria IPC. "+
				"Existe risco real de sobreposição de reivindicações — recomenda-se análise jurídica detalhada.",
			score*100,
		)
	case score >= 0.35:
		rec = "inconclusivo"
		narrative = fmt.Sprintf(
			"Similaridade moderada (%.0f%%). Existem elementos técnicos comuns mas também diferenciadores. "+
				"Análise de reivindicações específicas é necessária para determinar infração.",
			score*100,
		)
	default:
		rec = "sem_conflito"
		narrative = fmt.Sprintf(
			"Baixa similaridade técnica (%.0f%%). As PIs abordam problemas técnicos distintos. "+
				"Risco de conflito de reivindicações é baixo.",
			score*100,
		)
	}

	// Strengths.
	if a.FilingDate != nil {
		aStr = append(aStr, fmt.Sprintf("Depositada em %s", a.FilingDate.Format("02/01/2006")))
	}
	if len(a.Abstract) > 300 {
		aStr = append(aStr, "Abstract detalhado (reivindicação específica)")
	}
	if b.FilingDate != nil {
		bStr = append(bStr, fmt.Sprintf("Depositada em %s", b.FilingDate.Format("02/01/2006")))
	}
	if len(b.Abstract) > 300 {
		bStr = append(bStr, "Abstract detalhado (reivindicação específica)")
	}
	return
}

// trigramSimilarity computes Dice coefficient on character trigrams (lowercased).
func trigramSimilarity(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a == "" || b == "" {
		return 0
	}
	build := func(s string) map[string]int {
		m := map[string]int{}
		s = "  " + s + "  "
		for i := 0; i+3 <= len(s); i++ {
			m[s[i:i+3]]++
		}
		return m
	}
	ta, tb := build(a), build(b)
	var common int
	for k, cnt := range ta {
		if c2, ok := tb[k]; ok {
			if cnt < c2 {
				common += cnt
			} else {
				common += c2
			}
		}
	}
	total := 0
	for _, v := range ta { total += v }
	for _, v := range tb { total += v }
	if total == 0 { return 0 }
	return 2.0 * float64(common) / float64(total)
}

// ─── Public API ──────────────────────────────────────────────────────────────

func (a *ArbitrationAI) AddSubject(ctx context.Context, s *domain.DisputeSubject) error {
	return a.storage.AddSubject(ctx, s)
}

func (a *ArbitrationAI) ListSubjects(ctx context.Context, disputeID int64) ([]domain.DisputeSubject, error) {
	return a.storage.ListSubjects(ctx, disputeID)
}

func (a *ArbitrationAI) DeleteSubject(ctx context.Context, id int64) error {
	return a.storage.DeleteSubject(ctx, id)
}

func (a *ArbitrationAI) LatestVerdict(ctx context.Context, disputeID int64) (*domain.ArbitrationVerdict, error) {
	return a.storage.LatestVerdict(ctx, disputeID)
}

// Analyze runs the heuristic and persists a verdict.
func (a *ArbitrationAI) Analyze(ctx context.Context, disputeID int64) (*domain.ArbitrationVerdict, error) {
	dispute, err := a.disputes.GetByID(ctx, disputeID)
	if err != nil {
		return nil, fmt.Errorf("analyze: load dispute: %w", err)
	}

	subjects, err := a.storage.ListSubjects(ctx, disputeID)
	if err != nil {
		return nil, fmt.Errorf("analyze: list subjects: %w", err)
	}
	if len(subjects) < 2 {
		return nil, fmt.Errorf("analyze: need at least 2 subjects, got %d", len(subjects))
	}

	// Route by dispute kind. Default: trademark comparison.
	var scores []domain.SubjectScore
	var factors []string

	switch dispute.Kind {
	case domain.DisputeKindTrademarkInfringement, domain.DisputeKindOther:
		scores, factors, err = a.scoreTrademarks(ctx, subjects)
	case domain.DisputeKindPatentInfringement:
		scores, factors, err = a.scorePatents(ctx, subjects)
	case domain.DisputeKindAuthorship, domain.DisputeKindLicensing:
		scores, factors = a.scoreGeneric(subjects)
	default:
		scores, factors = a.scoreGeneric(subjects)
	}
	if err != nil {
		return nil, fmt.Errorf("analyze: scoring: %w", err)
	}

	// Pick winner.
	var winnerID *int64
	var winnerScore, runnerUp float64
	for _, s := range scores {
		if s.Score > winnerScore {
			runnerUp = winnerScore
			winnerScore = s.Score
			id := s.SubjectID
			winnerID = &id
		} else if s.Score > runnerUp {
			runnerUp = s.Score
		}
	}

	confidence := int(math.Round(winnerScore - runnerUp))
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	summary := buildSummary(scores, winnerID)

	reasoning := domain.VerdictReasoning{Subjects: scores, Factors: factors}
	reasonJSON, _ := json.Marshal(reasoning)

	v := &domain.ArbitrationVerdict{
		DisputeID:       disputeID,
		WinnerSubjectID: winnerID,
		Confidence:      confidence,
		Method:          domain.VerdictMethodHeuristic,
		Summary:         summary,
		Reasoning:       reasonJSON,
	}
	if err := a.storage.SaveVerdict(ctx, v); err != nil {
		return nil, fmt.Errorf("analyze: save verdict: %w", err)
	}
	return v, nil
}

// ─── Trademark scoring ───────────────────────────────────────────────────────

func (a *ArbitrationAI) scoreTrademarks(ctx context.Context, subjects []domain.DisputeSubject) ([]domain.SubjectScore, []string, error) {
	// Fetch the trademark for each subject (when ref_id present).
	type tmCtx struct {
		subj domain.DisputeSubject
		mark *domain.Trademark
	}
	contexts := make([]tmCtx, 0, len(subjects))
	for _, s := range subjects {
		c := tmCtx{subj: s}
		if s.Kind == domain.SubjectKindTrademark && s.RefID != nil {
			m, err := a.trademarks.GetByID(ctx, *s.RefID)
			if err == nil {
				c.mark = m
			}
		}
		contexts = append(contexts, c)
	}

	// Earliest filing date among subjects (priority anchor).
	var earliest *time.Time
	for _, c := range contexts {
		if c.mark != nil && c.mark.FilingDate != nil {
			if earliest == nil || c.mark.FilingDate.Before(*earliest) {
				earliest = c.mark.FilingDate
			}
		}
	}

	factors := []string{
		"Comparados nomes com normalização (sem acentos, maiúsculas)",
		"Verificada similaridade fonética via Soundex adaptado para PT-BR",
		"Considerado overlap de classes Nice entre as marcas",
		"Aplicada regra 'first-to-file' (prioridade temporal do INPI)",
		"Avaliado status atual no INPI (granted / filed / opposition)",
	}

	scores := make([]domain.SubjectScore, len(contexts))
	for i, c := range contexts {
		var (
			score float64 = 50 // base
			pros, cons []string
		)

		if c.mark == nil {
			scores[i] = domain.SubjectScore{
				SubjectID: c.subj.ID, Label: c.subj.Label, Score: 30,
				ProArguments: nil,
				ConArguments: []string{"Marca não vinculada ao banco — análise limitada a texto livre"},
			}
			continue
		}

		// 1. First-to-file priority (up to +25)
		if c.mark.FilingDate != nil && earliest != nil {
			daysAfter := c.mark.FilingDate.Sub(*earliest).Hours() / 24
			if daysAfter < 1 {
				score += 25
				pros = append(pros, fmt.Sprintf("Primeira a depositar no INPI (%s)", c.mark.FilingDate.Format("02/01/2006")))
			} else {
				penalty := math.Min(15, daysAfter/365*3) // ~3 pts per year delayed
				score -= penalty
				cons = append(cons, fmt.Sprintf("Depositada %.0f dias após a primeira concorrente", daysAfter))
			}
		}

		// 2. Status (granted = +15, filed/published = neutral, denied/expired = -20)
		switch c.mark.Status {
		case domain.TrademarkStatusGranted:
			score += 15
			pros = append(pros, "Marca já registrada (status: granted)")
		case domain.TrademarkStatusDenied, domain.TrademarkStatusExpired, domain.TrademarkStatusArchived:
			score -= 20
			cons = append(cons, fmt.Sprintf("Status desfavorável no INPI: %s", c.mark.Status))
		case domain.TrademarkStatusPublished:
			score += 5
			pros = append(pros, "Em fase de publicação — proteção em curso")
		}

		// 3. Distinctiveness (length × character variety)
		distinctiveness := nameDistinctiveness(c.mark.Name)
		score += distinctiveness
		switch {
		case distinctiveness >= 8:
			pros = append(pros, fmt.Sprintf("Nome distintivo (%d caracteres únicos)", len(c.mark.Name)))
		case distinctiveness < 3:
			cons = append(cons, "Nome curto/genérico — proteção mais fraca")
		}

		// 4. Phonetic conflict with siblings (cons)
		for j, other := range contexts {
			if i == j || other.mark == nil {
				continue
			}
			sim := phoneticSimilarity(c.mark.NormalizedName, other.mark.NormalizedName)
			if sim >= 0.75 {
				cons = append(cons, fmt.Sprintf(
					"Alta semelhança fonética com %q (%d%%)",
					other.mark.Name, int(sim*100),
				))
			}
		}

		// 5. Nice class overlap (cons if many overlap with competitors)
		for j, other := range contexts {
			if i == j || other.mark == nil {
				continue
			}
			overlap := classOverlap(c.mark.NiceClasses, other.mark.NiceClasses)
			if overlap > 0 {
				cons = append(cons, fmt.Sprintf(
					"Sobrepõe %d classe(s) Nice com %q",
					overlap, other.mark.Name,
				))
			}
		}

		// Clamp & store
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		scores[i] = domain.SubjectScore{
			SubjectID: c.subj.ID, Label: c.subj.Label, Score: score,
			ProArguments: pros, ConArguments: cons,
		}
	}

	return scores, factors, nil
}

// ─── Patent scoring ──────────────────────────────────────────────────────────

func (a *ArbitrationAI) scorePatents(ctx context.Context, subjects []domain.DisputeSubject) ([]domain.SubjectScore, []string, error) {
	type pCtx struct {
		subj   domain.DisputeSubject
		patent *domain.Patent
	}
	contexts := make([]pCtx, 0, len(subjects))
	for _, s := range subjects {
		c := pCtx{subj: s}
		if s.Kind == domain.SubjectKindPatent && s.RefID != nil {
			p, err := a.patents.GetByID(ctx, *s.RefID)
			if err == nil {
				c.patent = p
			}
		}
		contexts = append(contexts, c)
	}

	// Earliest filing date.
	var earliest *time.Time
	for _, c := range contexts {
		if c.patent != nil && c.patent.FilingDate != nil {
			if earliest == nil || c.patent.FilingDate.Before(*earliest) {
				earliest = c.patent.FilingDate
			}
		}
	}

	factors := []string{
		"Avaliada prioridade temporal de depósito INPI",
		"Comparada categoria IPC predita por BERT (8 classes A..H)",
		"Considerado tamanho/especificidade do abstract como proxy de escopo da reivindicação",
		"Verificado status (classified / pending / failed)",
	}

	scores := make([]domain.SubjectScore, len(contexts))
	for i, c := range contexts {
		var (
			score float64 = 50
			pros, cons []string
		)

		if c.patent == nil {
			scores[i] = domain.SubjectScore{
				SubjectID: c.subj.ID, Label: c.subj.Label, Score: 30,
				ConArguments: []string{"Patente não vinculada — análise limitada"},
			}
			continue
		}

		// First-to-file priority.
		if c.patent.FilingDate != nil && earliest != nil {
			delta := c.patent.FilingDate.Sub(*earliest).Hours() / 24
			if delta < 1 {
				score += 30
				pros = append(pros, fmt.Sprintf("Primeira a depositar (%s)", c.patent.FilingDate.Format("02/01/2006")))
			} else {
				penalty := math.Min(20, delta/365*5)
				score -= penalty
				cons = append(cons, fmt.Sprintf("Depositada %.0f dias após", delta))
			}
		}

		// Status.
		switch c.patent.Status {
		case "classified":
			score += 10
			pros = append(pros, "Já classificada pela IA do INPI")
		case "failed":
			score -= 10
			cons = append(cons, "Status 'failed' — classificação não concluída")
		}

		// Specificity (abstract length).
		l := len(c.patent.Abstract)
		switch {
		case l > 400:
			score += 8
			pros = append(pros, "Abstract detalhado — reivindicação específica")
		case l < 100:
			score -= 5
			cons = append(cons, "Abstract muito breve — reivindicação ampla/genérica (mais frágil)")
		}

		// IPC overlap.
		for j, other := range contexts {
			if i == j || other.patent == nil {
				continue
			}
			if c.patent.IPCCategory == other.patent.IPCCategory && c.patent.IPCCategory.IsValid() {
				cons = append(cons, fmt.Sprintf("Mesma categoria IPC de %q", other.patent.ApplicationNumber))
			}
		}

		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		scores[i] = domain.SubjectScore{
			SubjectID: c.subj.ID, Label: c.subj.Label, Score: score,
			ProArguments: pros, ConArguments: cons,
		}
	}

	return scores, factors, nil
}

// ─── Generic fallback (authorship / licensing / other) ────────────────────────

func (a *ArbitrationAI) scoreGeneric(subjects []domain.DisputeSubject) ([]domain.SubjectScore, []string) {
	factors := []string{
		"Análise heurística limitada (sem dados estruturados de mercado)",
		"Score baseado em metadados das partes",
		"Recomenda-se incluir documentos como evidência e re-analisar",
	}

	scores := make([]domain.SubjectScore, len(subjects))
	for i, s := range subjects {
		scores[i] = domain.SubjectScore{
			SubjectID: s.ID, Label: s.Label, Score: 50,
			ConArguments: []string{"Tipo de disputa requer análise manual ou modelo LLM (Phase 2)"},
		}
	}
	return scores, factors
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// nameDistinctiveness rewards uncommon characters and length.
func nameDistinctiveness(name string) float64 {
	clean := normalizeForCompare(name)
	if len(clean) == 0 {
		return 0
	}
	seen := map[rune]bool{}
	for _, r := range clean {
		if unicode.IsLetter(r) {
			seen[r] = true
		}
	}
	uniqueLetters := len(seen)
	// 0..15 range
	return math.Min(15, float64(uniqueLetters)*1.5)
}

// phoneticSimilarity returns 0..1 — pure character-overlap proxy.
// For an MVP this catches "ARGOS" vs "ARGUS" and "VEGABRAS" vs "VEGA NATURAL"
// without requiring a real PT-BR phonetic library.
func phoneticSimilarity(a, b string) float64 {
	a = normalizeForCompare(a)
	b = normalizeForCompare(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}

	// Trigram overlap (Dice coefficient)
	tri := func(s string) map[string]bool {
		out := map[string]bool{}
		s = "  " + s + "  "
		for i := 0; i+3 <= len(s); i++ {
			out[s[i:i+3]] = true
		}
		return out
	}
	ta, tb := tri(a), tri(b)
	common := 0
	for k := range ta {
		if tb[k] {
			common++
		}
	}
	if len(ta)+len(tb) == 0 {
		return 0
	}
	return 2.0 * float64(common) / float64(len(ta)+len(tb))
}

// normalizeForCompare strips accents and non-letters, lowercases.
func normalizeForCompare(s string) string {
	repl := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "ê", "e", "è", "e", "ë", "e",
		"í", "i", "î", "i", "ï", "i",
		"ó", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"Á", "A", "Ã", "A", "Â", "A",
		"É", "E", "Ê", "E", "Í", "I",
		"Ó", "O", "Ô", "O", "Õ", "O",
		"Ú", "U", "Ç", "C",
	)
	s = repl.Replace(s)
	var out strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(unicode.ToLower(r))
		}
	}
	return out.String()
}

// classOverlap counts how many Nice classes appear in both lists.
func classOverlap(a, b []int) int {
	set := map[int]bool{}
	for _, c := range a {
		set[c] = true
	}
	overlap := 0
	for _, c := range b {
		if set[c] {
			overlap++
		}
	}
	return overlap
}

// buildSummary turns scores into a one-line PT-BR sentence.
func buildSummary(scores []domain.SubjectScore, winnerID *int64) string {
	if winnerID == nil {
		return "Análise não conclusiva — todos os candidatos com score zero."
	}
	for _, s := range scores {
		if s.SubjectID == *winnerID {
			return fmt.Sprintf(
				"A análise heurística aponta %q como o candidato com maior fundamento jurídico (score %.1f/100).",
				s.Label, s.Score,
			)
		}
	}
	return "Análise concluída."
}
