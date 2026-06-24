package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePromptCacheTTL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", PromptCacheTTL5m},
		{"5m", PromptCacheTTL5m},
		{"1h", PromptCacheTTL1h},
		{"invalid", PromptCacheTTL5m},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, parsePromptCacheTTL(tc.in))
	}
}

func TestConfigPromptCache(t *testing.T) {
	cfg := Config{PromptCacheEnabled: true, PromptCacheTTL: PromptCacheTTL1h}
	pc := cfg.PromptCache()
	require.True(t, pc.Enabled)
	require.Equal(t, PromptCacheTTL1h, pc.TTL)
}

func TestValidatePromptCacheTTL(t *testing.T) {
	cfg := Config{
		Provider:           ProviderAnthropic,
		AnthropicAPIKey:    "key",
		PromptCacheTTL:     "bad",
		PromptCacheEnabled: true,
	}
	require.Error(t, cfg.Validate())
}

func TestSessionDirPathProject(t *testing.T) {
	cfg := Config{SessionScope: SessionScopeProject}
	dir, err := cfg.SessionDirPath("/tmp/myproject")
	require.NoError(t, err)
	want := filepath.Join("/tmp/myproject", ".coding-agent", "sessions")
	require.Equal(t, want, dir)
}

func TestSessionDirPathGlobal(t *testing.T) {
	cfg := Config{SessionScope: SessionScopeGlobal}
	dir, err := cfg.SessionDirPath("/tmp/myproject")
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(dir))
	require.Equal(t, "sessions", filepath.Base(dir))
}

func TestSessionDirPathOverride(t *testing.T) {
	cfg := Config{SessionScope: SessionScopeGlobal, SessionDir: "/custom/sessions"}
	dir, err := cfg.SessionDirPath("/tmp/myproject")
	require.NoError(t, err)
	require.Equal(t, "/custom/sessions", dir)
}

func TestApplySessionFlags(t *testing.T) {
	cfg := Config{}
	cfg.ApplySessionFlags("global", "/data/sessions")
	require.Equal(t, SessionScopeGlobal, cfg.SessionScope)
	require.Equal(t, "/data/sessions", cfg.SessionDir)
}

func TestApplyPermissionFlags(t *testing.T) {
	cfg := Config{PermissionEnabled: true}
	cfg.ApplyPermissionFlags(true)
	require.False(t, cfg.PermissionEnabled)
}

func TestParsePermissionHooksFile(t *testing.T) {
	require.Equal(t, ".coding-agent/hooks.json", parsePermissionHooksFile(""))
	require.Equal(t, "/custom/hooks.json", parsePermissionHooksFile("/custom/hooks.json"))
}

func TestValidateSessionScope(t *testing.T) {
	cfg := Config{
		Provider:        ProviderAnthropic,
		AnthropicAPIKey: "key",
		SessionScope:    "invalid",
	}
	require.Error(t, cfg.Validate())
}
