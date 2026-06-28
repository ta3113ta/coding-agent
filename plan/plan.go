package plan

import "maps"

import "time"

type Mode string

const (
	ModeAgent Mode = "agent"
	ModePlan  Mode = "plan"
)

type PlanStatus string

const (
	PlanStatusDraft    PlanStatus = "draft"
	PlanStatusApproved PlanStatus = "approved"
)

type TodoStatus string

const (
	TodoPending    TodoStatus = "pending"
	TodoInProgress TodoStatus = "in_progress"
	TodoCompleted  TodoStatus = "completed"
	TodoCancelled  TodoStatus = "cancelled"
)

type TodoItem struct {
	ID      string     `json:"id"`
	Content string     `json:"content"`
	Status  TodoStatus `json:"status"`
}

type Plan struct {
	Title     string     `json:"title"`
	Overview  string     `json:"overview"`
	Body      string     `json:"body"`
	Status    PlanStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

var planModeTools = map[string]bool{
	"read_file":   true,
	"list_dir":    true,
	"grep":        true,
	"glob":        true,
	"todo_write":  true,
	"create_plan": true,
}

func AllowedToolsInPlanMode() map[string]bool {
	out := make(map[string]bool, len(planModeTools))
	maps.Copy(out, planModeTools)
	return out
}

func IsAllowedInPlanMode(name string) bool {
	return planModeTools[name]
}

func ValidTodoStatus(s TodoStatus) bool {
	switch s {
	case TodoPending, TodoInProgress, TodoCompleted, TodoCancelled:
		return true
	default:
		return false
	}
}

const PlanModePromptSuffix = `You are in PLAN MODE (read-only).

Rules:
- Use read_file, list_dir, grep, and glob to research the codebase. Do not guess file contents.
- Do not attempt to edit files or run shell commands.
- When you have enough context, call create_plan with a structured markdown plan.
- Use todo_write to track investigation steps if helpful.
- Wait for the user to approve the plan before implementation begins.`
