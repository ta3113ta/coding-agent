# ADR-0004: Prompt caching via top-level automatic cache_control

**Status:** Accepted  
**Date:** 2026-06-18

## Context

Agent loop เรียก `provider.Complete` ซ้ำหลายครั้งต่อ user turn (tool use loop) โดยส่ง `system prompt + tool definitions + message history` ที่ prefix ส่วนใหญ่เหมือนเดิม — ทำให้เสีย token และ latency ซ้ำซ้อน

Anthropic และ OpenRouter รองรับ prompt caching ผ่าน `cache_control: { type: "ephemeral" }` ที่ลดต้นทุน input token บน prefix ที่ cache hit

## Decision

เพิ่ม **`PromptCacheConfig` บน `types.CompleteRequest`** แล้วให้ provider ใส่ **top-level automatic `cache_control`** เมื่อ enabled:

- Config: `PROMPT_CACHE_ENABLED` (default `true`), `PROMPT_CACHE_TTL` (`5m` หรือ `1h`)
- Agent forward config ทุก `Complete` call โดยไม่เปลี่ยน loop logic
- Anthropic: `MessageNewParams.CacheControl`
- OpenRouter: `ChatRequest.CacheControl`

Top-level automatic caching วาง breakpoint ที่ block สุดท้ายที่ cache ได้และเลื่อนไปตาม conversation โต — เหมาะกับ multi-step tool loop โดยไม่ต้องจัดการ breakpoint เอง

Implementation:
- [`types/types.go`](../../types/types.go) — `PromptCacheConfig`, `CompleteRequest.PromptCache`
- [`config/config.go`](../../config/config.go) — env parsing + validation
- [`agent/agent.go`](../../agent/agent.go) — forward config
- [`plugins/providers/anthropic/anthropic.go`](../../plugins/providers/anthropic/anthropic.go) — `applyPromptCache`
- [`plugins/providers/openrouter/openrouter.go`](../../plugins/providers/openrouter/openrouter.go) — `applyPromptCache`

## Alternatives Considered

| แนวทาง | เหตุผลที่ไม่เลือก v1 |
|--------|----------------------|
| Top-level automatic `cache_control` | **เลือกใช้** — minimal code, auto-advancing breakpoint |
| Explicit breakpoints (system, tools, messages) | ซับซ้อน: 4-breakpoint limit, 20-block lookback, TTL ordering |
| Provider-internal always-on | ซ่อน control, debug ยาก |
| OpenRouter `session_id` sticky routing | out of scope v1 |
| `CompleteResponse.Usage` cache stats | deferred — roadmap item Cost / token tracking |

## Consequences

**ข้อดี**

- ลด cost/latency บน tool-loop iterations โดยไม่เปลี่ยน agent history model
- Backward compatible — `PromptCache.Enabled == false` ใช้ path เดิม
- ทั้ง Anthropic direct และ OpenRouter ใช้ pattern เดียวกัน

**ข้อเสีย / trade-offs**

- Cache hit ต้องการ prefix ขั้นต่ำตาม model (เช่น 1,024 tokens สำหรับ Sonnet) — prompt สั้นอาจไม่ cache
- Tool definitions ต้อง stable ระหว่าง session (ลำดับใน `builtin.Default` คงที่)
- Long sessions (>20 content blocks) อาจต้อง explicit breakpoints ในอนาคต
