package domain

import "time"

// Schedule defines a recurring task configuration tied to a project.
type Schedule struct {
	ID          string
	ProjectID   string
	Name        string
	Description string
	CronExpr    string
	TaskTitle   string
	TaskDesc    string
	ProviderID  string
	Enabled     bool
	LastRunAt   *time.Time
	NextRunAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
