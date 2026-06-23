package filestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"coding-agent/session"
	"coding-agent/types"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	created, err := fs.Create(ctx, "anthropic", "claude-sonnet-4-5")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.Messages = []types.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	if err := fs.Save(ctx, created); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := fs.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Messages) != 2 || got.Messages[0].Content != "hello" {
		t.Fatalf("messages = %+v, want hello/hi", got.Messages)
	}

	metas, err := fs.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 1 || metas[0].MessageCount != 2 {
		t.Fatalf("List = %+v, want 1 session with 2 messages", metas)
	}
}

func TestFileStoreGetMissing(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = fs.Get(context.Background(), "missing-id")
	if err == nil {
		t.Fatal("expected error for missing session")
	}
}

func TestFileStoreAtomicSave(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	s, err := fs.Create(ctx, "anthropic", "model")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	path := filepath.Join(dir, s.ID+".json")
	s.Messages = []types.Message{{Role: "user", Content: "test"}}
	if err := fs.Save(ctx, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("session file missing: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file should not remain after save")
	}
}

func TestSessionTimestampsUpdatedOnSave(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	s, err := fs.Create(ctx, "openrouter", "gpt-4o")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	before := s.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	s.Messages = []types.Message{{Role: "user", Content: "x"}}
	if err := fs.Save(ctx, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !s.UpdatedAt.After(before) {
		t.Fatalf("UpdatedAt should advance on save")
	}
}

func TestFileStoreInvalidID(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = fs.Get(context.Background(), "../escape")
	if err == nil {
		t.Fatal("expected error for invalid session id")
	}
}

func TestFileStoreNameRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	s, err := fs.Create(ctx, "anthropic", "model")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	s.Name = "my task"
	if err := fs.Save(ctx, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := fs.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "my task" {
		t.Fatalf("name = %q, want my task", got.Name)
	}
	metas, err := fs.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 1 || metas[0].Name != "my task" {
		t.Fatalf("metas = %+v", metas)
	}
}

func TestFileStoreBackwardCompatNoCompactions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	legacy := `{"id":"legacy","created_at":"2026-06-18T00:00:00Z","updated_at":"2026-06-18T00:00:00Z","provider":"anthropic","model":"m","messages":[{"role":"user","content":"hi"}]}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := fs.Get(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Messages) != 1 || got.Compactions != nil {
		t.Fatalf("messages = %+v compactions = %+v", got.Messages, got.Compactions)
	}
}

func TestFileStoreCompactionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()
	s, err := fs.Create(ctx, "anthropic", "model")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	s.Messages = []types.Message{
		{Role: "user", Content: "old"},
		{Role: "user", Content: "recent"},
	}
	s.Compactions = []session.CompactionRecord{{
		ID:             "c1",
		Timestamp:      time.Now().UTC(),
		Summary:        "summarized old",
		FirstKeptIndex: 1,
		TokensBefore:   500,
		ReadFiles:      []string{"foo.go"},
	}}
	if err := fs.Save(ctx, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := fs.Get(ctx, s.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Compactions) != 1 || got.Compactions[0].Summary != "summarized old" {
		t.Fatalf("compactions = %+v", got.Compactions)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("archive messages = %d, want 2", len(got.Messages))
	}
}

func TestFileStoreBackwardCompatNoName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	legacy := `{"id":"legacy","created_at":"2026-06-18T00:00:00Z","updated_at":"2026-06-18T00:00:00Z","provider":"anthropic","model":"m","messages":[]}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}
	fs, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := fs.Get(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "" {
		t.Fatalf("name = %q, want empty", got.Name)
	}
}

var _ session.Store = (*FileStore)(nil)
