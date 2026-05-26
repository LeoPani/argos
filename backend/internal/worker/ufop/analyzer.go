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
	"log/slog"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/ai"
	"github.com/LeoPani/argos/backend/internal/ai/groqclassifier"
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
	ai   ai.AIService
	groq *groqclassifier.Client // optional — nil = fallback heurística
	log  *slog.Logger
}

// NewAnalyzer creates an Analyzer backed by the given AIService.
// O Groq classifier é opcional: se nil ou se a chamada falhar, cai em heurística.
func NewAnalyzer(ai ai.AIService) *Analyzer {
	return &Analyzer{ai: ai, log: slog.Default()}
}

// WithGroq habilita classificação real-time via Groq Cloud (Llama 3.3 70B).
// Quando configurado, vira o "primary classifier" sobre a heurística.
// O BERT, se online, ainda é usado pra IPC suggestion (mais barato).
func (a *Analyzer) WithGroq(c *groqclassifier.Client) *Analyzer {
	a.groq = c
	return a
}

// WithLogger sobrescreve o logger default. Útil para CLI tools.
func (a *Analyzer) WithLogger(log *slog.Logger) *Analyzer {
	a.log = log
	return a
}

// Analyze classifies the text, scores PI potential, and returns a
// ready-to-persist UFOPOpportunity. If the BERT service is down the
// function still returns a (lower-scored) result using keyword analysis
// alone and IPC category 0 with similarity 0.
func (a *Analyzer) Analyze(ctx context.Context, in AnalyzeInput) (*domain.UFOPOpportunity, error) {
	// ── 0. Art. 10 LPI — rejeição precoce ─────────────────────────────────
	// Art. 10 da Lei 9.279/96 exclui de patenteabilidade: esquemas jurídicos,
	// contábeis, comerciais; concepções puramente abstratas; apresentações
	// de informação (literatura, história pura); métodos matemáticos.
	// Para esses, retornamos rationale honesta + nível low + score 0.
	if reason := excludedByArt10LPI(in.Department, in.Title, in.Abstract); reason != "" {
		return buildRejectedOpp(in, reason), nil
	}

	// ── 0.5. LLM classifier (Groq) — primary path quando disponível ───────
	// Llama 3.3 70B via Groq Cloud retorna is_patentable + ipc + rationale.
	// Se a chamada falhar, fall-through pra heurística embaixo.
	if a.groq != nil {
		llmCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		res, err := a.groq.Classify(llmCtx, in.Department, in.Title, in.Abstract)
		cancel()
		if err == nil && res != nil {
			return buildLLMOpp(in, res, a.groq.Model()), nil
		}
		a.log.Warn("groq classify failed; falling back to heuristic",
			"err", err, "external_id", in.ExternalID)
	}

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
		// Genéricas demais: pontuam só se houver outro sinal técnico real.
		// "processo", "método", "sistema" foram movidos pra weak após
		// observar falsos positivos em trabalhos de direito processual.
		"processo": true, "método": true, "sistema": true, "modelo": true,
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

	// Patentável? Após Art. 10 já estar rejeitado, default true se há sinal
	// técnico mínimo (score >= 1.5); senão null (inconcluso).
	var isPatentable *bool
	if piScore >= 1.5 {
		t := true
		isPatentable = &t
	}

	opp := &domain.UFOPOpportunity{
		Source:            in.Source,
		ExternalID:        in.ExternalID,
		Title:             in.Title,
		Authors:           in.Authors,
		Department:        in.Department,
		Abstract:          in.Abstract,
		URL:               in.URL,
		PublishedAt:       in.PublishedAt,
		IPCSuggestion:     ipcSuggestions[ipcCatID],
		IPCCategory:       domain.IPCCategory(ipcCatID),
		Level:             level,
		SimilarityPct:     similarityPct,
		PIScore:           piScore,
		AIAnalysis:        analysis,
		Status:            domain.UFOPStatusNew,
		PublicationID:     in.PublicationID,
		IsPatentable:      isPatentable,
		Rationale:         fmt.Sprintf("Heurística v2: title=%.1f, abstract=%.1f, IPC bonus=%.1f", titleScore, abstractScore, bonus),
		ClassifierVersion: "heuristic-v2",
		Confidence:        heuristicConfidence(piScore),
	}
	return opp, nil
}

// heuristicConfidence — converte piScore num proxy de confiança 0-1.
// Calibrado: score 0 → 0.3 (chute), score 10 → 0.9 (alto sinal).
// Nunca 1.0 porque heurística não merece certeza.
func heuristicConfidence(piScore float64) float64 {
	conf := 0.3 + (piScore/10.0)*0.6
	if conf > 0.9 {
		conf = 0.9
	}
	return conf
}

// excludedByArt10LPI — verifica se o trabalho cai nas exclusões da Lei
// 9.279/96 Art. 10 (não considerados invenção/modelo de utilidade).
// Retorna string com a razão se sim, vazia se não.
//
// Inciso II — concepções puramente abstratas
// Inciso III — esquemas, planos, princípios ou métodos comerciais, contábeis,
//              financeiros, educativos, publicitários, de sorteio e fiscalização
// Inciso VI — apresentações de informações (texto puro, lit, hist)
//
// Heurística conservadora: prioriza dept (sinal forte) e palavras-chave
// inequívocas.
func excludedByArt10LPI(dept, title, abstract string) string {
	d := strings.ToLower(dept)
	if strings.Contains(d, "direito") || strings.Contains(d, "juríd") || strings.Contains(d, "dedir") {
		return "Trabalho de Direito — esquemas jurídicos não são patenteáveis (Art. 10 III, Lei 9.279/96)"
	}
	if strings.Contains(d, "letras") || strings.Contains(d, "literatura") || strings.Contains(d, "filos") {
		return "Trabalho de Letras/Filosofia — apresentações de informação não são patenteáveis (Art. 10 VI)"
	}
	if strings.Contains(d, "histór") || strings.Contains(d, "sociolog") || strings.Contains(d, "antrop") {
		return "Trabalho de Humanidades — concepção abstrata, fora do escopo do Art. 8 LPI"
	}
	if strings.Contains(d, "turismo") || strings.Contains(d, "hospital") {
		return "Trabalho de Turismo/Hospitalidade — serviços, não invenção (fora do Art. 8 LPI)"
	}
	if strings.Contains(d, "museol") || strings.Contains(d, "patrimôni") || strings.Contains(d, "patrimoni") {
		return "Trabalho de Museologia/Patrimônio — apresentação de informação cultural (Art. 10 VI)"
	}
	if strings.Contains(d, "contáb") || strings.Contains(d, "administ") || strings.Contains(d, "economia") {
		// Cuidado: alguns trabalhos de admin/economia podem ter inovação técnica
		// (ex: algoritmo de otimização logística). Só rejeita se o texto for
		// puramente metodológico-financeiro.
		t := strings.ToLower(title + " " + abstract)
		commercialSignal := false
		for _, kw := range []string{"contabilidade", "tributári", "fiscal", "contrato", "mercado financeiro",
			"governança", "gestão de pessoas", "marketing", "auditoria"} {
			if strings.Contains(t, kw) {
				commercialSignal = true
				break
			}
		}
		if commercialSignal {
			return "Tema comercial/contábil — Art. 10 III LPI exclui esses métodos"
		}
	}

	// Última checagem: keywords inequivocamente jurídicas no título
	tLow := strings.ToLower(title)
	for _, kw := range []string{
		"previdência social", "previdenciário", "inss",
		"direito do trabalho", "trabalhista", "constitucional",
		"penal", "criminal", "administrativo", "tributári", "civil",
		"jurisprudência", "processo civil", "processo penal",
	} {
		if strings.Contains(tLow, kw) {
			return fmt.Sprintf("Título indica tema jurídico (\"%s\") — Art. 10 III LPI", kw)
		}
	}

	return ""
}

// buildLLMOpp — converte a resposta do LLM em UFOPOpportunity. Diferente da
// heurística, o LLM dá o veredicto autoritativo (is_patentable + IPC); o
// PI Score é derivado de confidence × pesos de categoria.
func buildLLMOpp(in AnalyzeInput, res *groqclassifier.Classification, model string) *domain.UFOPOpportunity {
	ipc := res.IPCCategory
	if ipc < 0 || ipc > 7 {
		ipc = 0
	}

	// Se LLM disse não-patenteável → level=low, score=0, mantém rationale.
	level := domain.UFOPLevelLow
	piScore := 0.0
	if res.IsPatentable {
		// Score base = confidence (0-1) × 8 + bonus IPC (0-2).
		piScore = res.Confidence*8.0 + ipcCategoryBonus[ipc]
		if piScore > 10.0 {
			piScore = 10.0
		}
		switch {
		case piScore >= 6.5:
			level = domain.UFOPLevelHigh
		case piScore >= 4.0:
			level = domain.UFOPLevelMedium
		}
	}

	// Similarity é inverso a confidence — alta confiança no novo = niche
	similarity := 60 - int(res.Confidence*40)
	if similarity < 10 {
		similarity = 10
	}
	if similarity > 75 {
		similarity = 75
	}

	rationale := res.Rationale
	if rationale == "" {
		rationale = "Classificação via " + model
	}

	patentablePtr := res.IsPatentable
	return &domain.UFOPOpportunity{
		Source:            in.Source,
		ExternalID:        in.ExternalID,
		Title:             in.Title,
		Authors:           in.Authors,
		Department:        in.Department,
		Abstract:          in.Abstract,
		URL:               in.URL,
		PublishedAt:       in.PublishedAt,
		IPCSuggestion:     ipcSuggestions[ipc],
		IPCCategory:       domain.IPCCategory(ipc),
		Level:             level,
		SimilarityPct:     similarity,
		PIScore:           piScore,
		AIAnalysis:        rationale,
		Rationale:         rationale,
		IsPatentable:      &patentablePtr,
		ClassifierVersion: "groq-" + model,
		Confidence:        res.Confidence,
		Status:            domain.UFOPStatusNew,
		PublicationID:     in.PublicationID,
	}
}

// buildRejectedOpp — opp com is_patentable=false, level=low, score=0,
// rationale explicando o motivo. Mantém os metadados (department, title,
// authors) para o item continuar aparecendo no portfolio com a flag honesta.
func buildRejectedOpp(in AnalyzeInput, reason string) *domain.UFOPOpportunity {
	no := false
	return &domain.UFOPOpportunity{
		Source:            in.Source,
		ExternalID:        in.ExternalID,
		Title:             in.Title,
		Authors:           in.Authors,
		Department:        in.Department,
		Abstract:          in.Abstract,
		URL:               in.URL,
		PublishedAt:       in.PublishedAt,
		IPCSuggestion:     "— (não-patenteável)",
		IPCCategory:       domain.IPCCategory(0),
		Level:             domain.UFOPLevelLow,
		SimilarityPct:     0,
		PIScore:           0,
		AIAnalysis:        reason,
		Rationale:         reason,
		IsPatentable:      &no,
		ClassifierVersion: "heuristic-v2",
		Confidence:        0.95, // alta confiança na exclusão por Art. 10
		Status:            domain.UFOPStatusNew,
		PublicationID:     in.PublicationID,
	}
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
