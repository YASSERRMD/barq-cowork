package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ContextFileStore implements persistence for domain.ContextFile.
type ContextFileStore struct{ db *sql.DB }

// NewContextFileStore returns a new ContextFileStore.
func NewContextFileStore(db *sql.DB) *ContextFileStore { return &ContextFileStore{db: db} }

const ctxFileCols = `id, project_id, name, file_path, content, description, created_at, updated_at`

func scanContextFile(row interface{ Scan(...any) error }) (*domain.ContextFile, error) {
	var cf domain.ContextFile
	var createdAt, updatedAt string
	if err := row.Scan(
		&cf.ID, &cf.ProjectID, &cf.Name, &cf.FilePath,
		&cf.Content, &cf.Description, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	cf.CreatedAt = parseTime(createdAt)
	cf.UpdatedAt = parseTime(updatedAt)
	return &cf, nil
}

// Create inserts a new context file.
func (s *ContextFileStore) Create(ctx context.Context, cf *domain.ContextFile) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO context_files
		 (id, project_id, name, file_path, content, description, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		cf.ID, cf.ProjectID, cf.Name, cf.FilePath,
		cf.Content, cf.Description,
		formatTime(cf.CreatedAt), formatTime(cf.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("context_file create: %w", err)
	}
	return nil
}

// GetByID retrieves a single context file by ID.
func (s *ContextFileStore) GetByID(ctx context.Context, id string) (*domain.ContextFile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+ctxFileCols+` FROM context_files WHERE id=?`, id)
	cf, err := scanContextFile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return cf, err
}

// ListByProject returns all context files for a project, ordered by name.
func (s *ContextFileStore) ListByProject(ctx context.Context, projectID string) ([]*domain.ContextFile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+ctxFileCols+` FROM context_files WHERE project_id=? ORDER BY name ASC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("context_file list: %w", err)
	}
	defer rows.Close()
	var out []*domain.ContextFile
	for rows.Next() {
		cf, err := scanContextFile(rows)
		if err != nil {
			return nil, fmt.Errorf("context_file scan: %w", err)
		}
		out = append(out, cf)
	}
	return out, rows.Err()
}

// Update replaces the mutable fields of a context file.
func (s *ContextFileStore) Update(ctx context.Context, cf *domain.ContextFile) error {
	cf.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE context_files
		 SET name=?, file_path=?, content=?, description=?, updated_at=?
		 WHERE id=?`,
		cf.Name, cf.FilePath, cf.Content, cf.Description,
		formatTime(cf.UpdatedAt), cf.ID,
	)
	if err != nil {
		return fmt.Errorf("context_file update: %w", err)
	}
	return nil
}

// Delete removes a context file by ID.
func (s *ContextFileStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM context_files WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("context_file delete: %w", err)
	}
	return nil
}
