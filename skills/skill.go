package skills

import (
	"fmt"
	"strings"
)

type Source string

const (
	SourceProject  Source = "project"
	SourcePersonal Source = "personal"
	SourceBundled  Source = "bundled"
)

type Skill struct {
	Name                   string
	Description            string
	Path                   string
	Source                 Source
	DisableModelInvocation bool
}

type Registry struct {
	ordered []Skill
	byName  map[string]Skill
}

func NewRegistry(skills []Skill) *Registry {
	byName := make(map[string]Skill, len(skills))
	ordered := make([]Skill, 0, len(skills))
	for _, s := range skills {
		if _, exists := byName[s.Name]; exists {
			continue
		}
		byName[s.Name] = s
		ordered = append(ordered, s)
	}
	return &Registry{ordered: ordered, byName: byName}
}

func (r *Registry) Skills() []Skill {
	if r == nil {
		return nil
	}
	out := make([]Skill, len(r.ordered))
	copy(out, r.ordered)
	return out
}

func (r *Registry) Get(name string) (Skill, bool) {
	if r == nil {
		return Skill{}, false
	}
	s, ok := r.byName[name]
	return s, ok
}

func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.ordered)
}

const usageInstructions = `## Available Skills

When the user's task matches a skill description:
1. Read SKILL.md with read_file before working.
2. Follow the instructions in the skill immediately.
3. If a skill has disable-model-invocation, do not load it unless the user explicitly names the skill.`

func (r *Registry) IndexPrompt() string {
	if r == nil || len(r.ordered) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(usageInstructions)
	b.WriteString("\n\n<available_skills>\n")
	for _, s := range r.ordered {
		line := fmt.Sprintf("- %s (%s): %s", s.Name, s.Path, s.Description)
		if s.DisableModelInvocation {
			line += " [disable-model-invocation]"
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("</available_skills>")
	return b.String()
}
