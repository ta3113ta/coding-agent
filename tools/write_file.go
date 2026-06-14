package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
)

// WriteFile เขียนไฟล์ทั้งไฟล์ (สร้างใหม่หรือเขียนทับ)
// หมายเหตุ: ใน Phase 2 ควรเสริม str_replace เพื่อแก้บางส่วนโดยไม่เปลือง token
type WriteFile struct{}

func (WriteFile) Name() string { return "write_file" }

func (WriteFile) Definition() anthropic.ToolParam {
	return anthropic.ToolParam{
		Name:        "write_file",
		Description: anthropic.String("เขียนเนื้อหาลงไฟล์ สร้าง directory ให้อัตโนมัติถ้ายังไม่มี เขียนทับของเดิมทั้งหมด"),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"path":    map[string]any{"type": "string", "description": "path ของไฟล์ที่จะเขียน"},
				"content": map[string]any{"type": "string", "description": "เนื้อหาทั้งหมดของไฟล์"},
			},
			Required: []string{"path", "content"},
		},
	}
}

func (WriteFile) Execute(input json.RawMessage) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}
	if dir := filepath.Dir(args.Path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(args.Path, []byte(args.Content), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("เขียนไฟล์ %s สำเร็จ (%d bytes)", args.Path, len(args.Content)), nil
}
