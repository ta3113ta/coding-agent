package glob

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"coding-agent/plugin"
	"coding-agent/plugins/tools/rghelper"
	"coding-agent/tools"
	"coding-agent/types"
)

const defaultHeadLimit = 500

type Glob struct{}

func (Glob) Name() string { return "glob" }

func (Glob) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name:        "glob",
		Description: "Find files by glob pattern (e.g. **/*.go) using ripgrep --files; respects .gitignore. Patterns not starting with **/ are auto-prefixed with **/",
		Properties: map[string]any{
			"glob_pattern": map[string]any{
				"type":        "string",
				"description": "glob pattern, e.g. *.go or **/*_test.go",
			},
			"target_directory": map[string]any{
				"type":        "string",
				"description": "directory to search; omit for '.'",
			},
			"head_limit": map[string]any{
				"type":        "integer",
				"description": "limit number of files (default 500)",
			},
		},
		Required: []string{"glob_pattern"},
	}
}

type globArgs struct {
	GlobPattern     string `json:"glob_pattern"`
	TargetDirectory string `json:"target_directory"`
	HeadLimit       int    `json:"head_limit"`
}

func normalizePattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return pattern
	}
	if !strings.HasPrefix(pattern, "**/") {
		return "**/" + pattern
	}
	return pattern
}

func (Glob) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var args globArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	pattern := normalizePattern(args.GlobPattern)
	if pattern == "" {
		return "", fmt.Errorf("glob_pattern is required")
	}

	root := args.TargetDirectory
	if root == "" {
		root = "."
	}

	limit := args.HeadLimit
	if limit <= 0 {
		limit = defaultHeadLimit
	}

	rgArgs := []string{"--files", "-g", pattern, root}
	raw, err := rghelper.Run(ctx, rgArgs)
	if err != nil {
		return "", err
	}
	if raw == "" {
		return "(no files found)", nil
	}

	paths := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	type fileEntry struct {
		path string
		mtime int64
	}
	entries := make([]fileEntry, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		info, statErr := os.Stat(p)
		if statErr != nil {
			entries = append(entries, fileEntry{path: p, mtime: 0})
			continue
		}
		entries = append(entries, fileEntry{path: p, mtime: info.ModTime().UnixNano()})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].mtime != entries[j].mtime {
			return entries[i].mtime > entries[j].mtime
		}
		return entries[i].path < entries[j].path
	})

	truncated := len(entries) > limit
	if truncated {
		entries = entries[:limit]
	}

	var b strings.Builder
	for _, e := range entries {
		b.WriteString(filepath.ToSlash(e.path))
		b.WriteByte('\n')
	}

	result := strings.TrimRight(b.String(), "\n")
	if result == "" {
		return "(no files found)", nil
	}
	result = rghelper.Truncate(result, rghelper.DefaultMaxBytes)
	if truncated {
		result += fmt.Sprintf("\n... (showing first %d files, sorted by most recent mtime)", limit)
	}
	return result, nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/glob" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, Glob{})
	return nil
}

var _ tools.Tool = Glob{}
