package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"

	"coding-agent/tools"
)

const systemPrompt = `คุณคือ coding agent ที่ช่วยผู้ใช้แก้ปัญหาเขียนโค้ดใน working directory ปัจจุบัน

หลักการทำงาน:
- ใช้ tool ที่มีเพื่อสำรวจ อ่าน เขียน และรันคำสั่ง อย่าเดาเนื้อหาไฟล์ ให้อ่านจริงก่อนเสมอ
- ก่อนแก้ไฟล์ ให้ read_file ดูของเดิมก่อน
- หลังแก้โค้ด ถ้าทำได้ให้ลอง build/test ด้วย run_bash เพื่อยืนยันว่าทำงาน
- เมื่องานเสร็จ ตอบสรุปสั้นๆ เป็นภาษาไทยว่าทำอะไรไป โดยไม่ต้องเรียก tool อีก
- ถ้าคำสั่งอันตราย (เช่น ลบไฟล์จำนวนมาก) ให้ถามยืนยันก่อน`

type Agent struct {
	client   anthropic.Client
	registry *tools.Registry
	messages []anthropic.MessageParam
	model    anthropic.Model
	verbose  bool
}

func New(client anthropic.Client, registry *tools.Registry, verbose bool) *Agent {
	return &Agent{
		client:   client,
		registry: registry,
		model:    anthropic.ModelClaudeSonnet4_5, // เปลี่ยนเป็นรุ่นที่ต้องการได้
		verbose:  verbose,
	}
}

// Run รับ input จากผู้ใช้ แล้ววน loop จน LLM หยุดเรียก tool
// คืน text สุดท้ายที่ LLM ตอบ
func (a *Agent) Run(ctx context.Context, userInput string) (string, error) {
	a.messages = append(a.messages, anthropic.NewUserMessage(
		anthropic.NewTextBlock(userInput),
	))

	for {
		// ===== 1. เรียก LLM =====
		resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
			Model:     a.model,
			MaxTokens: 8096,
			System: []anthropic.TextBlockParam{
				{Text: systemPrompt},
			},
			Messages: a.messages,
			Tools:    a.registry.Definitions(),
		})
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		// ===== 2. แปลง response content เป็น param blocks เพื่อเก็บเข้า history
		//          + แยกว่ามี tool call ไหม ในรอบเดียว =====
		// SDK รุ่นนี้ไม่มี resp.ToParam() จึงต้องประกอบ assistant turn เอง
		var assistantBlocks []anthropic.ContentBlockParamUnion
		var toolResults []anthropic.ContentBlockParamUnion
		var finalText string

		for _, block := range resp.Content {
			switch b := block.AsAny().(type) {
			case anthropic.TextBlock:
				finalText += b.Text
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(b.Text))
				if a.verbose {
					fmt.Printf("\n💭 %s\n", b.Text)
				}
			case anthropic.ToolUseBlock:
				assistantBlocks = append(assistantBlocks,
					anthropic.NewToolUseBlock(b.ID, b.Input, b.Name))
				if a.verbose {
					fmt.Printf("🔧 %s(%s)\n", b.Name, string(b.Input))
				}
				// ===== 3. รัน tool =====
				result, err := a.registry.Dispatch(b.Name, json.RawMessage(b.Input))
				isError := false
				if err != nil {
					result = fmt.Sprintf("error: %v", err)
					isError = true
				}
				toolResults = append(toolResults,
					anthropic.NewToolResultBlock(b.ID, result, isError))
			}
		}

		// เก็บคำตอบ assistant เข้า history
		a.messages = append(a.messages, anthropic.NewAssistantMessage(assistantBlocks...))

		// ===== 4. ไม่มี tool call = จบงาน =====
		if len(toolResults) == 0 {
			return finalText, nil
		}

		// ===== 5. ส่งผล tool กลับเข้า history แล้ววนต่อ =====
		a.messages = append(a.messages, anthropic.NewUserMessage(toolResults...))
	}
}
