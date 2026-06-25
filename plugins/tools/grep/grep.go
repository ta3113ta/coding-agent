package grep

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"coding-agent/plugin"
	"coding-agent/plugins/tools/rghelper"
	"coding-agent/tools"
	"coding-agent/types"
)

type Grep struct{}

func (Grep) Name() string { return "grep" }

func (Grep) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name: "grep",
		Description: "Search the codebase with ripgrep (rg). Supports regex; respects .gitignore by default. " +
			"Use output_mode: content (default), files_with_matches, count",
		Properties: map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "regex pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "path or directory to search; omit for '.'",
			},
			"glob": map[string]any{
				"type":        "string",
				"description": "filter files with glob, e.g. *.go",
			},
			"output_mode": map[string]any{
				"type":        "string",
				"description": "content | files_with_matches | count",
				"enum":        []string{"content", "files_with_matches", "count"},
			},
			"-i": map[string]any{
				"type":        "boolean",
				"description": "case insensitive search",
			},
			"-A": map[string]any{"type": "integer", "description": "lines of context after match"},
			"-B": map[string]any{"type": "integer", "description": "lines of context before match"},
			"-C": map[string]any{"type": "integer", "description": "lines of context on both sides"},
			"type": map[string]any{
				"type":        "string",
				"description": "file type, e.g. go, py, js",
			},
			"head_limit": map[string]any{
				"type":        "integer",
				"description": "limit number of lines/results",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "skip the first N results",
			},
			"multiline": map[string]any{
				"type":        "boolean",
				"description": "enable multiline mode (. matches newlines)",
			},
		},
		Required: []string{"pattern"},
	}
}

type grepArgs struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	Glob       string `json:"glob"`
	OutputMode string `json:"output_mode"`
	IgnoreCase bool   `json:"-i"`
	ContextA   int    `json:"-A"`
	ContextB   int    `json:"-B"`
	ContextC   int    `json:"-C"`
	Type       string `json:"type"`
	HeadLimit  int    `json:"head_limit"`
	Offset     int    `json:"offset"`
	Multiline  bool   `json:"multiline"`
}

func (Grep) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var args grepArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Pattern) == "" {
		return "", fmt.Errorf("pattern is required")
	}

	searchPath := args.Path
	if searchPath == "" {
		searchPath = "."
	}

	mode := args.OutputMode
	if mode == "" {
		mode = "content"
	}

	rgArgs := []string{"--color=never"}
	switch mode {
	case "files_with_matches":
		rgArgs = append(rgArgs, "-l")
	case "count":
		rgArgs = append(rgArgs, "--count-matches")
	default:
		rgArgs = append(rgArgs, "-n", "--no-heading")
	}

	if args.IgnoreCase {
		rgArgs = append(rgArgs, "-i")
	}
	if args.ContextC > 0 {
		rgArgs = append(rgArgs, "-C", fmt.Sprintf("%d", args.ContextC))
	} else {
		if args.ContextA > 0 {
			rgArgs = append(rgArgs, "-A", fmt.Sprintf("%d", args.ContextA))
		}
		if args.ContextB > 0 {
			rgArgs = append(rgArgs, "-B", fmt.Sprintf("%d", args.ContextB))
		}
	}
	if args.Type != "" {
		rgArgs = append(rgArgs, "-t", args.Type)
	}
	if args.Glob != "" {
		rgArgs = append(rgArgs, "--glob", args.Glob)
	}
	if args.Multiline {
		rgArgs = append(rgArgs, "-U", "--multiline", "--multiline-dotall")
	}

	rgArgs = append(rgArgs, args.Pattern, searchPath)

	raw, err := rghelper.Run(ctx, rgArgs)
	if err != nil {
		return "", err
	}
	if raw == "" {
		return "(No results found)", nil
	}

	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if args.Offset > 0 && args.Offset < len(lines) {
		lines = lines[args.Offset:]
	} else if args.Offset >= len(lines) {
		return "(No results found)", nil
	}

	truncated := false
	if args.HeadLimit > 0 && len(lines) > args.HeadLimit {
		lines = lines[:args.HeadLimit]
		truncated = true
	}

	result := strings.Join(lines, "\n")
	result = rghelper.Truncate(result, rghelper.DefaultMaxBytes)
	if truncated {
		result += fmt.Sprintf("\n... (showing first %d results; use offset/head_limit for more)", args.HeadLimit)
	}
	return result, nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/grep" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, Grep{})
	return nil
}

var _ tools.Tool = Grep{}
