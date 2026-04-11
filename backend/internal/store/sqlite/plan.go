package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// PlanStore implements persistence for domain.Plan and domain.PlanStep.
type PlanStore struct{ db *sql.DB }

// NewPlanStore returns a new PlanStore.
func NewPlanStore(db *sql.DB) *PlanStore { return &PlanStore{db: db} }

// ─────────────────────────────────────────────
// Plan CRUD
// ─────────────────────────────────────────────

// CreatePlan inserts a new plan (without steps).
func (s *PlanStore) CreatePlan(ctx context.Context, p *domain.Plan) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO plans (id, task_id, created_at) VALUES (?, ?, ?)`,
		p.ID, p.TaskID, formatTime(p.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("plan create: %w", err)
	}
	return nil
}

// GetPlanByTask loads the plan and all its steps for a task.
func (s *PlanStore) GetPlanByTask(ctx context.Context, taskID string) (*domain.Plan, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, task_id, created_at FROM plans WHERE task_id=? ORDER BY created_at DESC LIMIT 1`, taskID)
	var p domain.Plan
	var createdAt string
	if err := row.Scan(&p.ID, &p.TaskID, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("plan scan: %w", err)
	}
	p.CreatedAt = parseTime(createdAt)

	steps, err := s.listSteps(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Steps = steps
	return &p, nil
}

// ─────────────────────────────────────────────
// PlanStep CRUD
// ─────────────────────────────────────────────

const stepCols = `id, plan_id, step_order, title, description, status,
                  tool_name, tool_input, tool_output, started_at, completed_at`

func scanStep(row interface{ Scan(...any) error }) (*domain.PlanStep, error) {
	var s domain.PlanStep
	var startedAt, completedAt sql.NullString
	if err := row.Scan(
		&s.ID, &s.PlanID, &s.Order, &s.Title, &s.Description, &s.Status,
		&s.ToolName, &s.ToolInput, &s.ToolOutput, &startedAt, &completedAt,
	); err != nil {
		return nil, err
	}
	if startedAt.Valid {
		ts := parseTime(startedAt.String)
		s.StartedAt = &ts
	}
	if completedAt.Valid {
		ts := parseTime(completedAt.String)
		s.CompletedAt = &ts
	}
	return &s, nil
}

// CreateStep inserts a plan step.
func (s *PlanStore) CreateStep(ctx context.Context, step *domain.PlanStep) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO plan_steps
		 (id, plan_id, step_order, title, description, status,
		  tool_name, tool_input, tool_output, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		step.ID, step.PlanID, step.Order, step.Title, step.Description, step.Status,
		step.ToolName, step.ToolInput, step.ToolOutput,
		nullableTime(step.StartedAt), nullableTime(step.CompletedAt),
	)
	if err != nil {
		return fmt.Errorf("step create: %w", err)
	}
	return nil
}

// UpdateStep replaces all mutable fields of a plan step.
func (s *PlanStore) UpdateStep(ctx context.Context, step *domain.PlanStep) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE plan_steps
		 SET status=?, tool_output=?, started_at=?, completed_at=? WHERE id=?`,
		step.Status, step.ToolOutput,
		nullableTime(step.StartedAt), nullableTime(step.CompletedAt), step.ID,
	)
	if err != nil {
		return fmt.Errorf("step update: %w", err)
	}
	return nil
}

// listSteps loads all steps for a plan ordered by step_order.
func (s *PlanStore) listSteps(ctx context.Context, planID string) ([]*domain.PlanStep, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+stepCols+` FROM plan_steps WHERE plan_id=? ORDER BY step_order ASC`, planID)
	if err != nil {
		return nil, fmt.Errorf("steps list: %w", err)
	}
	defer rows.Close()
	var out []*domain.PlanStep
	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return nil, fmt.Errorf("step scan: %w", err)
		}
		out = append(out, step)
	}
	return out, rows.Err()
}
