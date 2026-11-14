# Linux บทที่ 2: ข้อความ ท่อ และการเปลี่ยนทาง (text, pipes, redirects)

ก่อนเริ่ม — แผน 8 บทที่ผมใช้อยู่ (เว็บนี้เผยแพร่บท 1–3; แก้ลำดับได้ถ้าคุณต้องการ):

| บท | หัวข้อ | สถานะ |
|---|---|---|
| 1 | ระบบไฟล์, path, symlink, `find` | ✅ |
| **2** | **ข้อความ, pipe, redirect, grep/sed/awk** | **วันนี้** |
| 3 | permission & ownership (`chmod`, `chown`, umask) | ถัดไป |
| 4 | process, signal, job control | |
| 5 | user, env, sudo | |
| 6 | package & service (เครื่องนี้ไม่มี systemd) | |
| 7 | network (`ss`, `curl`, `ssh`) | |
| 8 | shell script + log troubleshooting | |

เป้าหมายบทนี้: **ให้คุณตอบคำถามจาก log/ไฟล์ขนาดใหญ่ได้ในหนึ่งบรรทัด** — เป็นสกิลที่ใช้ทุกวันกับ OpenCode, Go, และ server จริง

---

## 2.1 สามท่อของทุกคำสั่ง

ทุกโปรแกรมมี 3 channel:

```
        ┌─────────────┐
stdin  ─▶│  command    │─▶ stdout   (ผลลัพธ์ปกติ)
(0)     │             │─▶ stderr   (error)   (2)
        └─────────────┘
```

| สัญลักษณ์ | ความหมาย | ตัวอย่าง |
|---|---|---|
| `>` | เขียน stdout ทับไฟล์ ( truncate!) | `ls > out.txt` |
| `>>` | ต่อท้าย | `echo hi >> out.txt` |
| `2>` | เขียน stderr ลงไฟล์ | `ls /nope 2> err.txt` |
| `2>&1` | รวม stderr เข้า stdout | `cmd 2>&1 \| grep err` |
| `<` | อ่านจากไฟล์เป็น stdin | `grep ERROR < access.log` |
| `\|` | ต่อ stdout ของตัวซ้าย → stdin ของตัวขวา | `cat f \| wc -l` |
| `tee` | เขียนทั้งจอและไฟล์ | `cmd \| tee out.txt` |

**ผลจริง** (lab ที่ `/tmp/opencode/linux2`):

```sh
$ ls /nope > out.txt 2> err.txt
$ wc -c < out.txt
0
$ cat err.txt
ls: cannot access '/nope': No such file or directory

$ ls /nope 2>&1 | head -2
ls: cannot access '/nope': No such file or directory
```

> ⚠️ `>` **ลบของเดิมทันที** ก่อนคำสั่งจะรัน — ถ้าสงสัยให้ใช้ `>>` หรือ `tee` และระวัง `> file` ในลูป

---

## 2.2 ดูไฟล์ให้เป็น

| คำสั่ง | ใช้ |
|---|---|
| `cat f` | ทั้งไฟล์ (ไฟล์เล็กเท่านั้น) |
| `head -n 20 f` / `tail -n 20 f` | 20 บรรทัดแรก/ท้าย |
| `tail -f f` | **ตาม log สด** (Ctrl+C ออก) |
| `less f` | เลื่อนดู (`/คำ` ค้น, `q` ออก, `G` ท้ายไฟล์) |
| `nl f` | ใส่เลขบรรทัด |
| `wc -l -w -c f` | จำนวนบรรทัด / คำ / ไบต์ |

```sh
$ wc -l -w -c /tmp/opencode/linux2/access.log
 10  50 490 access.log

$ tail -f ~/.local/share/opencode/log/opencode.log     # เปิดไว้แล้วสั่งงาน OpenCode ในอีกหน้าต่าง
```

> เครื่องนี้ไม่มี `file` (Termux ไม่ได้ลง) — ใช้ `stat` หรือ `od -c` แทนได้

---

## 2.3 ตัวกรอง 5 ตัวที่ต้องแม่น

### `grep` — หาบรรทัด

```sh
grep ERROR access.log            # บรรทัดที่มี ERROR
grep -c ERROR access.log         # นับ (3)
grep -n ERROR access.log         # พร้อมเลขบรรทัด
grep -v INFO access.log          # เอาบรรทัดที่ไม่ใช่
grep -i warn access.log          # ไม่สนตัวพิมพ์
grep -E "ERROR|WARN" access.log  # regex (หรือ egrep)
grep -r "context7" ~/.config/opencode   # ค้นทั้งโฟลเดอร์
grep -r --include='*.jsonc' "plugins" ~/.config/opencode
grep -q DEBUG access.log; echo $?       # เงียบ, ดู exit code
```

### `cut` — ตัดคอลัมน์

```sh
$ cut -d' ' -f1,2 access.log | head -3
2026-11-14T10:01:02Z INFO
2026-11-14T10:01:07Z INFO
2026-11-14T10:02:11Z WARN
```
`-d' '` = ตัวคั่น, `-f1,2` = ฟิลด์ 1 และ 2 (ของจริง log นี้มี space หลายตัว → ระวัง!)

### `tr` — แปลงตัวอักษร

```sh
$ echo "a    b     c" | tr -s ' '
a b c
tr 'A-Z' 'a-z' < f      # ทำเป็นตัวเล็ก
```

### `sort` + `uniq` — จัดกลุ่มนับ

```sh
$ awk '{print $3}' access.log | sort | uniq -c | sort -rn
      4 alice
      2 carol
      2 bob
      1 erin
      1 dave
```

แพทเทิร์นนี้ใช้บ่อยที่สุดในชีวิตจริง: **`... | sort | uniq -c | sort -rn | head`** = "ใครมากที่สุด"

### `sed` — แทนที่/เลือกบรรทัด

```sh
sed -n '2,4p' access.log        # บรรทัด 2-4
sed 's/ERROR/ERR/g' access.log  # แทนที่ทั้งไฟล์ (ไม่แก้ต้นฉบับ)
sed -i 's/old/new/g' file       # แก้จริง (-i) — ควร backup ก่อน
```

### `awk` — ฟิลด์ + เงื่อนไข (ทรงพลังสุด)

```sh
$ awk '{print $5}' access.log | sort | uniq -c | sort -rn
      5 200
      2 500
      2 403
      1 502

$ awk '$5 ~ /^5/ {print $3, $4, $5}' access.log
carol /api/pay 500
alice /api/pay 502
carol /api/pay 500
```

อ่านว่า: *ถ้าฟิลด์ 5 ขึ้นต้นด้วย 5 → พิมพ์ฟิลด์ 3,4,5* — นี่คือ "หา 5xx ทั้งหมด" ในบรรทัดเดียว

---

## 2.4 `xargs` — แปลงบรรทัดเป็น argument

`find` กับ `xargs` ต่างกัน:

```sh
# find -exec: เรียกคำสั่งทีละไฟล์
find . -name '*.log' -exec wc -l {} \;

# find | xargs: รวมหลายไฟล์เป็น argument เดียว (เร็วกว่า)
find . -name '*.log' -print0 | xargs -0 wc -l
```

ผลจริง:

```sh
$ find . -name '*.log' -print0 | xargs -0 wc -l
10 ./access.log
```

> `-print0` + `xargs -0` จำเป็นเมื่อชื่อไฟล์มีช่องว่าง/ขึ้นบรรทัด — เป็นนิสัยที่ควรติดตัว

---

## 2.5 Exit code: กาวที่ทำให้ script ฉลาด

ทุกคำสั่งคืนค่า `0` = สำเร็จ, ไม่ใช่ `0` = ล้มเหลว

```sh
$ grep -q ERROR access.log; echo $?
0
$ grep -q DEBUG access.log; echo $?
1

$ grep -q DEBUG access.log && echo yes || echo no
no
```

| ตัวเชื่อม | ทำงานเมื่อ |
|---|---|
| `a && b` | b ทำงานเมื่อ a สำเร็จ |
| `a \|\| b` | b ทำงานเมื่อ a ล้มเหลว |
| `a ; b` | b ทำงานเสมอ |

นี่คือเหตุผลที่ permission system ของ OpenCode ใช้ scanner แยก compound command (บทที่ 5 ของ OpenCode) — `&&` ทำให้คำสั่งเดียวกลายเป็นหลายคำสั่ง

---

## 2.6 Lab จริง: วิเคราะห์ log ของ OpenCode

ไฟล์จริง: `~/.local/share/opencode/log/opencode.log` (1,049 บรรทัด, 200 KB)

```sh
$ grep -o 'level=[A-Z]*' opencode.log | sort | uniq -c | sort -rn
   1013 level=INFO
     31 level=WARN
      5 level=ERROR

$ grep -o 'message="[^"]*"' opencode.log | sort | uniq -c | sort -rn | head -5
    210 message="watcher subscribe"
    171 message="cli starting"
    108 message="watcher started"
     91 message="watcher stopped"
     83 message="Sent HTTP response"

$ grep 'level=ERROR' opencode.log | tail -1 | cut -c1-120
timestamp=... level=ERROR ... message="Failed to fetch models.dev" cause="TimeoutError ..."
```

**อ่านผลแบบวิศวกร**

| หลักฐาน | แปลว่า |
|---|---|
| `watcher subscribe` 210 ครั้ง | มีการ reload config หลายรอบ (ปกติของ hot reload) |
| ERROR 5 ครั้ง แต่ไม่ใช่ crash | เป็น network/quota ล้วน — ไม่ต้องแก้โค้ด |
| `message="..."` อยู่ใน quote | ใช้ `grep -o 'message="[^"]*"'` แยกได้ เพราะ log เป็น `key=value` |

---

## 2.7 กับดักที่เจอบ่อย

| กับดัก | อาการ | ทางแก้ |
|---|---|---|
| `>` ทับไฟล์ | ข้อมูลหายก่อนคำสั่งรัน | ใช้ `>>` หรือ `tee` |
| ลืม quote `*` | shell ขยายก่อนคำสั่งเห็น | `'*.log'` |
| space หลายตัวใน log | `cut -d' '` ตัดผิด | ใช้ `awk` หรือ `tr -s ' '` |
| `grep` ไม่มี output แต่ exit 1 | ไม่ใช่ error — คือ "ไม่เจอ" | เช็ค `$?` |
| `xargs` กับชื่อไฟล์มี space | ไฟล์แตกเป็นหลาย argument | `-print0` + `-0` |
| `tail -f` แล้วเห็นแต่ของเก่า | ยังไม่ flush | รอ/ใช้ `stdbuf -oL` |

---

## แบบฝึกหัด (มีเฉลยอยู่ท้าย)

ทำใน `/tmp/opencode/linux2` (มี `access.log` 10 บรรทัด + `notes.md`)

```sh
cd /tmp/opencode/linux2

# 1) นับจำนวน request ที่ status ไม่ใช่ 200
# 2) หา IP/user ที่ทำให้เกิด 5xx และนับว่ากี่ครั้ง
# 3) แสดงเฉพาะบรรทัดที่มี /api/pay พร้อมเลขบรรทัด
# 4) สร้างไฟล์ errors.txt ที่มีเฉพาะ ERROR (ใช้ tee ให้เห็นบนจอด้วย)
# 5) เขียน one-liner: "path ไหนถูกเรียกมากที่สุด 3 อันดับ"
```

**เฉลย**

```sh
# 1
awk '$5 != 200' access.log | wc -l           # 5

# 2
awk '$5 ~ /^5/ {print $3}' access.log | sort | uniq -c   # carol 2, alice 1

# 3
grep -n '/api/pay' access.log

# 4
grep ERROR access.log | tee errors.txt

# 5
awk '{print $4}' access.log | sort | uniq -c | sort -rn | head -3
```

---

## เช็คความเข้าใจ 3 ข้อ (ตอบสั้น ๆ ได้)

1. `cmd > out.txt 2>&1` กับ `cmd 2>&1 > out.txt` ต่างกันอย่างไร?
2. ทำไม `grep ERROR f | wc -l` ได้เลข แต่ `grep -c ERROR f` เร็วกว่า?
3. ถ้าต้องการดู log ที่กำลังโตแบบ real-time แต่ยังอยากเก็บลงไฟล์ด้วย ใช้คำสั่งอะไร?
