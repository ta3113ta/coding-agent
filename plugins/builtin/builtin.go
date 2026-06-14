package builtin

import (
	"coding-agent/plugin"
	"coding-agent/plugins/prompt/coding"
	"coding-agent/plugins/providers/anthropic"
	"coding-agent/plugins/providers/openrouter"
	"coding-agent/plugins/runner/repl"
	"coding-agent/plugins/tools/listdir"
	"coding-agent/plugins/tools/readfile"
	"coding-agent/plugins/tools/runbash"
	"coding-agent/plugins/tools/strreplace"
	"coding-agent/plugins/tools/writefile"
)

// Default is the single compile-time registry of built-in plugins.
// Add new plugins here.
var Default = []plugin.Plugin{
	readfile.Plugin{},
	writefile.Plugin{},
	strreplace.Plugin{},
	listdir.Plugin{},
	runbash.Plugin{},
	anthropic.Plugin{},
	openrouter.Plugin{},
	coding.Plugin{},
	repl.Plugin{},
}
