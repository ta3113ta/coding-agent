package createplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"coding-agent/plan"
	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type Tool struct {
	state *plan.SessionState
}

func (Tool) Name() string { return "create_plan" }

func (Tool) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name: "create_plan",
		Description: "Create a structured implementation plan after research in plan mode. " +
			"The plan is saved as a draft for user review and approval before implementation.",
		Properties: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "Short 3-4 word title for the plan",
			},
			"overview": map[string]any{
				"type":        "string",
				"description": "One or two sentence summary of what will be accomplished",
			},
			"plan": map[string]any{
				"type":        "string",
				"description": "Detailed markdown plan with steps, files to change, and test plan",
			},
		},
		Required: []string{"name", "overview", "plan"},
	}
}

func (t Tool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	if t.state == nil {
		return "", errors.New("plan mode is disabled")
	}
	if t.state.Mode() != plan.ModePlan {
		return "", errors.New("create_plan is only available in plan mode; use /plan to switch")
	}

	var args struct {
		Name     string `json:"name"`
		Overview string `json:"overview"`
		Plan     string `json:"plan"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	name := strings.TrimSpace(args.Name)
	overview := strings.TrimSpace(args.Overview)
	body := strings.TrimSpace(args.Plan)
	if name == "" || overview == "" || body == "" {
		return "", errors.New("name, overview, and plan are required")
	}

	sessionID := strings.TrimSpace(t.state.SessionID())
	if sessionID == "" {
		return "", errors.New("no active session")
	}

	p := t.state.CreateDraftPlan(name, overview, body)

	path, err := writePlanFile(sessionID, p)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Plan %q saved as draft.\nFile: %s\nStatus: %s\nAsk the user to review and run /approve, then /agent to implement.",
		p.Title, path, p.Status), nil
}

func writePlanFile(sessionID string, p *plan.Plan) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	dir := filepath.Join(cwd, ".coding-agent", "plans")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir plans: %w", err)
	}
	path := filepath.Join(dir, sessionID+".md")

	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", p.Title)
	fmt.Fprintf(&b, "%s\n\n", p.Overview)
	b.WriteString(p.Body)
	b.WriteByte('\n')

	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write plan file: %w", err)
	}
	return path, nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/createplan" }

func (Plugin) Register(app *plugin.App) error {
	if !app.Config.PlanEnabled || app.PlanState == nil {
		return nil
	}
	plugin.RegisterTools(app, Tool{state: app.PlanState})
	return nil
}

var _ tools.Tool = Tool{}
