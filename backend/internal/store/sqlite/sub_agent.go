package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// SubAgentStore implements persistence for domain.SubAgent.
type SubAgentStore struct{ db *sql.DB }

// NewSubAgentStore returns a new SubAgentStore.
func NewSubAgentStore(db *sql.DB) *SubAgentStore { return &SubAgentStore{db: db} }

const agentCols = `id, parent_task_id, role, title, instructions,
                   status, plan_id, created_at, updated_at, started_at, completed_at`

func scanSubAgent(row interface{ Scan(...any) error }) (*domain.SubAgent, error) {
	var a domain.SubAgent
	var createdAt, updatedAt string
	var startedAt, completedAt sql.NullString
	if err := row.Scan(
		&a.ID, &a.ParentTaskID, &a.Role, &a.Title, &a.Instructions,
		&a.Status, &a.PlanID, &createdAt, &updatedAt, &startedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	a.CreatedAt = parseTime(createdAt)
	a.UpdatedAt = parseTime(updatedAt)
	if startedAt.Valid {
		ts := parseTime(startedAt.String)
		a.StartedAt = &ts
	}
	if completedAt.Valid {
		ts := parseTime(completedAt.String)
		a.CompletedAt = &ts
	}
	return &a, nil
}

// Create inserts a new sub-agent record.
func (s *SubAgentStore) Create(ctx context.Context, a *domain.SubAgent) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sub_agents
		 (id, parent_task_id, role, title, instructions,
		  status, plan_id, created_at, updated_at, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.ParentTaskID, a.Role, a.Title, a.Instructions,
		a.Status, a.PlanID,
		formatTime(a.CreatedAt), formatTime(a.UpdatedAt),
		nullableTime(a.StartedAt), nullableTime(a.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("sub_agent create: %w", err)
	}
	return nil
}

// GetByID retrieves a sub-agent by ID.
func (s *SubAgentStore) GetByID(ctx context.Context, id string) (*domain.SubAgent, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+agentCols+` FROM sub_agents WHERE id=?`, id)
	a, err := scanSubAgent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// ListByTask returns all sub-agents for a parent task ordered by creation time.
func (s *SubAgentStore) ListByTask(ctx context.Context, parentTaskID string) ([]*domain.SubAgent, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+agentCols+` FROM sub_agents WHERE parent_task_id=? ORDER BY created_at ASC`,
		parentTaskID)
	if err != nil {
		return nil, fmt.Errorf("sub_agent list: %w", err)
	}
	defer rows.Close()
	var out []*domain.SubAgent
	for rows.Next() {
		a, err := scanSubAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("sub_agent scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateStatus sets the status and timestamps of a sub-agent.
func (s *SubAgentStore) UpdateStatus(ctx context.Context, a *domain.SubAgent) error {
	a.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE sub_agents
		 SET status=?, plan_id=?, updated_at=?, started_at=?, completed_at=?
		 WHERE id=?`,
		a.Status, a.PlanID,
		formatTime(a.UpdatedAt),
		nullableTime(a.StartedAt), nullableTime(a.CompletedAt),
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("sub_agent update: %w", err)
	}
	return nil
}

// Delete removes a sub-agent record (only valid if pending/failed).
func (s *SubAgentStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sub_agents WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("sub_agent delete: %w", err)
	}
	return nil
}
