# บทที่ 3: Agents & Subagents

## 3.1 Agent คืออะไรกันแน่

Agent = **4 อย่างรวมกันเป็นโปรไฟล์เดียว**

```
agent = system prompt + model preference + permissions + display details
```

ไม่ใช่ "โหมด" หรือ "คำสั่ง" — มันคือ *ตัวตน* ที่กำหนดว่าใครคิด ใช้โมเดลไหน และแตะอะไรได้บ้าง

## 3.2 Agent ที่มีให้จริงบนเครื่องนี้

```sh
opencode debug agents
```

ผลจริง (สรุป):

| id | mode | hidden | จำนวนกฎ | หน้าที่ |
|---|---|---|---|---|
| `build` | primary | ❌ | 33 | agent เริ่มต้น ทำงานเต็มรูปแบบ |
| `plan` | primary | ❌ | 35 | สำรวจ/วางแผน **ห้ามแก้ไฟล์โปรเจกต์** |
| `explore` | subagent | ❌ | 44 | ค้นหา/อ่านโค้ด ไม่แก้อะไร (deny-all + allow เฉพาะอ่าน) |
| `general` | subagent | ❌ | 34 | งานวิจัยหลายขั้น **ห้ามถาม ห้ามเรียก subagent ต่อ** |
| `compaction` | primary | ✅ | 32 | งานเบื้องหลัง (บีบ context) |
| `title` | primary | ✅ | 33 | ตั้งชื่อ session |
| `summary` | primary | ✅ | 33 | สรุป session |

**อ่านตัวเลขนี้ให้เป็น:** `build` มี 33 กฎ ไม่ใช่เพราะ built-in เยอะ แต่เพราะ **กฎของคุณต่อท้ายเข้าไป** — ถ้าเปิดดูจะเห็น `.env` โผล่ซ้ำ 2 รอบ:

```json
[ { "action": "*", "resource": "*", "effect": "allow" },        // base
  { "action": "read", "resource": "*.env", "effect": "ask" },   // base
  ...
  { "action": "read", "resource": "*.env", "effect": "ask" },   // ← ของคุณ (บทที่ 2)
  { "action": "shell", "resource": "sudo *", "effect": "deny" } // ← ของคุณ
]
```

นี่คือหลัก **"permission rules append"** จาก docs: global → agent → (กฎของ client/harness) เรียงต่อกัน แล้วใช้ *last match wins* — ดังนั้นกฎของ agent มีสิทธิ์ refine กฎ global ได้ แต่ **ลบกฎ global ไม่ได้**

## 3.3 Modes: ที่อยู่ของ agent

| Mode | รันที่ไหน | ใช้เมื่อ |
|---|---|---|
| `primary` | session หลัก (คนคุยด้วย) | ค่าเริ่มต้นของ agent ใหม่ |
| `subagent` | child session ผ่าน tool `subagent` | งานที่อยากแยก context |
| `all` | ได้ทั้งสองแบบ | agent อเนกประสงค์ |

```jsonc
{ "default_agent": "plan" }   // เปลี่ยน agent เริ่มต้นของ session ใหม่
```

ข้อควรรู้: `default_agent` **ไม่เปลี่ยน agent ของ session ที่มีอยู่แล้ว** และ `model` ของ session ก็เก็บแยก — เลือก agent ไม่เท่ากับเปลี่ยนโมเดล

## 3.4 Subagent ทำงานอย่างไร (สำคัญที่สุดของบทนี้)

```
Parent session ──► subagent tool ──► Child session (context ใหม่ทั้งหมด)
      ▲                                      │
      └────────── report เมื่อ child idle ◄───┘
```

4 คุณสมบัติที่ต้องจำ:

1. **Fresh context** — child ไม่เห็นประวัติสนทนาของ parent ต้องส่ง context ไปให้ครบใน prompt
2. **Background เสมอ** — parent ไม่บล็อกรอ แต่ได้ report เมื่อ child ว่าง และได้อีกครั้งถ้ามีงานรอบใหม่
3. **สิทธิ์แยกกัน** — child ใช้ permission **ของตัวเอง** ไม่ใช่ subset ของ parent (สำคัญมาก: subagent ไม่ได้ "อ่อนกว่า" โดยอัตโนมัติ)
4. **ใครเรียกใครได้ ถูกคุมที่ parent** — ด้วย action `subagent`:

```jsonc
{
  "agents": {
    "orchestrator": {
      "permissions": [
        { "action": "subagent", "resource": "*", "effect": "deny" },
        { "action": "subagent", "resource": "reviewer", "effect": "allow" }
      ]
    }
  }
}
```

`general` ถูกตั้งมาแบบนี้อยู่แล้ว (ห้ามเรียก subagent ต่อ) — กันการระเบิดเป็นต้นไม้

## 3.5 สร้าง agent เอง: Markdown คือวิธีที่อ่านง่ายที่สุด

```md
<!-- ~/.config/opencode/agents/reviewer.md -->
---
description: Reviews changes for correctness and regressions
mode: subagent
model: opencode/big-pickle
permissions:
  - action: edit
    resource: "*"
    effect: deny
  - action: shell
    resource: "*"
    effect: deny
---

Review the current changes. List findings in severity order with file and line references.
Do not modify files.
```

- **frontmatter** = config, **body** = `system` prompt
- วางที่ `~/.config/opencode/agents/` (global) หรือ `.opencode/agents/` (โปรเจกต์)
- path ซ้อนกลายเป็น ID: `.opencode/agents/team/reviewer.md` → `team/reviewer`
- ทางเลือก: นิยามใน `opencode.jsonc` ภายใต้ `agents` (ใช้เมื่ออยากให้ทุกอย่างอยู่ในไฟล์เดียว)

## 3.6 Options ที่มีผลจริง

| Field | ค่า | ใช้ทำอะไร |
|---|---|---|
| `description` | string | **จำเป็นสำหรับ subagent** — โมเดลใช้เลือกว่าจะเรียกตัวไหน |
| `mode` | `primary`/`subagent`/`all` | ที่อยู่ |
| `model` | `provider/model#variant` | child ไม่ตั้ง = ใช้โมเดลของ parent |
| `system` | string | แทน base prompt ของ provider (instruction อื่นยังถูกเพิ่ม) |
| `permissions` | rules | ordered, last wins |
| `steps` | number | เพดานจำนวน step ของโมเดล แล้วบังคับสรุปเป็นข้อความ |
| `hidden` | bool | ซ่อนจาก listing — **ไม่ใช่ security** |
| `color` | `#ff6b6b` | สีใน UI |
| `disabled` | bool | ปิด agent นั้น ณ จุดนั้นของ config |
| `request` | headers/body | ⚠️ V2 เก็บไว้แต่ **ยังไม่ส่งจริง** |

## 3.7 กับดักที่ต้องระวัง

1. **`hidden: true` ≠ ปลอดภัย** — มันแค่ซ่อนจากรายการ ต้องใช้ `permissions` คุมพฤติกรรม
2. **V1 fields เลิกใช้** — `permission`, `tools`, `temperature`, `maxSteps`, `disable`, `prompt` อย่าเขียนใน V2 ให้ใช้ `permissions`, `steps`, `disabled`, `system`
3. **ทับ built-in ด้วย ID เดียวกัน** — `agents.build.permissions` จะ *ต่อท้าย* กฎเดิม ไม่ใช่แทนที่
4. **`plan` ไม่ได้แก้อะไรไม่ได้เลย** — มันเขียนไฟล์ plan ใต้ `~/.opencode/plan` ได้ และ shell ยังถูกคุมด้วย permission
5. **`explore` 44 กฎ** เพราะสร้างจาก deny-all แล้ว allow เฉพาะ read/glob/grep/shell/webfetch — อ่าน `.env` ยัง ask

## 3.8 แบบฝึกหัด 3 ข้อ

```sh
# 1) เทียบสิทธิ์ plan กับ explore แบบเห็นชัด ๆ
opencode debug agents | python3 -c '
import json,sys
for a in json.load(sys.stdin):
    if a["id"] in ("plan","explore"):
        print("==",a["id"],len(a["permissions"]))
        for r in a["permissions"][-4:]: print("  ",r["action"],r["resource"],"->",r["effect"])
'

# 2) ลองใช้ agent ที่มีอยู่กับงานจริง (ไม่ต้องมีไฟล์ใหม่)
cd /tmp/opencode/config-lab && opencode run --agent plan 'List the config files here and explain the precedence. Do not edit anything.'

# 3) ดูว่า id/mode ที่เราจะสร้างชนกับของเดิมไหม
ls ~/.config/opencode/agents/ 2>/dev/null || echo "(ยังไม่มีโฟลเดอร์ agents)"
```
