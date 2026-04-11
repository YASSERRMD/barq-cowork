package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// TaskTemplateStore implements persistence for domain.TaskTemplate.
type TaskTemplateStore struct{ db *sql.DB }

// NewTaskTemplateStore returns a new TaskTemplateStore.
func NewTaskTemplateStore(db *sql.DB) *TaskTemplateStore { return &TaskTemplateStore{db: db} }

const tmplCols = `id, project_id, name, title, description, provider_id, created_at, updated_at`

func scanTaskTemplate(row interface{ Scan(...any) error }) (*domain.TaskTemplate, error) {
	var t domain.TaskTemplate
	var createdAt, updatedAt string
	if err := row.Scan(
		&t.ID, &t.ProjectID, &t.Name, &t.Title,
		&t.Description, &t.ProviderID, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	return &t, nil
}

// Create inserts a new task template.
func (s *TaskTemplateStore) Create(ctx context.Context, t *domain.TaskTemplate) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO task_templates
		 (id, project_id, name, title, description, provider_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.Name, t.Title,
		t.Description, t.ProviderID,
		formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("task_template create: %w", err)
	}
	return nil
}

// GetByID retrieves a single task template by ID.
func (s *TaskTemplateStore) GetByID(ctx context.Context, id string) (*domain.TaskTemplate, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+tmplCols+` FROM task_templates WHERE id=?`, id)
	t, err := scanTaskTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return t, err
}

// ListByProject returns all templates for a project, ordered by name.
func (s *TaskTemplateStore) ListByProject(ctx context.Context, projectID string) ([]*domain.TaskTemplate, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+tmplCols+` FROM task_templates WHERE project_id=? ORDER BY name ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("task_template list: %w", err)
	}
	defer rows.Close()
	var out []*domain.TaskTemplate
	for rows.Next() {
		t, err := scanTaskTemplate(rows)
		if err != nil {
			return nil, fmt.Errorf("task_template scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Update replaces the mutable fields of a task template.
func (s *TaskTemplateStore) Update(ctx context.Context, t *domain.TaskTemplate) error {
	t.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE task_templates
		 SET name=?, title=?, description=?, provider_id=?, updated_at=?
		 WHERE id=?`,
		t.Name, t.Title, t.Description, t.ProviderID,
		formatTime(t.UpdatedAt), t.ID,
	)
	if err != nil {
		return fmt.Errorf("task_template update: %w", err)
	}
	return nil
}

// Delete removes a task template by ID.
func (s *TaskTemplateStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM task_templates WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("task_template delete: %w", err)
	}
	return nil
}
