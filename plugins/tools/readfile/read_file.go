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
		Description: "อ่านเนื้อหาไฟล์ตาม path ที่ระบุ คืนผลพร้อมเลขบรรทัด ใช้ offset/limit เมื่อไฟล์ใหญ่เพื่ออ่านทีละช่วง",
		Properties: map[string]any{
			"path":   map[string]any{"type": "string", "description": "path ของไฟล์ที่จะอ่าน"},
			"offset": map[string]any{"type": "integer", "description": "เริ่มอ่านจากบรรทัดที่ (1-indexed) ไม่ใส่ = เริ่มจากต้น"},
			"limit":  map[string]any{"type": "integer", "description": "อ่านกี่บรรทัด ไม่ใส่ = อ่านถึงท้าย (cap ที่ 2000)"},
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
			b.WriteString(fmt.Sprintf("... (ตัดที่บรรทัด %d ใช้ offset/limit อ่านต่อ)\n", lineNo-1))
			break
		}
		b.WriteString(fmt.Sprintf("%6d\t%s\n", lineNo, sc.Text()))
		printed++
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	if printed == 0 {
		return "(ไฟล์ว่าง หรือ offset เกินจำนวนบรรทัด)", nil
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
