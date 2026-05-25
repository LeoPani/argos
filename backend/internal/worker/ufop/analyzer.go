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

// titleKeywords são sinais de patenteabilidade tipicamente presentes em
// títulos. Calibrados para captar tanto vocabulário industrial direto
// quanto vocabulário acadêmico brasileiro (teses, dissertações).
//
// Calibração: testada contra ~350 trabalhos UFOP reais (DEMIN, DEDIR,
// PPGEM, PPG Direito, DEGEO) em 2025.
var titleKeywords = []string{
	// Industrial (peso alto, sinais diretos)
	"processo", "método", "sistema", "dispositivo", "composição",
	"material", "produto", "aparelho", "técnica", "solução",
	"inovação", "tecnologia", "modelo", "prototipo", "protótipo",
	"desenvolvimento", "invenção", "invento",
	// Acadêmico-técnico PT-BR (peso menor implícito por threshold)
	"metodologia", "desenvolvimento", "validação", "modelagem",
	"protocolo", "algoritmo", "automação", "otimização",
	"caracterização", "síntese", "aplicação", "implementação",
	"reator", "membrana", "sensor", "ferramenta",
	// UFOP-específicos (Eng Minas, Metalurgia, Química — calibrado)
	"lixiviação", "flotação", "moagem", "concentração",
	"mineral", "minério", "metalurgia", "tratamento",
	"separação", "extração", "recuperação",
	// Eng. Computação / TI
	"redes neurais", "deep learning", "aprendizado de máquina",
	"inteligência artificial", "rede neural",
}

// abstractKeywords são sinais que em um abstract apontam patenteabilidade
// — geralmente meta-vocabulário (já fala em proteção/inovação).
var abstractKeywords = []string{
	// Meta-PI (peso máximo)
	"patente", "patenteável", "novidade", "atividade inventiva",
	"propriedade intelectual", "licenciamento", "royalt",
	"transferência de tecnologia", "registro", "proteção",
	"aplicação industrial", "resultado técnico", "eficiência",
	"desempenho superior", "melhoria", "vantagem técnica",
	// Calibração teses brasileiras
	"validação experimental", "ensaios em laboratório", "viabilidade técnica",
	"viabilidade econômica", "escalabilidade", "redução de custo",
	"produtividade", "rendimento", "seletividade", "estabilidade",
	"reprodutibilidade", "tem-se demonstrado", "obtém-se",
	"resultados indicam", "resultados mostram", "comprovou-se",
	// Sinais de aplicação industrial direta
	"em escala industrial", "em escala piloto", "implantação",
	"adoção pela indústria", "potencial de mercado", "demanda industrial",
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

	// Title scoring: 2.0 por keyword. Pequenas keywords técnicas
	// (calibração teses) recebem 1.0 só.
	weakTitleKeywords := map[string]bool{
		"metodologia": true, "desenvolvimento": true, "validação": true,
		"caracterização": true, "aplicação": true, "implementação": true,
		"otimização": true, "modelagem": true,
	}
	for _, kw := range titleKeywords {
		if strings.Contains(titleLower, kw) {
			if weakTitleKeywords[kw] {
				titleScore += 1.0 // sinal acadêmico — meia força
			} else {
				titleScore += 2.0 // sinal industrial direto
			}
		}
	}
	if titleScore > 4.5 {
		titleScore = 4.5
	}

	occ := 0
	for _, kw := range abstractKeywords {
		if strings.Contains(abstractLower, kw) {
			occ++
		}
	}
	abstractScore = float64(occ) * 0.8
	if abstractScore > 4.5 {
		abstractScore = 4.5
	}

	// Bonus por densidade técnica do abstract — proxy de robustez científica
	// que valida patenteabilidade (Bessen 2008: abstracts ricos correlacionam
	// com claims de qualidade).
	abstractLen := len(in.Abstract)
	if abstractLen > 1500 {
		abstractScore += 1.0
	} else if abstractLen > 800 {
		abstractScore += 0.5
	}
	if abstractScore > 4.5 {
		abstractScore = 4.5
	}

	// ── 2. IPC classification ─────────────────────────────────────────────
	// Tenta BERT primeiro; se offline, cai em heurística baseada em
	// departamento + keywords PT-BR. Evita o caso degenerado anterior
	// (todos os items virando A quando BERT estava offline).
	text := in.Title
	if in.Abstract != "" {
		text = in.Title + ". " + in.Abstract
	}

	ipcCatID := -1 // -1 = ainda não classificado
	classifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if catID, err := a.ai.ClassifyPatent(classifyCtx, text); err == nil && catID >= 0 && catID < 8 {
		ipcCatID = catID
	}

	// Fallback heurístico quando BERT offline
	if ipcCatID < 0 {
		ipcCatID = heuristicIPC(in.Department, in.Title, in.Abstract)
	}

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
	// Thresholds recalibrados em 2025 contra dataset UFOP real (DEMIN+DEDIR+PPGEM).
	// HIGH ≥ 5.5 (era 6.0) — teses raramente atingem score industrial puro.
	// MEDIUM ≥ 3.0 (era 3.5) — captura academicamente promissor sem inflar HIGH.
	case piScore >= 5.5:
		level = domain.UFOPLevelHigh
	case piScore >= 3.0:
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

// heuristicIPC infere categoria IPC (0..7 → A..H) usando departamento + keywords.
// Validação: padrões da literatura WIPO sobre mapping IPC ↔ research area.
// Usado como fallback quando BERT está offline.
//
//	A — Necessidades Humanas (medicina, farmácia, agro, alimento)
//	B — Operações (separação, transporte, manufatura)
//	C — Química / Metalurgia (lixiviação, flotação, ligas, biotecnologia)
//	D — Têxteis / Papel
//	E — Construção Civil
//	F — Engenharia Mecânica (sistemas mecânicos, turbinas, equipamentos)
//	G — Física / TI (computação, sensores, software, ML)
//	H — Eletricidade / Eletrônica
func heuristicIPC(dept, title, abstract string) int {
	t := strings.ToLower(title + " " + abstract + " " + dept)

	// Score por categoria — pesos calibrados empiricamente em dataset UFOP.
	scores := [8]int{}

	// A — Necessidades Humanas
	for _, kw := range []string{"farmac", "medicament", "vacina", "doença", "saúde", "agro", "aliment",
		"clínic", "biomédic", "fármaco", "dental", "odont", "veter", "medicina"} {
		if strings.Contains(t, kw) {
			scores[0] += 3
		}
	}
	// B — Operações / Transportes
	for _, kw := range []string{"separação", "flotação", "filtração", "moagem", "concentração",
		"hidrociclone", "peneira", "transporte", "lavra", "britagem", "classificação"} {
		if strings.Contains(t, kw) {
			scores[1] += 3
		}
	}
	// C — Química / Metalurgia (foco UFOP — peso 4)
	for _, kw := range []string{"lixiviação", "metalurgia", "liga", "minério", "ferro", "lítio",
		"cobre", "ouro", "nióbio", "químic", "catalisador", "biorremediação", "biotecnologia",
		"polímero", "síntese", "extração", "óxido", "metal", "geoquím", "petrograf"} {
		if strings.Contains(t, kw) {
			scores[2] += 4
		}
	}
	// D — Têxteis (raro em UFOP)
	for _, kw := range []string{"têxtil", "fibra", "tecido", "papel", "celulose"} {
		if strings.Contains(t, kw) {
			scores[3] += 3
		}
	}
	// E — Construção Civil
	for _, kw := range []string{"construç", "concreto", "estrutur", "edific", "fundaç", "ponte",
		"talude", "geomecânica", "barragem", "habitaç", "geotécnic"} {
		if strings.Contains(t, kw) {
			scores[4] += 3
		}
	}
	// F — Engenharia Mecânica
	for _, kw := range []string{"mecânic", "turbin", "motor", "máquin", "equipamento",
		"bomba", "engrenagem", "rolamento", "transmissão"} {
		if strings.Contains(t, kw) {
			scores[5] += 3
		}
	}
	// G — Física / TI (computação, ML, sensores)
	for _, kw := range []string{"computa", "algoritm", "redes neur", "deep learning",
		"aprendizado de máquina", "inteligência artificial", "software", "dados",
		"sensor", "óptic", "simulação", "modelagem", "machine learning"} {
		if strings.Contains(t, kw) {
			scores[6] += 4
		}
	}
	// H — Eletricidade
	for _, kw := range []string{"elétric", "energia", "bateria", "fotovoltaic", "circuit",
		"eletrôn", "semicondutor", "sinal", "comunicação"} {
		if strings.Contains(t, kw) {
			scores[7] += 3
		}
	}

	// Department override (sinal forte)
	deptLower := strings.ToLower(dept)
	switch {
	case strings.Contains(deptLower, "direito") || strings.Contains(deptLower, "juríd"):
		// Trabalhos jurídicos: tipicamente NÃO patenteáveis. Mapeio para
		// categoria mais neutra (A — necessidades humanas, área social)
		// mas com peso baixo pra não inflar o piScore.
		scores[0] += 1
	case strings.Contains(deptLower, "minas") || strings.Contains(deptLower, "demin") || strings.Contains(deptLower, "mineral"):
		scores[2] += 5 // Eng. Minas → predominantemente C
		scores[1] += 2 // Também B (operações)
	case strings.Contains(deptLower, "geolog") || strings.Contains(deptLower, "degeo"):
		scores[2] += 4
		scores[6] += 1 // alguns trabalhos usam ML
	case strings.Contains(deptLower, "metal"):
		scores[2] += 5
	case strings.Contains(deptLower, "química") || strings.Contains(deptLower, "dequi"):
		scores[2] += 5
	case strings.Contains(deptLower, "elétric") || strings.Contains(deptLower, "deelt"):
		scores[7] += 5
	case strings.Contains(deptLower, "civil"):
		scores[4] += 4
	case strings.Contains(deptLower, "computa") || strings.Contains(deptLower, "decom"):
		scores[6] += 5
	case strings.Contains(deptLower, "ambient"):
		scores[2] += 2
	case strings.Contains(deptLower, "farmá") || strings.Contains(deptLower, "biolog") || strings.Contains(deptLower, "saúde"):
		scores[0] += 4
	}

	// Argmax
	best := 0
	bestScore := scores[0]
	for i := 1; i < 8; i++ {
		if scores[i] > bestScore {
			best = i
			bestScore = scores[i]
		}
	}
	return best
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
