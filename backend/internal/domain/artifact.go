package domain

import "time"

// ArtifactType classifies the kind of artifact produced by a task.
type ArtifactType string

const (
	ArtifactTypeMarkdown ArtifactType = "markdown"
	ArtifactTypeJSON     ArtifactType = "json"
	ArtifactTypeFile     ArtifactType = "file"
	ArtifactTypeLog      ArtifactType = "log"
)

// Artifact is any output produced by a task execution — a file, report,
// JSON export, or log bundle.
type Artifact struct {
	ID            string
	TaskID        string
	ProjectID     string
	Name          string
	Type          ArtifactType
	ContentPath   string // absolute path on disk (for large artifacts)
	ContentInline string // for small artifacts stored directly in the DB
	Size          int64
	CreatedAt     time.Time
}
