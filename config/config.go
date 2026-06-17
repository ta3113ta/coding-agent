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
	Provider             ProviderName
	AnthropicAPIKey      string
	AnthropicModel       string
	OpenRouterAPIKey     string
	OpenRouterModel      string
	SkillsEnablePersonal bool
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
		Provider:             provider,
		AnthropicAPIKey:      strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicModel:       anthropicModel,
		OpenRouterAPIKey:     strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")),
		OpenRouterModel:      openRouterModel,
		SkillsEnablePersonal: parseBoolEnv("SKILLS_ENABLE_PERSONAL", true),
	}
}

func parseBoolEnv(key string, defaultVal bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return defaultVal
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultVal
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
			return fmt.Errorf("ANTHROPIC_API_KEY must be set")
		}
	case ProviderOpenRouter:
		if c.OpenRouterAPIKey == "" {
			return fmt.Errorf("OPENROUTER_API_KEY must be set")
		}
	default:
		return fmt.Errorf("unsupported provider: %s (use anthropic or openrouter)", c.Provider)
	}
	return nil
}
