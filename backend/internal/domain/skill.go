package domain

import "context"

// SkillKind classifies the primary output format of a skill.
type SkillKind string

const (
	SkillKindDoc   SkillKind = "doc"
	SkillKindSheet SkillKind = "sheet"
	SkillKindDeck  SkillKind = "deck"
	SkillKindPDF   SkillKind = "pdf"
	SkillKindText  SkillKind = "text"
)

// SkillSpec describes a single skill — its identity, capabilities, and
// template for driving LLM execution.
type SkillSpec struct {
	ID             string
	Name           string
	Kind           SkillKind
	Description    string
	OutputMimeType string
	OutputFileExt  string
	PromptTemplate string
	BuiltIn        bool
	Enabled        bool
	Tags           []string        // comma-joined in storage
	InputMimeTypes []string        // comma-joined in storage
}

// SkillRepository is the storage port for skills.
type SkillRepository interface {
	Create(ctx context.Context, s *SkillSpec) error
	GetByID(ctx context.Context, id string) (*SkillSpec, bool)
	List(ctx context.Context) ([]*SkillSpec, error)
	Update(ctx context.Context, s *SkillSpec) error
	Delete(ctx context.Context, id string) error
}
