// Package memory implements the WorkspaceMemory abstraction.
// The simple implementation performs exact-match recall by loading all
// context files for a project from the store. The interface is designed so
// a future implementation can swap in vector/semantic retrieval without
// changing the planner or orchestrator code.
package memory

import (
	"context"
	"fmt"

	"github.com/barq-cowork/barq-cowork/internal/domain"
)

// ContextFileReader is the narrow store interface the memory package needs.
type ContextFileReader interface {
	ListByProject(ctx context.Context, projectID string) ([]*domain.ContextFile, error)
}

// SimpleWorkspaceMemory loads all context files for a project and returns
// them as MemoryEntries. The query parameter is ignored in this implementation.
// Future implementations can rank by cosine similarity to query.
type SimpleWorkspaceMemory struct {
	store ContextFileReader
}

// New creates a SimpleWorkspaceMemory backed by the given store.
func New(store ContextFileReader) *SimpleWorkspaceMemory {
	return &SimpleWorkspaceMemory{store: store}
}

// Recall returns all context files for the project as MemoryEntries.
// All entries are given a score of 1.0 since no ranking is performed.
func (m *SimpleWorkspaceMemory) Recall(projectID, _ string) ([]domain.MemoryEntry, error) {
	files, err := m.store.ListByProject(context.Background(), projectID)
	if err != nil {
		return nil, fmt.Errorf("workspace memory recall: %w", err)
	}

	entries := make([]domain.MemoryEntry, 0, len(files))
	for _, cf := range files {
		content := cf.Content
		if content == "" && cf.FilePath != "" {
			// Placeholder: in a future iteration, read the file from workspace.
			content = fmt.Sprintf("[file: %s — content not loaded inline]", cf.FilePath)
		}
		if content == "" {
			continue // skip empty entries
		}
		entries = append(entries, domain.MemoryEntry{
			Source:  "context_file:" + cf.ID,
			Label:   cf.Name,
			Content: content,
			Score:   1.0,
		})
	}
	return entries, nil
}
