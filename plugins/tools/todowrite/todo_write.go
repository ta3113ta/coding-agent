package todowrite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"coding-agent/plan"
	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type Tool struct {
	state *plan.SessionState
}

func (Tool) Name() string { return "todo_write" }

func (Tool) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name: "todo_write",
		Description: "Create and manage a structured task list for the current session. " +
			"Use for multi-step tasks to track progress. " +
			"Only one task should be in_progress at a time.",
		Properties: map[string]any{
			"merge": map[string]any{
				"type":        "boolean",
				"description": "If true, merge todos by id into the existing list; if false, replace the list",
			},
			"todos": map[string]any{
				"type":        "array",
				"description": "Array of todo items to create or update",
				"minItems":    2,
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type":        "string",
							"description": "Unique identifier for the todo item",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "Description of the todo item",
						},
						"status": map[string]any{
							"type":        "string",
							"enum":        []string{"pending", "in_progress", "completed", "cancelled"},
							"description": "Current status of the todo item",
						},
					},
					"required": []string{"id", "content", "status"},
				},
			},
		},
		Required: []string{"merge", "todos"},
	}
}

func (t Tool) Execute(_ context.Context, input json.RawMessage) (string, error) {
	if t.state == nil {
		return "", errors.New("todo tracking is disabled")
	}

	var args struct {
		Merge bool `json:"merge"`
		Todos []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"todos"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if len(args.Todos) < 2 {
		return "", errors.New("todos must contain at least 2 items")
	}

	incoming := make([]plan.TodoItem, 0, len(args.Todos))
	for _, item := range args.Todos {
		id := strings.TrimSpace(item.ID)
		content := strings.TrimSpace(item.Content)
		if id == "" || content == "" {
			return "", errors.New("each todo requires non-empty id and content")
		}
		status := plan.TodoStatus(item.Status)
		if !plan.ValidTodoStatus(status) {
			return "", fmt.Errorf("invalid status %q for todo %q", item.Status, id)
		}
		incoming = append(incoming, plan.TodoItem{
			ID:      id,
			Content: content,
			Status:  status,
		})
	}

	if args.Merge {
		t.state.MergeTodos(incoming)
	} else {
		t.state.SetTodos(incoming)
	}

	return formatTodos(t.state.Todos()), nil
}

func formatTodos(todos []plan.TodoItem) string {
	if len(todos) == 0 {
		return "Todos updated (empty list)."
	}
	var b strings.Builder
	b.WriteString("Todos updated:\n")
	for _, todo := range todos {
		fmt.Fprintf(&b, "- [%s] %s: %s\n", todo.Status, todo.ID, todo.Content)
	}
	return strings.TrimRight(b.String(), "\n")
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/todowrite" }

func (Plugin) Register(app *plugin.App) error {
	if !app.Config.PlanEnabled || app.PlanState == nil {
		return nil
	}
	plugin.RegisterTools(app, Tool{state: app.PlanState})
	return nil
}

var _ tools.Tool = Tool{}
