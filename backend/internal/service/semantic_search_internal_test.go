// White-box tests for the SemanticSearchService internals.
// Uses package service (not service_test) to access unexported helpers.
package service

import (
	"math"
	"testing"
)

// ── tokenizeSemantic ──────────────────────────────────────────────────────────

func TestTokenizeSemantic_Basic(t *testing.T) {
	toks := tokenizeSemantic("Sistema de purificação por ozônio")
	// "de" and "por" are stopwords; "sistema", "purificação", "ozônio" must remain.
	want := map[string]bool{"sistema": true, "purificacao": true, "ozonio": true}
	for _, tok := range toks {
		delete(want, tok)
	}
	if len(want) != 0 {
		t.Errorf("missing expected tokens: %v (got %v)", want, toks)
	}
}

func TestTokenizeSemantic_StopwordsRemoved(t *testing.T) {
	toks := tokenizeSemantic("de a o e é")
	if len(toks) != 0 {
		t.Errorf("expected all stopwords to be removed, got %v", toks)
	}
}

func TestTokenizeSemantic_ShortTokensIgnored(t *testing.T) {
	// semanticMinToken = 3; tokens < 3 chars should be dropped
	toks := tokenizeSemantic("ab cd efg")
	for _, tok := range toks {
		if len(tok) < semanticMinToken {
			t.Errorf("token %q shorter than minimum %d", tok, semanticMinToken)
		}
	}
}

func TestTokenizeSemantic_Empty(t *testing.T) {
	if toks := tokenizeSemantic(""); len(toks) != 0 {
		t.Errorf("expected empty slice for empty input, got %v", toks)
	}
}

func TestTokenizeSemantic_AccentNormalization(t *testing.T) {
	toks := tokenizeSemantic("Química química")
	// After normalization both should collapse to the same token.
	if len(toks) != 2 {
		t.Fatalf("expected 2 tokens (duplicate allowed), got %v", toks)
	}
	if toks[0] != toks[1] {
		t.Errorf("accented and plain versions should normalize equal: %v", toks)
	}
}

// ── cosineSimilarity ──────────────────────────────────────────────────────────

func TestCosineSimilarity_Identical(t *testing.T) {
	v := map[string]float64{"patent": 1.0, "ozonio": 0.8}
	n := vectorNorm(v)
	score := cosineSimilarity(v, v, n, n)
	if math.Abs(score-1.0) > 1e-9 {
		t.Errorf("identical vectors should have cosine=1.0, got %f", score)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := map[string]float64{"patent": 1.0}
	b := map[string]float64{"trademark": 1.0}
	score := cosineSimilarity(a, b, vectorNorm(a), vectorNorm(b))
	if score != 0.0 {
		t.Errorf("orthogonal vectors should have cosine=0.0, got %f", score)
	}
}

func TestCosineSimilarity_ZeroNorm(t *testing.T) {
	v := map[string]float64{}
	score := cosineSimilarity(v, v, 0, 0)
	if score != 0.0 {
		t.Errorf("zero-norm vectors should return 0.0, got %f", score)
	}
}

func TestCosineSimilarity_Partial(t *testing.T) {
	a := map[string]float64{"patent": 1.0, "ozonio": 1.0}
	b := map[string]float64{"patent": 1.0, "other": 1.0}
	na, nb := vectorNorm(a), vectorNorm(b)
	score := cosineSimilarity(a, b, na, nb)
	if score <= 0 || score >= 1 {
		t.Errorf("partial overlap should give 0 < cosine < 1, got %f", score)
	}
}

// ── vectorNorm ────────────────────────────────────────────────────────────────

func TestVectorNorm_KnownValues(t *testing.T) {
	v := map[string]float64{"a": 3.0, "b": 4.0}
	got := vectorNorm(v)
	want := 5.0 // 3-4-5 triangle
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("vectorNorm({3,4}) = %f, want %f", got, want)
	}
}
