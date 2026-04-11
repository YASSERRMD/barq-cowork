package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/barq-cowork/barq-cowork/internal/store/sqlite"
	"github.com/google/uuid"
)

// openTestDB creates an in-memory SQLite database for testing.
func openTestDB(t *testing.T) interface {
	Close() error
} {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestWorkspaceStore_CRUD(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	store := sqlite.NewWorkspaceStore(db)
	ctx := context.Background()

	// Create
	w := &domain.Workspace{
		ID:          uuid.NewString(),
		Name:        "Test Workspace",
		Description: "desc",
		RootPath:    "/tmp/test",
		CreatedAt:   time.Now().UTC().Truncate(time.Second),
		UpdatedAt:   time.Now().UTC().Truncate(time.Second),
	}
	if err := store.Create(ctx, w); err != nil {
		t.Fatalf("create: %v", err)
	}

	// GetByID
	got, err := store.GetByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != w.Name {
		t.Errorf("name mismatch: got %q, want %q", got.Name, w.Name)
	}
	if got.RootPath != w.RootPath {
		t.Errorf("root_path mismatch: got %q, want %q", got.RootPath, w.RootPath)
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(list))
	}

	// Update
	w.Name = "Updated"
	w.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	if err := store.Update(ctx, w); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = store.GetByID(ctx, w.ID)
	if got.Name != "Updated" {
		t.Errorf("expected Updated, got %q", got.Name)
	}

	// Delete
	if err := store.Delete(ctx, w.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = store.GetByID(ctx, w.ID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestProjectStore_CRUD(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	wsStore := sqlite.NewWorkspaceStore(db)
	pStore := sqlite.NewProjectStore(db)
	ctx := context.Background()

	ws := &domain.Workspace{
		ID: uuid.NewString(), Name: "WS",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	_ = wsStore.Create(ctx, ws)

	p := &domain.Project{
		ID:           uuid.NewString(),
		WorkspaceID:  ws.ID,
		Name:         "My Project",
		Instructions: "do stuff",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := pStore.Create(ctx, p); err != nil {
		t.Fatalf("create project: %v", err)
	}

	got, err := pStore.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.Instructions != "do stuff" {
		t.Errorf("instructions mismatch: %q", got.Instructions)
	}

	list, _ := pStore.ListByWorkspace(ctx, ws.ID)
	if len(list) != 1 {
		t.Errorf("expected 1 project, got %d", len(list))
	}

	_ = pStore.Delete(ctx, p.ID)
	_, err = pStore.GetByID(ctx, p.ID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskStore_CRUD(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	wsStore := sqlite.NewWorkspaceStore(db)
	pStore := sqlite.NewProjectStore(db)
	tStore := sqlite.NewTaskStore(db)
	ctx := context.Background()

	ws := &domain.Workspace{ID: uuid.NewString(), Name: "WS",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	_ = wsStore.Create(ctx, ws)

	p := &domain.Project{ID: uuid.NewString(), WorkspaceID: ws.ID, Name: "P",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	_ = pStore.Create(ctx, p)

	task := &domain.Task{
		ID:        uuid.NewString(),
		ProjectID: p.ID,
		Title:     "Summarise the report",
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := tStore.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	got, err := tStore.GetByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Status != domain.TaskStatusPending {
		t.Errorf("expected pending, got %s", got.Status)
	}

	// UpdateStatus
	if err := tStore.UpdateStatus(ctx, task.ID, domain.TaskStatusRunning, time.Now().UTC()); err != nil {
		t.Fatalf("update status: %v", err)
	}
	got, _ = tStore.GetByID(ctx, task.ID)
	if got.Status != domain.TaskStatusRunning {
		t.Errorf("expected running, got %s", got.Status)
	}
	if got.StartedAt == nil {
		t.Error("started_at should be set when transitioning to running")
	}

	list, _ := tStore.ListByProject(ctx, p.ID)
	if len(list) != 1 {
		t.Errorf("expected 1 task, got %d", len(list))
	}
}
