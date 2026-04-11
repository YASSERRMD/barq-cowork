package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// WorkspaceStore implements domain.WorkspaceRepository on SQLite.
type WorkspaceStore struct{ db *sql.DB }

// NewWorkspaceStore returns a new WorkspaceStore.
func NewWorkspaceStore(db *sql.DB) *WorkspaceStore { return &WorkspaceStore{db: db} }

const workspaceCols = `id, name, description, root_path, created_at, updated_at`

func scanWorkspace(row interface{ Scan(...any) error }) (*domain.Workspace, error) {
	var w domain.Workspace
	var createdAt, updatedAt string
	if err := row.Scan(&w.ID, &w.Name, &w.Description, &w.RootPath, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	w.CreatedAt = parseTime(createdAt)
	w.UpdatedAt = parseTime(updatedAt)
	return &w, nil
}

func (s *WorkspaceStore) Create(ctx context.Context, w *domain.Workspace) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO workspaces (id, name, description, root_path, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		w.ID, w.Name, w.Description, w.RootPath,
		formatTime(w.CreatedAt), formatTime(w.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("workspace create: %w", err)
	}
	return nil
}

func (s *WorkspaceStore) GetByID(ctx context.Context, id string) (*domain.Workspace, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+workspaceCols+` FROM workspaces WHERE id = ?`, id)
	w, err := scanWorkspace(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return w, err
}

func (s *WorkspaceStore) List(ctx context.Context) ([]*domain.Workspace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+workspaceCols+` FROM workspaces ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("workspace list: %w", err)
	}
	defer rows.Close()

	var out []*domain.Workspace
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, fmt.Errorf("workspace scan: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *WorkspaceStore) Update(ctx context.Context, w *domain.Workspace) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE workspaces SET name=?, description=?, root_path=?, updated_at=? WHERE id=?`,
		w.Name, w.Description, w.RootPath, formatTime(w.UpdatedAt), w.ID,
	)
	if err != nil {
		return fmt.Errorf("workspace update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *WorkspaceStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("workspace delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
