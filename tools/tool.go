package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"coding-agent/types"
)

// Tool is the interface every tool must implement.
// It separates the "definition" (schema sent to the LLM) from the actual work (Execute).
type Tool interface {
	// Name is what the LLM uses to invoke the tool.
	Name() string
	// Definition is the schema sent to the LLM describing what the tool does and which params it accepts.
	Definition() types.ToolDefinition
	// Execute receives raw JSON input from the LLM and returns a string result.
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

// Registry holds all tools and dispatches by name.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry(ts ...Tool) *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	for _, t := range ts {
		r.Register(t)
	}
	return r
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for name := range r.tools {
		out = append(out, name)
	}
	return out
}

// Definitions returns all schemas in a format the provider can use.
func (r *Registry) Definitions() []types.ToolDefinition {
	out := make([]types.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Definition())
	}
	return out
}

// Dispatch finds a tool by name and calls Execute.
func (r *Registry) Dispatch(ctx context.Context, name string, input json.RawMessage) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, input)
}
