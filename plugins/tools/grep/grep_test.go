package grep

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireRg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not on PATH")
	}
}

func setupFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.go"), []byte("package main\n\nfunc Hello() {}\nfunc World() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bar.txt"), []byte("Hello world\n"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "baz.go"), []byte("package sub\n\nfunc hello() {}\n"), 0o644))
	return dir
}

func TestGrep_ContentMode(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Grep{}
	input, _ := json.Marshal(map[string]any{
		"pattern": `func Hello`,
		"path":    dir,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "foo.go")
	require.Contains(t, out, "func Hello")
}

func TestGrep_FilesWithMatches(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Grep{}
	input, _ := json.Marshal(map[string]any{
		"pattern":     `hello`,
		"path":        dir,
		"output_mode": "files_with_matches",
		"-i":          true,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "foo.go")
	require.Contains(t, out, "bar.txt")
}

func TestGrep_CountMode(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Grep{}
	input, _ := json.Marshal(map[string]any{
		"pattern":     `func`,
		"path":        dir,
		"glob":        "*.go",
		"output_mode": "count",
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "foo.go")
}

func TestGrep_HeadLimit(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Grep{}
	input, _ := json.Marshal(map[string]any{
		"pattern":     `.`,
		"path":        dir,
		"output_mode": "files_with_matches",
		"head_limit":  1,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "head_limit")
}

func TestGrep_NoMatches(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Grep{}
	input, _ := json.Marshal(map[string]any{
		"pattern": `zzznotfoundzzz`,
		"path":    dir,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "(No results found)", out)
}

func TestGrep_MissingPattern(t *testing.T) {
	tool := Grep{}
	input, _ := json.Marshal(map[string]any{"path": "."})
	_, err := tool.Execute(context.Background(), input)
	require.Error(t, err)
}
