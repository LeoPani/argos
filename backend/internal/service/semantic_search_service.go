// Package service — SemanticSearchService implementa busca semântica
// local via TF-IDF + cosine similarity (Salton & Buckley 1988).
//
// Não depende de Claude, BERT ou Lens — é a baseline gratuita do
// projeto. Documentada em METHODOLOGY.md como baseline #1.
//
// Index é construído sob demanda na primeira query e cached por 10
// minutos. Para corpus < 5k docs (UFOP), é instantâneo (~30ms).
package service

import (
	"context"
	"database/sql"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	semanticCacheTTL = 10 * time.Minute
	semanticMinToken = 3
	semanticTopK     = 20
)

// stopwords PT-BR — lista compacta cobrindo as ~60 mais frequentes.
var ptStopwords = map[string]struct{}{
	"a": {}, "as": {}, "o": {}, "os": {}, "um": {}, "uma": {}, "uns": {}, "umas": {},
	"de": {}, "do": {}, "da": {}, "dos": {}, "das": {}, "no": {}, "na": {}, "nos": {}, "nas": {},
	"em": {}, "por": {}, "para": {}, "com": {}, "sem": {}, "ao": {}, "aos": {},
	"e": {}, "ou": {}, "mas": {}, "que": {}, "se": {}, "como": {}, "porque": {},
	"é": {}, "são": {}, "foi": {}, "ser": {}, "ter": {}, "há": {}, "tem": {},
	"este": {}, "esta": {}, "isto": {}, "esse": {}, "essa": {}, "isso": {}, "aquele": {}, "aquela": {},
	"eu": {}, "tu": {}, "ele": {}, "ela": {}, "nós": {}, "vós": {}, "eles": {}, "elas": {},
	"meu": {}, "minha": {}, "seu": {}, "sua": {}, "nosso": {}, "nossa": {},
	"à": {}, "às": {}, "pela": {}, "pelo": {}, "pelos": {}, "pelas": {},
	"the": {}, "of": {}, "and": {}, "for": {}, "to": {}, "in": {}, "on": {}, "at": {}, "is": {}, "with": {},
}

type semanticDoc struct {
	Kind     string             // ufop_opp | patent
	ID       int64
	Title    string
	Snippet  string
	URL      string
	TFIDF    map[string]float64 // sparse vector
	Norm     float64            // ||v||
}

type SemanticHit struct {
	Kind    string  `json:"kind"`
	ID      int64   `json:"id"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
	URL     string  `json:"url"`
}

type SemanticSearchResponse struct {
	Query      string        `json:"query"`
	Method     string        `json:"method"`
	DocCount   int           `json:"doc_count"`
	BuiltAt    time.Time     `json:"built_at"`
	Hits       []SemanticHit `json:"hits"`
}

type SemanticSearchService struct {
	db *sql.DB

	mu      sync.RWMutex
	docs    []semanticDoc
	idf     map[string]float64
	builtAt time.Time
}

func NewSemanticSearchService(db *sql.DB) *SemanticSearchService {
	return &SemanticSearchService{db: db}
}

// Search é o ponto de entrada. Reconstrói o índice se expirado.
func (s *SemanticSearchService) Search(ctx context.Context, q string, topK int) (*SemanticSearchResponse, error) {
	if topK <= 0 || topK > 100 {
		topK = semanticTopK
	}
	q = strings.TrimSpace(q)
	if q == "" {
		return &SemanticSearchResponse{
			Query: q, Method: "tfidf_cosine_pt-br", Hits: []SemanticHit{},
		}, nil
	}

	if s.needsRebuild() {
		if err := s.rebuild(ctx); err != nil {
			return nil, err
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	queryTokens := tokenizeSemantic(q)
	if len(queryTokens) == 0 {
		return &SemanticSearchResponse{
			Query: q, Method: "tfidf_cosine_pt-br",
			DocCount: len(s.docs), BuiltAt: s.builtAt, Hits: []SemanticHit{},
		}, nil
	}
	qVec := s.tfidfVector(queryTokens)
	qNorm := vectorNorm(qVec)
	if qNorm == 0 {
		return &SemanticSearchResponse{
			Query: q, Method: "tfidf_cosine_pt-br",
			DocCount: len(s.docs), BuiltAt: s.builtAt, Hits: []SemanticHit{},
		}, nil
	}

	type scored struct {
		idx   int
		score float64
	}
	scores := make([]scored, 0, len(s.docs))
	for i, d := range s.docs {
		if d.Norm == 0 {
			continue
		}
		sim := cosineSimilarity(qVec, d.TFIDF, qNorm, d.Norm)
		if sim > 0 {
			scores = append(scores, scored{i, sim})
		}
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })

	if len(scores) > topK {
		scores = scores[:topK]
	}
	hits := make([]SemanticHit, 0, len(scores))
	for _, sc := range scores {
		d := s.docs[sc.idx]
		hits = append(hits, SemanticHit{
			Kind: d.Kind, ID: d.ID, Title: d.Title,
			Snippet: d.Snippet, Score: sc.score, URL: d.URL,
		})
	}
	return &SemanticSearchResponse{
		Query: q, Method: "tfidf_cosine_pt-br",
		DocCount: len(s.docs), BuiltAt: s.builtAt, Hits: hits,
	}, nil
}

func (s *SemanticSearchService) needsRebuild() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.builtAt.IsZero() || time.Since(s.builtAt) > semanticCacheTTL
}

// rebuild carrega UFOP opps + patents e computa TF-IDF.
func (s *SemanticSearchService) rebuild(ctx context.Context) error {
	raw, err := s.fetchCorpus(ctx)
	if err != nil {
		return err
	}

	// 1ª passada: tokenizar + computar DF
	tokenizedDocs := make([][]string, len(raw))
	df := make(map[string]int)
	for i, r := range raw {
		tokens := tokenizeSemantic(r.Title + " " + r.Snippet)
		tokenizedDocs[i] = tokens
		seen := make(map[string]struct{})
		for _, t := range tokens {
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			df[t]++
		}
	}

	// IDF = log(N / df)
	N := float64(len(raw))
	idf := make(map[string]float64, len(df))
	for term, freq := range df {
		idf[term] = math.Log(N / float64(freq))
	}

	// 2ª passada: TF-IDF por doc
	docs := make([]semanticDoc, len(raw))
	for i, r := range raw {
		vec := make(map[string]float64)
		tf := make(map[string]int)
		for _, t := range tokenizedDocs[i] {
			tf[t]++
		}
		for term, freq := range tf {
			vec[term] = float64(freq) * idf[term]
		}
		docs[i] = semanticDoc{
			Kind: r.Kind, ID: r.ID, Title: r.Title, Snippet: r.Snippet, URL: r.URL,
			TFIDF: vec, Norm: vectorNorm(vec),
		}
	}

	s.mu.Lock()
	s.docs = docs
	s.idf = idf
	s.builtAt = time.Now()
	s.mu.Unlock()
	return nil
}

type rawDoc struct {
	Kind    string
	ID      int64
	Title   string
	Snippet string
	URL     string
}

func (s *SemanticSearchService) fetchCorpus(ctx context.Context) ([]rawDoc, error) {
	out := make([]rawDoc, 0, 512)

	// UFOP opportunities
	rows1, err := s.db.QueryContext(ctx, `
		SELECT id,
		       COALESCE(title,'') AS title,
		       COALESCE(LEFT(description, 600),'') AS snippet
		FROM ufop_opportunities
		LIMIT 5000`)
	if err == nil {
		defer rows1.Close()
		for rows1.Next() {
			var d rawDoc
			d.Kind = "ufop_opp"
			if err := rows1.Scan(&d.ID, &d.Title, &d.Snippet); err == nil {
				d.URL = "/ufop"
				out = append(out, d)
			}
		}
	}

	// Patents
	rows2, err := s.db.QueryContext(ctx, `
		SELECT id,
		       COALESCE(title,'') AS title,
		       COALESCE(LEFT(abstract, 600),'') AS snippet
		FROM patents
		LIMIT 5000`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var d rawDoc
			d.Kind = "patent"
			if err := rows2.Scan(&d.ID, &d.Title, &d.Snippet); err == nil {
				d.URL = "/patentes"
				out = append(out, d)
			}
		}
	}

	// INPI publications — despachos das RPIs (harvest semanal).
	// Enriquece o índice com ~6k registros de publicações oficiais INPI
	// sem custo adicional de infra (mesma pipeline TF-IDF).
	rows3, err := s.db.QueryContext(ctx, `
		SELECT id,
		       COALESCE(process_number,'') || ' ' || COALESCE(title,'') AS title,
		       COALESCE(applicant,'') AS snippet
		FROM inpi_publications
		WHERE title != '' OR applicant != ''
		LIMIT 6000`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var d rawDoc
			d.Kind = "inpi"
			if err := rows3.Scan(&d.ID, &d.Title, &d.Snippet); err == nil {
				d.URL = ""
				out = append(out, d)
			}
		}
	}

	return out, nil
}

func (s *SemanticSearchService) tfidfVector(tokens []string) map[string]float64 {
	tf := make(map[string]int, len(tokens))
	for _, t := range tokens {
		tf[t]++
	}
	vec := make(map[string]float64, len(tf))
	for term, freq := range tf {
		w, ok := s.idf[term]
		if !ok {
			continue
		}
		vec[term] = float64(freq) * w
	}
	return vec
}

// tokenizeSemantic — lowercase, strip diacritics, drop stopwords/short.
func tokenizeSemantic(text string) []string {
	if text == "" {
		return nil
	}
	out := make([]string, 0, 32)
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(stripDiacritic(r))
		default:
			if b.Len() > 0 {
				emitToken(&b, &out)
			}
		}
	}
	if b.Len() > 0 {
		emitToken(&b, &out)
	}
	return out
}

func emitToken(b *strings.Builder, out *[]string) {
	t := b.String()
	b.Reset()
	if len(t) < semanticMinToken {
		return
	}
	if _, skip := ptStopwords[t]; skip {
		return
	}
	*out = append(*out, t)
}

// stripDiacritic: remove acentos comuns PT-BR mantendo a letra-base.
func stripDiacritic(r rune) rune {
	switch r {
	case 'á', 'à', 'â', 'ã', 'ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'õ', 'ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ç':
		return 'c'
	case 'ñ':
		return 'n'
	}
	return r
}

func vectorNorm(v map[string]float64) float64 {
	var s float64
	for _, x := range v {
		s += x * x
	}
	return math.Sqrt(s)
}

func cosineSimilarity(a, b map[string]float64, na, nb float64) float64 {
	var dot float64
	// itera pelo menor para economizar
	if len(a) > len(b) {
		a, b = b, a
	}
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (na * nb)
}
