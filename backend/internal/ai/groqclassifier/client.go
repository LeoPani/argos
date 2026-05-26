// Package groqclassifier — cliente HTTP para a API Groq Cloud usada na
// classificação de patenteabilidade real-time de oportunidades UFOP.
//
// Não confunde com internal/ai/llm/, que é stub pra geração de texto longo
// (relatórios de anterioridade). Aqui o foco é **decisão estruturada** com
// schema fixo (JSON), modelo Llama 3.3 70B versatile (free tier 14400/dia).
//
// Documentação Groq: https://console.groq.com/docs/api-reference
// Modelo escolhido validado em LLM-as-annotator (Honovich 2022) e
// AnnoLLM (He 2024).
package groqclassifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://api.groq.com/openai/v1/chat/completions"
	defaultModel    = "llama-3.3-70b-versatile"
	maxBodyBytes    = 64 * 1024
)

// Classification é a resposta estruturada do classificador.
type Classification struct {
	IsPatentable bool    `json:"is_patentable"`
	IPCCategory  int     `json:"ipc_category"` // 0=A..7=H
	Confidence   float64 `json:"confidence"`   // 0.0-1.0
	Rationale    string  `json:"rationale"`    // PT-BR curto
}

// Client encapsula chamadas à Groq Cloud Chat Completions.
type Client struct {
	apiKey   string
	model    string
	endpoint string
	http     *http.Client
}

type Config struct {
	APIKey         string
	Model          string        // default: llama-3.3-70b-versatile
	Endpoint       string        // default: groq cloud
	RequestTimeout time.Duration // default: 15s
}

// New cria o client. Retorna nil se APIKey vazia (caller deve cair em fallback).
func New(cfg Config) *Client {
	if cfg.APIKey == "" {
		return nil
	}
	model := cfg.Model
	if model == "" {
		model = defaultModel
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	timeout := cfg.RequestTimeout
	if timeout == 0 {
		timeout = 15 * time.Second
	}
	return &Client{
		apiKey:   cfg.APIKey,
		model:    model,
		endpoint: endpoint,
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) Model() string { return c.model }

// Classify pede ao LLM a classificação completa. Erro indica fallback necessário.
func (c *Client) Classify(ctx context.Context, dept, title, abstract string) (*Classification, error) {
	body := chatRequest{
		Model:          c.model,
		Temperature:    0.0,
		MaxTokens:      400,
		ResponseFormat: &responseFormat{Type: "json_object"},
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: buildUserPrompt(dept, title, abstract)},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("groq returned %d: %s", resp.StatusCode, truncate(string(respBody), 240))
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode envelope: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("groq: empty choices")
	}
	content := parsed.Choices[0].Message.Content
	// Llama as vezes envolve em ```json … ```
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var out Classification
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return nil, fmt.Errorf("decode classification: %w (raw: %s)", err, truncate(content, 200))
	}
	// Sanity clamps
	if out.IPCCategory < 0 || out.IPCCategory > 7 {
		out.IPCCategory = 0
	}
	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 1 {
		out.Confidence = 1
	}
	return &out, nil
}

func buildUserPrompt(dept, title, abstract string) string {
	if len(abstract) > 3000 {
		abstract = abstract[:3000]
	}
	if len(title) > 300 {
		title = title[:300]
	}
	return fmt.Sprintf(
		"Analise o trabalho acadêmico abaixo e retorne APENAS um JSON válido.\n\n"+
			"DEPARTAMENTO: %s\n\nTÍTULO: %s\n\nRESUMO:\n%s\n\n"+
			"Retorne JSON com: is_patentable (bool), ipc_category (0-7), "+
			"confidence (0-1), rationale (PT-BR, max 200 chars).",
		strOrDash(dept), strOrDash(title), strOrDash(abstract),
	)
}

func strOrDash(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ─── wire types (subset OpenAI-compatible) ─────────────────────────────────

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    float64         `json:"temperature"`
	MaxTokens      int             `json:"max_tokens"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// ─── prompt ────────────────────────────────────────────────────────────────

const systemPrompt = `Você é um especialista em Propriedade Intelectual (PI) brasileiro
trabalhando com o NIT-UFOP. Sua tarefa é avaliar trabalhos acadêmicos quanto ao
potencial de gerarem patentes industriais defensáveis junto ao INPI sob a Lei
n. 9.279/1996.

Retorne APENAS um JSON válido (sem markdown, sem texto extra) com:
  - "is_patentable": true/false — tem aspecto técnico patenteável (Art. 8 LPI)?
  - "ipc_category": 0-7, onde:
       0=A Necessidades humanas (saúde, farmácia, alimentos)
       1=B Operações/transportes (processos industriais)
       2=C Química e metalurgia
       3=D Têxteis e papel
       4=E Construção civil
       5=F Engenharia mecânica
       6=G Física / TI / sensores
       7=H Eletricidade e eletrônica
  - "confidence": 0.0 a 1.0
  - "rationale": frase curta (max 200 chars) em PT-BR justificando

REGRAS DURAS (Art. 10 LPI):
  ✗ Trabalhos de Direito, Letras, Filosofia, História, Sociologia → is_patentable=false
  ✗ Métodos comerciais, contábeis, jurídicos → is_patentable=false
  ✗ Concepções abstratas (teorias puras) → is_patentable=false
  ✗ Programa de computador per se → is_patentable=false

SINAIS POSITIVOS:
  ✓ Processo industrial concreto + aplicação técnica
  ✓ Dispositivo, composição ou material novo
  ✓ Algoritmo com resultado técnico (não só software)
  ✓ Beneficiamento mineral, metalurgia, química aplicada

Se in dúvida, prefira confidence baixa e is_patentable=false.`
