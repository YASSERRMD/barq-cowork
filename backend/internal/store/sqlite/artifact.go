package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ArtifactStore implements persistence for domain.Artifact.
type ArtifactStore struct{ db *sql.DB }

// NewArtifactStore returns a new ArtifactStore.
func NewArtifactStore(db *sql.DB) *ArtifactStore { return &ArtifactStore{db: db} }

const artifactCols = `id, task_id, project_id, name, type, content_path, content_inline, size, created_at`

func scanArtifact(row interface{ Scan(...any) error }) (*domain.Artifact, error) {
	var a domain.Artifact
	var createdAt string
	if err := row.Scan(
		&a.ID, &a.TaskID, &a.ProjectID, &a.Name, &a.Type,
		&a.ContentPath, &a.ContentInline, &a.Size, &createdAt,
	); err != nil {
		return nil, err
	}
	a.CreatedAt = parseTime(createdAt)
	return &a, nil
}

// Create inserts a new artifact record.
func (s *ArtifactStore) Create(ctx context.Context, a *domain.Artifact) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts
		 (id, task_id, project_id, name, type, content_path, content_inline, size, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TaskID, a.ProjectID, a.Name, a.Type,
		a.ContentPath, a.ContentInline, a.Size, formatTime(a.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("artifact create: %w", err)
	}
	return nil
}

// GetByID retrieves an artifact by ID.
func (s *ArtifactStore) GetByID(ctx context.Context, id string) (*domain.Artifact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+artifactCols+` FROM artifacts WHERE id=?`, id)
	a, err := scanArtifact(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// ListByTask returns all artifacts for a task ordered by creation time.
func (s *ArtifactStore) ListByTask(ctx context.Context, taskID string) ([]*domain.Artifact, error) {
	return s.list(ctx, `task_id=?`, taskID)
}

// ListByProject returns all artifacts for a project ordered by creation time.
func (s *ArtifactStore) ListByProject(ctx context.Context, projectID string) ([]*domain.Artifact, error) {
	return s.list(ctx, `project_id=?`, projectID)
}

// ListRecent returns the most recent artifacts across all tasks/projects.
// limit <= 0 defaults to 100.
func (s *ArtifactStore) ListRecent(ctx context.Context, limit int) ([]*domain.Artifact, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+artifactCols+` FROM artifacts ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("artifact list recent: %w", err)
	}
	defer rows.Close()
	var out []*domain.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, fmt.Errorf("artifact scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *ArtifactStore) list(ctx context.Context, where, arg string) ([]*domain.Artifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+artifactCols+` FROM artifacts WHERE `+where+` ORDER BY created_at DESC`, arg)
	if err != nil {
		return nil, fmt.Errorf("artifact list: %w", err)
	}
	defer rows.Close()
	var out []*domain.Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, fmt.Errorf("artifact scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
