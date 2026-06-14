package config

import (
	"fmt"
	"os"
	"strings"
)

type ProviderName string

const (
	ProviderAnthropic  ProviderName = "anthropic"
	ProviderOpenRouter ProviderName = "openrouter"

	defaultAnthropicModel  = "claude-sonnet-4-5"
	defaultOpenRouterModel = "anthropic/claude-sonnet-4"
)

type Config struct {
	Provider         ProviderName
	AnthropicAPIKey  string
	AnthropicModel   string
	OpenRouterAPIKey string
	OpenRouterModel  string
}

func LoadFromEnv() Config {
	provider := ProviderName(strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER"))))
	if provider == "" {
		provider = ProviderAnthropic
	}

	anthropicModel := strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL"))
	if anthropicModel == "" {
		anthropicModel = defaultAnthropicModel
	}

	openRouterModel := strings.TrimSpace(os.Getenv("OPENROUTER_MODEL"))
	if openRouterModel == "" {
		openRouterModel = defaultOpenRouterModel
	}

	return Config{
		Provider:         provider,
		AnthropicAPIKey:  strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicModel:   anthropicModel,
		OpenRouterAPIKey: strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")),
		OpenRouterModel:  openRouterModel,
	}
}

func (c *Config) ApplyFlags(provider, model string) {
	if p := strings.TrimSpace(provider); p != "" {
		c.Provider = ProviderName(strings.ToLower(p))
	}
	if m := strings.TrimSpace(model); m != "" {
		switch c.Provider {
		case ProviderAnthropic:
			c.AnthropicModel = m
		case ProviderOpenRouter:
			c.OpenRouterModel = m
		}
	}
}

func (c Config) Model() string {
	switch c.Provider {
	case ProviderAnthropic:
		return c.AnthropicModel
	case ProviderOpenRouter:
		return c.OpenRouterModel
	default:
		return ""
	}
}

func (c Config) Validate() error {
	switch c.Provider {
	case ProviderAnthropic:
		if c.AnthropicAPIKey == "" {
			return fmt.Errorf("ต้องตั้งค่า ANTHROPIC_API_KEY ก่อน")
		}
	case ProviderOpenRouter:
		if c.OpenRouterAPIKey == "" {
			return fmt.Errorf("ต้องตั้งค่า OPENROUTER_API_KEY ก่อน")
		}
	default:
		return fmt.Errorf("provider ไม่รองรับ: %s (ใช้ anthropic หรือ openrouter)", c.Provider)
	}
	return nil
}
