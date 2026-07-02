package interactive

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"coding-agent/permission"
	"coding-agent/plugin"
)

var autoAllowTools = map[string]bool{
	"read_file":   true,
	"list_dir":    true,
	"grep":        true,
	"glob":        true,
	"write_file":  true,
	"str_replace": true,
}

var gatedTools = map[string]bool{
	"run_bash": true,
	"task":     true,
}

type Hook struct {
	in    io.Reader
	out   io.Writer
	rules *permission.SessionRules
}

func NewHook(in io.Reader, out io.Writer, rules *permission.SessionRules) *Hook {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	if rules == nil {
		rules = permission.NewSessionRules()
	}
	return &Hook{in: in, out: out, rules: rules}
}

func (h *Hook) BeforeToolUse(ctx context.Context, req permission.ToolUseRequest) (permission.Result, error) {
	_ = ctx
	if autoAllowTools[req.ToolName] {
		return permission.Result{Decision: permission.Allow}, nil
	}

	needsPrompt := gatedTools[req.ToolName] || req.AskHint != "" || !autoAllowTools[req.ToolName]
	if !needsPrompt {
		return permission.Result{Decision: permission.Allow}, nil
	}

	if h.rules.Allows(req.ToolName) {
		return permission.Result{Decision: permission.Allow}, nil
	}

	choice, err := h.prompt(req)
	if err != nil {
		return permission.Result{}, err
	}
	switch choice {
	case choiceOnce:
		return permission.Result{Decision: permission.Allow}, nil
	case choiceAlwaysTool:
		h.rules.AllowTool(req.ToolName)
		return permission.Result{Decision: permission.Allow}, nil
	case choiceAlwaysAll:
		h.rules.AllowAll()
		return permission.Result{Decision: permission.Allow}, nil
	default:
		return permission.Result{
			Decision: permission.Deny,
			Message:  fmt.Sprintf("user denied %s", req.ToolName),
		}, nil
	}
}

type promptChoice int

const (
	choiceDeny promptChoice = iota
	choiceOnce
	choiceAlwaysTool
	choiceAlwaysAll
)

func (h *Hook) prompt(req permission.ToolUseRequest) (promptChoice, error) {
	fmt.Fprintf(h.out, "\n⚠️  Tool permission: %s\n", req.ToolName)
	if req.AskHint != "" {
		fmt.Fprintf(h.out, "    %s\n", req.AskHint)
	}
	if len(req.Input) > 0 {
		fmt.Fprintf(h.out, "    input: %s\n", summarizeInput(req.Input))
	}
	fmt.Fprint(h.out, "Allow? [y]es / [a]lways this tool / [A]ll tools / [n]o: ")

	reader := bufio.NewReader(h.in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return choiceDeny, err
	}
	switch strings.TrimSpace(line) {
	case "y", "yes", "Y", "Yes":
		return choiceOnce, nil
	case "a", "always":
		return choiceAlwaysTool, nil
	case "A":
		return choiceAlwaysAll, nil
	default:
		return choiceDeny, nil
	}
}

func summarizeInput(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

type Plugin struct{}

func (Plugin) Name() string { return "permission/interactive" }

func (Plugin) Register(app *plugin.App) error {
	if !app.Config.PermissionEnabled {
		return nil
	}
	if app.Permission == nil {
		app.Permission = permission.NewChain()
	}
	hook := NewHook(os.Stdin, os.Stdout, app.Permission.SessionRules())
	plugin.RegisterPermissionHook(app, hook)
	return nil
}

var _ permission.Hook = (*Hook)(nil)
