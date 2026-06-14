package skills

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"coding-agent/plugin"
	"coding-agent/skills"
)

//go:embed builtin/*/SKILL.md
var builtinFS embed.FS

type Plugin struct {
	bundledDir string
}

func (Plugin) Name() string { return "skills" }

func (p *Plugin) Register(app *plugin.App) error {
	bundled, err := p.loadBundled()
	if err != nil {
		return fmt.Errorf("bundled skills: %w", err)
	}

	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	registry, err := skills.Discover(skills.DiscoverOptions{
		Workspace:       workspace,
		HomeDir:         home,
		IncludePersonal: app.Config.SkillsEnablePersonal,
		Bundled:         bundled,
	})
	if err != nil {
		return err
	}

	app.Skills = registry
	if prompt := registry.IndexPrompt(); prompt != "" {
		plugin.AppendPrompt(app, prompt)
	}
	return nil
}

func (p *Plugin) loadBundled() ([]skills.Skill, error) {
	if p.bundledDir == "" {
		dir, err := extractBundledSkills()
		if err != nil {
			return nil, err
		}
		p.bundledDir = dir
	}
	return skills.ScanDir(p.bundledDir, skills.SourceBundled)
}

func extractBundledSkills() (string, error) {
	root, err := os.MkdirTemp("", "code-agent-skills-")
	if err != nil {
		return "", err
	}

	entries, err := fs.ReadDir(builtinFS, "builtin")
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		data, err := builtinFS.ReadFile(filepath.Join("builtin", name, "SKILL.md"))
		if err != nil {
			return "", fmt.Errorf("read bundled %s: %w", name, err)
		}
		destDir := filepath.Join(root, name)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return "", err
		}
		destPath := filepath.Join(destDir, "SKILL.md")
		if err := os.WriteFile(destPath, data, 0o644); err != nil {
			return "", err
		}
	}
	return root, nil
}

var _ plugin.Plugin = (*Plugin)(nil)
