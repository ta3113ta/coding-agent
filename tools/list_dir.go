package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// ListDir แสดงรายการไฟล์/โฟลเดอร์ใน path
type ListDir struct{}

func (ListDir) Name() string { return "list_dir" }

func (ListDir) Definition() anthropic.ToolParam {
	return anthropic.ToolParam{
		Name:        "list_dir",
		Description: anthropic.String("แสดงไฟล์และโฟลเดอร์ใน path ที่ระบุ ไม่ใส่ path = directory ปัจจุบัน"),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"path": map[string]any{"type": "string", "description": "path ของ directory ไม่ใส่ = '.'"},
			},
			Required: []string{},
		},
	}
}

func (ListDir) Execute(input json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	_ = json.Unmarshal(input, &args)
	if args.Path == "" {
		args.Path = "."
	}
	entries, err := os.ReadDir(args.Path)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		b.WriteString(fmt.Sprintf("%-40s %8d\n", name, size))
	}
	if b.Len() == 0 {
		return "(directory ว่าง)", nil
	}
	return b.String(), nil
}
