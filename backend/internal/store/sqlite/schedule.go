package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ScheduleStore implements persistence for domain.Schedule.
type ScheduleStore struct{ db *sql.DB }

func NewScheduleStore(db *sql.DB) *ScheduleStore {
	return &ScheduleStore{db: db}
}

const scheduleCols = `id, project_id, name, description, cron_expr, task_title, task_desc,
                      provider_id, enabled, last_run_at, next_run_at, created_at, updated_at`

func scanSchedule(row interface{ Scan(...any) error }) (*domain.Schedule, error) {
	var s domain.Schedule
	var enabled int
	var lastRunAt, nextRunAt sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(
		&s.ID, &s.ProjectID, &s.Name, &s.Description, &s.CronExpr,
		&s.TaskTitle, &s.TaskDesc, &s.ProviderID, &enabled,
		&lastRunAt, &nextRunAt, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	s.Enabled = enabled == 1
	if lastRunAt.Valid {
		t := parseTime(lastRunAt.String)
		s.LastRunAt = &t
	}
	if nextRunAt.Valid {
		t := parseTime(nextRunAt.String)
		s.NextRunAt = &t
	}
	s.CreatedAt = parseTime(createdAt)
	s.UpdatedAt = parseTime(updatedAt)
	return &s, nil
}

func (st *ScheduleStore) Create(ctx context.Context, s *domain.Schedule) error {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	var lastRunAt, nextRunAt interface{}
	if s.LastRunAt != nil {
		lastRunAt = formatTime(*s.LastRunAt)
	}
	if s.NextRunAt != nil {
		nextRunAt = formatTime(*s.NextRunAt)
	}
	_, err := st.db.ExecContext(ctx,
		`INSERT INTO schedules
		 (id, project_id, name, description, cron_expr, task_title, task_desc, provider_id, enabled, last_run_at, next_run_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.ProjectID, s.Name, s.Description, s.CronExpr, s.TaskTitle, s.TaskDesc,
		s.ProviderID, enabled, lastRunAt, nextRunAt,
		formatTime(s.CreatedAt), formatTime(s.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("schedule create: %w", err)
	}
	return nil
}

func (st *ScheduleStore) GetByID(ctx context.Context, id string) (*domain.Schedule, error) {
	row := st.db.QueryRowContext(ctx,
		`SELECT `+scheduleCols+` FROM schedules WHERE id=?`, id)
	s, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return s, err
}

func (st *ScheduleStore) ListByProject(ctx context.Context, projectID string) ([]*domain.Schedule, error) {
	rows, err := st.db.QueryContext(ctx,
		`SELECT `+scheduleCols+` FROM schedules WHERE project_id=? ORDER BY name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("schedule list: %w", err)
	}
	defer rows.Close()
	var out []*domain.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("schedule scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (st *ScheduleStore) List(ctx context.Context) ([]*domain.Schedule, error) {
	rows, err := st.db.QueryContext(ctx,
		`SELECT `+scheduleCols+` FROM schedules ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("schedule list all: %w", err)
	}
	defer rows.Close()
	var out []*domain.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("schedule scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (st *ScheduleStore) Update(ctx context.Context, s *domain.Schedule) error {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	var lastRunAt, nextRunAt interface{}
	if s.LastRunAt != nil {
		lastRunAt = formatTime(*s.LastRunAt)
	}
	if s.NextRunAt != nil {
		nextRunAt = formatTime(*s.NextRunAt)
	}
	res, err := st.db.ExecContext(ctx,
		`UPDATE schedules
		 SET name=?, description=?, cron_expr=?, task_title=?, task_desc=?, provider_id=?,
		     enabled=?, last_run_at=?, next_run_at=?, updated_at=?
		 WHERE id=?`,
		s.Name, s.Description, s.CronExpr, s.TaskTitle, s.TaskDesc, s.ProviderID,
		enabled, lastRunAt, nextRunAt, formatTime(s.UpdatedAt), s.ID,
	)
	if err != nil {
		return fmt.Errorf("schedule update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (st *ScheduleStore) Delete(ctx context.Context, id string) error {
	res, err := st.db.ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("schedule delete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
