package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

// RunBash รัน command ใน shell มี timeout + truncate output
// ครอบคลุม grep/find/git/test/build ทำให้ไม่ต้องเขียน tool แยกเยอะ
type RunBash struct{}

func (RunBash) Name() string { return "run_bash" }

func (RunBash) Definition() anthropic.ToolParam {
	return anthropic.ToolParam{
		Name:        "run_bash",
		Description: anthropic.String("รันคำสั่ง bash ใน working directory ปัจจุบัน ใช้สำหรับ grep, find, git, go build/test ฯลฯ มี timeout 60 วินาที"),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"command": map[string]any{"type": "string", "description": "คำสั่ง bash ที่จะรัน"},
			},
			Required: []string{"command"},
		},
	}
}

func (RunBash) Execute(input json.RawMessage) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()

	result := truncate(out.String(), 30_000) // กัน context พองจาก output ยาว
	if ctx.Err() == context.DeadlineExceeded {
		return result + "\n(timeout หลัง 60 วินาที)", nil
	}
	if err != nil {
		// ส่ง error กลับเป็น "ผลลัพธ์" ให้ LLM เห็นและแก้เอง ไม่ return error จริง
		return fmt.Sprintf("%s\n(exit error: %v)", result, err), nil
	}
	if result == "" {
		return "(ไม่มี output, exit 0)", nil
	}
	return result, nil
}

// truncate เก็บหัว 60% ท้าย 40% ตัดตรงกลาง — ส่วนใหญ่ข้อมูลสำคัญอยู่หัวกับท้าย
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	head := max * 6 / 10
	tail := max - head
	return s[:head] + fmt.Sprintf("\n... (ตัด %d bytes ตรงกลาง) ...\n", len(s)-max) + s[len(s)-tail:]
}
