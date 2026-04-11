package domain

import "time"

// ApprovalStatus tracks whether a destructive action has been reviewed.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

// ApprovalRequest is created when a tool wants to perform a destructive or
// sensitive action (delete, rename, move, HTTP POST). The user must resolve
// it before execution continues.
type ApprovalRequest struct {
	ID         string
	TaskID     string
	ToolName   string
	Action     string         // human-readable description of what will happen
	Payload    string         // JSON — the exact arguments the tool will receive
	Status     ApprovalStatus
	Resolution string         // "approved" | "rejected"
	CreatedAt  time.Time
	ResolvedAt *time.Time
}
