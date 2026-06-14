# ADR-0002: Load Skill แบบ two-phase discovery

**Status:** Accepted  
**Date:** 2026-06-14

## Context

Coding agent มี system prompt แบบ static ผ่าน prompt plugin (`coding`) ที่ inject ตอน bootstrap เท่านั้น — ไม่มีวิธีเพิ่ม procedural knowledge ตาม project หรือ user workflow

Cursor Agent Skills แก้ปัญหานี้ด้วย `SKILL.md` (YAML frontmatter + markdown body) ที่ agent โหลดเมื่อ task relevant แทนการใส่ instructions ทั้งหมดใน system prompt ตลอด session

Phase 2 ต้องการ skill loading ที่:
- compatible กับ Cursor SKILL.md format
- ประหยัด token (index เท่านั้นใน system prompt)
- ไม่แก้ agent loop

## Decision

ใช้ **two-phase lazy loading** คู่กับ **`read_file` tool ที่มีอยู่**:

1. **Bootstrap** — discover `SKILL.md` จาก 3 แหล่ง แล้ว merge ด้วย priority
2. **Index** — inject เฉพาะ `name` + `description` + `path` เข้า system prompt
3. **Full load** — agent อ่าน `SKILL.md` ด้วย `read_file` เมื่อ task ตรงกับ description

### Discovery paths (priority: project > personal > bundled)

| Source | Path |
|--------|------|
| Project | `<workspace>/.cursor/skills/*/SKILL.md` |
| Personal | `~/.cursor/skills/*/SKILL.md` (ปิดได้ด้วย `SKILLS_ENABLE_PERSONAL=false`) |
| Bundled | `plugins/skills/builtin/*/SKILL.md` (embed + extract ไป temp dir สำหรับ `read_file`) |

Implementation:
- Core contract: [`skills/`](../../skills/)
- Plugin: [`plugins/skills/skills.go`](../../plugins/skills/skills.go)

## Alternatives Considered

| แนวทาง | Token efficiency | เหตุผลที่ไม่เลือก v1 |
|--------|------------------|----------------------|
| Two-phase index + `read_file` | สูงสุด | **เลือกใช้** — ไม่แก้ agent loop |
| Eager load ทุก skill body เข้า prompt | ต่ำ | token cost สูงเมื่อ skill เยอะ |
| `load_skill` tool แยก | ปานกลาง | ต้อง inject agent state เข้า tool; `read_file` พอแล้ว |
| Compile-time prompt plugin เท่านั้น | สูง | ไม่ dynamic ตาม project/user |
| REPL `/skill-name` explicit invoke | ดีสำหรับ slash commands | defer v2 — ยังพึ่ง model routing จาก index |
| Scan `~/.cursor/skills-cursor/` | — | Cursor internal — ไม่ใช้ |

## Consequences

**ข้อดี**

- ไม่แก้ [`agent/agent.go`](../../agent/agent.go) — `systemPrompt` static ต่อ session แต่รวม skill index ตอน bootstrap
- Compatible กับ Cursor SKILL.md format (`name`, `description`, `disable-model-invocation`)
- Token-efficient — descriptions เท่านั้นใน prompt ตลอด session
- Bundled skills ผ่าน `embed.FS` — ship กับ binary ได้

**ข้อเสีย / trade-offs**

- พึ่ง model ตัดสินใจว่า skill ไหน relevant จาก description
- Bundled skills extract ไป temp dir — path เปลี่ยนทุก session
- ไม่รองรับ `environments` / `metadata.surfaces` filtering ใน v1
- ไม่มี explicit `/skill-name` ใน REPL v1

## Skill Index Format

```
## Available Skills

เมื่องานของผู้ใช้ตรงกับ description ของ skill ใด:
1. อ่าน SKILL.md ด้วย read_file ก่อนทำงาน
2. ทำตาม instructions ใน skill ทันที
3. ถ้า skill มี disable-model-invocation อย่าโหลดเอง ยกเว้นผู้ใช้ระบุชื่อ skill ชัดเจน

<available_skills>
- commit-message (/path/to/SKILL.md): สร้าง git commit message...
</available_skills>
```

## v2 (defer)

- REPL `/skill-name` explicit invocation
- `environments` / `metadata.surfaces` filtering
- `embed.FS` path แทน temp dir extraction (ถ้า `read_file` รองรับ virtual FS)
- Skill hot-reload ระหว่าง session

## References

- Cursor `create-skill` SKILL.md spec (`~/.cursor/skills-cursor/create-skill/SKILL.md`)
- Cursor two-phase skill index model (`<agent_skills>` block)
