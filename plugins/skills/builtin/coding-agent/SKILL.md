---
name: coding-agent
description: สอน agent วิธีเพิ่ม tool, provider, prompt หรือ runner ตาม architecture ของ coding-agent project นี้ ใช้เมื่อผู้ใช้ขอเพิ่ม feature ใหม่ใน repo
---

# Coding Agent Architecture Guide

อ่าน [AGENTS.md](../../../AGENTS.md) ก่อนทำงาน

## กฎหลัก

1. **Contract** → core package (`agent/`, `types/`, `llm/`, `tools/`, `config/`, `plugin/`)
2. **Implementation** → `plugins/` เท่านั้น
3. ลงทะเบียนทุก plugin ใน `plugins/builtin/builtin.go`
4. Feature ที่กระทบ architecture → เขียน ADR ใน `docs/adr/` ก่อน

## เพิ่ม tool

1. สร้าง `plugins/tools/mytool/my_tool.go`
2. Implement `tools.Tool` (`Name`, `Definition`, `Execute`)
3. เพิ่ม `Plugin` struct + `Register()` เรียก `plugin.RegisterTools()`
4. Append ใน `builtin.Default`

## เพิ่ม provider

1. สร้าง `plugins/providers/myprovider/myprovider.go`
2. Implement `llm.Provider` (`Complete`)
3. เพิ่ม constant ใน `config/config.go`
4. Register ใน `builtin.Default`

## แก้ไฟล์

- อ่านก่อนแก้ด้วย `read_file`
- แก้ไฟล์เดิมด้วย `str_replace` (primary)
- สร้างไฟล์ใหม่ด้วย `write_file`
