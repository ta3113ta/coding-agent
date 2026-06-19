package config

import (
	"path/filepath"
	"testing"
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
		if got := parsePromptCacheTTL(tc.in); got != tc.want {
			t.Errorf("parsePromptCacheTTL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestConfigPromptCache(t *testing.T) {
	cfg := Config{PromptCacheEnabled: true, PromptCacheTTL: PromptCacheTTL1h}
	pc := cfg.PromptCache()
	if !pc.Enabled || pc.TTL != PromptCacheTTL1h {
		t.Fatalf("PromptCache() = %+v, want enabled 1h", pc)
	}
}

func TestValidatePromptCacheTTL(t *testing.T) {
	cfg := Config{
		Provider:           ProviderAnthropic,
		AnthropicAPIKey:    "key",
		PromptCacheTTL:     "bad",
		PromptCacheEnabled: true,
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid PROMPT_CACHE_TTL")
	}
}

func TestSessionDirPathProject(t *testing.T) {
	cfg := Config{SessionScope: SessionScopeProject}
	dir, err := cfg.SessionDirPath("/tmp/myproject")
	if err != nil {
		t.Fatalf("SessionDirPath: %v", err)
	}
	want := filepath.Join("/tmp/myproject", ".coding-agent", "sessions")
	if dir != want {
		t.Fatalf("SessionDirPath = %q, want %q", dir, want)
	}
}

func TestSessionDirPathGlobal(t *testing.T) {
	cfg := Config{SessionScope: SessionScopeGlobal}
	dir, err := cfg.SessionDirPath("/tmp/myproject")
	if err != nil {
		t.Fatalf("SessionDirPath: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("global session dir should be absolute, got %q", dir)
	}
	if filepath.Base(dir) != "sessions" {
		t.Fatalf("expected .../sessions, got %q", dir)
	}
}

func TestSessionDirPathOverride(t *testing.T) {
	cfg := Config{SessionScope: SessionScopeGlobal, SessionDir: "/custom/sessions"}
	dir, err := cfg.SessionDirPath("/tmp/myproject")
	if err != nil {
		t.Fatalf("SessionDirPath: %v", err)
	}
	if dir != "/custom/sessions" {
		t.Fatalf("SessionDirPath = %q, want override", dir)
	}
}

func TestApplySessionFlags(t *testing.T) {
	cfg := Config{}
	cfg.ApplySessionFlags("global", "/data/sessions")
	if cfg.SessionScope != SessionScopeGlobal || cfg.SessionDir != "/data/sessions" {
		t.Fatalf("ApplySessionFlags = %+v", cfg)
	}
}

func TestValidateSessionScope(t *testing.T) {
	cfg := Config{
		Provider:        ProviderAnthropic,
		AnthropicAPIKey: "key",
		SessionScope:    "invalid",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid SESSION_SCOPE")
	}
}
