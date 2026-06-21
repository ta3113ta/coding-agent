package script

import (
	"context"
	"fmt"
	"regexp"

	"coding-agent/permission"
	"coding-agent/plugin"
)

type Hook struct {
	hooks   []matchedHook
	workDir string
}

type matchedHook struct {
	def     Def
	matcher *regexp.Regexp
}

func NewHook(cfg *Config, workDir string) (*Hook, error) {
	if cfg == nil {
		return &Hook{workDir: workDir}, nil
	}

	var hooks []matchedHook
	for _, def := range cfg.PreToolUseHooks() {
		if def.Type != "" && def.Type != "command" {
			continue
		}
		if def.Command == "" {
			continue
		}
		mh := matchedHook{def: def}
		if def.Matcher != "" {
			re, err := regexp.Compile(def.Matcher)
			if err != nil {
				return nil, fmt.Errorf("invalid matcher %q: %w", def.Matcher, err)
			}
			mh.matcher = re
		}
		hooks = append(hooks, mh)
	}
	return &Hook{hooks: hooks, workDir: workDir}, nil
}

func (h *Hook) BeforeToolUse(ctx context.Context, req permission.ToolUseRequest) (permission.Result, error) {
	for _, mh := range h.hooks {
		if mh.matcher != nil && !mh.matcher.MatchString(req.ToolName) {
			continue
		}
		res, err := runHook(ctx, mh.def, req, h.workDir)
		if err != nil {
			if mh.def.FailClosed {
				return permission.Result{
					Decision: permission.Deny,
					Message:  fmt.Sprintf("permission hook error: %v", err),
				}, nil
			}
			continue
		}
		if res.Decision != permission.Allow {
			return res, nil
		}
		if res.UpdatedInput != nil {
			req.Input = res.UpdatedInput
		}
	}
	return permission.Result{Decision: permission.Allow}, nil
}

type Plugin struct{}

func (Plugin) Name() string { return "permission/script" }

func (Plugin) Register(app *plugin.App) error {
	if !app.Config.PermissionEnabled {
		return nil
	}

	cfg, err := LoadConfig(app.Config.PermissionHooksFile)
	if err != nil {
		return err
	}
	hook, err := NewHook(cfg, workDir())
	if err != nil {
		return err
	}
	if len(hook.hooks) == 0 {
		return nil
	}
	plugin.RegisterPermissionHook(app, hook)
	return nil
}

var _ permission.Hook = (*Hook)(nil)
