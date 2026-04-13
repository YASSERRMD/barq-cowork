package v1

import (
	"path/filepath"
	"testing"
)

func TestArtifactFullPath_AllowsAbsoluteArtifactPaths(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "server-workspace")
	abs := filepath.Join(string(filepath.Separator), "tmp", "task-workspace", "slides", "deck.pptx")

	got, err := artifactFullPath(root, abs)
	if err != nil {
		t.Fatalf("expected absolute artifact path to resolve, got error %v", err)
	}
	if got != abs {
		t.Fatalf("expected %q, got %q", abs, got)
	}
}

func TestArtifactFullPath_RelativePathStaysInsideServerRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "server-workspace")
	got, err := artifactFullPath(root, "slides/deck.pptx")
	if err != nil {
		t.Fatalf("expected relative artifact path to resolve, got error %v", err)
	}

	want := filepath.Join(root, "slides", "deck.pptx")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
