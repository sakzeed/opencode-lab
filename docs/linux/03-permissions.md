# Linux บทที่ 3: สิทธิ์ไฟล์และเจ้าของ (permissions & ownership)

บทนี้ตอบคำถามที่เจอบ่อยที่สุดเวลา server มีปัญหา: **"ทำไมไฟล์นี้แก้ไม่ได้ / ทำไมคนอื่นอ่านได้ / ทำไมสคริปต์รันไม่ขึ้น"**

และมันเชื่อมกับ OpenCode ตรง ๆ — `service.json` ที่เก็บ token ของคุณเป็น mode `600` ไม่ใช่เรื่องบังเอิญ

---

## 3.1 โมเดลพื้นฐาน: 3 กลุ่ม × 3 สิทธิ์

```sh
$ ls -l /usr/bin/passwd
-rwsr-xr-x. 1 root root 142424 Apr 19  2025 /usr/bin/passwd
 │└┬┘└┬┘└┬┘
 │ │  │  └── other  (คนอื่นทั้งหมด)
 │ │  └───── group  (กลุ่มเจ้าของ)
 │ └──────── owner  (เจ้าของ)
 └────────── type: - file, d dir, l symlink
```

| ตัวอักษร | ไฟล์ | ไดเรกทอรี |
|---|---|---|
| `r` | อ่านเนื้อหา (`cat`) | **ดูรายชื่อ**ไฟล์ (`ls`) |
| `w` | เขียน/แก้/ตัดทอน (`>`, `sed -i`) | **สร้าง/ลบ/rename** entry (ต้องมี `x` ด้วย) |
| `x` | รันเป็นโปรแกรม | **เข้าได้** (`cd`), เข้าถึงชื่อไฟล์ที่รู้ชื่อ |
| `-` | ไม่มีสิทธิ์ | ไม่มีสิทธิ์ |

> `r` กับ `x` ของไดเรกทอรี **แยกกันจริง**:
> - `r--` → ดูชื่อไฟล์ได้ แต่ `cd` ไม่ได้
> - `--x` → `cd` ได้ แต่ `ls` ไม่ได้ — แต่ถ้ารู้ชื่อไฟล์ก็ `cat d2/f.txt` ได้
>
> นี่คือเหตุผลที่ web server ใช้ `711` กับ directory ที่ต้องการให้เข้าถึงไฟล์ที่รู้ชื่อ แต่ไม่ให้ list ของทั้งหมด

---

## 3.2 เลข octal: 4 = r, 2 = w, 1 = x

บวกกันเป็นตัวเลข 1 หลักต่อ 1 กลุ่ม:

| อยากได้ | บวก | เลข |
|---|---|---|
| `rwx` | 4+2+1 | **7** |
| `rw-` | 4+2 | **6** |
| `r-x` | 4+1 | **5** |
| `r--` | 4 | **4** |
| `-wx` | 2+1 | **3** |
| `-w-` | 2 | **2** |
| `--x` | 1 | **1** |
| `---` | 0 | **0** |

**ชุดที่ต้องจำ**

| mode | ใช้กับ |
|---|---|
| `600` | ไฟล์ลับส่วนตัว (token, key, db) |
| `644` | ไฟล์ทั่วไปที่คนอื่นอ่านได้ |
| `700` | ไดเรกทอรีส่วนตัว |
| `755` | โปรแกรม/ไดเรกทอรีสาธารณะ |
| `777` | 🚫 อย่าใช้ — ทุกคนเขียนได้ |
| `4755` | setuid (อันตรายถ้าไม่จำเป็น) |
| `1777` | sticky dir (`/tmp`) |

**ผลจริงบนเครื่องนี้**

```sh
$ mkdir -p /tmp/opencode/linux3 && cd /tmp/opencode/linux3
$ touch a.txt b.txt c.txt
$ chmod 600 a.txt; chmod 644 b.txt; chmod 755 c.txt
$ stat -c '%a %n' a.txt b.txt c.txt
600 a.txt
644 b.txt
755 c.txt
```

---

## 3.3 `chmod` แบบตัวอักษร (อ่านง่ายกว่าเมื่อแก้ทีละนิด)

```sh
$ chmod u+x,g-w,o-r c.txt
$ stat -c '%a %n' c.txt
751 c.txt
$ ls -l c.txt
-rwxr-x--x. 1 root root 0 Nov 14 21:21 c.txt
```

| สัญลักษณ์ | ความหมาย |
|---|---|
| `u` `g` `o` `a` | owner / group / other / all |
| `+` `-` `=` | เพิ่ม / ลบ / ตั้งให้เท่านั้น |
| `X` (ใหญ่) | +x **เฉพาะ** dir หรือไฟล์ที่มี x อยู่แล้ว → ใช้กับ `chmod -R a+rX` |
| `s` `t` | setuid/setgid, sticky |

```sh
chmod u=rw,go=r file          # = 644 (ตั้งใหม่ทั้งชุด)
chmod -R a+rX /srv/www        # ให้อ่านทั้ง tree แต่ไม่ทำให้ไฟล์ธรรมดากลายเป็น exec
chmod g+s shared/             # setgid: ไฟล์ใหม่ในนี้ใช้ group ของ dir
```

> จุดที่คนพลาดบ่อย: `chmod -R 777 .` แก้ปัญหาชั่วคราว แต่ทำลาย security และ **ไม่แก้ root cause** (มักเป็นเรื่อง ownership หรือ umask)

---

## 3.4 `umask`: สิทธิ์ตั้งต้นของไฟล์ใหม่

`umask` คือ "bit ที่จะถูก **ปิด**" ตอนสร้างไฟล์:

```
ไฟล์ใหม่ = 666 & ~umask
dir ใหม่ = 777 & ~umask
```

**ผลจริง**

```sh
$ umask
0022
$ python3 -c 'print(oct(0o666 & ~0o022), oct(0o777 & ~0o022))'
0o644 0o755
$ touch default.txt; stat -c '%a %n' default.txt
644 default.txt
```

ลอง umask เข้มขึ้นเฉพาะ subshell:

```sh
$ ( umask 077; touch secret.txt )
$ stat -c '%a %n' secret.txt
600 secret.txt
$ stat -c '%a %n' default.txt
644 default.txt          # ← ไม่ถูกกระทบ เพราะ umask อยู่ใน subshell
```

> `umask 077` = ค่าเริ่มต้นที่ควรใช้กับโฟลเดอร์เก็บ key/token
> ใน script ให้ตั้ง umask **ก่อน**สร้างไฟล์ลับเสมอ แล้วค่อย `chmod` ซ้ำอีกชั้น

---

## 3.5 `chown` / `chgrp`

```sh
chown alice file              # เปลี่ยนเจ้าของ
chown alice:www-data file     # เปลี่ยนทั้ง owner + group
chown -R www-data:www-data /srv/app
chgrp docker /var/run/docker.sock
```

และดูว่าใครเป็นเจ้าของจริง:

```sh
$ stat -c '%a %U:%G %n' /root/.config/opencode/service.json
600 root:root /root/.config/opencode/service.json
```

### ⚠️ caveat ของเครื่องนี้ (ทดสอบแล้ว)

```sh
$ chown 65534:65534 b.txt && echo ok
ok
$ stat -c '%a %u:%g %n' b.txt
644 0:0 b.txt          # ← ไม่เปลี่ยนจริง
```

`chown` **คืน exit 0 แต่ไม่เปลี่ยน owner** เพราะ proot จำลอง uid ไว้ ไม่ได้แตะ inode จริง — บน Linux จริงคำสั่งนี้จะเปลี่ยนทันที (และต้องเป็น root หรือมี `CAP_CHOWN`)

---

## 3.6 special bits 3 ตัว

### setuid (4xxx) — รันด้วยสิทธิ์เจ้าของไฟล์

```sh
$ find /usr/bin /usr/sbin -perm -4000 -type f 2>/dev/null | wc -l
8
$ find /usr/bin -perm -4000 -type f | head -5
/usr/bin/chfn
/usr/bin/chsh
/usr/bin/gpasswd
/usr/bin/mount
/usr/bin/newgrp
$ ls -l /usr/bin/passwd
-rwsr-xr-x. 1 root root 142424 ... /usr/bin/passwd
```

`s` ตรง owner = setuid + x  
`S` = setuid แต่ **ไม่มี** x (มักเป็น config ผิด — bit ไม่ทำงาน)

เหตุที่ `passwd` ต้อง setuid: ผู้ใช้ทั่วไปต้องเขียน `/etc/shadow` ซึ่งเป็นของ root → โปรแกรมรับสิทธิ์ root ชั่วคราวเพื่อทำงานนี้เท่านั้น

**ทดลอง**

```sh
$ cp /bin/true suid-demo && chmod 4755 suid-demo
$ ls -l suid-demo
-rwsr-xr-x. 1 root root 68048 ... suid-demo
$ stat -c '%a %n' suid-demo
4755 suid-demo
```

> 🔐 setuid บน script ใช้ไม่ได้บน Linux (kernel เพิกเฉย) — setuid มีผลกับ binary เท่านั้น และเป็นช่องโหว่ยอดนิยมถ้าเขียนไม่ระวัง

### setgid (2xxx) — ไฟล์ใหม่ inherits group ของ directory

```sh
chmod 2775 /srv/shared     # drwxrwsr-x
```

ใช้กับโฟลเดอร์ที่หลายคนทำงานร่วมกัน (ไฟล์ใหม่จะได้ group เดียวกันอัตโนมัติ)

### sticky bit (1xxx) — ลบได้เฉพาะเจ้าของ

```sh
$ stat -c '%a %n' /tmp
1777 /tmp
$ ls -ld /tmp
drwxrwxrwt. 5 root root 3452 Nov 14 12:42 /tmp
```

`t` ตัวสุดท้าย = ทุกคนเขียนได้ แต่ **ลบ/rename ไฟล์ของคนอื่นไม่ได้** — จำเป็นมากสำหรับ `/tmp` ที่ทุก process ใช้ร่วมกัน

> caveat เครื่องนี้: ทดสอบ `setpriv --reuid=65534 ... rm -f /tmp/sticky-demo.txt` แล้ว **ลบสำเร็จ** ซึ่งผิดจาก Linux จริง (proot ไม่บังคับใช้ sticky) — อย่าใช้ sandbox นี้ตัดสินพฤติกรรม sticky

---

## 3.7 ค้นหาไฟล์ตามสิทธิ์ด้วย `find -perm`

```sh
find . -maxdepth 1 -perm 600 -printf '%m %f\n' | sort
600 a.txt
600 secret.txt
```

| รูปแบบ | ความหมาย |
|---|---|
| `-perm 644` | **ตรงเป๊ะ** 644 |
| `-perm -644` | มี bit เหล่านี้ **ครบ** (อาจมีมากกว่า) |
| `-perm /4000` | มี bit ใดก็ได้ในกลุ่มนี้ |
| `-perm -o+w` | world-writable |

**สูตรตรวจเครื่อง**

```sh
# ไฟล์ที่ใครก็เขียนได้ (น่าสงสัย)
find / -xdev -type f -perm -0002 -ls 2>/dev/null

# setuid ทั้งเครื่อง
find / -xdev -type f -perm -4000 -ls 2>/dev/null

# ไฟล์ที่ group/other อ่านได้ ในโฟลเดอร์ config
find ~/.config/opencode -type f ! -perm 600 -printf '%m %p\n'
```

---

## 3.8 ของจริง: ทำไม `service.json` ต้องเป็น 600

```sh
$ stat -c '%a %n' ~/.config/opencode/service.json ~/.config/opencode/opencode.jsonc ~/.local/share/opencode/opencode.db
600 /root/.config/opencode/service.json
600 /root/.config/opencode/opencode.jsonc
600 /root/.local/share/opencode/opencode.db
```

| ไฟล์ | ทำไมต้องปิด |
|---|---|
| `service.json` | เก็บ `password` = token ของ API (บทที่ 7: `curl` ไม่มี token → **401**) |
| `opencode.jsonc` | มี policy/permission + path ภายใน |
| `opencode.db` | ประวัติ session ทั้งหมด (27 MB บนเครื่องนี้) |

ถ้าเผลอ `chmod 644` ไฟล์เหล่านี้ ผู้ใช้อื่นบนเครื่องเดียวกันอ่าน token ได้ทันที — **permission ของ OpenCode ไม่ได้ป้องกันระดับ OS** มันป้องกันเฉพาะ "agent ทำอะไรได้" แต่ OS permission คือด่านสุดท้าย

```sh
# ตรวจและแก้
ls -l ~/.config/opencode/service.json
chmod 600 ~/.config/opencode/service.json
```

**เทียบกับโลกจริง**

| สิ่งที่เจอ | mode ที่ถูก |
|---|---|
| private key (`id_ed25519`) | `600` |
| `~/.ssh` | `700` |
| `.env` ที่มี secret | `600` |
| script ที่รันเอง | `700` หรือ `755` |
| web root | `755` dir, `644` file |

---

## 3.9 กับดักที่เจอบ่อย

| กับดัก | อาการ | ทางแก้ |
|---|---|---|
| `chmod -R 777` | แก้ชั่วคราว พัง security | หา owner/group ที่ถูกแล้ว `chown -R` |
| dir เป็น `644` | `cd: Permission denied` แม้ไฟล์ข้างในเปิดอยู่ | dir ต้องมี `x` (`755`/`711`) |
| `r` แต่ไม่มี `x` บน dir | `ls` ได้ แต่ `cd` ไม่ได้ | เพิ่ม `x` |
| ลืม `umask` ใน script | ไฟล์ลับกลายเป็น `644` | `umask 077` ก่อนสร้าง |
| `chmod +x` ทั้งโฟลเดอร์ | ไฟล์ข้อมูลกลายเป็น exec | `chmod -R a+rX` (X ใหญ่) |
| `S`/`T` ตัวใหญ่ | special bit ไม่ทำงานเพราะขาด `x` | ใส่ `x` ด้วย |
| แก้ permission บน symlink | ไม่มีผล / ไปโดน target | ระวัง `chmod -R` ข้าม symlink |
| เห็น `.` ท้าย mode (`-rw-r--r--.`) | ไม่ใช่ permission | เป็น security label (SELinux context) |

---

## 3.10 caveat ของเครื่องนี้ (ทดสอบแล้ว — อย่าเข้าใจผิด)

| เรื่อง | ผลจริง |
|---|---|
| `sudo` | **ไม่มี** (ใช้ `su` / `runuser` / `setpriv`) |
| `chattr` / `lsattr` / `file` | **ไม่มี** |
| `getfacl` / `setfacl` | มี แต่มาจาก Termux (`/data/data/com.termux/...`) — ACL อาจไม่ตรงกับ proot |
| **การบังคับใช้สิทธิ์อ่าน** | proot บังคับไม่ครบ: `setpriv --reuid=65534` อ่านไฟล์ `600`, `640`, `604`, `400` **สำเร็จหมด** — เฉพาะ `000` ที่ได้ `Permission denied` |
| `root` bypass | root อ่านไฟล์ `000` ได้ (exit 0) — ตรงกับ Linux จริง |
| `chown` | exit 0 แต่ owner ยังเป็น `0:0` |
| sticky bit | ลบไฟล์คนอื่นใน `/tmp` ได้ (ผิดจากของจริง) |

**แปลว่า:** ใช้เครื่องนี้ฝึก **คำสั่งและตัวเลข** ได้เต็มที่ แต่ถ้าจะพิสูจน์ว่า "user A อ่านไฟล์ B ไม่ได้" ต้องใช้ Linux จริง / container / VM — อย่าเอา proot ไปเป็นหลักฐาน

---

## แบบฝึกหัด (ทำใน `/tmp/opencode/linux3`)

```sh
cd /tmp/opencode/linux3

# 1) สร้างโฟลเดอร์ private ที่เจ้าของเท่านั้นเข้าได้
mkdir private && chmod ??? private

# 2) สร้างไฟล์ token ให้ได้ mode 600 โดยใช้ umask (ไม่ใช้ chmod)
( umask ??? ; echo "TOKEN=abc" > token.env )
stat -c '%a %n' token.env

# 3) หาไฟล์ทั้งหมดในโฟลเดอร์นี้ที่ group หรือ other เขียนได้
find . -type f -perm /0022 -printf '%m %p\n'

# 4) เปลี่ยน c.txt จาก 751 เป็น 640 ด้วย symbolic mode อย่างเดียว
chmod ??? c.txt && stat -c '%a' c.txt

# 5) ทำให้ suid-demo กลับเป็นไฟล์ปกติ
chmod ??? suid-demo && ls -l suid-demo

# 6) หา setuid binary ทั้งเครื่อง แล้วนับ
find /usr/bin /usr/sbin -perm -4000 -type f | wc -l
```

**เฉลย**

```sh
# 1
chmod 700 private

# 2
( umask 077 ; echo "TOKEN=abc" > token.env )      # → 600

# 3
find . -type f -perm /0022 -printf '%m %p\n'      # ไม่มี output = ดี

# 4
chmod u-x,g+w,o-rwx,o-r,g+r c.txt                 # 751 → 640 (หรือ chmod 640 ก็ได้)
stat -c '%a' c.txt                                 # 640

# 5
chmod 755 suid-demo && ls -l suid-demo             # -rwxr-xr-x (s หาย)

# 6
find /usr/bin /usr/sbin -perm -4000 -type f | wc -l   # 8
```

---

## เช็คความเข้าใจ 3 ข้อ

1. ทำไมไดเรกทอรี `644` ทำให้ `cd` ไม่ได้ ทั้งที่ `r` มีอยู่?
2. `umask 027` ทำให้ไฟล์ใหม่และ dir ใหม่ได้ mode อะไร?
3. ทำไม `chmod 777` แก้ปัญหา "Permission denied" ได้ แต่เป็นวิธีที่ผิด?

---

## สรุปบทที่ 3

```
owner / group / other  ×  r(4) w(2) x(1)  =  ตัวเลข 3 หลัก
ไฟล์ใหม่ = 666 & ~umask      dir ใหม่ = 777 & ~umask
chmod = เปลี่ยนสิทธิ์        chown = เปลี่ยนเจ้าของ
setuid(4) = รันด้วยเจ้าของ   setgid(2) = inherit group   sticky(1) = ลบได้เฉพาะเจ้าของ
```

**กฎ 3 ข้อที่ควรติดตัว**

1. ไฟล์ลับ → `600`, โฟลเดอร์ลับ → `700`, อย่าใช้ `777` เด็ดขาด
2. ถ้าเจอ permission error ให้ถามว่า "ใครเป็น owner" ก่อนเสมอ — `chmod` แก้ที่อาการ, `chown` แก้ที่ причина
3. `find -perm` คือเครื่องมือตรวจสอบ ไม่ใช่แค่คำสั่งท่องจำ
