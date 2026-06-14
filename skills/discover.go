package skills

import (
	"fmt"
	"os"
	"path/filepath"
)

type DiscoverOptions struct {
	Workspace       string
	HomeDir         string
	IncludePersonal bool
	Bundled         []Skill
}

func Discover(opts DiscoverOptions) (*Registry, error) {
	byName := make(map[string]Skill)

	merge := func(skills []Skill) {
		for _, s := range skills {
			byName[s.Name] = s
		}
	}

	merge(opts.Bundled)

	if opts.IncludePersonal && opts.HomeDir != "" {
		personal, err := scanSkillsDir(filepath.Join(opts.HomeDir, ".cursor", "skills"), SourcePersonal)
		if err != nil {
			return nil, fmt.Errorf("personal skills: %w", err)
		}
		merge(personal)
	}

	if opts.Workspace != "" {
		project, err := scanSkillsDir(filepath.Join(opts.Workspace, ".cursor", "skills"), SourceProject)
		if err != nil {
			return nil, fmt.Errorf("project skills: %w", err)
		}
		merge(project)
	}

	var ordered []Skill
	seen := make(map[string]bool)

	appendWinners := func(skills []Skill) {
		for _, s := range skills {
			if seen[s.Name] {
				continue
			}
			winner, ok := byName[s.Name]
			if !ok || winner.Path != s.Path {
				continue
			}
			ordered = append(ordered, winner)
			seen[s.Name] = true
		}
	}

	if opts.Workspace != "" {
		project, _ := scanSkillsDir(filepath.Join(opts.Workspace, ".cursor", "skills"), SourceProject)
		appendWinners(project)
	}
	if opts.IncludePersonal && opts.HomeDir != "" {
		personal, _ := scanSkillsDir(filepath.Join(opts.HomeDir, ".cursor", "skills"), SourcePersonal)
		appendWinners(personal)
	}
	appendWinners(opts.Bundled)

	return NewRegistry(ordered), nil
}

func ScanDir(root string, source Source) ([]Skill, error) {
	return scanSkillsDir(root, source)
}

func scanSkillsDir(root string, source Source) ([]Skill, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var skills []Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirName := entry.Name()
		skillPath := filepath.Join(root, dirName, "SKILL.md")
		data, err := os.ReadFile(skillPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("%s: %w", skillPath, err)
		}
		absPath, err := filepath.Abs(skillPath)
		if err != nil {
			absPath = skillPath
		}
		skill, err := ParseFile(absPath, data, source, dirName)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", skillPath, err)
		}
		skills = append(skills, skill)
	}
	return skills, nil
}
