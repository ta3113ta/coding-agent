package script

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"coding-agent/permission"
)

type hookInput struct {
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolCallID string          `json:"tool_call_id"`
}

type hookOutput struct {
	Permission   string          `json:"permission"`
	UserMessage  string          `json:"user_message"`
	AgentMessage string          `json:"agent_message"`
	UpdatedInput json.RawMessage `json:"updated_input"`
}

func runHook(ctx context.Context, def Def, req permission.ToolUseRequest, workDir string) (permission.Result, error) {
	timeout := time.Duration(def.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	in := hookInput{
		ToolName:   req.ToolName,
		ToolInput:  req.Input,
		ToolCallID: req.ToolCallID,
	}
	inJSON, err := json.Marshal(in)
	if err != nil {
		return permission.Result{}, err
	}

	cmdPath := def.Command
	if !filepath.IsAbs(cmdPath) {
		cmdPath = filepath.Join(workDir, cmdPath)
	}

	cmd := exec.CommandContext(runCtx, cmdPath)
	cmd.Dir = workDir
	cmd.Stdin = bytes.NewReader(inJSON)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if runCtx.Err() == context.DeadlineExceeded {
		if def.FailClosed {
			return permission.Result{
				Decision: permission.Deny,
				Message:  fmt.Sprintf("permission hook timed out after %s", timeout),
			}, nil
		}
		return permission.Result{Decision: permission.Allow}, nil
	}

	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			if def.FailClosed {
				return permission.Result{
					Decision: permission.Deny,
					Message:  fmt.Sprintf("permission hook failed: %v", err),
				}, nil
			}
			return permission.Result{Decision: permission.Allow}, nil
		}
	}

	if exitCode == 2 {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = "permission denied by hook"
		}
		return permission.Result{Decision: permission.Deny, Message: msg}, nil
	}
	if exitCode != 0 {
		if def.FailClosed {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = fmt.Sprintf("permission hook exited with code %d", exitCode)
			}
			return permission.Result{Decision: permission.Deny, Message: msg}, nil
		}
		return permission.Result{Decision: permission.Allow}, nil
	}

	outText := strings.TrimSpace(stdout.String())
	if outText == "" {
		return permission.Result{Decision: permission.Allow}, nil
	}

	var out hookOutput
	if err := json.Unmarshal([]byte(outText), &out); err != nil {
		if def.FailClosed {
			return permission.Result{
				Decision: permission.Deny,
				Message:  fmt.Sprintf("invalid hook output JSON: %v", err),
			}, nil
		}
		return permission.Result{Decision: permission.Allow}, nil
	}

	return parseHookOutput(out)
}

func parseHookOutput(out hookOutput) (permission.Result, error) {
	switch strings.ToLower(strings.TrimSpace(out.Permission)) {
	case "allow", "":
		res := permission.Result{Decision: permission.Allow}
		if len(out.UpdatedInput) > 0 {
			res.UpdatedInput = out.UpdatedInput
		}
		return res, nil
	case "deny":
		msg := out.AgentMessage
		if msg == "" {
			msg = "permission denied by hook"
		}
		return permission.Result{Decision: permission.Deny, Message: msg}, nil
	case "ask":
		msg := out.UserMessage
		if msg == "" {
			msg = out.AgentMessage
		}
		return permission.Result{Decision: permission.Ask, Message: msg}, nil
	default:
		return permission.Result{}, fmt.Errorf("unknown permission value: %q", out.Permission)
	}
}

func workDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
