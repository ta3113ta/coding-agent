# ADR-0001: ใช้ str_replace สำหรับแก้ไฟล์แทน write_file

**Status:** Accepted  
**Date:** 2026-06-14

## Context

Coding agent มี `read_file` และ `write_file` แล้ว แต่ `write_file` เขียนทับทั้งไฟล์ทุกครั้งที่แก้ — ทำให้ LLM ต้องส่ง output token เท่ากับขนาดไฟล์ทั้งหมด แม้จะเปลี่ยนแค่ไม่กี่บรรทัด

Phase 2 ต้องการ edit tool ที่ประหยัด token สำหรับการแก้ไฟล์ทั่วไปใน agent loop

## Decision

ใช้ **`str_replace`** (exact match + uniqueness check + optional `replace_all`) เป็น **primary edit tool** คู่กับ:

- `read_file` — inspect ก่อนแก้ (พร้อมเลขบรรทัด)
- `write_file` — สร้างไฟล์ใหม่ หรือ fallback หลัง str_replace ล้มเหลวหลายครั้ง

Implementation: [`plugins/tools/strreplace/str_replace.go`](../../plugins/tools/strreplace/str_replace.go)

## Alternatives Considered

| แนวทาง | Token efficiency | เหตุผลที่ไม่เลือก v1 |
|--------|------------------|----------------------|
| `str_replace` | สูงสุด (~1-5% ของไฟล์) | **เลือกใช้** |
| `apply_patch` (unified diff) | ดี (~5-15%) | retry rate สูง, fragile line offsets |
| line-range edit | ปานกลาง | line numbers stale หลัง edit หลายรอบใน session เดียว |
| fuzzy matching (Aider/Cline) | ดี (ลด retry) | เสี่ยง silent wrong edit ใน Go (imports, struct tags, string literals) |
| `write_file` (full overwrite) | ต่ำสุด (100%) | เก็บเป็น fallback เท่านั้น |

## Consequences

**ข้อดี**

- ประหยัด output token ~90-95% สำหรับ typical edit (เช่น แก้ 8 บรรทัดในไฟล์ 400 บรรทัด: ~80-120 tokens แทน ~1,600)
- ตรงกับ industry practice: Cursor `StrReplace`, Anthropic `str_replace_based_edit_tool`, OpenHands `str_replace_editor`
- แยก concern ชัดเจนใน plugin architecture — ไม่ต้องสร้าง monolithic editor

**ข้อเสีย / trade-offs**

- Model ต้อง `read_file` ก่อนและ copy `old_string` ให้ตรง whitespace/indentation
- ไม่รองรับ multi-hunk atomic edit ใน tool เดียว (defer เป็น `apply_patch` v2)
- Tab/space mismatch แก้ด้วย `expandTabs` เท่านั้น ไม่มี fuzzy cascade ใน v1

## Tool Contract

### Schema

```json
{
  "name": "str_replace",
  "required": ["path", "old_string", "new_string"],
  "properties": {
    "path": "string — path ของไฟล์",
    "old_string": "string — ข้อความเดิม exact match",
    "new_string": "string — ข้อความใหม่ (ว่าง = ลบ)",
    "replace_all": "boolean — default false"
  }
}
```

### Semantics

| `replace_all` | matches | behavior |
|---------------|---------|----------|
| `false` | 0 | error + hint re-read |
| `false` | 1 | replace |
| `false` | >1 | error + line numbers |
| `true` | ≥1 | replace all |
| `true` | 0 | error |

### Implementation details

1. อ่านไฟล์จาก disk ทันทีใน `Execute()` — ไม่ trust conversation context
2. Normalize: `\r\n` → `\n`, `\t` → 4 spaces (ก่อน match)
3. Preserve line ending style และ file mode เมื่อเขียนกลับ
4. Success response: snippet ±4 บรรทัดรอบจุดแก้ พร้อมเลขบรรทัด (format เดียวกับ `read_file`)

### Error examples

```
# 0 matches
old_string not found in main.go. อ่านใหม่ด้วย read_file ก่อนลองอีกครั้ง

# >1 matches
old_string matched 3 times at lines [14 52 103]. เพิ่มบริบทใน old_string หรือตั้ง replace_all=true
```

## Token analysis (ตัวอย่าง)

แก้ nil-check ในไฟล์ Go 400 บรรทัด (~8 บรรทัดเปลี่ยน):

| Tool | Output tokens (ประมาณ) |
|------|------------------------|
| `write_file` | ~1,600 |
| `str_replace` | ~80-120 |

Input tokens คล้ายกัน (model ยังต้องอ่านไฟล์) แต่ **output tokens ใน tool-calling agents มีต้นทุนสูงกว่า** — `str_replace` ลดส่วนนี้ได้มาก

## v2 (defer)

- Line-trimmed fallback (2nd pass ถ้า exact match fail)
- `insert` at line (เพิ่ม import บนสุด)
- `apply_patch` (multi-file atomic refactor)

## References

- Cursor `StrReplace` tool
- Anthropic `str_replace_based_edit_tool` (computer-use demo)
- OpenHands `str_replace_editor`
- SWE-Edit / AdaEdit papers — search-replace accuracy > unified diff สำหรับ LLM code editing
