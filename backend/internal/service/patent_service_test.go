package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LeoPani/argos/backend/internal/domain"
	"github.com/LeoPani/argos/backend/internal/service"
)

// ── Mocks ─────────────────────────────────────────────────────────────────────

// mockAI satisfies ai.AIService.
type mockAI struct {
	category int
	err      error
}

func (m *mockAI) ClassifyPatent(_ context.Context, _ string) (int, error) {
	return m.category, m.err
}
func (m *mockAI) GeneratePriorArtReport(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (m *mockAI) SummarizeDispute(_ context.Context, _ string) (string, error) {
	return "", nil
}

// mockPatentRepo satisfies repository.PatentRepository.
type mockPatentRepo struct {
	inserted *domain.Patent
	byID     *domain.Patent
	insertErr error
}

func (r *mockPatentRepo) Insert(_ context.Context, p *domain.Patent) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	p.ID = 42
	r.inserted = p
	return nil
}
func (r *mockPatentRepo) GetByID(_ context.Context, _ int64) (*domain.Patent, error) {
	if r.byID == nil {
		return nil, domain.ErrNotFound
	}
	return r.byID, nil
}
func (r *mockPatentRepo) GetByApplicationNumber(_ context.Context, _ string) (*domain.Patent, error) {
	return nil, domain.ErrNotFound
}
func (r *mockPatentRepo) List(_ context.Context, _ domain.PatentFilter) ([]domain.Patent, error) {
	return nil, nil
}
func (r *mockPatentRepo) Count(_ context.Context, _ domain.PatentFilter) (int64, error) {
	return 0, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newValidPatent() *domain.Patent {
	return &domain.Patent{
		Title:    "Sistema de purificação por ozônio para tratamento de efluentes",
		Abstract: "A presente invenção descreve um sistema de purificação de água utilizando geração de ozônio por descarga elétrica, com aplicação em estações de tratamento de efluentes industriais.",
		ApplicationNumber: "BR102024000001",
		Inventors:         []string{"Maria Silva", "João Ferreira"},
		Applicant:         "UFOP",
	}
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestIngest_ClassificationSuccess(t *testing.T) {
	repo := &mockPatentRepo{}
	aiSvc := &mockAI{category: 2} // Química e Metalurgia
	svc := service.NewPatentService(repo, aiSvc)

	got, err := svc.Ingest(context.Background(), newValidPatent())
	if err != nil {
		t.Fatalf("Ingest() error = %v, want nil", err)
	}
	if got.ID != 42 {
		t.Errorf("got ID %d, want 42 (from mock)", got.ID)
	}
	if got.IPCCategory != 2 {
		t.Errorf("got IPCCategory %d, want 2", got.IPCCategory)
	}
	if got.Status != domain.PatentStatusClassified {
		t.Errorf("got status %q, want %q", got.Status, domain.PatentStatusClassified)
	}
}

func TestIngest_ClassificationFails_SavesAsFailed(t *testing.T) {
	repo := &mockPatentRepo{}
	aiSvc := &mockAI{err: errors.New("bert unavailable")}
	svc := service.NewPatentService(repo, aiSvc)

	got, err := svc.Ingest(context.Background(), newValidPatent())
	if err != nil {
		t.Fatalf("Ingest() error = %v, want nil (AI failure is tolerated)", err)
	}
	if got.Status != domain.PatentStatusFailed {
		t.Errorf("got status %q, want %q", got.Status, domain.PatentStatusFailed)
	}
	if got.IPCCategory != domain.IPCCategoryUnknown {
		t.Errorf("got IPCCategory %d, want %d (unknown)", got.IPCCategory, domain.IPCCategoryUnknown)
	}
	// Patent must still be persisted.
	if repo.inserted == nil {
		t.Error("patent should be inserted even when classification fails")
	}
}

func TestIngest_ValidationError_EmptyTitle(t *testing.T) {
	repo := &mockPatentRepo{}
	aiSvc := &mockAI{category: 0}
	svc := service.NewPatentService(repo, aiSvc)

	p := newValidPatent()
	p.Title = ""

	_, err := svc.Ingest(context.Background(), p)
	if err == nil {
		t.Fatal("Ingest() with empty title should return error")
	}
}

func TestIngest_ValidationError_EmptyAbstract(t *testing.T) {
	repo := &mockPatentRepo{}
	aiSvc := &mockAI{category: 0}
	svc := service.NewPatentService(repo, aiSvc)

	p := newValidPatent()
	p.Abstract = ""

	_, err := svc.Ingest(context.Background(), p)
	if err == nil {
		t.Fatal("Ingest() with empty abstract should return error")
	}
}

func TestIngest_RepoError_Propagated(t *testing.T) {
	repoErr := errors.New("unique constraint violation")
	repo := &mockPatentRepo{insertErr: repoErr}
	aiSvc := &mockAI{category: 1}
	svc := service.NewPatentService(repo, aiSvc)

	_, err := svc.Ingest(context.Background(), newValidPatent())
	if err == nil {
		t.Fatal("Ingest() should propagate repository error")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error chain should contain repoErr; got: %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	repo := &mockPatentRepo{} // byID is nil → ErrNotFound
	aiSvc := &mockAI{}
	svc := service.NewPatentService(repo, aiSvc)

	_, err := svc.GetByID(context.Background(), 999)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetByID() error = %v, want domain.ErrNotFound", err)
	}
}

func TestGetByID_Found(t *testing.T) {
	patent := &domain.Patent{ID: 7, Title: "Test Patent", Status: domain.PatentStatusClassified}
	repo := &mockPatentRepo{byID: patent}
	aiSvc := &mockAI{}
	svc := service.NewPatentService(repo, aiSvc)

	got, err := svc.GetByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetByID() error = %v, want nil", err)
	}
	if got.ID != 7 {
		t.Errorf("got ID %d, want 7", got.ID)
	}
}
