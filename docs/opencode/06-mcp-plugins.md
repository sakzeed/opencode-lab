# บทที่ 6: MCP Servers & Plugins

สองกลไกนี้เป็น "ทางต่อขยาย" ของ OpenCode แต่คนละชั้นกัน:

| | **MCP server** | **Plugin** |
|---|---|---|
| คืออะไร | กระบวนการ/บริการภายนอกที่คุยผ่าน protocol | โค้ดที่โหลดเข้าไปอยู่ใน OpenCode เอง |
| ให้อะไร | tools, prompts, resources, instructions | tools, hooks, commands, agents, UI (Desktop/TUI) |
| ใช้กับ client อื่นได้ไหม | ได้ (มาตรฐานกลาง) | ไม่ได้ ผูกกับ OpenCode |
| ต้นทุน context | สูง — tool ทุกตัวกิน context | แล้วแต่ plugin |
| ใครส่งมาให้ได้ | config + **Console workspace** | config + โฟลเดอร์ discovery |

> ⚠️ เอกสารเตือนตรง ๆ: *"MCP tools consume model context, so add only the servers you need."* — บทนี้จะเน้นวิธี "เปิดเฉพาะที่ใช้ และปิดให้เป็น"

---

## 6.1 Config V2: จุดที่คน migrate ผิดบ่อย

```jsonc
// ✅ V2
{ "mcp": { "servers": { "my-server": { ... } } } }

// ❌ V1 (V2 ไม่รับ)
{ "mcp": { "my-server": { ... } } }
```

ชื่อ server ต้องไม่ซ้ำ — และ **config ที่ precedence สูงกว่าจะ *แทนที่* object ทั้งก้อน** ไม่ merge ทีละ field ถ้าจะ override ต้องใส่ field ที่จำเป็นครบ หรือใช้ชื่อใหม่

### Local server (stdio)

```jsonc
{
  "mcp": {
    "servers": {
      "everything": {
        "type": "local",
        "command": ["npx", "-y", "@modelcontextprotocol/server-everything"],
        "cwd": ".",
        "environment": {
          "LOG_LEVEL": "info",
          "MCP_API_KEY": "{env:MCP_API_KEY}"
        },
        "disabled": false,
        "codemode": true,
        "timeout": { "startup": 45000 },
        "protocol": "legacy",
        "pool": false
      }
    }
  }
}
```

### Remote server (Streamable HTTP)

```jsonc
{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp",
        "oauth": false,
        "headers": { "CONTEXT7_API_KEY": "{env:CONTEXT7_API_KEY}" }
      }
    }
  }
}
```

**กฎที่ต้องจำ 4 ข้อ**

1. `{env:NAME}` เท่านั้น — `$NAME` **ไม่ถูกขยาย** ใน JSON string
2. ใช้ `disabled: true` ไม่ใช่ `enabled` (ไม่มี field `enabled`)
3. remote เปิด OAuth ให้อัตโนมัติ — ปิดด้วย `oauth: false` เมื่อใช้ API key ล้วน
4. `codemode: false` = เอา tool ไปวางบน native tool list (กิน context มากกว่า แต่บาง provider ต้องการ)

---

## 6.2 ของจริงบนเครื่องนี้: `opencode_console` โผล่มาได้อย่างไร

ใน config ของเรา **ไม่มี** `mcp` เลย แต่:

```sh
$ opencode mcp list
✓ opencode_console  connected
```

คำตอบอยู่ใน `opencode debug config` — Console workspace ส่ง document มาให้:

```jsonc
{
  "policies": [
    { "action": "provider.use", "resource": "*",      "effect": "deny"  },
    { "action": "provider.use", "resource": "opencode", "effect": "allow" }
  ],
  "mcp": {
    "servers": {
      "opencode_console": { "type": "remote", "url": "https://opencode.ai/console/mcp" }
    }
  },
  "websearch": { "provider": "opencode" }
}
```

เชื่อมกับบทที่ 5 ได้ทันที: **Console อยู่เหนือ global** ในลำดับ policy → repo เปิด provider อื่นไม่ได้ แม้แก้ global ก็ตาม

และ log ยืนยันว่าต่อจริง:

```
message="mcp connected" server=opencode_console tools=11 directory=/root
message="mcp connected" server=opencode_console tools=11 directory=/tmp/opencode/perm-lab
message="mcp connected" server=opencode_console tools=11 directory=/tmp
```

**3 directory = 3 connection (connectionID ต่างกัน)** — นี่คือพฤติกรรม default "1 connection ต่อ 1 directory" ซึ่งจะกลับมาในหัวข้อ `pool`

---

## 6.3 CLI: เพิ่ม / ดู / auth

```sh
# remote (ค่าเริ่มต้นเขียนลง project config)
opencode mcp add context7 --url https://mcp.context7.com/mcp

# ให้ใช้ทุก project
opencode mcp add context7 --global --url https://mcp.context7.com/mcp

# local: command ต้องอยู่หลัง --
opencode mcp add everything --env LOG_LEVEL=info -- npx -y @modelcontextprotocol/server-everything

# header ที่มี {env:...} ต้อง quote single
opencode mcp add context7 --url https://mcp.context7.com/mcp \
  --header 'CONTEXT7_API_KEY={env:CONTEXT7_API_KEY}'

opencode mcp list
opencode mcp auth sentry      # เริ่ม OAuth flow
opencode mcp logout sentry    # ลบ credential ที่เก็บไว้
```

**ผลทดสอบจริง** (ใน sandbox `/tmp/opencode/mcp-add-lab`):

```
$ opencode mcp add context7 --url https://mcp.context7.com/mcp \
    --header 'CONTEXT7_API_KEY={env:CONTEXT7_API_KEY}'
MCP server "context7" added to /tmp/opencode/mcp-add-lab/opencode.json

$ cat opencode.json
{
  "mcp": {
    "servers": {
      "context7": {
        "type": "remote",
        "url": "https://mcp.context7.com/mcp",
        "headers": { "CONTEXT7_API_KEY": "{env:CONTEXT7_API_KEY}" }
      }
    }
  }
}

$ opencode mcp list
✓ context7          connected
✓ opencode_console  connected
```

> 🐛 **ข้อสังเกตจริง:** เรียก `mcp list` ทันทีหลัง `mcp add` ครั้งแรกได้ `No MCP servers configured` — พอรันซ้ำอีกครั้งจึงเห็น server ครบ (น่าจะเป็น cache/รอบเชื่อมของ service) **ถ้าเจออาการนี้ให้รันซ้ำก่อนไปแก้ config**

---

## 6.4 สถานะที่ `mcp list` แสดง

```
$ opencode mcp list        # ใน mcp-lab ที่จงใจทำให้พัง
✗ demo_local        failed: Connection closed: MCP server process exited with code 1
○ demo_remote       pending
✓ opencode_console  connected
```

| สัญลักษณ์ | ความหมาย |
|---|---|
| `✓ connected` | ต่อได้ พร้อมให้ agent ใช้ |
| `○ pending` | กำลังเชื่อม/รอ handshake |
| `✗ failed: ...` | ต่อไม่สำเร็จ — อ่านข้อความต่อท้ายคือสาเหตุ |
| `needs authentication` | remote ที่ต้อง OAuth → `/mcps` หรือ `opencode mcp auth <name>` |

**ข้อเท็จจริงของเครื่องนี้:** ไม่มี `node`/`npx` (`command -v node` → exit 1) ดังนั้น local server แบบ `npx ...` **รันไม่ได้** ตอนนี้ — ใช้ remote ได้เลย (context7 พิสูจน์แล้ว) ถ้าจะใช้ local ต้องลง Node ก่อน

---

## 6.5 ชื่อ tool และ Code Mode

OpenCode ตั้งชื่อ tool เป็น `<server>_<tool>` และแทนอักขระอื่นด้วย `_`:

```
server: context 7   +   tool: resolve.library/id
→ context_7_resolve_library_id
```

ใต้ Code Mode (ค่าเริ่มต้น) tool จะถูกจัดกลุ่มตามชื่อ server:

```js
tools.context_7.resolve_library_id(...)
```

**ของจริงใน session นี้:** MCP catalog ที่ผมเห็นคือ `tools.opencode_console.*` — 11 tools เช่น

```
tools.opencode_console.create_model
tools.opencode_console.create_provider
...
```

และ instructions ที่ server ส่งมาตอน initialize ถูกฉีดเข้า system prompt ตรง ๆ:

```xml
<mcp_instructions>
  <server name="opencode_console">
    Use tools from this server through `execute` under `tools["opencode_console"]`.
    ...
  </server>
</mcp_instructions>
```

> ถ้า agent ถูก deny tool ของ server ครบทุกตัว instructions ของ server นั้นจะถูกตัดออก — เป็นการประหยัด context อีกชั้น

MCP **prompts** กลายเป็น command: `<server>:<prompt>` → `/context_7:find_docs react "server components"` (argument คั่นด้วย space, ใส่ quote ถ้ามี space)

---

## 6.6 Permission ของ MCP

ซ่อน/บล็อก tool โดยไม่ต้องตัด connection — match ชื่อ normalized:

```jsonc
{
  "permissions": [
    { "action": "context7_*", "resource": "*", "effect": "deny" }
  ]
}
```

Resources (อ่านผ่าน tool built-in 2 ตัว) มี action ของตัวเอง:

| Action | Resource |
|---|---|
| `opencode_list_mcp_resources` | ชื่อ server |
| `opencode_read_mcp_resource` | `<server>:<uri>` |

```jsonc
{ "permissions": [
  { "action": "opencode_read_mcp_resource", "resource": "docs:*", "effect": "allow" }
]}
```

ถ้าอยากกันถาวรจน saved approval ชนะไม่ได้ → ใช้ **policy** `mcp.use` (บทที่ 5)

---

## 6.7 OAuth, timeout, protocol, pool

**OAuth** เปิดให้ remote อัตโนมัติ: discover auth server → PKCE → refresh token → เก็บ credential **นอก** project config

- รันบน SSH/container แล้วเบราว์เซอร์เปิด callback ไม่ได้ → copy URL จาก address bar ไปวางใน prompt (`/mcps`, `opencode mcp auth`, หรือ dialog ของแอป)
- client registration เลือกอัตโนมัติ: pre-registered → CIMD → DCR
- ถ้า server ไม่ประกาศ metadata → ตั้ง `oauth.auth_server_metadata_url` เอง

**Timeouts** (ms, จำนวนเต็มบวก):

| ค่า | default | ใช้กับ |
|---|---|---|
| `startup` | 30 s | ต่อ transport + initialize |
| `catalog` | 30 s | list tools/prompts/resources |
| `execution` | **12 ชั่วโมง** | เรียก tool / อ่าน resource |

ตั้ง default ที่ `mcp.timeout` แล้ว override ต่อ server ได้

**Protocol:** `legacy` (default, ถึง revision 2025-11-25) / `auto` (probe `server/discover` เพื่อหา 2026-07-28 แล้ว fallback) / `2026-07-28` (บังคับ)

> `auto` กับ local server เก่า = spawn process เพิ่มชั่วคราว และอาจรอถึง startup timeout — ใส่เฉพาะที่จำเป็น

**Pool:** `pool: true` ให้หลาย directory ใช้ connection + tool list ร่วมกัน — แต่**แชร์ได้เฉพาะ protocol 2026-07-28** และทุกอย่างที่เปลี่ยน connection ต้องตรงกัน (URL, headers, OAuth/บัญชี, command, cwd, env, timeout)

นี่คือคำอธิบาย log ที่เห็นใน 6.2: default = 1 connection ต่อ directory; ถ้าอยากให้ `/root` กับ `/tmp` ใช้ร่วมกัน ต้องเปิด `pool` และคู่กับ protocol ใหม่

---

## 6.8 Plugins

### รูปแบบที่ใส่ใน `plugins` array

```jsonc
{
  "plugins": [
    "opencode-acme-plugin",           // npm
    "opencode-acme-plugin@1.2.0",     // pinned
    "@acme/opencode-plugin",          // scoped
    "./plugins/local",                // relative (จากไฟล์ config ที่ประกาศ)
    "file:///home/me/plugins/local",
    { "package": "@acme/opencode-plugin",
      "options": { "agent": "reviewer", "strict": true } }
  ]
}
```

- path ต้องเป็น **directory** — ชี้ไปไฟล์เดี่ยวจะถูกข้าม
- array จาก config หลายชั้น **merge กันจากต่ำ→สูง** ไม่ใช่แทนที่:

```
~/.config/opencode/opencode.jsonc   (ต่ำสุด)
./opencode.jsonc
./.opencode/opencode.jsonc          (สูงสุด)
```

### Discovery: ไม่ต้องประกาศก็โหลด

```
.opencode/plugins/          ← .ts/.js ตรง ๆ + package directory ชั้นแรก
~/.config/opencode/plugins/ ← global layout เดียวกัน
```

**ของจริง:** `~/.config/opencode/plugins/` **ยังไม่มี** บนเครื่องนี้ — superpowers มาจาก config array + cache ไม่ใช่ discovery

> กับดัก: `plugins/` ที่วางข้าง `opencode.json(c)` ของ project root **ไม่** ถูก discover อัตโนมัติ ต้องประกาศ path เอง หรือย้ายไปใต้ `.opencode/`

### Control: ปิด/เปิดแบบ declarative

```jsonc
{ "plugins": ["*", "-opencode.provider.*", "opencode.provider.openai", "-acme.reviewer"] }
```

- `-id` = ปิด, `*` = ทุกตัว, `.*` = match prefix ของ ID
- ID ที่ประกาศทีหลัง **เปิดคืนได้**
- **built-in 2 ตัวลบไม่ได้เด็ดขาด**: `opencode.config.policy` และ `opencode.provider.opencode` (กัน repo ปิด policy enforcement ขององค์กร)

**ทดสอบจริง** — superpowers ถูกปิดด้วย project config:

```sh
# /tmp/opencode/plugin-lab/opencode.jsonc
{ "plugins": ["-superpowers"] }

$ opencode plugin list --standalone
No plugins found          ← ปิดได้จริง

$ opencode plugin list
superpowers  22f0b46  superpowers@git+https://github.com/obra/superpowers.git
```

> ⚠️ **กับดักจริงที่เจอ:** `plugin list` **ไม่ใส่ `--standalone`** จะไปถาม background service (ซึ่งรันด้วย config ของ `/root`) → เห็น superpowers ทั้งที่ cwd ปิดอยู่ ต้องเติม `--standalone` เมื่อต้องการตรวจ config ของ directory ปัจจุบัน

### Built-in มี 95 ตัว

```sh
$ opencode plugin list --builtin | wc -l
95

$ opencode plugin list --builtin | grep -E 'config\.policy|provider\.opencode|config\.mcp'
opencode.config.policy      builtin
opencode.config.mcp         builtin
opencode.provider.opencode  builtin
```

### Manage + reload

```sh
opencode plugin add opencode-acme-plugin@1.2.0
opencode plugin list
opencode plugin check          # เช็ค update (server/GUI/TUI)
opencode plugin update         # อัปเดตทุกตัวที่เก่า
opencode plugin remove opencode-acme-plugin@1.2.0
```

รับ npm name + version/tag/range และ git spec:

```sh
opencode plugin add github:acme/opencode-plugin
opencode plugin add git+ssh://git@github.com/acme/opencode-plugin.git#main
opencode plugin add 'github:acme/plugins#main::path:packages/opencode-plugin'
```

- **pinned** (exact version / full commit) = ไม่ถูกแตะ
- **unpinned** = ตรวจ update ให้ แต่**ไม่เปลี่ยนของที่ติดตั้งแล้ว**อัตโนมัติ
- config ที่ watched → reload เอง; local dependency ที่ไม่ watched → `touch` แล้ว `opencode service restart`

Desktop: plugin ที่ export `./gui-experimental` จะเพิ่ม panel/command/settings — ตั้งใน global config เพื่อให้ทุก window ใช้; สวิตช์ใน Settings ปิดแค่ UI ส่วน `-id` ปิดทั้ง plugin
TUI-only: ประกาศใน `cli.json` (`plugins: ["opencode-acme-cli"]`) และยังทำงานเมื่อต่อ remote server

---

## 6.9 สถานะจริงของเครื่องนี้ + เลือกใช้อะไร

| ชั้น | สถานะ |
|---|---|
| `mcp` ใน global config | **ไม่มี** — `opencode_console` มาจาก Console |
| `policies` ใน global config | **ไม่มี** — Console ส่งมา (deny provider `*`, allow `opencode`) |
| `plugins` | `superpowers@git+...#22f0b46` (v7.0.2, 15 skills) |
| `~/.config/opencode/plugins/` | ไม่มี |
| local MCP ที่ใช้ได้ | ไม่มี (ไม่มี node/npx) |
| remote MCP ที่ทดสอบผ่าน | `context7` (ใน sandbox), `opencode_console` |

**แผนตัดสินใจ**

| ต้องการ | ใช้ |
|---|---|
| ให้ agent เรียก service ภายนอก และอยากใช้ข้าม client | MCP server |
| อยากเพิ่ม hook/พฤติกรรม/command/agent/UI ใน OpenCode | Plugin |
| ลด context ที่ MCP กิน | `codemode: false` เฉพาะที่จำเป็น + permission deny tool ที่ไม่ใช้ |
| ปิด MCP ทั้ง server ชั่วคราว | `disabled: true` |
| ห้ามใช้ MCP server นั้นตลอดกาล | policy `mcp.use` |

---

## แบบฝึกหัด (รันได้จริง)

```sh
# 1) ดูว่า MCP อะไรต่ออยู่ และมาจากไหน
opencode mcp list
opencode debug config | python3 -c '
import json,sys
for d in json.load(sys.stdin):
    i=d.get("info",{})
    if "mcp" in i: print(list(i["mcp"]["servers"]), "|", d.get("source") or d.get("type"))
'

# 2) ของที่ Console ส่งมา (policy + mcp)
opencode debug config | grep -A6 '"policies"'

# 3) พิสูจน์ว่า plugin array merge + negation ทำงาน (ใน sandbox)
mkdir -p /tmp/opencode/plug-demo && cd /tmp/opencode/plug-demo
echo '{ "plugins": ["-superpowers"] }' > opencode.jsonc
opencode plugin list --standalone    # → No plugins found
rm opencode.jsonc
opencode plugin list --standalone    # → superpowers กลับมา
```

---

## ก่อนไปบทถัดไป — เลือกได้ 1 อย่าง

1. **ลงมือจริง:** เพิ่ม MCP server ระดับ global 1 ตัว (context7 จาก sandbox ขึ้น global) แล้วทดสอบว่า tool โผล่ใน catalog + ลอง deny ด้วย permission
2. **เขียน plugin แรก:** สร้าง `~/.config/opencode/plugins/*.ts` ตัวเล็ก ๆ (เช่น hook log/คำสั่ง) แล้วดูว่า reload เองหรือต้อง restart
3. **ไปบทที่ 7 เลย:** CLI / TUI / API / SDK — ต่อยอดจากที่เราใช้ `opencode api` มาตลอด

sandbox ที่ทิ้งไว้ให้ลอง: `/tmp/opencode/mcp-add-lab` (context7 connected), `/tmp/opencode/mcp-lab` (server พังโดยตั้งใจ), `/tmp/opencode/plugin-lab` — ลบได้ทุกเมื่อ บอกได้ถ้าอยากให้เก็บกวาด
