package plugin

import (
	"context"

	"coding-agent/config"
	"coding-agent/llm"
	"coding-agent/permission"
	"coding-agent/session"
	"coding-agent/skills"
	"coding-agent/tools"
	"coding-agent/types"
)

type App struct {
	Config       config.Config
	Tools        *tools.Registry
	Prompt       string
	Provider     llm.Provider
	Runner       Runner
	Skills       *skills.Registry
	SessionStore session.Store
	Permission   *permission.Chain
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
	Run(ctx context.Context, userInput string, onStream func(types.StreamEvent)) (string, error)
	SessionManager
}

type SessionManager interface {
	ResetSession(ctx context.Context) error
	ResumeSession(ctx context.Context, id string) error
	ListSessions(ctx context.Context) ([]session.Meta, error)
	CurrentSessionID() string
	SetSessionName(ctx context.Context, name string) error
	CurrentSessionName() string
	SessionLabel() string
}

type Runner interface {
	Run(ctx context.Context, ag AgentHandle) error
}
