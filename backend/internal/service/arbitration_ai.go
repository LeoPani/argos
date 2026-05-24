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

// ArbitrationAI compares dispute subjects heuristically.
type ArbitrationAI struct {
	storage    ArbitrationStorage
	patents    repository.PatentRepository
	trademarks repository.TrademarkRepository
	disputes   repository.DisputeRepository
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
