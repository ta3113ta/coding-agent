# Coding Agent — Phase 1

Minimal coding agent ที่รันได้จริง แกนกลางคือ **agent loop**: วน LLM + tools จน LLM หยุดเรียก tool

ออกแบบแบบ **minimal core + compile-time plugins** — ดู [AGENTS.md](AGENTS.md) สำหรับวิธีเพิ่ม feature ใหม่

## โครงสร้าง

```
coding-agent/
├── main.go                          # bootstrap only
├── AGENTS.md                        # architecture guide
├── .cursor/rules/
│   └── agent-architecture.mdc       # Cursor rule
├── config/
│   └── config.go                    # env + flag config
├── plugin/
│   ├── plugin.go                    # Plugin interfaces + App
│   └── registry.go                  # Bootstrap
├── agent/
│   └── agent.go                     # agent loop (core)
├── types/
│   └── types.go                     # neutral shared types
├── llm/
│   └── provider.go                  # Provider interface + registry
├── tools/
│   └── tool.go                      # Tool interface + Registry
├── skills/
│   └── ...                          # skill discovery contract (parse, discover, registry)
└── plugins/
    ├── builtin/builtin.go           # default plugin registry
    ├── tools/                       # readfile, writefile, strreplace, listdir, runbash
    ├── providers/                   # anthropic, openrouter
    ├── skills/builtin/              # bundled SKILL.md files
    ├── prompt/coding/               # system prompt
    └── runner/repl/                 # stdin REPL
```

## วิธีรัน

```bash
cp .env-example .env   # แล้วใส่ API key
go mod tidy
go run .
```

### Anthropic (default)

```bash
export ANTHROPIC_API_KEY=sk-ant-...
go run .
```

### OpenRouter ผ่าน env

```bash
export LLM_PROVIDER=openrouter
export OPENROUTER_API_KEY=sk-or-...
go run .
```

### OpenRouter ผ่าน CLI flags

```bash
go run . --provider openrouter --model openai/gpt-4o
```

### ตัวแปร config

| ตัวแปร / flag | ค่าเริ่มต้น | คำอธิบาย |
|---|---|---|
| `LLM_PROVIDER` / `--provider` | `anthropic` | `anthropic` หรือ `openrouter` |
| `ANTHROPIC_API_KEY` | — | API key สำหรับ Anthropic |
| `ANTHROPIC_MODEL` / `--model` | `claude-sonnet-4-5` | model สำหรับ Anthropic |
| `OPENROUTER_API_KEY` | — | API key สำหรับ OpenRouter |
| `OPENROUTER_MODEL` / `--model` | `anthropic/claude-sonnet-4` | model สำหรับ OpenRouter |
| `SKILLS_ENABLE_PERSONAL` | `true` | เปิด/ปิด discovery จาก `~/.cursor/skills/` (`false` เพื่อปิด) |

CLI flags จะ override ค่าจาก env

แล้วลองสั่ง เช่น:
- `สร้างไฟล์ fizzbuzz.go ที่พิมพ์ 1-20 แล้ว build ให้ดูว่าผ่าน`
- `อ่าน main.go แล้วอธิบายว่าทำงานยังไง`

## แกนของ loop (agent/agent.go)

```
วน:
  1. เรียก provider.Complete ด้วย messages + tool definitions
  2. เก็บ assistant response (text + tool calls) เข้า history
  3. ถ้ามี tool call → รัน tool → เก็บผลเป็น role=tool message
  4. ไม่มี tool call → จบ คืน text
  5. ส่ง tool results กลับเข้า history → วนต่อ
```

## เพิ่ม feature ใหม่

1. เขียน ADR ใน `docs/adr/` ถ้า feature กระทบ architecture (ดู [AGENTS.md](AGENTS.md#architecture-decision-records))
2. เพิ่ม plugin ใน `plugins/` แล้วลงทะเบียนใน `plugins/builtin/builtin.go`

## Phase 2 ต่อยอด

1. ~~`str_replace` tool plugin~~ — [ADR-0001](docs/adr/0001-str-replace-for-file-editing.md)
2. ~~Load skill (two-phase discovery)~~ — [ADR-0002](docs/adr/0002-load-skill-two-phase-discovery.md)
3. **Prompt caching**
4. **Permission hook plugin**
5. **Context compaction**
6. **Parallel tool execution**
7. **Streaming runner plugin**
