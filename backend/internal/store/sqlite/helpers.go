package sqlite

import (
	"database/sql"
	"time"
)

const timeLayout = time.RFC3339Nano

// formatTime formats a time.Time as an RFC3339Nano string for DB storage.
func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

// parseTime parses an RFC3339Nano string from the DB back to time.Time.
// Returns zero time on error (handles empty strings gracefully).
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		// Fallback for SQLite datetime() strings (no timezone suffix)
		t, _ = time.Parse("2006-01-02T15:04:05", s)
	}
	return t.UTC()
}

// nullableTime converts a *time.Time to a sql.NullString for DB writes.
func nullableTime(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: formatTime(*t), Valid: true}
}
