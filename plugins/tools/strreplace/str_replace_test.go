package strreplace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStrReplace_UniqueMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "package main\n\nfunc foo() {\n    return 1\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":        path,
		"old_string":  "return 1",
		"new_string":  "return 2",
	})
	out, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Replaced 1 occurrence") {
		t.Fatalf("unexpected output: %s", out)
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "return 2") {
		t.Fatalf("file not updated: %s", got)
	}
}

func TestStrReplace_ZeroMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "missing",
		"new_string": "x",
	})
	_, err := tool.Execute(input)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStrReplace_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "foo\nfoo\nfoo\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "foo",
		"new_string": "bar",
	})
	_, err := tool.Execute(input)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "matched 3 times") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "[1 2 3]") {
		t.Fatalf("expected line numbers: %v", err)
	}
}

func TestStrReplace_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "foo\nfoo\nfoo\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":         path,
		"old_string":   "foo",
		"new_string":   "bar",
		"replace_all":  true,
	})
	out, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Replaced 3 occurrences") {
		t.Fatalf("unexpected output: %s", out)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "bar\nbar\nbar\n" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestStrReplace_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "before\nREMOVE\nafter\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "REMOVE\n",
		"new_string": "",
	})
	_, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "before\nafter\n" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestStrReplace_TabNormalization(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "func foo() {\n\treturn 1\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "    return 1",
		"new_string": "    return 2",
	})
	_, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "\treturn 2") && !strings.Contains(string(got), "    return 2") {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestStrReplace_CRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "line1\r\nline2\r\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "line2",
		"new_string": "LINE2",
	})
	_, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "\r\n") {
		t.Fatalf("CRLF not preserved: %q", got)
	}
	if string(got) != "line1\r\nLINE2\r\n" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestStrReplace_EmptyOldString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := StrReplace{}
	input, _ := json.Marshal(map[string]any{
		"path":       path,
		"old_string": "",
		"new_string": "y",
	})
	_, err := tool.Execute(input)
	if err == nil || !strings.Contains(err.Error(), "old_string ต้องไม่ว่าง") {
		t.Fatalf("expected empty old_string error, got: %v", err)
	}
}
