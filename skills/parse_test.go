package skills

import (
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	content := `---
name: code-review
description: Review code for quality and security.
disable-model-invocation: true
---
# Code Review

Follow these steps.
`
	meta, body, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if meta.Name != "code-review" {
		t.Fatalf("name = %q", meta.Name)
	}
	if meta.Description != "Review code for quality and security." {
		t.Fatalf("description = %q", meta.Description)
	}
	if !meta.DisableModelInvocation {
		t.Fatal("expected disable-model-invocation true")
	}
	if !strings.Contains(body, "# Code Review") {
		t.Fatalf("body = %q", body)
	}
}

func TestParseMissingFrontmatter(t *testing.T) {
	_, _, err := Parse([]byte("# No frontmatter"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseMissingName(t *testing.T) {
	content := `---
description: Do something.
---
# Body
`
	_, _, err := Parse([]byte(content))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseMissingDescription(t *testing.T) {
	content := `---
name: foo
---
# Body
`
	_, _, err := Parse([]byte(content))
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestParseInvalidName(t *testing.T) {
	content := `---
name: Invalid_Name
description: Bad name format.
---
`
	_, _, err := Parse([]byte(content))
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestParseFile(t *testing.T) {
	content := `---
name: commit-message
description: Create git commit messages.
---
# Commit
`
	skill, err := ParseFile("/tmp/commit-message/SKILL.md", []byte(content), SourceBundled, "commit-message")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if skill.Path != "/tmp/commit-message/SKILL.md" {
		t.Fatalf("path = %q", skill.Path)
	}
	if skill.Source != SourceBundled {
		t.Fatalf("source = %q", skill.Source)
	}
}
