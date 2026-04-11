package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ApprovalStore implements persistence for domain.ApprovalRequest.
type ApprovalStore struct{ db *sql.DB }

// NewApprovalStore returns a new ApprovalStore.
func NewApprovalStore(db *sql.DB) *ApprovalStore { return &ApprovalStore{db: db} }

const approvalCols = `id, task_id, tool_name, action, payload, status, resolution, created_at, resolved_at`

func scanApproval(row interface{ Scan(...any) error }) (*domain.ApprovalRequest, error) {
	var a domain.ApprovalRequest
	var createdAt string
	var resolvedAt sql.NullString
	if err := row.Scan(
		&a.ID, &a.TaskID, &a.ToolName, &a.Action, &a.Payload,
		&a.Status, &a.Resolution, &createdAt, &resolvedAt,
	); err != nil {
		return nil, err
	}
	a.CreatedAt = parseTime(createdAt)
	if resolvedAt.Valid {
		ts := parseTime(resolvedAt.String)
		a.ResolvedAt = &ts
	}
	return &a, nil
}

// Create inserts a new approval request.
func (s *ApprovalStore) Create(ctx context.Context, a *domain.ApprovalRequest) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO approval_requests
		 (id, task_id, tool_name, action, payload, status, resolution, created_at, resolved_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.TaskID, a.ToolName, a.Action, a.Payload,
		a.Status, a.Resolution, formatTime(a.CreatedAt), nullableTime(a.ResolvedAt),
	)
	if err != nil {
		return fmt.Errorf("approval create: %w", err)
	}
	return nil
}

// GetByID retrieves an approval request by ID.
func (s *ApprovalStore) GetByID(ctx context.Context, id string) (*domain.ApprovalRequest, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+approvalCols+` FROM approval_requests WHERE id=?`, id)
	a, err := scanApproval(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// ListPending returns all unresolved approval requests.
func (s *ApprovalStore) ListPending(ctx context.Context) ([]*domain.ApprovalRequest, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+approvalCols+` FROM approval_requests WHERE status='pending' ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("approval list pending: %w", err)
	}
	defer rows.Close()
	var out []*domain.ApprovalRequest
	for rows.Next() {
		a, err := scanApproval(rows)
		if err != nil {
			return nil, fmt.Errorf("approval scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Resolve sets the status and resolution of an approval request.
func (s *ApprovalStore) Resolve(ctx context.Context, id, resolution string) error {
	status := domain.ApprovalStatusApproved
	if resolution == "rejected" {
		status = domain.ApprovalStatusRejected
	}
	now := formatTime(time.Now().UTC())
	res, err := s.db.ExecContext(ctx,
		`UPDATE approval_requests SET status=?, resolution=?, resolved_at=? WHERE id=?`,
		status, resolution, now, id,
	)
	if err != nil {
		return fmt.Errorf("approval resolve: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
