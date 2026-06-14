package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, dir, name, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func skillContent(name, desc string) string {
	return `---
name: ` + name + `
description: ` + desc + `
---
# ` + name + `
`
}

func TestDiscoverDedupPriority(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	workspace := filepath.Join(tmp, "workspace")

	projectSkills := filepath.Join(workspace, ".cursor", "skills")
	personalSkills := filepath.Join(home, ".cursor", "skills")

	writeSkill(t, projectSkills, "shared-skill", skillContent("shared-skill", "project version"))
	writeSkill(t, personalSkills, "shared-skill", skillContent("shared-skill", "personal version"))
	writeSkill(t, personalSkills, "personal-only", skillContent("personal-only", "personal only"))

	bundled := []Skill{{
		Name:        "shared-skill",
		Description: "bundled version",
		Path:        "/bundled/shared-skill/SKILL.md",
		Source:      SourceBundled,
	}, {
		Name:        "bundled-only",
		Description: "bundled only",
		Path:        "/bundled/bundled-only/SKILL.md",
		Source:      SourceBundled,
	}}

	reg, err := Discover(DiscoverOptions{
		Workspace:       workspace,
		HomeDir:         home,
		IncludePersonal: true,
		Bundled:         bundled,
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	shared, ok := reg.Get("shared-skill")
	if !ok {
		t.Fatal("shared-skill not found")
	}
	if shared.Source != SourceProject {
		t.Fatalf("shared-skill source = %q, want project", shared.Source)
	}
	if shared.Description != "project version" {
		t.Fatalf("shared-skill description = %q", shared.Description)
	}

	personal, ok := reg.Get("personal-only")
	if !ok || personal.Source != SourcePersonal {
		t.Fatalf("personal-only: %+v", personal)
	}

	bonly, ok := reg.Get("bundled-only")
	if !ok || bonly.Source != SourceBundled {
		t.Fatalf("bundled-only: %+v", bonly)
	}
}

func TestDiscoverMissingDirs(t *testing.T) {
	reg, err := Discover(DiscoverOptions{
		Workspace:       filepath.Join(t.TempDir(), "empty"),
		HomeDir:         filepath.Join(t.TempDir(), "home"),
		IncludePersonal: true,
	})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if reg.Len() != 0 {
		t.Fatalf("expected empty registry, got %d", reg.Len())
	}
}

func TestRegistryIndexPrompt(t *testing.T) {
	reg := NewRegistry([]Skill{{
		Name:        "foo",
		Description: "Does foo things.",
		Path:        "/path/foo/SKILL.md",
		Source:      SourceProject,
	}})
	prompt := reg.IndexPrompt()
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "Available Skills") || !strings.Contains(prompt, "foo") {
		t.Fatalf("prompt missing content: %q", prompt)
	}
}

func TestRegistryIndexPromptEmpty(t *testing.T) {
	reg := NewRegistry(nil)
	if reg.IndexPrompt() != "" {
		t.Fatal("expected empty prompt for empty registry")
	}
}
