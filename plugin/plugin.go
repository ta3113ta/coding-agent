package plugin

import (
	"context"

	"coding-agent/config"
	"coding-agent/llm"
	"coding-agent/skills"
	"coding-agent/tools"
)

type App struct {
	Config   config.Config
	Tools    *tools.Registry
	Prompt   string
	Provider llm.Provider
	Runner   Runner
	Skills   *skills.Registry
}

type Plugin interface {
	Name() string
	Register(app *App) error
}

type ToolPlugin interface {
	Tools() []tools.Tool
}

type ProviderPlugin interface {
	ProviderName() config.ProviderName
	NewProvider(cfg config.Config) (llm.Provider, error)
}

type PromptContributor interface {
	SystemPrompt() string
}

// AgentHandle is the minimal agent surface runners need (avoids import cycle with agent package).
type AgentHandle interface {
	Run(ctx context.Context, userInput string) (string, error)
}

type Runner interface {
	Run(ctx context.Context, ag AgentHandle) error
}
