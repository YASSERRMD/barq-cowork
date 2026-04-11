package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/barq-cowork/barq-cowork/internal/tools"
)

// newTestCtx creates an InvocationContext scoped to a temporary directory.
func newTestCtx(t *testing.T) (tools.InvocationContext, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "barq-tools-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return tools.InvocationContext{WorkspaceRoot: dir}, dir
}

// ─── scope tests ───────────────────────────────────────────────────

func TestScopedPath_RejectsEscape(t *testing.T) {
	_, dir := newTestCtx(t)
	ictx := tools.InvocationContext{WorkspaceRoot: dir}

	// write_file to path that escapes workspace should fail
	result := tools.WriteFileTool{}.Execute(
		context.Background(), ictx,
		`{"path":"../../etc/passwd","content":"hacked"}`,
	)
	if result.Status != tools.ResultError {
		t.Errorf("expected error for path escape, got %s: %s", result.Status, result.Content)
	}
}

// ─── list_files ────────────────────────────────────────────────────

func TestListFilesTool(t *testing.T) {
	ictx, dir := newTestCtx(t)
	_ = os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hi"), 0o644)

	result := tools.ListFilesTool{}.Execute(context.Background(), ictx, `{"path":"."}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("expected ok, got %s: %s", result.Status, result.Error)
	}
}

// ─── read_file / write_file ────────────────────────────────────────

func TestWriteAndReadFile(t *testing.T) {
	ictx, _ := newTestCtx(t)

	// Write
	wr := tools.WriteFileTool{}.Execute(context.Background(), ictx,
		`{"path":"sub/test.txt","content":"hello world"}`)
	if wr.Status != tools.ResultOK {
		t.Fatalf("write: %s", wr.Error)
	}

	// Read back
	rr := tools.ReadFileTool{}.Execute(context.Background(), ictx,
		`{"path":"sub/test.txt"}`)
	if rr.Status != tools.ResultOK {
		t.Fatalf("read: %s", rr.Error)
	}
}

func TestReadFileTool_Missing(t *testing.T) {
	ictx, _ := newTestCtx(t)
	result := tools.ReadFileTool{}.Execute(context.Background(), ictx,
		`{"path":"does-not-exist.txt"}`)
	if result.Status != tools.ResultError {
		t.Errorf("expected error, got %s", result.Status)
	}
}

// ─── create_folder ────────────────────────────────────────────────

func TestCreateFolderTool(t *testing.T) {
	ictx, dir := newTestCtx(t)
	result := tools.CreateFolderTool{}.Execute(context.Background(), ictx,
		`{"path":"a/b/c"}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("create folder: %s", result.Error)
	}
	if _, err := os.Stat(filepath.Join(dir, "a/b/c")); err != nil {
		t.Errorf("directory not created: %v", err)
	}
}

// ─── search_files ─────────────────────────────────────────────────

func TestSearchFilesTool(t *testing.T) {
	ictx, dir := newTestCtx(t)
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle in a haystack"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("nothing here"), 0o644)

	result := tools.SearchFilesTool{}.Execute(context.Background(), ictx,
		`{"pattern":"needle"}`)
	if result.Status != tools.ResultOK {
		t.Fatalf("search: %s", result.Error)
	}
}

// ─── move_file (approval denied) ──────────────────────────────────

func TestMoveFileTool_DeniedByApproval(t *testing.T) {
	ictx, dir := newTestCtx(t)
	_ = os.WriteFile(filepath.Join(dir, "src.txt"), []byte("data"), 0o644)

	// Set require_approval that always denies
	ictx.RequireApproval = func(_ context.Context, _, _ string) bool { return false }

	result := tools.MoveFileTool{}.Execute(context.Background(), ictx,
		`{"source":"src.txt","destination":"dst.txt"}`)
	if result.Status != tools.ResultDenied {
		t.Errorf("expected denied, got %s", result.Status)
	}
}

// ─── tool registry ─────────────────────────────────────────────────

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := tools.NewRegistry()
	r.Register(tools.ListFilesTool{})
	r.Register(tools.ReadFileTool{})

	if _, ok := r.Get("list_files"); !ok {
		t.Error("expected list_files to be registered")
	}
	if _, ok := r.Get("nonexistent"); ok {
		t.Error("expected nonexistent tool to be missing")
	}
	if len(r.List()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(r.List()))
	}
}
