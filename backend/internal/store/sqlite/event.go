package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// EventStore implements persistence for domain.Event.
type EventStore struct{ db *sql.DB }

// NewEventStore returns a new EventStore.
func NewEventStore(db *sql.DB) *EventStore { return &EventStore{db: db} }

const eventCols = `id, task_id, type, payload, created_at`

func scanEvent(row interface{ Scan(...any) error }) (*domain.Event, error) {
	var e domain.Event
	var createdAt string
	if err := row.Scan(&e.ID, &e.TaskID, &e.Type, &e.Payload, &createdAt); err != nil {
		return nil, err
	}
	e.CreatedAt = parseTime(createdAt)
	return &e, nil
}

// Create inserts a new event.
func (s *EventStore) Create(ctx context.Context, e *domain.Event) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (id, task_id, type, payload, created_at) VALUES (?, ?, ?, ?, ?)`,
		e.ID, e.TaskID, e.Type, e.Payload, formatTime(e.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("event create: %w", err)
	}
	return nil
}

// ListRecent returns the most recent events across all tasks, newest first.
// limit <= 0 defaults to 200.
func (s *EventStore) ListRecent(ctx context.Context, limit int) ([]*domain.Event, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+eventCols+` FROM events ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("event list recent: %w", err)
	}
	defer rows.Close()
	var out []*domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("event scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ListByTask returns all events for a task ordered by creation time.
func (s *EventStore) ListByTask(ctx context.Context, taskID string) ([]*domain.Event, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+eventCols+` FROM events WHERE task_id=? ORDER BY created_at ASC`, taskID)
	if err != nil {
		return nil, fmt.Errorf("event list: %w", err)
	}
	defer rows.Close()
	var out []*domain.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("event scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
