package builtin

import (
	"coding-agent/plugin"
	"coding-agent/plugins/compaction/summarize"
	"coding-agent/plugins/permission/interactive"
	"coding-agent/plugins/permission/script"
	"coding-agent/plugins/prompt/coding"
	"coding-agent/plugins/providers/anthropic"
	"coding-agent/plugins/providers/openrouter"
	"coding-agent/plugins/runner/repl"
	"coding-agent/plugins/session/filestore"
	"coding-agent/plugins/session/memory"
	"coding-agent/plugins/skills"
	spawnrunner "coding-agent/plugins/spawn/runner"
	"coding-agent/plugins/tools/glob"
	"coding-agent/plugins/tools/grep"
	"coding-agent/plugins/tools/listdir"
	"coding-agent/plugins/tools/readfile"
	"coding-agent/plugins/tools/runbash"
	"coding-agent/plugins/tools/strreplace"
	"coding-agent/plugins/tools/task"
	"coding-agent/plugins/tools/writefile"
)

// Default is the single compile-time registry of built-in plugins.
// Add new plugins here.
var Default = []plugin.Plugin{
	// core tools plugins
	readfile.Plugin{},
	writefile.Plugin{},
	strreplace.Plugin{},
	listdir.Plugin{},
	grep.Plugin{},
	glob.Plugin{},
	runbash.Plugin{},

	// permission plugins (script before interactive)
	script.Plugin{},
	interactive.Plugin{},

	// skills plugins
	&skills.Plugin{},

	// provider plugins
	anthropic.Plugin{},
	openrouter.Plugin{},

	// compaction plugins (after providers)
	summarize.Plugin{},

	// spawn plugins (after tools + providers)
	spawnrunner.Plugin{},
	task.Plugin{},

	// session plugins
	filestore.Plugin{},
	memory.Plugin{},

	// other plugins
	coding.Plugin{},
	repl.Plugin{},
}
