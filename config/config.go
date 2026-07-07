package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"coding-agent/types"
)

type ProviderName string

const (
	ProviderAnthropic  ProviderName = "anthropic"
	ProviderOpenRouter ProviderName = "openrouter"

	defaultAnthropicModel  = "claude-sonnet-4-5"
	defaultOpenRouterModel = "anthropic/claude-sonnet-4"

	PromptCacheTTL5m = "5m"
	PromptCacheTTL1h = "1h"

	SessionScopeProject = "project"
	SessionScopeGlobal  = "global"

	defaultCompactionReserveTokens    = 16384
	defaultCompactionKeepRecentTokens = 20000
	defaultCompactionContextWindow    = 200000
	defaultSpawnMaxTurns              = 25
)

type Config struct {
	Provider                   ProviderName
	AnthropicAPIKey            string
	AnthropicModel             string
	OpenRouterAPIKey           string
	OpenRouterModel            string
	SkillsEnablePersonal       bool
	PromptCacheEnabled         bool
	PromptCacheTTL             string
	SessionScope               string
	SessionDir                 string
	PermissionEnabled          bool
	PermissionHooksFile        string
	CompactionEnabled          bool
	CompactionReserveTokens    int
	CompactionKeepRecentTokens int
	CompactionContextWindow    int
	SpawnEnabled               bool
	SpawnMaxTurns              int
	PlanEnabled                bool
	ParallelToolsEnabled       bool
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
		Provider:                   provider,
		AnthropicAPIKey:            strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")),
		AnthropicModel:             anthropicModel,
		OpenRouterAPIKey:           strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")),
		OpenRouterModel:            openRouterModel,
		SkillsEnablePersonal:       parseBoolEnv("SKILLS_ENABLE_PERSONAL", true),
		PromptCacheEnabled:         parseBoolEnv("PROMPT_CACHE_ENABLED", true),
		PromptCacheTTL:             parsePromptCacheTTL(os.Getenv("PROMPT_CACHE_TTL")),
		SessionScope:               parseSessionScope(os.Getenv("SESSION_SCOPE")),
		SessionDir:                 strings.TrimSpace(os.Getenv("SESSION_DIR")),
		PermissionEnabled:          parseBoolEnv("PERMISSION_ENABLED", true),
		PermissionHooksFile:        parsePermissionHooksFile(os.Getenv("PERMISSION_HOOKS_FILE")),
		CompactionEnabled:          parseBoolEnv("COMPACTION_ENABLED", true),
		CompactionReserveTokens:    parseIntEnv("COMPACTION_RESERVE_TOKENS", defaultCompactionReserveTokens),
		CompactionKeepRecentTokens: parseIntEnv("COMPACTION_KEEP_RECENT_TOKENS", defaultCompactionKeepRecentTokens),
		CompactionContextWindow:    parseIntEnv("COMPACTION_CONTEXT_WINDOW", defaultCompactionContextWindow),
		SpawnEnabled:               parseBoolEnv("SPAWN_ENABLED", true),
		SpawnMaxTurns:              parseIntEnv("SPAWN_MAX_TURNS", defaultSpawnMaxTurns),
		PlanEnabled:                parseBoolEnv("PLAN_ENABLED", true),
		ParallelToolsEnabled:       parseBoolEnv("PARALLEL_TOOLS_ENABLED", true),
	}
}

func parseSessionScope(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case SessionScopeGlobal:
		return SessionScopeGlobal
	default:
		return SessionScopeProject
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

func parsePromptCacheTTL(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", PromptCacheTTL5m:
		return PromptCacheTTL5m
	case PromptCacheTTL1h:
		return PromptCacheTTL1h
	default:
		return PromptCacheTTL5m
	}
}

func parseIntEnv(key string, defaultVal int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}

func parsePermissionHooksFile(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ".coding-agent/hooks.json"
	}
	return raw
}

func (c *Config) ApplySpawnFlags(noSpawn bool) {
	if noSpawn {
		c.SpawnEnabled = false
	}
}

func (c *Config) ApplyPlanFlags(noPlan bool) {
	if noPlan {
		c.PlanEnabled = false
	}
}

func (c *Config) ApplyCompactionFlags(noCompaction bool) {
	if noCompaction {
		c.CompactionEnabled = false
	}
}

func (c *Config) ApplyPermissionFlags(noPermission bool) {
	if noPermission {
		c.PermissionEnabled = false
	}
}

func (c *Config) ApplyParallelToolsFlags(noParallel bool) {
	if noParallel {
		c.ParallelToolsEnabled = false
	}
}

func (c *Config) ApplySessionFlags(sessionScope, sessionDir string) {
	if s := strings.TrimSpace(strings.ToLower(sessionScope)); s != "" {
		c.SessionScope = parseSessionScope(s)
	}
	if d := strings.TrimSpace(sessionDir); d != "" {
		c.SessionDir = d
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
	if c.PromptCacheTTL != PromptCacheTTL5m && c.PromptCacheTTL != PromptCacheTTL1h {
		return fmt.Errorf("PROMPT_CACHE_TTL must be %q or %q", PromptCacheTTL5m, PromptCacheTTL1h)
	}
	if c.SessionScope != SessionScopeProject && c.SessionScope != SessionScopeGlobal {
		return fmt.Errorf("SESSION_SCOPE must be %q or %q", SessionScopeProject, SessionScopeGlobal)
	}
	if c.CompactionReserveTokens <= 0 {
		return fmt.Errorf("COMPACTION_RESERVE_TOKENS must be positive")
	}
	if c.CompactionKeepRecentTokens <= 0 {
		return fmt.Errorf("COMPACTION_KEEP_RECENT_TOKENS must be positive")
	}
	if c.CompactionContextWindow <= 0 {
		return fmt.Errorf("COMPACTION_CONTEXT_WINDOW must be positive")
	}
	if c.SpawnMaxTurns <= 0 {
		return fmt.Errorf("SPAWN_MAX_TURNS must be positive")
	}
	return nil
}

func (c Config) SessionDirPath(cwd string) (string, error) {
	if c.SessionDir != "" {
		return c.SessionDir, nil
	}
	if c.SessionScope == SessionScopeGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home dir: %w", err)
		}
		return filepath.Join(home, ".coding-agent", "sessions"), nil
	}
	return filepath.Join(cwd, ".coding-agent", "sessions"), nil
}

func (c Config) PromptCache() types.PromptCacheConfig {
	return types.PromptCacheConfig{
		Enabled: c.PromptCacheEnabled,
		TTL:     c.PromptCacheTTL,
	}
}
