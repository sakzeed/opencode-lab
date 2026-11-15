# OpenCode & Linux Lab

บทเรียนที่เขียนจากการ **ลงมือจริง** บนเครื่อง Debian 13 (trixie) บน Android/aarch64 ผ่าน proot-distro — ทุกบทมีผลลัพธ์คำสั่งจริง ไม่ใช่ทฤษฎีล้วน

## tracks

| Track | เนื้อหา | เริ่มที่ |
|---|---|---|
| **OpenCode V2** (7 บท) | client–server, config, agents, commands/skills, permissions, MCP/plugins, CLI/API | [บทที่ 1](opencode/01-architecture.md) |
| **Linux** (4 บท) | ระบบไฟล์, pipes/redirects, permissions & ownership, process/signal/job control | [บทที่ 1](linux/01-filesystem.md) |
| **Git & GitHub** (2 บท) | commit graph, branch/merge, remote, PR, Pages, GitHub Actions & CI | [บทที่ 1](git/01-git-github.md) |

## สิ่งที่ต้องมี

- OpenCode V2 (บทเรียนอ้าง `2.3.0`)
- Linux shell (bash 5.x) — คำสั่งส่วนใหญ่อยู่ใน POSIX พื้นฐาน
- ไม่ต้องมี `node`/`npm` สำหรับบท plugin และ MCP

## หมายเหตุเรื่องสภาพแวดล้อม

เครื่องที่ใช้เรียนเป็น **proot** จึงมีข้อจำกัดที่บทเรียนระบุไว้ตรงจุด เช่น `sudo`/`chattr` ไม่มี, การบังคับใช้สิทธิ์ไฟล์ไม่สมบูรณ์, และ `chown` ไม่เปลี่ยน owner จริง — จุดเหล่านี้ถูกทำเครื่องหมาย "caveat" ไว้ในบทที่เกี่ยวข้อง

## แก้ไข / ต่อยอด

โค้ดและเนื้อหาอยู่ที่ [sakzeed/opencode-lab](https://github.com/sakzeed/opencode-lab) — แก้ใน `docs/` แล้วเปิด PR ได้เลย
