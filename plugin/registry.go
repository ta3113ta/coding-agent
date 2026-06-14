package plugin

import (
	"fmt"
	"strings"

	"coding-agent/config"
	"coding-agent/llm"
	"coding-agent/tools"
)

func LoadConfigFromEnv() config.Config {
	return config.LoadFromEnv()
}

func Bootstrap(cfg config.Config, plugins ...Plugin) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	app := &App{
		Config: cfg,
		Tools:  tools.NewRegistry(),
	}

	for _, p := range plugins {
		if err := p.Register(app); err != nil {
			return nil, fmt.Errorf("plugin %s: %w", p.Name(), err)
		}
	}

	if app.Provider == nil {
		provider, err := llm.NewProvider(cfg)
		if err != nil {
			return nil, err
		}
		app.Provider = provider
	}

	if app.Prompt == "" {
		return nil, fmt.Errorf("ไม่มี prompt plugin ลงทะเบียน")
	}

	if app.Runner == nil {
		return nil, fmt.Errorf("ไม่มี runner plugin ลงทะเบียน")
	}

	return app, nil
}

func RegisterTools(app *App, ts ...tools.Tool) {
	for _, t := range ts {
		app.Tools.Register(t)
	}
}

func RegisterProvider(app *App, p ProviderPlugin) {
	llm.RegisterProvider(p.ProviderName(), func(cfg config.Config) (llm.Provider, error) {
		return p.NewProvider(cfg)
	})
}

func AppendPrompt(app *App, prompt string) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return
	}
	if app.Prompt == "" {
		app.Prompt = prompt
		return
	}
	app.Prompt = app.Prompt + "\n\n" + prompt
}
