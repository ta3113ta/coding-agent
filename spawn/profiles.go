package spawn

import "fmt"

type Profile struct {
	Type         Type
	SystemSuffix string
	Tools        []string
	AllowAll     bool
	Exclude      []string
}

var profiles = map[Type]Profile{
	TypeExplore: {
		Type: TypeExplore,
		SystemSuffix: "You are a read-only codebase exploration sub-agent. " +
			"Use read_file and list_dir to investigate. Do not modify files. " +
			"Return findings concisely.",
		Tools: []string{"read_file", "list_dir"},
	},
	TypeShell: {
		Type: TypeShell,
		SystemSuffix: "You are a shell command execution sub-agent. " +
			"Use run_bash to execute commands. Return stdout/stderr summary.",
		Tools: []string{"run_bash"},
	},
	TypeGeneralPurpose: {
		Type: TypeGeneralPurpose,
		SystemSuffix: "You are a general-purpose sub-agent handling an isolated sub-task. " +
			"Complete the task autonomously and return a concise final summary.",
		AllowAll: true,
		Exclude:  []string{"task"},
	},
}

func ProfileFor(t Type) (Profile, error) {
	p, ok := profiles[t]
	if !ok {
		return Profile{}, fmt.Errorf("unknown subagent_type: %q (use generalPurpose, explore, or shell)", t)
	}
	return p, nil
}

func AllowedTools(p Profile, all []string) map[string]bool {
	allowed := make(map[string]bool)
	if p.AllowAll {
		excluded := make(map[string]bool, len(p.Exclude))
		for _, name := range p.Exclude {
			excluded[name] = true
		}
		for _, name := range all {
			if !excluded[name] {
				allowed[name] = true
			}
		}
		return allowed
	}
	for _, name := range p.Tools {
		allowed[name] = true
	}
	return allowed
}

func Types() []Type {
	return []Type{TypeGeneralPurpose, TypeExplore, TypeShell}
}
