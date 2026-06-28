package runner

import (
	"context"
	"fmt"
	"strings"

	"coding-agent/agent"
	"coding-agent/llm"
	"coding-agent/permission"
	"coding-agent/plugin"
	"coding-agent/plugins/session/memory"
	"coding-agent/spawn"
	"coding-agent/tools"
	"coding-agent/types"
)

type Runner struct {
	provider     llm.Provider
	basePrompt   string
	model        string
	promptCache  types.PromptCacheConfig
	providerName string
	tools        *tools.Registry
	permission   *permission.Chain
	maxTurns     int
}

func (r *Runner) Run(ctx context.Context, req spawn.Request) (spawn.Result, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return spawn.Result{}, fmt.Errorf("prompt is required")
	}

	profile, err := spawn.ProfileFor(req.Type)
	if err != nil {
		return spawn.Result{}, err
	}

	allowed := spawn.AllowedTools(profile, r.tools.Names())
	registry := tools.Filter(r.tools, allowed)
	if len(registry.Definitions()) == 0 {
		return spawn.Result{}, fmt.Errorf("no tools available for subagent_type %q", req.Type)
	}

	sysPrompt := r.basePrompt
	if profile.SystemSuffix != "" {
		sysPrompt = r.basePrompt + "\n\n" + profile.SystemSuffix
	}

	store := memory.New()
	ag, err := agent.New(
		r.provider,
		registry,
		r.model,
		sysPrompt,
		r.promptCache,
		false,
		store,
		r.providerName,
		r.permission,
		nil,
		nil,
		false,
	)
	if err != nil {
		return spawn.Result{}, fmt.Errorf("child agent: %w", err)
	}
	if err := ag.InitNewSession(ctx); err != nil {
		return spawn.Result{}, fmt.Errorf("child session: %w", err)
	}

	text, err := ag.RunSubtask(ctx, req.Prompt, r.maxTurns)
	if err != nil {
		return spawn.Result{}, err
	}
	return spawn.Result{Text: text}, nil
}

type Plugin struct{}

func (Plugin) Name() string { return "spawn/runner" }

func (Plugin) Register(app *plugin.App) error {
	if !app.Config.SpawnEnabled {
		return nil
	}

	provider := app.Provider
	if provider == nil {
		var err error
		provider, err = llm.NewProvider(app.Config)
		if err != nil {
			return fmt.Errorf("spawn runner: %w", err)
		}
	}

	plugin.RegisterSpawner(app, &Runner{
		provider:     provider,
		basePrompt:   app.Prompt,
		model:        app.Config.Model(),
		promptCache:  app.Config.PromptCache(),
		providerName: string(app.Config.Provider),
		tools:        app.Tools,
		permission:   app.Permission,
		maxTurns:     app.Config.SpawnMaxTurns,
	})
	return nil
}

var _ spawn.Runner = (*Runner)(nil)
