package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/repository/postgres"
)

// IPTimestampService cria e lista registros de anterioridade com prova de existência.
//
// O hash é gerado assim:
//   content = title|description|authors_joined|created_at_iso8601
//   content_hash = SHA-256(content) em hex
//
// Cada registro armazena o hash do anterior (prev_hash), criando uma
// cadeia auditável que detecta qualquer adulteração retroativa.
type IPTimestampService struct {
	repo *postgres.IPTimestampRepo
}

func NewIPTimestampService(db *sql.DB) *IPTimestampService {
	return &IPTimestampService{repo: postgres.NewIPTimestampRepo(db)}
}

// CreateRequest é o payload de entrada para criar um registro.
type IPTimestampCreateRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Authors     []string `json:"authors"`
	Category    string   `json:"category"` // opcional; default "invenção"
}

// CreateResponse inclui o registro criado + metadados para exibição no certificado.
type IPTimestampCreateResponse struct {
	domain.IPTimestamp
	CanonicalContent string `json:"canonical_content"` // o texto exato que foi hasheado
}

// Create gera o hash, encadeia ao anterior e persiste.
func (s *IPTimestampService) Create(ctx context.Context, req IPTimestampCreateRequest) (*IPTimestampCreateResponse, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if req.Category == "" {
		req.Category = "invenção"
	}
	if req.Authors == nil {
		req.Authors = []string{}
	}

	// Busca o último registro para encadear
	prev, err := s.repo.GetLatest(ctx)
	if err != nil {
		return nil, fmt.Errorf("get latest for chaining: %w", err)
	}

	now := time.Now().UTC()

	// Conteúdo canônico que é hasheado (determinístico e auditável)
	canonical := fmt.Sprintf("%s|%s|%s|%s",
		req.Title,
		req.Description,
		strings.Join(req.Authors, ","),
		now.Format(time.RFC3339),
	)
	rawHash := sha256.Sum256([]byte(canonical))
	contentHash := fmt.Sprintf("%x", rawHash)

	prevHash := ""
	chainIndex := int64(0)
	if prev != nil {
		prevHash = prev.ContentHash
		chainIndex = prev.ChainIndex + 1
	}

	t := &domain.IPTimestamp{
		Title:       req.Title,
		Description: req.Description,
		Authors:     req.Authors,
		Category:    req.Category,
		ContentHash: contentHash,
		PrevHash:    prevHash,
		ChainIndex:  chainIndex,
	}

	if err := s.repo.Insert(ctx, t); err != nil {
		return nil, fmt.Errorf("insert ip_timestamp: %w", err)
	}

	return &IPTimestampCreateResponse{
		IPTimestamp:      *t,
		CanonicalContent: canonical,
	}, nil
}

// List retorna registros paginados.
func (s *IPTimestampService) List(ctx context.Context, limit, offset int) ([]domain.IPTimestamp, int64, error) {
	items, err := s.repo.List(ctx, domain.IPTimestampFilter{Limit: limit, Offset: offset})
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx)
	return items, total, err
}

// GetByID retorna um registro por ID.
func (s *IPTimestampService) GetByID(ctx context.Context, id int64) (*domain.IPTimestamp, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return t, err
}

// Verify recalcula o hash de um registro e confirma integridade.
func (s *IPTimestampService) Verify(ctx context.Context, id int64) (bool, string, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return false, "", err
	}
	canonical := fmt.Sprintf("%s|%s|%s|%s",
		t.Title, t.Description,
		strings.Join(t.Authors, ","),
		t.CreatedAt.UTC().Format(time.RFC3339),
	)
	raw := sha256.Sum256([]byte(canonical))
	recomputed := fmt.Sprintf("%x", raw)
	ok := recomputed == t.ContentHash
	return ok, recomputed, nil
}
