package service

import (
	"context"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/google/uuid"
)

// SkillService manages the skills registry.
type SkillService struct {
	repo domain.SkillRepository
}

// NewSkillService creates a SkillService and seeds built-in skills.
func NewSkillService(repo domain.SkillRepository) *SkillService {
	svc := &SkillService{repo: repo}
	svc.seedBuiltIns(context.Background())
	return svc
}

// List returns all registered skills.
func (s *SkillService) List(ctx context.Context) ([]*domain.SkillSpec, error) {
	return s.repo.List(ctx)
}

// GetByID returns a skill by its ID.
func (s *SkillService) GetByID(ctx context.Context, id string) (*domain.SkillSpec, bool) {
	return s.repo.GetByID(ctx, id)
}

// Create registers a new custom skill.
func (s *SkillService) Create(ctx context.Context, sk *domain.SkillSpec) (*domain.SkillSpec, error) {
	if sk.Name == "" {
		return nil, &domain.ValidationError{Field: "name", Message: "required"}
	}
	sk.ID = uuid.NewString()
	sk.BuiltIn = false
	sk.Enabled = true
	if err := s.repo.Create(ctx, sk); err != nil {
		return nil, fmt.Errorf("skill service create: %w", err)
	}
	return sk, nil
}

// UpdateEnabled toggles the enabled state of a skill.
func (s *SkillService) UpdateEnabled(ctx context.Context, id string, enabled bool) (*domain.SkillSpec, error) {
	sk, ok := s.repo.GetByID(ctx, id)
	if !ok {
		return nil, domain.ErrNotFound
	}
	sk.Enabled = enabled
	if err := s.repo.Update(ctx, sk); err != nil {
		return nil, err
	}
	return sk, nil
}

// Delete removes a custom skill (built-in skills cannot be deleted).
func (s *SkillService) Delete(ctx context.Context, id string) error {
	sk, ok := s.repo.GetByID(ctx, id)
	if !ok {
		return domain.ErrNotFound
	}
	if sk.BuiltIn {
		return &domain.ValidationError{Field: "id", Message: "cannot delete a built-in skill"}
	}
	return s.repo.Delete(ctx, id)
}

// ── Built-in seeds ────────────────────────────────────────────────

var builtInSkills = []domain.SkillSpec{
	{
		ID:             "builtin-docx",
		Name:           "Word Document",
		Kind:           domain.SkillKindDoc,
		Description:    "Create and transform Word documents — summaries, reports, business documents, and structured content.",
		OutputMimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		OutputFileExt:  ".docx",
		BuiltIn:        true,
		Enabled:        true,
		Tags:           []string{"document", "report", "word"},
		InputMimeTypes: []string{"text/plain", "application/pdf", "text/markdown"},
		PromptTemplate: "You are a professional document writer. Based on the provided input, create a well-structured Word document. Use clear headings, paragraphs, and formatting. Output as structured Markdown that will be converted to DOCX.",
	},
	{
		ID:             "builtin-xlsx",
		Name:           "Spreadsheet",
		Kind:           domain.SkillKindSheet,
		Description:    "Create and analyze Excel spreadsheets — tables, summaries, comparisons across multiple files.",
		OutputMimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		OutputFileExt:  ".xlsx",
		BuiltIn:        true,
		Enabled:        true,
		Tags:           []string{"spreadsheet", "excel", "data"},
		InputMimeTypes: []string{"text/csv", "application/json", "text/plain"},
		PromptTemplate: "You are a data analyst. Analyze the provided data and create a structured spreadsheet output as CSV with clear column headers and organized rows.",
	},
	{
		ID:             "builtin-pptx",
		Name:           "Presentation",
		Kind:           domain.SkillKindDeck,
		Description:    "Generate PowerPoint slide decks from documents, notes, or outlines with structured content.",
		OutputMimeType: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		OutputFileExt:  ".pptx",
		BuiltIn:        true,
		Enabled:        true,
		Tags:           []string{"slides", "powerpoint", "deck"},
		InputMimeTypes: []string{"text/plain", "text/markdown", "application/pdf"},
		PromptTemplate: "You are a presentation designer. Create a clear, engaging slide deck outline from the provided content. Format output as: # Slide Title\\n- bullet point\\n- bullet point\\n\\n# Next Slide...",
	},
	{
		ID:             "builtin-pdf",
		Name:           "PDF",
		Kind:           domain.SkillKindPDF,
		Description:    "Summarize, extract, compare, and generate PDF documents. Extract tables, text, and structured data.",
		OutputMimeType: "application/pdf",
		OutputFileExt:  ".pdf",
		BuiltIn:        true,
		Enabled:        true,
		Tags:           []string{"pdf", "extract", "summarize"},
		InputMimeTypes: []string{"application/pdf", "text/plain"},
		PromptTemplate: "You are a document analyst. Analyze the provided PDF content and produce a clear, structured summary with key findings, tables, and insights.",
	},
	{
		ID:             "builtin-text",
		Name:           "Text & Markdown",
		Kind:           domain.SkillKindText,
		Description:    "Process and generate plain text, Markdown, CSV, and JSON content for any workflow.",
		OutputMimeType: "text/markdown",
		OutputFileExt:  ".md",
		BuiltIn:        true,
		Enabled:        true,
		Tags:           []string{"text", "markdown", "csv", "json"},
		InputMimeTypes: []string{"text/plain", "text/markdown", "text/csv", "application/json"},
		PromptTemplate: "Process the provided text content and produce clean, well-formatted output. Use Markdown for structure where appropriate.",
	},
}

// seedBuiltIns upserts built-in skills on startup. Existing records are left
// unchanged so user modifications (e.g. enabled flag) are preserved.
func (s *SkillService) seedBuiltIns(ctx context.Context) {
	for _, sk := range builtInSkills {
		if _, exists := s.repo.GetByID(ctx, sk.ID); exists {
			continue
		}
		_ = s.repo.Create(ctx, &sk)
	}
}
