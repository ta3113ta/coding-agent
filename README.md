# Coding Agent — Phase 1

Minimal coding agent ที่รันได้จริง แกนกลางคือ **agent loop**: วน LLM + tools จน LLM หยุดเรียก tool

## โครงสร้าง

```
coding-agent/
├── go.mod
├── main.go              # REPL entry point + ลงทะเบียน tool
├── agent/
│   └── agent.go         # ❤️ agent loop (อ่านตรงนี้ก่อน)
└── tools/
    ├── tool.go          # Tool interface + Registry (dispatch ตามชื่อ)
    ├── read_file.go     # อ่านไฟล์ + offset/limit
    ├── write_file.go    # เขียนไฟล์ทั้งไฟล์
    ├── list_dir.go      # ls
    └── run_bash.go      # รัน bash + timeout + truncate output
```

## วิธีรัน

```bash
export ANTHROPIC_API_KEY=sk-ant-...
go mod tidy      # ดึง SDK
go run .
```

แล้วลองสั่ง เช่น:
- `สร้างไฟล์ fizzbuzz.go ที่พิมพ์ 1-20 แล้ว build ให้ดูว่าผ่าน`
- `อ่าน main.go แล้วอธิบายว่าทำงานยังไง`

## แกนของ loop (agent/agent.go)

```
วน:
  1. เรียก LLM ด้วย messages + tool definitions
  2. แปลง response → param blocks เก็บเข้า history
  3. ถ้ามี tool_use → รัน tool → เก็บผลเป็น tool_result
  4. ไม่มี tool_use → จบ คืน text
  5. ส่ง tool_result กลับเข้า history → วนต่อ
```

ทั้งหมดของ opencode/Claude Code คือ loop นี้ + ของห่อรอบๆ

## จุดออกแบบที่ควรสังเกต

- **Tool เป็น interface** — เพิ่ม tool ใหม่แค่ implement 3 method แล้ว append ใน `main.go`
- **error ของ tool ไม่ทำให้ loop พัง** — ส่ง error กลับเป็น tool_result ให้ LLM แก้เอง (isError=true)
- **กัน context พอง** — `read_file` มี offset/limit, `run_bash` truncate output หัว-ท้าย
- **assistant turn ประกอบเอง** — SDK รุ่นนี้ไม่มี `resp.ToParam()` จึง map content blocks → param blocks เอง

## Phase 2 ต่อยอด (ดู comment ในโค้ด)

1. `str_replace` tool — แก้ไฟล์บางส่วน ไม่เขียนทั้งไฟล์ (ประหยัด token มหาศาล)
2. **Prompt caching** — cache system prompt + tool defs + history ลด cost ~90%
3. **Permission layer** — ขออนุมัติก่อนรันคำสั่งอันตราย
4. **Context compaction** — สรุป history เมื่อใกล้เต็ม window
5. **Parallel tool execution** — รัน tool หลายตัวพร้อมกันด้วย goroutine
6. **Streaming** — `client.Messages.NewStreaming` เพื่อ UX
