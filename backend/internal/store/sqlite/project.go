package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ProjectStore implements domain.ProjectRepository on SQLite.
type ProjectStore struct{ db *sql.DB }

// NewProjectStore returns a new ProjectStore.
func NewProjectStore(db *sql.DB) *ProjectStore { return &ProjectStore{db: db} }

const projectCols = `id, workspace_id, name, description, instructions, created_at, updated_at`

func scanProject(row interface{ Scan(...any) error }) (*domain.Project, error) {
	var p domain.Project
	var createdAt, updatedAt string
	if err := row.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Description,
		&p.Instructions, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	p.CreatedAt = parseTime(createdAt)
	p.UpdatedAt = parseTime(updatedAt)
	return &p, nil
}

func (s *ProjectStore) Create(ctx context.Context, p *domain.Project) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO projects (id, workspace_id, name, description, instructions, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.WorkspaceID, p.Name, p.Description, p.Instructions,
		formatTime(p.CreatedAt), formatTime(p.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("project create: %w", err)
	}
	return nil
}

func (s *ProjectStore) GetByID(ctx context.Context, id string) (*domain.Project, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+projectCols+` FROM projects WHERE id = ?`, id)
	p, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return p, err
}

func (s *ProjectStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*domain.Project, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+projectCols+` FROM projects WHERE workspace_id=? ORDER BY created_at DESC`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("project list: %w", err)
	}
	defer rows.Close()

	var out []*domain.Project
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("project scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProjectStore) Update(ctx context.Context, p *domain.Project) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE projects SET name=?, description=?, instructions=?, updated_at=? WHERE id=?`,
		p.Name, p.Description, p.Instructions, formatTime(p.UpdatedAt), p.ID,
	)
	if err != nil {
		return fmt.Errorf("project update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *ProjectStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("project delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
