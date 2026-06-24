package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"coding-agent/types"
)

// Tool คือ interface ที่ทุก tool ต้อง implement
// แยก "นิยาม" (schema ที่ส่งให้ LLM) ออกจาก "การทำงานจริง" (Execute)
type Tool interface {
	// Name ที่ LLM ใช้เรียก
	Name() string
	// Definition คือ schema ที่ส่งให้ LLM รู้ว่า tool นี้ทำอะไร รับ param อะไร
	Definition() types.ToolDefinition
	// Execute รับ raw JSON input จาก LLM แล้วคืนผลลัพธ์เป็น string
	Execute(ctx context.Context, input json.RawMessage) (string, error)
}

// Registry เก็บ tool ทั้งหมด ใช้ dispatch ตามชื่อ
type Registry struct {
	tools map[string]Tool
}

func NewRegistry(ts ...Tool) *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	for _, t := range ts {
		r.Register(t)
	}
	return r
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for name := range r.tools {
		out = append(out, name)
	}
	return out
}

// Definitions คืน schema ทั้งหมดในรูปแบบที่ provider ใช้ได้
func (r *Registry) Definitions() []types.ToolDefinition {
	out := make([]types.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t.Definition())
	}
	return out
}

// Dispatch หา tool ตามชื่อแล้วเรียก Execute
func (r *Registry) Dispatch(ctx context.Context, name string, input json.RawMessage) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, input)
}
