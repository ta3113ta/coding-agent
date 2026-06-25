package readfile

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type ReadFile struct{}

func (ReadFile) Name() string { return "read_file" }

func (ReadFile) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name:        "read_file",
		Description: "Read file contents at the given path. Returns line numbers. Use offset/limit for large files to read in chunks.",
		Properties: map[string]any{
			"path":   map[string]any{"type": "string", "description": "path of the file to read"},
			"offset": map[string]any{"type": "integer", "description": "start reading at this line (1-indexed); omit to start from the beginning"},
			"limit":  map[string]any{"type": "integer", "description": "number of lines to read; omit to read to end (capped at 2000)"},
		},
		Required: []string{"path"},
	}
}

func (ReadFile) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	_ = ctx
	var args struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	f, err := os.Open(args.Path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	const maxLines = 2000
	if args.Limit <= 0 || args.Limit > maxLines {
		args.Limit = maxLines
	}
	start := args.Offset
	if start < 1 {
		start = 1
	}

	var b strings.Builder
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNo := 0
	printed := 0
	for sc.Scan() {
		lineNo++
		if lineNo < start {
			continue
		}
		if printed >= args.Limit {
			b.WriteString(fmt.Sprintf("... (truncated at line %d; use offset/limit to read more)\n", lineNo-1))
			break
		}
		b.WriteString(fmt.Sprintf("%6d\t%s\n", lineNo, sc.Text()))
		printed++
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	if printed == 0 {
		return "(empty file or offset beyond line count)", nil
	}
	return b.String(), nil
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/readfile" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, ReadFile{})
	return nil
}

var _ tools.Tool = ReadFile{}
