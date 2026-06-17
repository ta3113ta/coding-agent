# ADR-0003: Streaming LLM responses via optional callback

**Status:** Accepted  
**Date:** 2026-06-17

## Context

REPL runner รอ `provider.Complete` คืนค่าทั้งก้อนก่อนพิมพ์คำตอบ — UX ช้าและไม่รู้สึก interactive โดยเฉพาะคำตอบยาว

Streaming ต้องไหลจาก LLM API → provider → agent loop → runner แต่ agent loop ยังต้องเก็บ history และ tool calls แบบเดิมหลังจบแต่ละ turn

## Decision

เพิ่ม **optional `OnStream` callback** บน `types.CompleteRequest`:

- เมื่อ `OnStream != nil` — provider ใช้ SDK streaming API, emit `types.StreamEvent{TextDelta}` ระหว่างรับ chunk, แล้วคืน `CompleteResponse` ที่ accumulate ครบเหมือน non-streaming
- เมื่อ `OnStream == nil` — provider ใช้ path เดิม (non-streaming)

Runner (REPL) ส่ง callback ที่ `fmt.Print` text delta ทันที; tool calls ยังแสดงหลังจบ LLM turn ผ่าน verbose mode ใน agent loop

Implementation:
- [`types/types.go`](../../types/types.go) — `StreamEvent`, `CompleteRequest.OnStream`
- [`agent/agent.go`](../../agent/agent.go) — forward callback
- [`plugins/providers/anthropic/anthropic.go`](../../plugins/providers/anthropic/anthropic.go) — `Messages.NewStreaming`
- [`plugins/providers/openrouter/openrouter.go`](../../plugins/providers/openrouter/openrouter.go) — `Stream: true` + EventStream
- [`plugins/runner/repl/repl.go`](../../plugins/runner/repl/repl.go) — print deltas live

## Alternatives Considered

| แนวทาง | เหตุผลที่ไม่เลือก |
|--------|-------------------|
| `StreamingProvider` interface แยก | type assertion ซ้ำในหลายจุด |
| `CompleteStream` method บน `Provider` | บังคับทุก provider implement แม้ไม่ใช้ |
| Runner เรียก provider โดยตรง | ทำลาย layering, agent loop ข้าม |
| Stream tool-call JSON | out of scope v1 — แสดงหลัง turn เหมือน verbose |

## Consequences

**ข้อดี**

- REPL ตอบแบบ token-by-token โดยไม่เปลี่ยน agent history model
- Backward compatible — `OnStream == nil` ใช้ path เดิม
- Provider เดียวกัน serve ทั้ง streaming และ non-streaming caller

**ข้อเสีย / trade-offs**

- Provider implementations ซับซ้อนขึ้น (สอง path + accumulation)
- Tool calls ยังไม่ stream ระหว่าง generate (แสดงหลังจบ turn)
