# opencode-lab

ชุด config / plugin / command ที่สร้างระหว่างเรียน **OpenCode V2 (7 บท)** — ใช้เป็นตัวอย่างและจุดเริ่มต้นของ workspace ตัวเอง

## มีอะไรในนี้

| ไฟล์ | คืออะไร |
|---|---|
| `config/opencode.jsonc` | config กลาง: model `opencode/big-pickle`, permission 26 กฎ, MCP `context7`, plugin `superpowers` |
| `commands/god-agents.md` | slash command `/god-agents` — orchestrator 5 ขั้น (plan → dispatch → collect → report → verify) |
| `plugins/lab-sysinfo.js` | plugin แรก: เพิ่ม tool `lab.sysinfo` + logging lifecycle (hot reload) |

## วิธีใช้

```sh
# 1) backup ของเดิม
cp ~/.config/opencode/opencode.jsonc ~/.config/opencode/opencode.jsonc.bak

# 2) copy เข้า workspace
mkdir -p ~/.config/opencode/commands ~/.config/opencode/plugins
cp config/opencode.jsonc        ~/.config/opencode/
cp commands/god-agents.md       ~/.config/opencode/commands/
cp plugins/lab-sysinfo.js       ~/.config/opencode/plugins/

# 3) ตรวจว่าโหลดจริง
opencode debug config
opencode plugin list
opencode mcp list
```

## บทเรียนที่เกี่ยวข้อง

- **OpenCode V2 (7 บท)** — client/server, config & permissions, agents, commands & skills, policies, MCP & plugins, CLI/API/SDK
- **Linux** — filesystem, text processing (pipes/redirects), permissions & ownership

## ⚠️ Secret — ห้าม commit

ไฟล์ต่อไปนี้ **ไม่ถูก track** โดยเจตนา (อยู่ใน `.gitignore`):

```
service.json      # token ของ API (mode 600)
*.db              # ประวัติ session ทั้งหมด
.env              # secret ของโปรเจกต์
```

ถ้าเผลอ commit ไปแล้ว ให้ถือว่า token รั่ว → rotate ทันที (ลบใน commit ใหม่ไม่พอ เพราะยังอยู่ใน history)

## License

MIT
