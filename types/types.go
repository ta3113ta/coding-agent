package types

import "encoding/json"

type ToolDefinition struct {
	Name        string
	Description string
	Properties  map[string]any
	Required    []string
}

type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

type Message struct {
	Role       string // "user" | "assistant" | "tool"
	Content    string
	ToolCalls  []ToolCall // assistant only
	ToolCallID string     // tool only
	IsError    bool       // tool only
}

type CompleteRequest struct {
	SystemPrompt string
	Messages     []Message
	Tools        []ToolDefinition
	Model        string
	MaxTokens    int
}

type CompleteResponse struct {
	Text      string
	ToolCalls []ToolCall
}
