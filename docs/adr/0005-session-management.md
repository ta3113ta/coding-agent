# ADR-0005: File-based session management

**Status:** Accepted  
**Date:** 2026-06-18

## Context

Conversation history เก็บใน `agent.Agent.messages` ใน RAM เท่านั้น — REPL multi-turn ทำงานได้ภายใน process เดียว แต่ history หายเมื่อ exit และไม่สามารถ resume ข้ามการรันได้

## Decision

เพิ่ม **file-based JSON session persistence** ผ่าน core `session.Store` contract และ `plugins/session/filestore` implementation:

- **Persist:** `id`, `created_at`, `updated_at`, `provider`, `model`, `messages[]`
- **Do not persist:** `systemPrompt` — re-resolve จาก bootstrap ตอน resume (skills/prompt อาจเปลี่ยนระหว่าง runs)
- **Resume policy:** โหลด messages; ใช้ provider/model จาก config ปัจจุบันสำหรับ LLM calls ต่อไป (metadata ใน store ใช้แสดงใน list)
- **Auto-save:** หลัง `Run()` สำเร็จแต่ละครั้ง
- **Storage:** default `{cwd}/.coding-agent/sessions/`; `SESSION_SCOPE=global` → `~/.coding-agent/sessions/`; `SESSION_DIR` override ทั้งคู่
- **UX:** CLI `--resume`, `--new-session`, `--session-scope`, `--session-dir`; REPL `/new`, `/sessions`, `/resume <id>`, `/session`

Implementation:
- [`session/session.go`](../../session/session.go) — `Session`, `Meta`, `Store` interface
- [`config/config.go`](../../config/config.go) — `SESSION_SCOPE`, `SESSION_DIR`, `SessionDir()`
- [`agent/agent.go`](../../agent/agent.go) — persistence hooks around `Run()`
- [`plugins/session/filestore/filestore.go`](../../plugins/session/filestore/filestore.go) — JSON file store
- [`plugins/runner/repl/repl.go`](../../plugins/runner/repl/repl.go) — slash commands
- [`plugin/plugin.go`](../../plugin/plugin.go) — `App.SessionStore`, extended `AgentHandle`

## Alternatives Considered

| แนวทาง | เหตุผลที่ไม่เลือก v1 |
|--------|----------------------|
| File-based JSON (`session.Store` + filestore plugin) | **เลือกใช้** — minimal deps, human-readable, fits plugin architecture |
| SQLite | overkill สำหรับ learning scope |
| Provider-native session APIs (OpenRouter `session_id`) | coupled to provider; deferred |
| In-agent RAM only | ไม่แก้ปัญหา persistence |
| Persist `systemPrompt` | stale เมื่อ skills เปลี่ยน |

## Consequences

**ข้อดี**

- Resume conversation ข้าม process restarts
- Project-local sessions default; global scope สำหรับ cross-project resume
- Agent loop logic ไม่เปลี่ยน — hooks รอบ `Run()` เท่านั้น
- Serialization DTO อยู่ใน filestore plugin — `types/` ยัง provider-neutral

**ข้อเสีย / trade-offs**

- `--resume` บน startup สร้าง empty session ก่อน load (orphan file เล็กน้อย) — **แก้ใน v2** ด้วย `InitNewSession` แทน auto-create ใน `New`
- ไม่มี `/delete`, context compaction ใน v1
- `List` โหลด metadata จากทุกไฟล์ — พอใช้สำหรับ session จำนวนน้อย

## v2 amendment (2026-06-18)

### Context

v1 ขาด display name, ephemeral mode, และ startup UX ที่สะดวก (`-c` continue latest, `-r` interactive picker)

### Decision

- **Display name:** เพิ่ม `name` ใน session JSON; CLI `--name`, REPL `/name <name>`
- **Ephemeral mode:** `--no-session` ใช้ `plugins/session/memory` (in-RAM `Store`, ไม่เขียน disk)
- **Startup flags:** `-c` resume latest; `-r` interactive picker; flag precedence ใน `main.go`
- **Startup refactor:** `agent.New` ไม่ auto-create session; `main` เรียก `InitNewSession` หรือ `ResumeSession` ตาม flags — แก้ orphan file บน resume paths
- **Default:** ไม่มี flag → new empty session ทุกครั้ง (เหมือนเดิม)

Flag precedence: `--no-session` > `--new-session` > `--resume` > `-c` > `-r`

Implementation additions:
- [`plugins/session/memory/memory.go`](../../plugins/session/memory/memory.go) — ephemeral store
- [`plugins/session/picker/picker.go`](../../plugins/session/picker/picker.go) — `-r` interactive selection
- [`session/session.go`](../../session/session.go) — `Name`, `Latest()`

