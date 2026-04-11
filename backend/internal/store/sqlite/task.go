package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// TaskStore implements domain.TaskRepository on SQLite.
type TaskStore struct{ db *sql.DB }

// NewTaskStore returns a new TaskStore.
func NewTaskStore(db *sql.DB) *TaskStore { return &TaskStore{db: db} }

const taskCols = `id, project_id, title, description, status, provider_id,
                  created_at, updated_at, started_at, completed_at`

func scanTask(row interface{ Scan(...any) error }) (*domain.Task, error) {
	var t domain.Task
	var createdAt, updatedAt string
	var startedAt, completedAt sql.NullString

	if err := row.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description,
		&t.Status, &t.ProviderID,
		&createdAt, &updatedAt, &startedAt, &completedAt,
	); err != nil {
		return nil, err
	}

	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
	if startedAt.Valid {
		ts := parseTime(startedAt.String)
		t.StartedAt = &ts
	}
	if completedAt.Valid {
		ts := parseTime(completedAt.String)
		t.CompletedAt = &ts
	}
	return &t, nil
}

func (s *TaskStore) Create(ctx context.Context, t *domain.Task) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks
		 (id, project_id, title, description, status, provider_id, created_at, updated_at, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ProjectID, t.Title, t.Description, t.Status, t.ProviderID,
		formatTime(t.CreatedAt), formatTime(t.UpdatedAt),
		nullableTime(t.StartedAt), nullableTime(t.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("task create: %w", err)
	}
	return nil
}

func (s *TaskStore) GetByID(ctx context.Context, id string) (*domain.Task, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+taskCols+` FROM tasks WHERE id=?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return t, err
}

func (s *TaskStore) ListByProject(ctx context.Context, projectID string) ([]*domain.Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskCols+` FROM tasks WHERE project_id=? ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("task list: %w", err)
	}
	defer rows.Close()

	var out []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("task scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *TaskStore) Update(ctx context.Context, t *domain.Task) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET title=?, description=?, status=?, provider_id=?,
		 updated_at=?, started_at=?, completed_at=? WHERE id=?`,
		t.Title, t.Description, t.Status, t.ProviderID,
		formatTime(t.UpdatedAt), nullableTime(t.StartedAt), nullableTime(t.CompletedAt),
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("task update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *TaskStore) UpdateStatus(ctx context.Context, id string, status domain.TaskStatus, now time.Time) error {
	// Conditionally set started_at / completed_at based on the new status.
	switch status {
	case domain.TaskStatusRunning:
		_, err := s.db.ExecContext(ctx,
			`UPDATE tasks SET status=?, started_at=?, updated_at=? WHERE id=?`,
			status, formatTime(now), formatTime(now), id,
		)
		return err
	case domain.TaskStatusCompleted, domain.TaskStatusFailed, domain.TaskStatusCancelled:
		_, err := s.db.ExecContext(ctx,
			`UPDATE tasks SET status=?, completed_at=?, updated_at=? WHERE id=?`,
			status, formatTime(now), formatTime(now), id,
		)
		return err
	default:
		_, err := s.db.ExecContext(ctx,
			`UPDATE tasks SET status=?, updated_at=? WHERE id=?`,
			status, formatTime(now), id,
		)
		return err
	}
}

// RecoverStuck resets all tasks currently in 'planning' or 'running' state
// back to 'failed'. This is called on startup to handle tasks that were
// interrupted by a previous crash or forced shutdown.
func (s *TaskStore) RecoverStuck(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET status='failed', updated_at=?
		 WHERE status IN ('planning','running')`,
		formatTime(time.Now().UTC()),
	)
	if err != nil {
		return 0, fmt.Errorf("task recover stuck: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *TaskStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("task delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
