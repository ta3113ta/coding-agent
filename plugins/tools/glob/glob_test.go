package glob

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.go"), []byte("package main\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bar.txt"), []byte("hello\n"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "subdir", "baz.go"), []byte("package sub\n"), 0o644))
	return dir
}

func TestGlob_GoPattern(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Glob{}
	input, _ := json.Marshal(map[string]any{
		"glob_pattern":     "*.go",
		"target_directory": dir,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "foo.go")
	require.Contains(t, out, "baz.go")
	require.NotContains(t, out, "bar.txt")
}

func TestGlob_RecursivePattern(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Glob{}
	input, _ := json.Marshal(map[string]any{
		"glob_pattern":     "**/*.go",
		"target_directory": dir,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "foo.go")
	require.Contains(t, out, "baz.go")
}

func TestGlob_NoMatches(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Glob{}
	input, _ := json.Marshal(map[string]any{
		"glob_pattern":     "*.rs",
		"target_directory": dir,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "(no files found)", out)
}

func TestNormalizePattern(t *testing.T) {
	require.Equal(t, "**/*.go", normalizePattern("*.go"))
	require.Equal(t, "**/*.go", normalizePattern("**/*.go"))
	require.Equal(t, "**/foo", normalizePattern("foo"))
}

func TestGlob_HeadLimit(t *testing.T) {
	requireRg(t)
	dir := setupFixture(t)
	tool := Glob{}
	input, _ := json.Marshal(map[string]any{
		"glob_pattern":     "*.go",
		"target_directory": dir,
		"head_limit":       1,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 2) // 1 file + truncation notice
	require.Contains(t, out, "mtime")
}

func TestGlob_MissingPattern(t *testing.T) {
	tool := Glob{}
	input, _ := json.Marshal(map[string]any{"target_directory": "."})
	_, err := tool.Execute(context.Background(), input)
	require.Error(t, err)
}
