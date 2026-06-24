package strreplace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStrReplace_UniqueMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "package main\n\nfunc foo() {\n    return 1\n}\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":        path,
		"old_string":  "return 1",
		"new_string":  "return 2",
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "Replaced 1 occurrence")

	got, _ := os.ReadFile(path)
	require.Contains(t, string(got), "return 2")
}

func TestStrReplace_ZeroMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "missing",
		"new_string": "x",
	})
	_, err := tool.Execute(context.Background(), input)
	require.Error(t, err)
	require.ErrorContains(t, err, "not found")
}

func TestStrReplace_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "foo\nfoo\nfoo\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "foo",
		"new_string": "bar",
	})
	_, err := tool.Execute(context.Background(), input)
	require.Error(t, err)
	require.ErrorContains(t, err, "matched 3 times")
	require.ErrorContains(t, err, "[1 2 3]")
}

func TestStrReplace_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "foo\nfoo\nfoo\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":         path,
		"old_string":   "foo",
		"new_string":   "bar",
		"replace_all":  true,
	})
	out, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)
	require.Contains(t, out, "Replaced 3 occurrences")

	got, _ := os.ReadFile(path)
	require.Equal(t, "bar\nbar\nbar\n", string(got))
}

func TestStrReplace_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "before\nREMOVE\nafter\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "REMOVE\n",
		"new_string": "",
	})
	_, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	got, _ := os.ReadFile(path)
	require.Equal(t, "before\nafter\n", string(got))
}

func TestStrReplace_TabNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "func foo() {\n\treturn 1\n}\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "    return 1",
		"new_string": "    return 2",
	})
	_, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	got, _ := os.ReadFile(path)
	require.True(t, strings.Contains(string(got), "\treturn 2") || strings.Contains(string(got), "    return 2"))
}

func TestStrReplace_CRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "line1\r\nline2\r\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "line2",
		"new_string": "LINE2",
	})
	_, err := tool.Execute(context.Background(), input)
	require.NoError(t, err)

	got, _ := os.ReadFile(path)
	require.Contains(t, string(got), "\r\n")
	require.Equal(t, "line1\r\nLINE2\r\n", string(got))
}

func TestStrReplace_EmptyOldString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	require.NoError(t, os.WriteFile(path, []byte("x\n"), 0o644))

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "",
		"new_string": "y",
	})
	_, err := tool.Execute(context.Background(), input)
	require.ErrorContains(t, err, "old_string must not be empty")
}
