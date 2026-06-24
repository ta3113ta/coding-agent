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
	"read_file": true,
	"list_dir":  true,
}

var gatedTools = map[string]bool{
	"run_bash":    true,
	"write_file":  true,
	"str_replace": true,
	"task":        true,
}

type Hook struct {
	in  io.Reader
	out io.Writer
}

func NewHook(in io.Reader, out io.Writer) *Hook {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	return &Hook{in: in, out: out}
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

	approved, err := h.prompt(req)
	if err != nil {
		return permission.Result{}, err
	}
	if approved {
		return permission.Result{Decision: permission.Allow}, nil
	}
	return permission.Result{
		Decision: permission.Deny,
		Message:  fmt.Sprintf("user denied %s", req.ToolName),
	}, nil
}

func (h *Hook) prompt(req permission.ToolUseRequest) (bool, error) {
	fmt.Fprintf(h.out, "\n⚠️  Tool permission: %s\n", req.ToolName)
	if req.AskHint != "" {
		fmt.Fprintf(h.out, "    %s\n", req.AskHint)
	}
	if len(req.Input) > 0 {
		fmt.Fprintf(h.out, "    input: %s\n", summarizeInput(req.Input))
	}
	fmt.Fprint(h.out, "Allow? [y/N]: ")

	reader := bufio.NewReader(h.in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
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
	plugin.RegisterPermissionHook(app, NewHook(os.Stdin, os.Stdout))
	return nil
}

var _ permission.Hook = (*Hook)(nil)
