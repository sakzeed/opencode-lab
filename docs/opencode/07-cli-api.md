# บทที่ 7: CLI / TUI / API / SDK

บทสุดท้ายของซีรีส์ — และเป็นบทที่รวมทุกอย่างเข้าด้วยกัน เพราะ OpenCode V2 มีแนวคิดเดียว:

> **ทุก surface เป็น client ของ server เดียวกัน** — TUI, CLI, IDE, เบราว์เซอร์, สคริปต์ของคุณ

```
                   ┌─────────────────────────────┐
  TUI ────────────▶│  OpenCode server            │
  CLI (run/api) ──▶│  sessions · config · tools  │──▶ providers / MCP / plugins
  IDE (ACP) ──────▶│  permissions · plugins       │
  Web / Desktop ──▶│  OpenAPI 3.1 (170 operations)│
  SDK / curl ─────▶└─────────────────────────────┘
```

---

## 7.1 CLI: ภาพรวมคำสั่ง

| กลุ่ม | คำสั่ง | ใช้ทำอะไร |
|---|---|---|
| เปิดใช้งาน | `opencode [dir]` | TUI เต็มจอ |
| | `opencode mini` | TUI แบบมินิมอล (ไม่กินทั้งจอ) |
| | `opencode run "..."` | ส่ง prompt แบบไม่เปิด UI (script/CI) |
| จัดการ | `opencode session list/delete/export/import` | จัดการ session |
| | `opencode service start/restart/status` | background server |
| | `opencode reload` | โหลด config ใหม่ |
| | `opencode models` | ดู model ทั้งหมด |
| | `opencode stats [--json]` | สถิติการใช้งาน/ต้นทุน |
| | `opencode pair` | ลิงก์เชื่อม client (ครั้งเดียว) |
| | `opencode upgrade / uninstall` | อัปเกรด/ถอน |
| ต่อขยาย | `opencode mcp ...` / `opencode plugin ...` | บทที่ 6 |
| เชื่อมต่อ | `opencode acp` | ACP server สำหรับ editor |
| | `opencode serve` | HTTP + web server แบบ headless |
| | `opencode api` | ยิง HTTP request ไป server ที่รันอยู่ |
| Debug | `opencode debug config/agents/paths/heap-dump` | ตรวจของจริง |

### Global flags ที่ใช้บ่อย (ใช้ได้กับเกือบทุกคำสั่ง)

```sh
--standalone          # ใช้ private server (ไม่แตะ service กลาง)
--server <url>        # ต่อกับ server ที่ระบุ
--auto                # auto-approve permission ที่ไม่ใช่ deny
-c, --continue        # ต่อ session ล่าสุด
-s, --session <id>    # ต่อ session ที่ระบุ
-m, --model <p/m>     # เลือก model
--agent <id>          # เลือก agent
--format json         # (run) output เป็น NDJSON
```

### ของจริงบนเครื่องนี้

```sh
$ opencode service status
http://127.0.0.1:49374

$ opencode session list
ses_e25ed55bdffeKk1UODr0B0O8YI   ทักทาย                       11/14/2026, 9:17:25 PM
ses_e25ab9003ffeZdXa90f54WjgS5   god-agents dispatch test     11/14/2026, 2:18:55 PM
ses_e25ad9af0ffeUOYBn6UVqmuPxl   god-agents smoke test        11/14/2026, 1:51:15 PM

$ opencode stats --days 0
7 sessions · 3 subagents
36 prompts · 209 steps · 10.8m tokens
98.4% tool success · 1 active day · best streak 1 day

$ opencode debug paths
home    /root
data    /root/.local/share/opencode
cache   /root/.cache/opencode
config  /root/.config/opencode
state   /root/.local/state/opencode
tmp     /tmp/opencode
db      /root/.local/share/opencode/opencode.db
```

> `debug paths` มีประโยชน์มากเวลาจะ backup/ย้ายเครื่อง — และ `opencode debug paths db` พิมพ์ path เดียวให้เอาไปใช้ต่อ เช่น `sqlite3 "$(opencode debug paths db)"`

---

## 7.2 `opencode run` — headless mode สำหรับ script/CI

```sh
$ opencode run --format json "ตอบแค่คำว่า ok"
{"type":"step_start","timestamp":...,"sessionID":"ses_e241...","part":{...}}
{"type":"text","timestamp":...,"sessionID":"ses_e241...","part":{"type":"text","text":"ok",...}}
{"type":"step_finish","timestamp":...,"sessionID":"ses_e241...","part":{...}}
```

- **1 บรรทัด = 1 event (NDJSON)** → เอาไป pipe เข้า `jq` ได้ทันที
- ใช้เวลา **6.7 วินาที** สำหรับ prompt สั้น (รวม start server)
- `--file` แนบไฟล์, `--worktree` เปิดงานใน git worktree ใหม่, `--fork` แยก session ก่อนทำต่อ

**สูตรสำหรับ CI**

```sh
opencode run --format json --agent plan "review การเปลี่ยนแปลงใน git diff แล้วสรุปเป็น bullet" \
  | jq -r 'select(.type=="text") | .part.text'
```

> ⚠️ ข้อจำกัดที่เราพิสูจน์ในบทที่ 4: ถ้ารันแบบ headless แล้ว agent กระจาย subagent ต่อ **จะไม่มี client คอยรับ `<subagent>` report** ต้องให้ agent อ่าน session ของ child เอง (เราอุดช่องนี้ใน `/god-agents` แล้ว)

---

## 7.3 TUI: 3 สิ่งที่ทำให้เร็วขึ้น

| อยากได้ | พิมพ์ |
|---|---|
| อ้างไฟล์ (fuzzy search) | `@packages/functions/src/api/index.ts` |
| รัน shell แล้วเอาผลเข้า conversation | `!ls -la` |
| เรียก command | `/help`, `/models`, `/new`, `/sessions`, `/compact`, `/export`, `/init`, `/mcps`, `/undo`, `/redo` |
| shortcut (leader = `ctrl+x`) | `ctrl+x c` compact · `ctrl+x n` new · `ctrl+x l` sessions · `ctrl+x m` models · `ctrl+x t` themes · `ctrl+x e` editor |

- `/undo` + `/redo` ใช้ Git จัดการไฟล์ → **ต้องอยู่ใน git repo**
- `/export` ส่งบทสนทนาเป็น Markdown ไปเปิดใน `$EDITOR`
- `opencode mini` = เวอร์ชันเบา (มี `--replay-limit` จำกัดประวัติ)

---

## 7.4 HTTP API: OpenAPI 3.1 ที่ใช้งานได้จริง

```sh
$ opencode api get /openapi.json | ...
170 operations | 139 paths | 515 schemas
```

> เอกสารบนเว็บเขียน 170 operations / 501 schemas — ของจริงบนเครื่องนี้ **515 schemas** (docs คลาดเคลื่อนเล็กน้อย)

**2 วิธีเรียก**

```sh
opencode api server.info                    # ใช้ Operation ID
opencode api get /api/info                  # ใช้ method + path
opencode api post /api/session --data '{"title":"job"}'
opencode api get /api/agent --param limit=10
```

**ผลจริง**

```json
$ opencode api server.info
{"version":"2.3.0","pid":11624,"hostname":"localhost",
 "urls":["http://127.0.0.1:49374"],
 "paths":{"tmp":"/tmp/opencode"},
 "capabilities":{"persistentPty":true}}

$ opencode api get /api/location
{"directory":"/root","project":{"id":"cd31f212...","directory":"/root","canonical":"/root"}}

$ opencode api get /api/agent
7 agents: build, general, explore, compaction, title, summary, plan
```

**Authentication — สำคัญ**

```sh
$ curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:49374/api/info
401
```

แม้อยู่บน `127.0.0.1` ก็ต้อง auth — `opencode api` ทำงานได้เพราะอ่าน token จาก `~/.config/opencode/service.json` (mode `600`, เก็บ key `password`)

```sh
$ ls -l ~/.config/opencode/service.json
-rw-------. 1 root root 64 ... service.json
```

**สูตร automation ครบวงจร** (แบบที่เราใช้ทดสอบ `/god-agents`)

```sh
SID=$(opencode api post /api/session --data '{"title":"nightly"}' | jq -r .id)
opencode api post "/api/session/$SID/prompt" --data '{"text":"สรุป git log 7 วัน"}'
opencode api post "/api/experimental/session/$SID/wait" --data '{}'     # รอจน idle
opencode api get  "/api/session/$SID/message" | jq -r '...'             # อ่านผล
```

endpoint ที่มีให้ใช้ (ตัวอย่างจาก 139 paths): `/api/session/{id}/fork`, `/compact`, `/revert`, `/diff`, `/context`, `/shell`, `/inbox`, `/api/mcp`, `/api/plugin`, `/api/worktree`, `/api/event` (SSE), `/api/pty` (persistent PTY)

---

## 7.5 SDK

```sh
npm install @opencode-ai/sdk
```

```ts
import { createOpencode, createOpencodeClient } from "@opencode-ai/sdk"

// 1) เปิด server + client ในตัว
const { client } = await createOpencode({ hostname: "127.0.0.1", port: 4096 })

// 2) ต่อกับ server ที่มีอยู่
const c = createOpencodeClient({ baseUrl: "http://localhost:4096" })

const s = await c.session.create({ body: { title: "My session" } })
await c.session.prompt({
  path: { id: s.data.id },
  body: { parts: [{ type: "text", text: "Hello!" }] },
})
```

- Type ทั้งหมด generate จาก OpenAPI → import ได้: `import type { Session, Message, Part }`
- error: default **ไม่ throw** (อยู่ใน `result.error`) → ใส่ `throwOnError: true` ถ้าอยาก throw
- **v2 client** ใช้ flat params + structured output:

```ts
import { createOpencodeClient } from "@opencode-ai/sdk/v2"
const r = await c.session.prompt({
  sessionID, parts: [{ type: "text", text: "..." }],
  format: { type: "json_schema", schema: { type: "object", properties: {...} } },
})
console.log(r.data.info.structured)
```

> 🚫 **บนเครื่องนี้ใช้ SDK ไม่ได้** — ไม่มี `node`/`npm` (บทที่ 6 ตรวจแล้ว) ทางที่ใช้ได้คือ **HTTP ตรง** ผ่าน `opencode api` / `curl` / Go `net/http` หรือ ACP
> เอกสาร SDK/server/acp บนเว็บยังเป็น **V1** — API surface ของ V2 ดูที่หน้า API (170 operations) และ `/openapi.json` ของเครื่องจริง

---

## 7.6 ACP / Serve / Web

| ทาง | คำสั่ง | ใช้เมื่อ |
|---|---|---|
| **ACP** (Agent Client Protocol) | `opencode acp` | ต่อ Zed / JetBrains / Avante.nvim / CodeCompanion — คุย JSON-RPC ทาง stdio; ใช้ tool, MCP, AGENTS.md, permission ได้เหมือน TUI (ยกเว้น `/undo`, `/redo`) |
| **Serve** | `opencode serve --port 4096 --hostname 127.0.0.1` | เปิด API + web app แบบ headless (เช่นใน container) |
| **Web/Desktop** | ผ่าน `serve` + `opencode pair` | เปิดจากเบราว์เซอร์/แอป desktop |
| **Mini** | `opencode mini` | TUI เบา ๆ ใน terminal เล็ก |

`opencode serve` ยังรองรับ `OPENCODE_SERVER_PASSWORD` (HTTP basic auth, username default `opencode`) และ `--cors` สำหรับ frontend ของคุณ

> หมายเหตุ: `opencode web` **ไม่มีใน V2** — พิมพ์แล้วจะตกไปที่ help หลัก ใช้ `serve` แทน (ยืนยันจาก `opencode --help` บนเครื่องนี้)

---

## 7.7 Security checklist (จากของจริง)

| ความเสี่ยง | ข้อเท็จจริงบนเครื่องนี้ |
|---|---|
| API เปิดโล่ง? | ไม่ — `curl` ไม่มี token = **401** |
| Token อยู่ไหน | `~/.config/opencode/service.json` mode `600` |
| Pair link | ใช้ครั้งเดียว, 5 นาที, = credential |
| `--auto` | auto-approve เฉพาะ `ask`; `deny` ยังชนะ (บทที่ 2) |
| `opencode auth export` | **พิมพ์ secret ทั้งหมดออกมา** — ระวังตอน pipe/แชร์ |
| `opencode serve` บน `0.0.0.0` | ควรตั้ง `OPENCODE_SERVER_PASSWORD` เสมอ |

---

## 7.8 สรุปทั้งซีรีส์ 7 บท

| บท | แกนความคิด |
|---|---|
| 1 | client–server: ทุก client คุย server เดียว |
| 2 | config: precedence + permission 26 กฎของเรา |
| 3 | agents: build/plan/explore/general + subagent |
| 4 | commands & skills: markdown + `$ARGUMENTS` + superpowers 15 skills |
| 5 | permissions vs policies: ถามได้ vs เพดานแข็ง, precedence กลับด้าน |
| 6 | MCP & plugins: ต่อขยายจากนอก/ใน + hot reload |
| 7 | CLI/TUI/API/SDK/ACP: ทางเข้าเดียวกันหมด |

**ของที่คุณมีอยู่จริงตอนนี้**

```
~/.config/opencode/
├── opencode.jsonc          model + 26 permissions + mcp.context7 + plugins[superpowers]
├── commands/god-agents.md  5-step orchestrator
└── plugins/lab-sysinfo.js  plugin แรก (hot reload + tool จริง)
```

---

## แบบฝึกหัดปิดท้าย (เลือก 1)

```sh
# 1) ดู server ตัวเองแบบละเอียด
opencode api server.info | jq .
opencode debug paths | tail -3

# 2) headless pipeline
opencode run --format json "นับจำนวนไฟล์ .js ใน ~/.config/opencode" \
  | jq -r 'select(.type=="text") | .part.text'

# 3) เปิด API server แยกสำหรับทดลอง (ไม่กระทบ service กลาง)
opencode serve --port 4097 --standalone &
curl -s http://127.0.0.1:4097/api/info
```
