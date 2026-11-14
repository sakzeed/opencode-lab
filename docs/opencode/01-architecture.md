# บทเรียน OpenCode (V2)

## แผนหลักสูตร 7 บท

| บท | เรื่อง | ไฟล์/คำสั่งที่แตะ |
|---|---|---|
| 1 | สถาปัตยกรรม client–server + แผนที่ไฟล์ | `service.json`, `opencode.db`, `service status` |
| 2 | Configuration & precedence | `opencode.jsonc` 3 ระดับ, `{env:...}`, `{file:...}` |
| 3 | Agents & Subagents | `agents`, `mode`, `default_agent` |
| 4 | Commands & Skills | `/command`, `.opencode/skills/` |
| 5 | Permissions & Policies | `permissions`, `policies` |
| 6 | MCP servers & Plugins | `mcp.servers`, `plugins` |
| 7 | CLI/TUI, API, SDK | `cli.json`, `opencode api`, `@opencode/client` |

---

## บทที่ 1: OpenCode ไม่ใช่แค่ TUI — มันคือ client–server

## 1.1 หลักคิดที่ต้องเข้าใจก่อน

หลายคนคิดว่า OpenCode คือโปรแกรมใน terminal ตัวเดียว แต่จริง ๆ แยกเป็น 2 ส่วน:

```
┌─────────────┐   HTTP (127.0.0.1:49374)   ┌──────────────────────────┐
│  Client     │ ─────────────────────────► │  Server (background)     │
│  TUI / Web  │ ◄───────────────────────── │  sessions, config,       │
│  Desktop    │        streaming           │  plugins, tool execution │
│  สคริปต์คุณ │                            └──────────────────────────┘
└─────────────┘
```

ลองพิสูจน์บนเครื่องนี้:

```sh
opencode service status
# http://127.0.0.1:49374        ← ที่อยู่ของ server ที่กำลังรัน

opencode api get /api/info
# {"version":"2.3.0","pid":11624,"hostname":"localhost",
#  "urls":["http://127.0.0.1:49374"],
#  "paths":{"tmp":"/tmp/opencode"},
#  "capabilities":{"persistentPty":true}}
```

**ผลที่ตามมาที่ใช้จริง:**

1. **หลาย client ใช้ session เดียวกันได้** — เปิด TUI ทิ้งไว้ แล้วต่อ web ด้วย `opencode pair` หรือยิง API จากสคริปต์ ก็เห็นประวัติเดียวกัน
2. **แก้ config ไม่ต้อง restart** — server เฝ้าไฟล์อยู่ พอ save `opencode.jsonc` มัน reload agents/commands/skills/MCP/plugins ให้เอง (ถ้าไม่ติดใช้ `/reload` ใน TUI)
3. **ถ้าค้าง ให้ restart แค่ service ไม่ใช่ลบข้อมูล** — `opencode service restart` แล้วเช็คด้วย `opencode service status`
4. **ทุกอย่างผ่าน API ได้** — `opencode api <method> <path>` ใช้ auth flow เดียวกับ TUI

> 💡 ในสภาพแวดล้อมที่ผมทำงานอยู่ มี tool namespace `opencode` ให้เรียก session/worktree/models โดยตรง — นี่คือตัวอย่างจริงว่าสถาปัตยกรรมนี้เปิดให้ client อื่นเข้ามาใช้

## 1.2 แผนที่ไฟล์บนเครื่องคุณ (ของจริง)

```sh
~/.config/opencode/          ← config ระดับผู้ใช้ (global)
├── service.json             ← ข้อมูล service — อย่าแก้เอง
└── opencode.json(c)         ← ยังไม่มี! จะเกิดเมื่อคุณตั้งค่า

~/.local/share/opencode/     ← data ของ server
├── opencode.db              ← sessions + messages (12 MB ตอนเริ่มบท, SQLite)
├── opencode.db-wal          ← write-ahead log (ปกติ)
├── log/opencode.log         ← จุดแรกที่ต้องดูเวลา debug
├── repos/                   ← ข้อมูล git repo ที่เกี่ยวข้อง
└── shell/                   ← สถานะ shell ที่ agent ใช้
```

**ข้อควรระวัง:** `opencode.db` คือประวัติสนทนาทั้งหมดของคุณ — เวลา debug ห้ามลบ/แก้ด้วยมือ ให้ backup ก่อนเสมอ

## 1.3 Config มี 3 ระดับ และทับกันตามลำดับ

```sh
1. ~/.config/opencode/opencode.json(c)          ← global (ทุกโปรเจกต์)
2. <โปรเจกต์>/opencode.json(c)                  ← ไล่จากโฟลเดอร์บนสุดลงมาถึง cwd
3. <โปรเจกต์>/.opencode/opencode.json(c)        ← ทับ direct config ทุกไฟล์
4. /etc/opencode/opencode.json(c)               ← system (สูงสุด, ผู้ใช้ทับไม่ได้)
```

กติกาสำคัญ 2 ข้อที่คนพลาดบ่อย:

- **ค้นหาไล่ขึ้นไปจนถึง `/`** ไม่หยุดที่ root ของ git repo → ถ้ารัน opencode ใน `/tmp/opencode` มันจะอ่าน config ที่ `/tmp`, `/` ด้วย
- **`.opencode/` ชนะ `opencode.json` ทุกกรณี** ถ้าจะวาง config ไว้ที่เดียว ให้เลือกแบบเดียวทั้ง tree

## 1.4 อย่าสับสน: `opencode.json` vs `cli.json`

| ไฟล์ | คุมอะไร | มี project-level ไหม |
|---|---|---|
| `opencode.json(c)` | model, agents, permissions, MCP, plugins — **พฤติกรรม agent** | ✅ |
| `~/.config/opencode/cli.json` | ธีม, keybind, การแสดงผล — **หน้าตา terminal** | ❌ global เท่านั้น |
| `~/.config/opencode/service.json` | ข้อมูล service ที่ระบบเขียนเอง | ❌ |

บนเครื่องนี้ยังไม่มี `cli.json` เลย (ว่าง) — ถ้าอยากปรับธีม/keybind ให้แก้ไฟล์นั้น หรือกด `Ctrl+P` → **Open settings** ใน TUI

## 1.5 ตัวอย่าง config แรกที่ควรมี

```jsonc
// ~/.config/opencode/opencode.jsonc
{
  "$schema": "https://opencode.ai/config.json",

  // โมเดลเริ่มต้น (รูปแบบ provider/model)
  "model": "opencode/big-pickle",

  // ต้องอนุมัติก่อนรันคำสั่งที่อันตราย
  "permissions": [
    { "action": "shell", "resource": "git push *", "effect": "ask" }
  ],

  // ไฟล์ context เพิ่มเติมนอกเหนือจาก AGENTS.md
  "instructions": ["CONTRIBUTING.md"]
}
```

`$schema` ใส่ไว้เถอะ — editor จะ autocomplete และบอกทันทีว่าใส่ field ผิด

## 1.6 แบบฝึกหัด 3 ข้อ

```sh
# 1) พิสูจน์ว่าเป็น client-server จริง: หา pid ของ server
opencode api get /api/info          # เอา pid ไปเทียบ
ps -p <pid> -o pid,etime,args       # process นี้รันมานานแค่ไหน

# 2) ดูว่า config มีอะไรถูกโหลดอยู่ (ควรว่าง เพราะยังไม่มีไฟล์)
cat ~/.config/opencode/service.json

# 3) ดู log แบบกรองเฉพาะ client vs server
grep -c "role=server" ~/.local/share/opencode/log/opencode.log
grep -c "role=cli"    ~/.local/share/opencode/log/opencode.log
```
