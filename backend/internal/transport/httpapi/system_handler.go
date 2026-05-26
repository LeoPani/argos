package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/pkg/httputil"
)

// SystemHandler expõe metadados de transparência:
// qual modelo está rodando, qual modo de análise, fontes de dados.
type SystemHandler struct {
	aiBertURL string
	client    *http.Client
}

func NewSystemHandler(aiBertURL string) *SystemHandler {
	return &SystemHandler{
		aiBertURL: aiBertURL,
		client:    &http.Client{Timeout: 3 * time.Second},
	}
}

// AnalysisMode retorna qual classificador está realmente em uso.
//
// Possíveis valores:
//
//	"trained_sbert" — modelo Sentence-BERT supervisionado (argos_classifier.py)
//	"bert_fine_tuned" — BERTimbau fine-tuned legado (api_argos.py)
//	"heuristic"    — fallback: keywords + departamento + IPC heurística
type AnalysisModeResponse struct {
	Mode             string                 `json:"mode"`
	Description      string                 `json:"description"`
	BertOnline       bool                   `json:"bert_online"`
	BertHealth       map[string]any         `json:"bert_health,omitempty"`
	LensTokenSet     bool                   `json:"lens_token_set"`
	AnthropicKeySet  bool                   `json:"anthropic_key_set"`
	GroqKeySet       bool                   `json:"groq_key_set"`
	AnnotatorReady   bool                   `json:"annotator_ready"`
	AnnotatorProvider string                `json:"annotator_provider,omitempty"`
	DataSources      map[string]string      `json:"data_sources"`
	Limitations      []string               `json:"limitations"`
	NextSteps        []string               `json:"next_steps"`
}

// AnalysisMode — GET /api/v1/system/analysis-mode
func (h *SystemHandler) AnalysisMode(w http.ResponseWriter, r *http.Request) {
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY") != ""
	groqKey := os.Getenv("GROQ_API_KEY") != ""
	resp := AnalysisModeResponse{
		LensTokenSet:    os.Getenv("LENS_API_TOKEN") != "",
		AnthropicKeySet: anthropicKey,
		GroqKeySet:      groqKey,
		AnnotatorReady:  anthropicKey || groqKey,
		DataSources: map[string]string{
			"ufop_opportunities":  "OAI-PMH repositorio.ufop.br (REAL — Dublin Core)",
			"ufop_patents":        "Google Patents xhr/query (REAL — limited by rate-limit)",
			"patent_citations":    "Mock determinístico (Lens.org requer token)",
			"inpi_marcas":         "Não integrado (sem fonte free pública)",
		},
	}
	switch {
	case groqKey:
		resp.AnnotatorProvider = "groq (Llama 3.3 70B, free tier)"
	case anthropicKey:
		resp.AnnotatorProvider = "anthropic (Claude Sonnet 4.6)"
	}

	// Probe BERT
	health := h.probeBERT(r.Context())
	if health != nil {
		resp.BertOnline = true
		resp.BertHealth = health

		// Detecta se é o novo (argos_classifier) ou antigo (api_argos)
		if _, ok := health["has_sbert"]; ok {
			if hasSbert, _ := health["has_sbert"].(bool); hasSbert {
				resp.Mode = "trained_sbert"
				resp.Description = "Sentence-BERT supervisionado (Reimers & Gurevych 2020) " +
					"treinado em dataset UFOP anotado via Claude. Embeddings 384d."
			} else {
				resp.Mode = "bert_fine_tuned"
				resp.Description = "BERTimbau fine-tuned legado (api_argos.py). Modelo " +
					"original do projeto, sem reentreinamento recente."
			}
		} else {
			resp.Mode = "bert_fine_tuned"
			resp.Description = "BERTimbau fine-tuned (modelo original)."
		}
	} else {
		resp.Mode = "heuristic"
		resp.Description = "Sistema heurístico de classificação por keywords + " +
			"departamento + IPC inferida. Não é IA no sentido moderno. " +
			"Documentado em METHODOLOGY.md como baseline."
		resp.Limitations = append(resp.Limitations,
			"Sem BERT online: IPC inferida via regras (não captura semântica)",
			"PI Score = keyword count + threshold (Salton 1988 baseline)",
			"ai_analysis usa template (Mad Libs), não é gerada por LLM",
		)
		resp.NextSteps = append(resp.NextSteps,
			"Rodar pipeline em ai-service/training/ (necessita ANTHROPIC_API_KEY)",
			"Subir argos_classifier.py via uvicorn",
			"Re-rodar harvest UFOP — classificação ficará pelo modelo treinado",
		)
	}

	if !resp.LensTokenSet {
		resp.Limitations = append(resp.Limitations,
			"Lens.org sem token: forward citations são mock determinístico",
		)
	}
	if !resp.AnnotatorReady {
		resp.Limitations = append(resp.Limitations,
			"Nenhuma API key de LLM configurada (ANTHROPIC_API_KEY ou GROQ_API_KEY): "+
				"anotação ML está bloqueada. Groq é free (gsk_... em groq.com).",
		)
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// probeBERT tenta GET no FastAPI; retorna o JSON do /health ou nil.
func (h *SystemHandler) probeBERT(ctx context.Context) map[string]any {
	if h.aiBertURL == "" {
		return nil
	}
	url := strings.TrimRight(h.aiBertURL, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}
	return data
}
