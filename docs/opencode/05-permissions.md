# บทที่ 5: Permissions & Policies

สองระบบนี้ดูคล้ายกันแต่ทำคนละหน้าที่ — เข้าใจความต่างแล้วจะตอบคำถาม "ทำไมบล็อกไม่ได้/ทำไมถามรัว" ได้ทันที

## 5.1 ภาพเดียวจบ: ใครคุมอะไร

```
เครื่องมือของ agent (shell, edit, read, subagent, skill, ...)
        │
        ▼
   ┌─────────────┐   allow / ask / deny    ← ตัดสินใจได้ 3 ทาง, ถามได้
   │ Permissions │
   └──────┬──────┘
          ▼
   ┌─────────────┐   allow / deny เท่านั้น  ← ไม่ถาม, บล็อกอย่างเดียว
   │  Policies   │   (ตัดสินใจหลังสุด = เพดานแข็ง)
   └──────┬──────┘
          ▼
      ดำเนินการ / "Blocked by configuration policy"
```

| | **Permissions** | **Policies** |
|---|---|---|
| ค่า effect | `allow` / `ask` / `deny` | `allow` / `deny` |
| ถามผู้ใช้ได้ไหม | ได้ | **ไม่เคย** |
| ทำงานเมื่อ | ก่อน (ต่อ agent, ต่อ session) | หลัง (เพดานสุดท้าย) |
| ใช้คุม | เครื่องมือแต่ละครั้ง | provider, tool, MCP, skill, plugin, websearch, integration |
| `--auto` มีผลไหม | มี (auto-approve `ask`) | **ไม่มี** |
| precedence | ใกล้ชนะ (global→project→agent) | **กลับด้าน** (กว้างชนะ) |

## 5.2 Permissions: กติกาที่ต้องแม่น

### รูปแบบกฎ

```jsonc
{ "action": "shell", "resource": "git push *", "effect": "ask" }
```

- **last matching rule wins** — ไม่มีกฎ "specific กว่าชนะ" ต้องเรียงกว้าง→แคบเอง
- **ถ้าไม่มีกฎตรงเลย → `ask`** (แต่ base policy มี `* → allow` อยู่แล้ว จึงไม่ค่อยเกิด)
- pattern เป็น whole-value wildcard: `*` (ครอบ `/`), `?`; shell ที่ลงท้าย ` *` ยัง match คำสั่งเปล่าด้วย

### กฎที่ระบบใส่ให้ทุก agent (base policy)

```jsonc
[
  { "action": "*", "resource": "*", "effect": "allow" },
  { "action": "read", "resource": "*.env", "effect": "ask" },
  { "action": "read", "resource": "*.env.*", "effect": "ask" },
  { "action": "read", "resource": "*.env.example", "effect": "allow" }
]
```

แล้วแต่ละ agent เติมของตัวเอง: `plan` ห้าม edit (ยกเว้น `~/.opencode/plan`), `general` ห้าม question/subagent, `explore` deny-all + allow เฉพาะอ่าน

### ลำดับการต่อกฎ (สำคัญ)

```
global permissions  →  agent permissions  →  saved approvals (project)
```

agent **ต่อท้าย ไม่ใช่แทนที่** — และ saved approval เป็น `allow` ที่ **ไม่เคยชนะ `deny`**

### หลาย resource ต่อการกระทำเดียว

patch ที่แตะ 5 ไฟล์ = ตรวจ 5 path พร้อมกัน:

```
มี deny ตัวไหน → deny ทั้ง operation
ไม่มี deny แต่มี ask → ask
ไม่งั้น → allow
```

### Approvals: "Allow always" เก็บที่ไหน

| ตัวเลือก | ผล |
|---|---|
| Allow once | อนุมัติเฉพาะครั้งนี้ |
| **Allow always** | บันทึก pattern ที่เครื่องมือเสนอ เป็น `allow` ระดับ **project** |
| Reject | ปฏิเสธ + ปฏิเสธ request อื่นที่ค้างใน session |

เครื่องมือแต่ละตัวเสนอ pattern ไม่เหมือนกัน: shell เสนอ prefix ของคำสั่ง, skill/subagent เสนอ ID, บางตัวเสนอ `*`

> 💡 ตรวจ/ลบของที่บันทึกไว้ได้:
> ```sh
> opencode api get /api/permission/saved     # บนเครื่องนี้ = {"data":[]}
> ```

### Directories (กับดักที่เราเจอจริง)

- path นอก Location + worktree ถูกตรวจ `external_directory` **ก่อน** `read`/`edit`
- ค่าเริ่มต้น **allow** — ต้องเพิ่ม `ask`/`deny` เองถ้าอยากให้ถาม (เราทำแล้วใน config)
- `~`, `$HOME` ถูกขยายเฉพาะ `read`/`edit`/`external_directory` — **shell ไม่ขยาย**
- shell ตรวจ working directory นอก Location และ directory ที่ scanner อนุมานจากคำสั่งด้วย

### Scanner

shell ใช้ scanner แยก compound command (`a && b | c`) ออกเป็นหลายคำสั่งเพื่อตรวจทีละตัว — เราเห็นผลจริงในบทที่ 2 (คำสั่งที่มี `sudo` ถูกปฏิเสธทั้งก้อน) ถ้า syntax ผิดจนวิเคราะห์ไม่ได้ → **scanner error** ไม่ใช่รันแบบเดา

## 5.3 Policies: เพดานที่ถามไม่ได้

### 8 actions ที่มีจริง

| Action | Resource | ตัวอย่าง |
|---|---|---|
| `provider.use` | provider id | `"openai"`, `"opencode"` |
| `tool.use` | `<action>:<value>` | `"shell:git push *"`, `"edit:*.env"` |
| `integration.connect` | integration id | `"opencode"` |
| `integration.use` | `<integration>#<account>` | `"opencode#org_01..."` |
| `mcp.use` | ชื่อ MCP server | `"github"` |
| `skill.use` | skill ID | `"browser"` |
| `plugin.use` | ชื่อ package/absolute path | `"opencode-acme-cli"` |
| `websearch.use` | provider id | `"exa"` |

### `tool.use` คือตัวที่ใช้บ่อยที่สุด

```jsonc
{
  "policies": [
    { "action": "tool.use", "resource": "shell:sudo *", "effect": "deny" },
    { "action": "tool.use", "resource": "edit:*.env", "effect": "deny" },
    { "action": "tool.use", "resource": "read:*/.ssh/*", "effect": "deny" }
  ]
}
```

- `allow` ใน `tool.use` **ไม่ได้ให้สิทธิ์** — แค่ยก `deny` กว้างที่อยู่ก่อนหน้า แล้วปล่อยให้ permission ปกติตัดสิน
- ผลลัพธ์เมื่อโดน: `Blocked by configuration policy` (ไม่ใช่ "Permission denied") — ข้อความนี้ช่วยแยกได้ทันทีว่ามาจากชั้นไหน

### Precedence กลับด้าน!

นี่คือจุดที่คนเข้าใจผิดบ่อยที่สุด:

| | Settings ทั่วไป | Policies |
|---|---|---|
| ใครชนะ | ไฟล์ที่ **ใกล้ที่สุด** | ไฟล์ที่ **กว้างที่สุด** (ประเมินทีหลัง) |

ลำดับอำนาจ (สูง→ต่ำ):

```
1. macOS managed preferences
2. /etc/opencode/opencode.json(c)          ← system
3. OpenCode Console workspace
4. ~/.config/opencode/opencode.json(c)    ← global (ของเรา)
5. opencode.json(c) (outer ชนะ inner)
6. .opencode/opencode.json(c)
```

ผลจริง: **repo เปิด provider ที่คุณ deny ระดับ global ไม่ได้** — `deny` ที่กว้างกว่าชนะเสมอ

ภายในไฟล์เดียว: เรียงตามที่เขียน (last match wins)

### กับดัก 3 ข้อ

1. **statement ที่ผิด schema ถูกทิ้งเงียบ ๆ** + warning ใน log (`role=server`) — ถ้า deny ไม่ทำงาน ให้ดู log ก่อน
2. **`integration.use` เปลี่ยนความหมายตั้งแต่ 2.2.3** — `{action:"integration.use", resource:"*", effect:"deny"}` แบบเก่าเคยกั้น MCP/skill/plugin แต่ตอนนี้มันกั้น **provider ทั้งหมด** ต้องแยกเป็น `mcp.use` + `skill.use` + `plugin.use`
3. **policy ไม่ได้ sandbox โค้ด plugin** — มันบล็อกที่ทางผ่านปกติ (provider/tool) เท่านั้น

## 5.4 เลือกใช้ตัวไหน: แผนตัดสินใจ

| สถานการณ์ | ใช้อะไร | ทำไม |
|---|---|---|
| อยากให้ถามก่อน `rm -rf` แต่ยอมอนุมัติได้ | permission `ask` | ยังทำงานได้หลังอนุมัติ |
| ห้าม `sudo` เด็ดขาด ไม่ว่ากดอะไร | **policy `tool.use`** | permission `deny` ก็ได้ แต่ policy กันแม้แต่ "allow always" ที่เคยบันทึกไว้ |
| ห้าม agent อ่าน `.env` | permission `read` (มี default) | |
| ห้ามแก้ `.env` ตลอดกาล | policy `edit:*.env` | permission deny ก็พอ แต่ policy ชัดกว่าเมื่อมีหลายชั้น config |
| ปิด provider ที่ไม่ต้องการใช้เลย | policy `provider.use` | provider หายจาก catalog |
| ซ่อน skill ที่อันตรายจากโมเดล | permission `skill` deny | โฆษณา/โหลดไม่ได้ |
| ห้ามโหลด plugin ตัวหนึ่งทุกกรณี | policy `plugin.use` | plugin ไม่ถูกโหลด |

## 5.5 สถานะจริงบนเครื่องนี้

```sh
opencode api get /api/permission/saved     # {"data":[]}  ← ยังไม่มี approval ที่บันทึก
opencode debug config                      # 26 กฎของเรา + model + plugins
```

`opencode.jsonc` ของเราตอนนี้ใช้ **permission 26 กฎ** และยังไม่มี `policies` เลย — ซึ่งแปลว่า:

- `sudo` ถูก **deny** ด้วย permission → ถ้ามี saved approval `sudo *` จากอดีต มันจะยังชนะได้ (แต่ตอนนี้ไม่มี)
- ถ้าอยากให้กันถาวรจริง ๆ ควรย้าย `sudo` ไปเป็น policy

## แบบฝึกหัด 3 ข้อ

```sh
# 1) ดูว่า approval ที่บันทึกไว้มีอะไร (ตอนนี้ว่าง)
opencode api get /api/permission/saved

# 2) ดู error message ต่างกันระหว่างสองชั้น (จาก transcript บทที่ 2)
#    permission → "Permission denied: shell"
#    policy     → "Blocked by configuration policy"

# 3) ตรวจว่ากฎของเราถูก merge เข้า agent จริง
opencode debug agents | python3 -c '
import json,sys
for a in json.load(sys.stdin):
    if a["id"]=="build":
        print(len(a["permissions"]),"rules")
        for r in a["permissions"][-3:]: print(" ",r["action"],r["resource"],"->",r["effect"])
'
```
