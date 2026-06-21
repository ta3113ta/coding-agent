package script

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Version int              `json:"version"`
	Hooks   map[string][]Def `json:"hooks"`
}

type Def struct {
	Command    string `json:"command"`
	Type       string `json:"type"`
	Timeout    int    `json:"timeout"`
	Matcher    string `json:"matcher"`
	FailClosed bool   `json:"failClosed"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read hooks config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse hooks config: %w", err)
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]Def{}
	}
	return &cfg, nil
}

func (c *Config) PreToolUseHooks() []Def {
	if c == nil {
		return nil
	}
	return c.Hooks["preToolUse"]
}
