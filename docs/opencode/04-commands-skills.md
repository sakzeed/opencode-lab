# บทที่ 4: Commands & Skills

สองอย่างนี้หน้าตาคล้ายกัน (Markdown + frontmatter) แต่ **คนละบทบาทโดยสิ้นเชิง**

| | **Commands** | **Skills** |
|---|---|---|
| ใครเรียก | **คุณ** พิมพ์ `/review` | **โมเดล** เรียกเองเมื่อเห็นว่าเกี่ยว |
| เก็บที่ | `commands/<name>.md` | `skills/<id>/SKILL.md` |
| เนื้อหา | prompt template (แทนที่ด้วย argument) | คำสั่ง/ขั้นตอนทำงาน + ไฟล์ประกอบ |
| โหลดเมื่อ | ตอนคุณเรียก | ตอนโมเดลขอ (`skill` tool) |
| ใช้ทำ | งานที่ทำซ้ำและอยากเรียกสั้น ๆ | องค์ความรู้ที่อยากให้โผล่เมื่อจำเป็น |

จำง่าย ๆ: **command = ทางลัดของคน / skill = ความรู้ของ agent**

## ส่วนที่ 1: Commands

### 1.1 ของที่มีอยู่จริงบนเครื่องนี้

```sh
opencode api get /api/command
```
```
3 commands
- init   | guided AGENTS.md setup
- review | review changes [commit|branch|pr], defaults to uncommitted
- goal   | keep working until an objective is met [pause|resume|clear]
```

ทั้งสามเป็น built-in — ถ้าคุณสร้าง `~/.config/opencode/commands/review.md` จะ **ทับ** built-in ตัวนั้นทันที

### 1.2 รูปแบบ Markdown (แนะนำ)

```md
<!-- ~/.config/opencode/commands/review.md -->
---
description: ตรวจโค้ดที่แก้ล่าสุด
agent: plan
---

ตรวจ $ARGUMENTS หาบั๊กและ test ที่ขาด รายงานบั๊กก่อน
```

- **body = template** (ห้ามใส่ `template` ใน frontmatter)
- วางที่ `~/.config/opencode/commands/` (global) หรือ `.opencode/commands/` (โปรเจกต์)
- path ซ้อน → ชื่อมี `/`: `.opencode/commands/team/review.md` → `/team/review`
- โฟลเดอร์ `command/` (เอกพจน์) ยังถูกค้นหา แต่ใช้ `commands/` สำหรับไฟล์ใหม่

### 1.3 Arguments 4 แบบ

| แบบ | ตัวอย่าง invocation | ผลลัพธ์ |
|---|---|---|
| `$ARGUMENTS` | `/review src/auth.ts` | ทั้งสตริงดิบ `src/auth.ts` |
| `$1`, `$2` | `/check src/auth.ts "error handling"` | `src/auth.ts` / `error handling` (quote ถูกถอด) |
| position สูงสุด | `/compare api stable branch` (template `Compare $1 with $2`) | `Compare api with stable branch` |
| ไม่มี placeholder | `/explain src/cache.ts` | ต่อท้ายหลังบรรทัดว่างให้อัตโนมัติ |

รายละเอียดสำคัญ: **position สูงสุดกิน arguments ที่เหลือทั้งหมด** และ position ที่ไม่มีจะกลายเป็นสตริงว่าง

### 1.4 Shell block — ทรงพลังและอันตราย

```md
ตรวจ diff นี้:

!`git diff --stat && git diff`
```

`!` + backtick = รันคำสั่ง shell **แล้วแทรก output** ก่อนส่ง prompt (argument ถูกแทนก่อน shell รัน)

> ⚠️ **อันตราย:** shell block รัน **นอก** permission flow ของ agent — คือ *ไม่ถาม* ตามกฎ `shell` ที่เราตั้งในบทที่ 2 และ **ห้ามใส่ argument ที่ไม่น่าเชื่อถือลงใน shell block** (`/history $1` กับ `git log -$1` = command injection)

### 1.5 Fields ทั้งหมด

| Field | ค่า | หมายเหตุ |
|---|---|---|
| `description` | string | โชว์ในรายการ command |
| `agent` | agent id | สลับ agent **ก่อน** ส่ง prompt เช่น `plan` |
| `model` | `provider/model#variant` | ทับทุกอย่าง (ดูกฎข้อ 1.7) |
| `subagent` | `true`/`false` | `true` = รันใน child session เบื้องหลัง |
| `subtask` | — | deprecated alias ของ `subagent` (ถ้าใส่ทั้งคู่ `subagent` ชนะ) |

ใน JSON config ใช้ `commands.<name>` และ **ต้องมี `template`**:

```jsonc
{
  "commands": {
    "audit": {
      "template": "Audit $ARGUMENTS.",
      "agent": "general",
      "subagent": true
    }
  }
}
```

### 1.6 Background (`subagent: true`)

```
parent (ทำงานต่อได้) ──► child session ──► ส่งผลลัพธ์กลับเมื่อเสร็จ
```

| ค่า | พฤติกรรม |
|---|---|
| `true` | ใช้ child เสมอ แม้ agent นั้นเป็น `primary` |
| `false` | ใช้ session ปัจจุบันเสมอ แม้ agent เป็น `subagent` |
| ไม่ระบุ | ใช้ child เฉพาะเมื่อ agent ที่เลือกมี `mode: subagent` |

โมเดลของ child: `command.model` → `agent.model` → `parent.model`

### 1.7 กฎการเลือกโมเดล (เรียงตามสิทธิ์)

1. `model` ของ command ชนะทุกอย่าง
2. ถ้าไม่มี → โมเดลของ agent ที่ command เลือก
3. ถ้าไม่มีอีก → โมเดลของ session ปัจจุบัน

### 1.8 กับดัก

- **`@path` ใน template ไม่ถูกขยาย** — มันเป็นข้อความธรรมดา ต้องแนบไฟล์ผ่าน composer ตอนเรียก
- **โปรเจกต์ทับ global** และไฟล์ที่ใกล้กว่าทับไฟล์ที่ไกลกว่า
- **แก้ไฟล์แล้วมีผลทันที** ไม่ต้อง restart

## ส่วนที่ 2: Skills

### 2.1 ของจริงบนเครื่องนี้

```sh
opencode api get /api/skill
# 18 skills — opencode, report, brainstorming, systematic-debugging, ...
```

สังเกตว่า 14 ตัวมาจาก **ปลั๊กอิน Superpowers** ที่เราเพิ่งติดตั้ง ไม่ได้อยู่ใน `~/.config/opencode/skills` เลย — นี่คือบทเรียนสำคัญ: skill มาจากได้ 3 ทาง คือ **ไฟล์ / config `skills` / plugin**

### 2.2 โครงสร้าง

```
.opencode/skills/
└── git-release/
    ├── SKILL.md
    ├── scripts/changelog.ts
    └── references/release-policy.md
```

```md
---
name: Git Release
description: Prepare release notes, version bumps, and GitHub releases
---

## Workflow
1. อ่าน `references/release-policy.md`
2. สรุปการเปลี่ยนแปลงตั้งแต่ tag ก่อนหน้า
3. เสนอ version bump ก่อนแก้ไฟล์
```

- path ที่เขียนใน skill **relative กับโฟลเดอร์ของ `SKILL.md`**
- รองรับ 2 แบบ: `skills/review.md` (flat) หรือ `skills/<id>/SKILL.md` (directory — แนะนำ เพราะมี base dir ให้ไฟล์ประกอบ)
- ชื่อไฟล์ต้องเป็น `SKILL.md` เป๊ะ ๆ และ **ไม่ค้นลึกเข้าไปในโฟลเดอร์ที่มี `SKILL.md` แล้ว**

### 2.3 ค้นหาอัตโนมัติจากที่ไหน

| Scope | ที่อยู่ |
|---|---|
| Global | `~/.config/opencode/skills` |
| Global (compat) | `~/.claude/skills`, `~/.agents/skills` |
| Project | `.opencode/skills` |
| Project (compat) | `.claude/skills`, `.agents/skills` |

Project scope ไล่จาก cwd ขึ้นไปถึง project root, ข้าม hidden/node_modules, ลึกสุด 6 ชั้น

### 2.4 ID มาจาก path ไม่ใช่ `name`

| ไฟล์ | ID |
|---|---|
| `<source>/git-release.md` | `git-release` |
| `<source>/git-release/SKILL.md` | `git-release` |
| `<source>/teams/release/SKILL.md` | `release` |

ID **case-sensitive** และควรเป็น kebab-case — ถ้า skill "ไม่เจอ" ให้สงสัยข้อนี้ก่อนเสมอ (เช่น `name: Git Release` แต่ ID คือ `release`)

### 2.5 Frontmatter ที่มีผลจริง

| Field | ผล |
|---|---|
| `description` | ใช้โชว์ให้โมเดลเห็น — **ไม่มี description = ไม่ถูกโฆษณา** |
| `name` | แค่ชื่อแสดงผล |
| `metadata.opencode/autoinvoke: false` | ซ่อนจากรายการของโมเดล แต่ยังโหลดด้วย ID ได้ |
| `disable-model-invocation: true` | ผลเดียวกับข้างบน (ถ้าใส่ทั้งคู่ ตัวบนชนะ) |

### 2.6 ลำดับความสำคัญ (ต่ำ → สูง)

```
1. built-in skills
2. .claude/skills
3. .agents/skills
4. ~/.config/opencode/skills
5. project .opencode/skills (ใกล้ cwd ชนะ)
6. skills ที่ประกาศใน config (ตามลำดับ array)
```

### 2.7 กลไกโหลด (เข้าใจแล้วจะ debug ง่ายมาก)

```
ทุก step ของโมเดล:
  OpenCode ส่งรายการ skill ที่อนุญาต → แค่ [ID, name, description]
                                        (ไม่ส่งเนื้อหา!)

โมเดลเห็นว่าเกี่ยว → เรียก skill tool { id: "..." }
  → ตรวจ permission action "skill"
  → แทรก body (ตัด frontmatter)
  → ให้ base dir + path ของไฟล์ประกอบสูงสุด 10 ไฟล์
```

ไฟล์ประกอบ **ไม่ถูกโหลดอัตโนมัติ** — agent ต้อง `read` เองตามที่ skill สั่ง

คุณโหลดเองได้ด้วยการพิมพ์ `@skill-id` ใน prompt

### 2.8 Permission ของ skill (ต่อจากบทที่ 2)

```jsonc
{
  "permissions": [
    { "action": "skill", "resource": "*", "effect": "allow" },
    { "action": "skill", "resource": "internal-*", "effect": "deny" },
    { "action": "skill", "resource": "experimental-*", "effect": "ask" }
  ]
}
```

| Effect | ผล |
|---|---|
| `allow` | โฆษณา + โหลดได้เลย |
| `ask` | โฆษณา แต่ถามก่อนโหลด |
| `deny` | **ซ่อนจากโมเดล** + ปฏิเสธการโหลด |

### 2.9 กับดัก

1. **ชื่อโฟลเดอร์ ≠ ID** ถ้าใช้ `SKILL.md` — ID คือชื่อโฟลเดอร์ ไม่ใช่ `name` ใน frontmatter
2. **ไม่มี description = มองไม่เห็น** (แต่ยังโหลดด้วย `@id` ได้)
3. **`SKILL.md` ต้องตัวพิมพ์ใหญ่เป๊ะ** — `skill.md` ไม่ถูกค้น
4. **ID ซ้ำ = ตัวหลังชนะ** — ระวังตอนติดตั้ง plugin ที่มี skill ชื่อชนของเดิม
5. **HTTP catalog** ต้องมี `index.json` และเพิ่ม `version` เมื่อไฟล์เปลี่ยน ไม่งั้น cache ค้าง

## แบบฝึกหัด 3 ข้อ

```sh
# 1) ดู command ที่ลงทะเบียนอยู่ (ก่อน/หลังสร้างไฟล์)
opencode api get /api/command

# 2) สร้าง command ง่าย ๆ แล้วดูว่ามันโผล่ทันที
mkdir -p ~/.config/opencode/commands
printf -- '---\ndescription: อธิบายโค้ดสั้น ๆ\n---\n\nอธิบาย $ARGUMENTS แบบสั้น กระชับ\n' \
  > ~/.config/opencode/commands/explain.md
opencode api get /api/command | grep -o '"explain"'

# 3) ดูว่า skill แต่ละตัวมี description ไหม (ตัวไม่มี = โมเดลมองไม่เห็น)
opencode api get /api/skill | python3 -c '
import json,sys
for s in json.load(sys.stdin)["data"]:
    print(("OK " if s.get("description") else "NO ") + s["id"])
'
```
