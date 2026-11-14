# บทที่ 2: Configuration & Precedence

เราจะไม่ท่องจำ field ทั้งหมด (มีเป็นร้อย) แต่จะเข้าใจ **"config มาจากไหน ใครชนะใคร"** เพราะนั่นคือ 90% ของปัญหาจริง — แล้วค่อยดู field ที่ใช้บ่อย 10 ตัว

## 2.1 เครื่องมือสำคัญที่เพิ่งมีให้ใช้: `opencode debug config`

คำสั่งนี้บอกว่า **"ในโฟลเดอร์นี้ OpenCode โหลด config จากที่ไหนบ้าง เรียงตามลำดับ"**

```sh
cd /root && opencode debug config
```
```json
[
  { "type": "directory", "path": "/root/.config/opencode" },
  { "type": "directory", "path": "/root/.opencode" }
]
```

ตอนนี้ยังว่าง (ยังไม่มี `opencode.json` ที่ไหนเลย) — ถ้าเห็น `"type": "document"` นั่นคือไฟล์ config ที่ถูกโหลดจริง พร้อมเนื้อหาที่ parse แล้ว

## 2.2 ทดลอง precedence ด้วยมือ (ทำจริงบนเครื่องนี้แล้ว)

ผมสร้างห้องแล็บไว้ให้แล้วที่ `/tmp/opencode/config-lab`:

```
config-lab/
├── opencode.jsonc                  ← model + instructions (ระดับ repo)
└── apps/web/
    ├── opencode.jsonc              ← instructions (ระดับ package)
    └── .opencode/
        └── opencode.jsonc          ← formatter (ระดับ .opencode)
```

รันจาก `apps/web`:

```sh
cd /tmp/opencode/config-lab/apps/web && opencode debug config
```

ผลลัพธ์จริง (ย่อ):

```json
[
  { "type": "directory", "path": "/root/.config/opencode" },

  { "type": "document",
    "path": "/tmp/opencode/config-lab/opencode.jsonc",
    "info": {
      "model": { "providerID": "opencode", "model": "big-pickle" },
      "instructions": ["root.md"]
    } },

  { "type": "document",
    "path": "/tmp/opencode/config-lab/apps/web/opencode.jsonc",
    "info": { "instructions": ["web.md"] } },

  { "type": "document",
    "path": "/tmp/opencode/config-lab/apps/web/.opencode/opencode.jsonc",
    "info": { "formatter": true } },

  { "type": "directory", "path": "/tmp/opencode/config-lab/apps/web/.opencode" }
]
```

**อ่านลำดับนี้ให้เป็น:** รายการบน→ล่าง = ลำดับโหลด **ตัวล่างสุดมีสิทธิ์ชนะ** เมื่อ key ชนกัน

```
global config
    ↓
opencode.jsonc ที่ไกลสุด (repo root)
    ↓
opencode.jsonc ที่ใกล้สุด (cwd)
    ↓
.opencode/opencode.jsonc  ← ชนะ direct config ทุกไฟล์
    ↓
/etc/opencode/            ← system/managed ชนะทุกอย่าง (ผู้ใช้ทับไม่ได้)
```

สังเกต 2 เรื่องจาก output จริง:

- `model: "opencode/big-pickle"` ถูก normalize เป็น `{providerID, model}` — รูปแบบที่ใช้คือ `provider/model`
- config ระดับ `apps/web` **ไม่ได้** มี `model` ติดมาด้วย นี่คือสิ่งที่พิสูจน์ว่าแต่ละ document ถูกอ่านแยก แล้วค่อย merge ตอนใช้งาน ไม่ใช่ copy ต่อกัน

## 2.3 กับดัก 5 ข้อที่คนพลาดบ่อย

| กับดัก | ความจริง |
|---|---|
| "วาง config ไว้ที่ root ของ git repo แล้วจบ" | OpenCode ไล่ขึ้นไปถึง `/` เลย — config ที่ `/tmp` หรือ `/` ก็มีผล |
| "มีทั้ง `opencode.json` และ `.opencode/opencode.jsonc` น่าจะรวมกัน" | รวมก็จริง แต่ `.opencode/` **ชนะ** ทุกไฟล์ direct — ใช้แบบเดียวทั้ง tree ดีที่สุด |
| "monorepo วาง config ที่ package ได้" | ได้ แต่ระวัง config ของ package พี่น้องกันไม่ทับกัน (คนละ subtree) |
| "รันใน `/tmp` ไม่เป็นไร" | MOTD ของคุณเอง: `/tmp` เป็น tmpfs — config หายเมื่อ host เคลียร์ |
| "`cli.json` วางในโปรเจกต์ได้" | ไม่ได้ — **CLI settings เป็น global เท่านั้น** (`~/.config/opencode/cli.json`) |

## 2.4 JSON vs JSONC — เลือก JSONC

นามสกุล `.jsonc` ใส่ comment และ trailing comma ได้ ซึ่งมีค่ามากเวลา config โตขึ้น:

```jsonc
// ~/.config/opencode/opencode.jsonc
{
  "$schema": "https://opencode.ai/config.json",

  // โมเดลเริ่มต้น
  "model": "opencode/big-pickle",

  // ต้องอนุมัติก่อนคำสั่งเสี่ยง
  "permissions": [
    { "action": "shell", "resource": "git push *", "effect": "ask" },
  ],
}
```

`$schema` ทำให้ editor ตรวจ field ผิดให้ทันที — ใส่ไว้เถอะ ไม่มีข้อเสีย

## 2.5 10 field ที่ควรรู้ (และที่เหลือไปดู schema)

| Field | ทำอะไร | หมายเหตุ |
|---|---|---|
| `model` | โมเดลเริ่มต้น `provider/model` | root config ไม่รับ `#variant` |
| `default_agent` | agent เริ่มต้นเมื่อ session ไม่ระบุ | เช่น `"build"` |
| `permissions` | กฎ allow/deny/ask ตาม tool+resource | บทที่ 5 |
| `agents` | นิยาม/ทับ agent | บทที่ 3 |
| `commands` | slash command แบบ prompt template | บทที่ 4 |
| `skills` | เพิ่มโฟลเดอร์/URL ของ skill | `.opencode/skills/` ถูกค้นหาอัตโนมัติ |
| `instructions` | ไฟล์ context เพิ่มจาก `AGENTS.md` | รับ glob และ URL |
| `mcp.servers` | MCP server local/remote | ใช้ CLI `opencode mcp add` ดีกว่าแก้มือ |
| `plugins` | โหลด plugin จาก package/โฟลเดอร์ | object form ใส่ options ได้ |
| `formatter` | format ไฟล์หลัง write/edit/patch | `true` = ใช้ built-in ที่มี |

**global-only field ที่สำคัญ:** `update` — `"notify"` (ค่าเริ่มต้น), `"auto"`, `"disable"` ค่าจาก project config จะถูกเมิน

## 2.6 Variables: `{env:...}` และ `{file:...}`

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "model": "{env:OPENCODE_MODEL}",
  "providers": {
    "anthropic": {
      "settings": { "apiKey": "{env:ANTHROPIC_API_KEY}" }
    }
  }
}
```

```jsonc
{
  "providers": {
    "openai": { "settings": { "apiKey": "{file:~/.secrets/openai-key}" } }
  }
}
```

⚠️ **กับดักเงียบ:** ถ้า env var ไม่ถูกตั้ง หรือไฟล์ไม่มี → ค่าจะกลายเป็น **empty string** โดยไม่ error คุณจึงต้องตรวจเอง:

```sh
echo "${OPENCODE_MODEL:-<<unset>>}"
ls -l ~/.secrets/openai-key
```

นี่คือวิธีเก็บ secret โดยไม่เขียนลง config และแชร์ config ให้คนอื่นได้

## 2.7 แบบฝึกหัด (แล็บอยู่แล้ว ไม่ต้องสร้างใหม่)

```sh
# 1) ดูว่า config ในแล็บโหลดยังไง
cd /tmp/opencode/config-lab && opencode debug config

# 2) เพิ่ม field ที่ระดับ repo แล้วดูว่า document ไหนเปลี่ยน
#    (แก้ /tmp/opencode/config-lab/opencode.jsonc เพิ่ม "tool_output": { "max_lines": 500 })
cd /tmp/opencode/config-lab/apps/web && opencode debug config

# 3) ดูว่า "ข้อมูลระบบ" อยู่ไหน (คนละเรื่องกับ config)
opencode debug paths
```

จากข้อ 3 คุณจะเห็นแผนที่จริง:

```
data   /root/.local/share/opencode     ← sessions, log, db
cache  /root/.cache/opencode           ← binary, ของที่โหลดซ้ำได้
config /root/.config/opencode          ← config
state  /root/.local/state/opencode     ← สถานะ service
tmp    /tmp/opencode                   ← ไฟล์ชั่วคราว (หายได้)
```

**กฎคิดง่าย ๆ:** config = ตั้งใจเขียน / state+data = ระบบเขียน ห้ามลบ / cache = ลบได้เสมอ

## 2.8 ข้อควรรู้เวลา config พัง

- **แก้แล้วไม่ต้อง restart** — server เฝ้าไฟล์และ reload ให้เอง ถ้าสงสัยให้ `/reload` ใน TUI
- **JSON ผิด = config ไม่โหลด** มักเห็นอาการ "ค่ำที่ตั้งไว้ไม่มีผล" ไม่ใช่ error ชัด ๆ → ตรวจด้วย `opencode debug config` ทันที
- **ดู log ที่ `~/.local/share/opencode/log/opencode.log`** กรอง `role=server` เพื่อดู error ตอน parse
