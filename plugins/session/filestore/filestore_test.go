package filestore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"coding-agent/session"
	"coding-agent/types"

	"github.com/stretchr/testify/require"
)

func TestFileStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()

	created, err := fs.Create(ctx, "anthropic", "claude-sonnet-4-5")
	require.NoError(t, err)

	created.Messages = []types.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	require.NoError(t, fs.Save(ctx, created))

	got, err := fs.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, got.Messages, 2)
	require.Equal(t, "hello", got.Messages[0].Content)

	metas, err := fs.List(ctx)
	require.NoError(t, err)
	require.Len(t, metas, 1)
	require.Equal(t, 2, metas[0].MessageCount)
}

func TestFileStoreGetMissing(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	_, err = fs.Get(context.Background(), "missing-id")
	require.Error(t, err)
}

func TestFileStoreAtomicSave(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	s, err := fs.Create(ctx, "anthropic", "model")
	require.NoError(t, err)
	path := filepath.Join(dir, s.ID+".json")
	s.Messages = []types.Message{{Role: "user", Content: "test"}}
	require.NoError(t, fs.Save(ctx, s))
	_, err = os.Stat(path)
	require.NoError(t, err)
	_, err = os.Stat(path + ".tmp")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestSessionTimestampsUpdatedOnSave(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	s, err := fs.Create(ctx, "openrouter", "gpt-4o")
	require.NoError(t, err)
	before := s.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	s.Messages = []types.Message{{Role: "user", Content: "x"}}
	require.NoError(t, fs.Save(ctx, s))
	require.True(t, s.UpdatedAt.After(before))
}

func TestFileStoreInvalidID(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	_, err = fs.Get(context.Background(), "../escape")
	require.Error(t, err)
}

func TestFileStoreNameRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	s, err := fs.Create(ctx, "anthropic", "model")
	require.NoError(t, err)
	s.Name = "my task"
	require.NoError(t, fs.Save(ctx, s))
	got, err := fs.Get(ctx, s.ID)
	require.NoError(t, err)
	require.Equal(t, "my task", got.Name)
	metas, err := fs.List(ctx)
	require.NoError(t, err)
	require.Len(t, metas, 1)
	require.Equal(t, "my task", metas[0].Name)
}

func TestFileStoreBackwardCompatNoCompactions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	legacy := `{"id":"legacy","created_at":"2026-06-18T00:00:00Z","updated_at":"2026-06-18T00:00:00Z","provider":"anthropic","model":"m","messages":[{"role":"user","content":"hi"}]}`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o644))
	fs, err := New(dir)
	require.NoError(t, err)
	got, err := fs.Get(context.Background(), "legacy")
	require.NoError(t, err)
	require.Len(t, got.Messages, 1)
	require.Nil(t, got.Compactions)
}

func TestFileStoreCompactionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	fs, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	s, err := fs.Create(ctx, "anthropic", "model")
	require.NoError(t, err)
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
	require.NoError(t, fs.Save(ctx, s))
	got, err := fs.Get(ctx, s.ID)
	require.NoError(t, err)
	require.Len(t, got.Compactions, 1)
	require.Equal(t, "summarized old", got.Compactions[0].Summary)
	require.Len(t, got.Messages, 2)
}

func TestFileStoreBackwardCompatNoName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.json")
	legacy := `{"id":"legacy","created_at":"2026-06-18T00:00:00Z","updated_at":"2026-06-18T00:00:00Z","provider":"anthropic","model":"m","messages":[]}`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o644))
	fs, err := New(dir)
	require.NoError(t, err)
	got, err := fs.Get(context.Background(), "legacy")
	require.NoError(t, err)
	require.Empty(t, got.Name)
}

var _ session.Store = (*FileStore)(nil)
