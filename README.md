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

## เว็บไซต์บทเรียน

📖 **https://sakzeed.github.io/opencode-lab/**

เนื้อหาอยู่ใน `docs/` (Markdown) และ build ด้วย MkDocs Material — ทุกครั้งที่ push `main` ที่แก้ `docs/` หรือ `mkdocs.yml` GitHub Actions จะ build และ deploy ขึ้น Pages อัตโนมัติ

```sh
# preview ในเครื่อง
python3 -m venv .venv
.venv/bin/pip install "mkdocs-material==9.*"
.venv/bin/mkdocs serve        # http://127.0.0.1:8000

# ตรวจก่อน push (CI รันสองคำสั่งนี้)
python3 tools/check_docs.py
.venv/bin/mkdocs build --strict
```

- **OpenCode V2 (7 บท)** — client/server, config & permissions, agents, commands & skills, policies, MCP & plugins, CLI/API/SDK
- **Linux (4 บท)** — filesystem, text processing (pipes/redirects), permissions & ownership, process/signal/job control
- **Git & GitHub (2 บท)** — commit graph, branch/merge, remote, PR, GitHub Actions/CI

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
