# Linux บทที่ 4: process, signal และ job control — design

วันที่: 2026-11-15
สถานะ: อนุมัติแล้ว (ผู้ใช้: "อนุมัติ ออโต้ เลย ทำให้หมดเลย")

## ทำไม

บทที่ 1–3 ปิดเรื่องไฟล์/ข้อความ/สิทธิ์แล้ว แต่ยังไม่แตะ "อะไรกำลังรันอยู่บนเครื่อง" ซึ่งเป็นสิ่งที่เจอบ่อยที่สุดเวลา server มีปัญหา: คำสั่งค้าง, ปิด terminal แล้วงานตาย, kill ไม่ตาย, zombie เต็ม process table. บทนี้ปิด track Linux ให้ครบ 4 บท และเผยแพร่บนเว็บร่วมกับบทที่มีอยู่

## ขอบเขต

**ทำ**
- ไฟล์ `docs/linux/04-processes.md` 1 ไฟล์ (8 ส่วน + แบบฝึกหัด)
- ทุกคำสั่งมี output จริงจากเครื่องนี้ พร้อม caveat ของ proot ที่ทดสอบแล้ว
- แก้ `mkdocs.yml` (nav), `docs/index.md` (Linux 3→4 บท), `README.md` (Linux 3→4 บท)
- `tools/check_docs.py` เพิ่ม EXPECTED ให้ validator ตรวจหน้านี้

**ไม่ทำ**
- systemd / service management (เครื่องนี้ไม่มี systemd)
- cgroups, namespace, container, user management
- network / port (ไม่มี `ss`/`lsof` ที่เชื่อถือได้ — ใช้ `/proc/<pid>/fd` แทน)
- strace/dtrace (ไม่มี)
- shell scripting (อยู่ใน roadmap บท 8)

## ทางเลือกที่พิจารณา

| ทาง | เนื้อหา |  trade-off | สรุป |
|---|---|---|---|
| **A. Balanced + lab (เลือก)** | แนวคิด → `/proc` → signal → job control → งานยาว → zombie/orphan → runaway lab | ครบตาม roadmap และต่อยอดบท 1–3; ยาวประมาณบท 3 | **เลือก** |
| B. Job control อย่างเดียว | `&`, `jobs`, `fg/bg`, `nohup`, `disown` | ใช้งานได้เร็ว แต่ไม่เข้าใจ signal/state | ไม่เลือก |
| C. Incident response | หา/ฆ่า process ที่ค้าง, ดู `/proc` | น่าสนใจ แต่ไม่มี strace/ss ทำให้ lab อ่อน | ไม่เลือก |

## ข้อเท็จจริงจากเครื่อง (constraints ที่บทต้องสะท้อน)

- process view ของ proot จำกัด: เห็นแค่ subtree (44 processes), PPID ราก = 12343, หน่วยความจำที่ `top` แสดงเป็นของ host (11 GB) ไม่ใช่ limit ของ container
- มี **31 zombie** ค้าง (PPID 12343) ตลอด session — ฆ่าไม่ได้, จะหายเมื่อ parent reap หรือ parent ตาย
- **orphan / nohup อยู่รอดจริง**: child ถูก reparent ไป PID 1 (ทดสอบแล้ว `sleep` ยัง ALIVE)
- **Ctrl-Z ทำงานแบบ interactive** (PTY ทดสอบแล้ว: `[1]+ Stopped`), แต่ `kill -STOP <pid>` จาก script ไม่ suspend จริง (state ยัง `S`) — ต้อง caveat ชัด
- signal ทำงานครบ: `trap` จับ TERM, exit 128+N, `timeout` คืน 124, SIGKILL คืน 137
- `pkill -f` สามารถฆ่า shell ของตัวเองได้ (ทดสอบแล้ว exit 143) — ต้องสอน `pgrep -a` ก่อนเสมอ
- `kill -0` บน zombie คืน 0 (ดูเหมือนยังมีชีวิต) — ต้อง caveat
- `setsid`, `renice`, `timeout`, `pgrep`/`pkill`, `lsof` มี; `htop`, `strace`, `ss`, `tmux`, `screen` ไม่มี

## โครงบท

1. สถานะเครื่องนี้: proot เห็นอะไร/ไม่เห็นอะไร
2. process คืออะไร — PID, PPID, state (`R/S/D/T/Z`), `ps` ที่ใช้จริง, `top -b`
3. `/proc/<pid>` — อ่าน `status`, `cmdline`, `cwd`, `fd` แทน strace
4. signal — `kill -l`, TERM/INT/HUP/KILL/STOP/CONT, `trap`, exit 128+N, `kill -0`
5. job control — `&`, `$!`, `jobs`, `fg`, `bg`, `%1`, `wait`, `disown`
6. งานยาว — `nohup`, `setsid`, `timeout`, log, `pgrep`/`pkill` แบบตรวจก่อนฆ่า
7. zombie & orphan — เกิดอย่างไร, ทำไมฆ่าไม่ได้, แก้ที่ parent
8. lab: ตามล่า runaway process (TERM → รอ → KILL) + แบบฝึกหัด 4 ข้อ + เช็คความเข้าใจ 3 ข้อ + สรุป

## ไฟล์ที่แตะ

- Create: `docs/linux/04-processes.md`
- Modify: `tools/check_docs.py` (EXPECTED +1)
- Modify: `mkdocs.yml` (nav)
- Modify: `docs/index.md` (Linux 3→4 บท)
- Modify: `README.md` (Linux 3→4 บท)

## Verification

- `python3 tools/check_docs.py` → `ok: 12 pages`
- `/tmp/opencode/venv/bin/mkdocs build --strict` → exit 0
- `site/linux/04-processes/index.html` มีอยู่; nav มี 4 ลิงก์ Linux
- push → workflow `docs` success; `https://sakzeed.github.io/opencode-lab/linux/04-processes/` คืน 200 และมี marker (`zombie`, `SIGTERM`)
- grep หา chat text / mixed script ผ่าน validator

## Left to builder

- คำอธิบายละเอียดและลำดับย่อยในแต่ละส่วน
- จำนวนแบบฝึกหัด (4) และข้อเช็คความเข้าใจ (3) — ตรงกับบทที่ 1–3
- ตัดสินใจว่าจะใส่ `nice`/`renice` ในส่วน 4.5 หรือ 4.7 (แนะนำ 4.5)
- รูปแบบ admonition ใช้ blockquote `> ⚠️ caveat` ตามบทที่ 3
