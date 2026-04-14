package service

import (
	"context"
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/prompting"
)

type stubSkillRepo struct {
	byID map[string]*domain.SkillSpec
}

func newStubSkillRepo() *stubSkillRepo {
	return &stubSkillRepo{byID: map[string]*domain.SkillSpec{}}
}

func cloneSkill(sk *domain.SkillSpec) *domain.SkillSpec {
	if sk == nil {
		return nil
	}
	cp := *sk
	cp.Tags = append([]string(nil), sk.Tags...)
	cp.InputMimeTypes = append([]string(nil), sk.InputMimeTypes...)
	return &cp
}

func (r *stubSkillRepo) Create(_ context.Context, s *domain.SkillSpec) error {
	r.byID[s.ID] = cloneSkill(s)
	return nil
}

func (r *stubSkillRepo) GetByID(_ context.Context, id string) (*domain.SkillSpec, bool) {
	sk, ok := r.byID[id]
	return cloneSkill(sk), ok
}

func (r *stubSkillRepo) List(_ context.Context) ([]*domain.SkillSpec, error) {
	out := make([]*domain.SkillSpec, 0, len(r.byID))
	for _, sk := range r.byID {
		out = append(out, cloneSkill(sk))
	}
	return out, nil
}

func (r *stubSkillRepo) Update(_ context.Context, s *domain.SkillSpec) error {
	r.byID[s.ID] = cloneSkill(s)
	return nil
}

func (r *stubSkillRepo) Delete(_ context.Context, id string) error {
	delete(r.byID, id)
	return nil
}

func TestSkillService_PromptForKindPrefersEnabledBuiltin(t *testing.T) {
	repo := newStubSkillRepo()
	svc := NewSkillService(repo)

	got := svc.PromptForKind(context.Background(), domain.SkillKindDeck)
	if got != prompting.BuiltinPresentationPromptTemplate {
		t.Fatalf("expected built-in presentation prompt, got %q", got)
	}
}

func TestSkillService_SeedBuiltInsRefreshesPromptAndPreservesEnabled(t *testing.T) {
	repo := newStubSkillRepo()
	repo.byID["builtin-pptx"] = &domain.SkillSpec{
		ID:             "builtin-pptx",
		Name:           "Presentation",
		Kind:           domain.SkillKindDeck,
		PromptTemplate: "old prompt",
		Enabled:        false,
		BuiltIn:        true,
	}

	svc := NewSkillService(repo)
	sk, ok := svc.GetByID(context.Background(), "builtin-pptx")
	if !ok {
		t.Fatalf("expected builtin-pptx to exist")
	}
	if sk.Enabled {
		t.Fatalf("expected enabled flag to be preserved")
	}
	if sk.PromptTemplate != prompting.BuiltinPresentationPromptTemplate {
		t.Fatalf("expected prompt to refresh, got %q", sk.PromptTemplate)
	}
}
