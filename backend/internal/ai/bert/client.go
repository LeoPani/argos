// Package bert is the BERT-backed adapter satisfying ai.Classifier.
// It speaks HTTP to the Python FastAPI service in ai-service/api_argos.py.
//
// Wire it up in main.go like:
//
//	bertClient := bert.New(bert.DefaultConfig("http://localhost:8000"))
//	aiSvc := ai.NewComposite(bertClient, nil)
package bert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config tunes the HTTP behavior of the BERT classifier client.
type Config struct {
	BaseURL        string        // e.g. "http://localhost:8000"
	RequestTimeout time.Duration // per-call ceiling
	MaxRetries     int           // retry on 5xx / transient network errors
	RetryBackoff   time.Duration // base backoff; doubled per attempt
}

// DefaultConfig returns production-sensible defaults. Override per env.
func DefaultConfig(baseURL string) Config {
	return Config{
		BaseURL:        baseURL,
		RequestTimeout: 15 * time.Second,
		MaxRetries:     2,
		RetryBackoff:   500 * time.Millisecond,
	}
}

// Client implements ai.Classifier against the Python BERTimbau service.
type Client struct {
	cfg  Config
	http *http.Client
}

// New builds a Client with a tuned HTTP transport that reuses TCP
// connections across the worker pool. We intentionally do NOT set a
// client-level Timeout; per-call deadlines come from the caller's
// context via context.WithTimeout.
func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		http: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 16, // matches default worker concurrency
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// classifyRequest mirrors the FastAPI contract in api_argos.py:
//
//	class PatentRequest(BaseModel):
//	    text: str
type classifyRequest struct {
	Text string `json:"text"`
}

// classifyResponse mirrors the FastAPI response shape:
//
//	{"text_received": "...", "predicted_category_id": 3}
type classifyResponse struct {
	TextReceived        string `json:"text_received"`
	PredictedCategoryID int    `json:"predicted_category_id"`
}

// ClassifyPatent sends the abstract to /classify and returns the
// predicted IPC category id (0..7). The provided context bounds the
// total call duration including retries.
func (c *Client) ClassifyPatent(ctx context.Context, abstract string) (int, error) {
	if abstract == "" {
		return -1, fmt.Errorf("bert: classify: empty abstract")
	}

	body, err := json.Marshal(classifyRequest{Text: abstract})
	if err != nil {
		return -1, fmt.Errorf("bert: marshal request: %w", err)
	}

	endpoint := c.cfg.BaseURL + "/classify"
	var lastErr error

	for attempt := 0; attempt <= c.cfg.MaxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return -1, fmt.Errorf("bert: context done: %w", err)
		}

		category, err := c.doOnce(ctx, endpoint, body)
		if err == nil {
			return category, nil
		}
		lastErr = err

		// Don't sleep after the final attempt.
		if attempt == c.cfg.MaxRetries {
			break
		}
		backoff := c.cfg.RetryBackoff * time.Duration(1<<attempt) // 1x, 2x, 4x
		select {
		case <-ctx.Done():
			return -1, fmt.Errorf("bert: context done during backoff: %w", ctx.Err())
		case <-time.After(backoff):
		}
	}

	return -1, fmt.Errorf("bert: all %d retries exhausted: %w", c.cfg.MaxRetries+1, lastErr)
}

// doOnce performs a single HTTP attempt against /classify.
func (c *Client) doOnce(ctx context.Context, endpoint string, body []byte) (int, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return -1, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return -1, fmt.Errorf("http do: %w", err)
	}
	defer func() {
		// Drain body so the TCP connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= 500 {
		return -1, fmt.Errorf("upstream %d (retryable)", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return -1, fmt.Errorf("upstream %d: %s", resp.StatusCode, string(snippet))
	}

	var decoded classifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return -1, fmt.Errorf("decode response: %w", err)
	}

	// BERTimbau was trained for 8 categories (labels 0..7).
	if decoded.PredictedCategoryID < 0 || decoded.PredictedCategoryID > 7 {
		return -1, fmt.Errorf("upstream returned out-of-range category: %d", decoded.PredictedCategoryID)
	}
	return decoded.PredictedCategoryID, nil
}
