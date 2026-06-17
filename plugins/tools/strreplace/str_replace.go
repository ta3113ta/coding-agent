package strreplace

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"coding-agent/plugin"
	"coding-agent/tools"
	"coding-agent/types"
)

type StrReplace struct{}

func (StrReplace) Name() string { return "str_replace" }

func (StrReplace) Definition() types.ToolDefinition {
	return types.ToolDefinition{
		Name: "str_replace",
		Description: "แทนที่ข้อความในไฟล์แบบ exact match อ่านไฟล์ด้วย read_file ก่อนเสมอ " +
			"old_string ต้องตรงกับไฟล์ทุกตัวอักษร (whitespace, indentation) " +
			"ใส่บริบทรอบๆ 2-5 บรรทัดให้ unique หรือตั้ง replace_all=true สำหรับไฟล์ใหม่ใช้ write_file",
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "path ของไฟล์ที่จะแก้",
			},
			"old_string": map[string]any{
				"type":        "string",
				"description": "ข้อความเดิมที่จะแทนที่ ต้องปรากฏครั้งเดียว ยกเว้น replace_all=true",
			},
			"new_string": map[string]any{
				"type":        "string",
				"description": "ข้อความใหม่ ใส่ค่าว่างเพื่อลบ old_string",
			},
			"replace_all": map[string]any{
				"type":        "boolean",
				"description": "ถ้า true แทนที่ทุกจุดที่พบ old_string (default false)",
			},
		},
		Required: []string{"path", "old_string", "new_string"},
	}
}

func (StrReplace) Execute(input json.RawMessage) (string, error) {
	var args struct {
		Path        string `json:"path"`
		OldString   string `json:"old_string"`
		NewString   string `json:"new_string"`
		ReplaceAll  bool   `json:"replace_all"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if args.OldString == "" {
		return "", fmt.Errorf("old_string must not be empty")
	}

	info, err := os.Stat(args.Path)
	if err != nil {
		return "", err
	}

	raw, err := os.ReadFile(args.Path)
	if err != nil {
		return "", err
	}

	original := string(raw)
	ending := detectLineEnding(original)

	normalized := normalizeForMatch(original)
	oldNorm := normalizeForMatch(args.OldString)
	newNorm := normalizeForMatch(args.NewString)

	count := strings.Count(normalized, oldNorm)
	switch {
	case count == 0:
		return "", fmt.Errorf(
			"old_string not found in %s. re-read with read_file before trying again",
			args.Path,
		)
	case count > 1 && !args.ReplaceAll:
		lines := findMatchLines(normalized, oldNorm)
		return "", fmt.Errorf(
			"old_string matched %d times at lines %v. add context to old_string or set replace_all=true",
			count, lines,
		)
	}

	var result string
	if args.ReplaceAll {
		result = strings.ReplaceAll(normalized, oldNorm, newNorm)
	} else {
		result = strings.Replace(normalized, oldNorm, newNorm, 1)
	}

	output := restoreLineEnding(result, ending)
	if err := os.WriteFile(args.Path, []byte(output), info.Mode()); err != nil {
		return "", err
	}

	firstIdx := strings.Index(normalized, oldNorm)
	snippet := snippetAround(result, firstIdx, 4)

	var summary string
	if args.ReplaceAll {
		summary = fmt.Sprintf("Replaced %d occurrences in %s.\n", count, args.Path)
	} else {
		summary = fmt.Sprintf("Replaced 1 occurrence in %s.\n", args.Path)
	}
	return summary + snippet, nil
}

func expandTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "    ")
}

func normalizeForMatch(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return expandTabs(s)
}

func detectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func restoreLineEnding(content, ending string) string {
	if ending == "\r\n" {
		return strings.ReplaceAll(content, "\n", "\r\n")
	}
	return content
}

func findMatchLines(content, oldString string) []int {
	var lines []int
	start := 0
	for {
		idx := strings.Index(content[start:], oldString)
		if idx < 0 {
			break
		}
		abs := start + idx
		lines = append(lines, 1+strings.Count(content[:abs], "\n"))
		start = abs + len(oldString)
	}
	return lines
}

func snippetAround(content string, matchIdx int, radius int) string {
	if matchIdx < 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	charCount := 0
	matchLine := 0
	for i, line := range lines {
		lineEnd := charCount + len(line)
		if matchIdx >= charCount && matchIdx <= lineEnd {
			matchLine = i
			break
		}
		charCount += len(line) + 1
	}

	start := matchLine - radius
	if start < 0 {
		start = 0
	}
	end := matchLine + radius
	if end >= len(lines) {
		end = len(lines) - 1
	}

	var b strings.Builder
	for i := start; i <= end; i++ {
		b.WriteString(fmt.Sprintf("%6d\t%s\n", i+1, lines[i]))
	}
	return b.String()
}

type Plugin struct{}

func (Plugin) Name() string { return "tools/strreplace" }

func (Plugin) Register(app *plugin.App) error {
	plugin.RegisterTools(app, StrReplace{})
	return nil
}

var _ tools.Tool = StrReplace{}
