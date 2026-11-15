# Linux บทที่ 4: process, signal และ job control

บทนี้ตอบคำถามที่เจอบ่อยที่สุดเวลาเครื่องมีปัญหา: **"คำสั่งค้างอยู่ได้ยังไง / ปิด terminal แล้วงานตายทำไม / kill แล้วทำไมไม่ตาย / zombie คืออะไร"**

ทุก output ในบทนี้มาจากการรันจริงบนเครื่อง Debian 13 + proot เครื่องเดียวกับบทก่อน ๆ — ตัวเลข PID และเวลาเปลี่ยนได้ตาม session แต่รูปแบบเหมือนกัน

---

## 4.1 process คืออะไร — PID, PPID และ state

process คือโปรแกรมที่กำลังรันอยู่ + สถานะของมันใน kernel ทุก process มีเลข **PID** (process ID) และมี **PPID** (parent PID) ที่บอกว่าใครสร้างมันขึ้นมา

shell สร้าง child process ทุกครั้งที่รันคำสั่ง และเก็บ PID ล่าสุดไว้ใน `$!` ส่วน `$$` คือ PID ของ shell เอง

```sh
$ bash -c 'sleep 5 & echo "child=$! parent=$$"; wait'
child=21276 parent=21271
```

`ps` แบบที่ใช้อ่านจริงมีไม่กี่รูปแบบ — จำสามอันนี้พอ:

```sh
$ ps -o pid,ppid,stat,etime,comm -p $$
  PID  PPID STAT     ELAPSED COMMAND
20406 12343 S          00:01 /usr/bin/bash
```

| คอลัมน์ | ความหมาย |
|---|---|
| `PID` / `PPID` | ตัวเอง / พ่อแม่ |
| `STAT` | state (ดูตารางถัดไป) |
| `ETIME` | รันมานานเท่าไร |
| `COMM` | ชื่อโปรแกรม (สั้น) |
| `%CPU` / `%MEM` | สัดส่วน CPU / หน่วยความจำ |

### state ที่ต้องรู้

| STAT | ชื่อ | ความหมาย |
|---|---|---|
| `R` | running / runnable | กำลังใช้ CPU หรือรอคิว CPU |
| `S` | interruptible sleep | รอ event (sleep, รอ I/O, รอ input) — ปกติที่สุด |
| `D` | uninterruptible sleep | รอ I/O ที่ตัดไม่ได้ (มักเป็น disk/network) — `kill` ไม่ออกจนกว่ารอเสร็จ |
| `T` | stopped / traced | ถูกหยุดด้วย job control หรือถูก debugger trace |
| `Z` | zombie | ตายแล้ว แต่พ่อแม่ยังไม่ `wait()` เก็บสถานะ |

ตัวอักษรต่อท้าย เช่น `Ss`, `R+`, `Sl` คือ flag เพิ่ม (`s` = session leader, `+` = foreground, `l` = multi-threaded)

```sh
$ ps aux | wc -l
44

$ ps -eo pid,ppid,stat,comm --sort=pid | head -6
  PID  PPID STAT COMMAND
  440 12343 Z    bash <defunct>
  509 12343 Z    gh <defunct>
 1243 12343 Z    curl <defunct>
 1402 12343 Z    bash <defunct>
 1581 12343 Z    bash <defunct>
 1742 12343 Z    curl <defunct>
```

`Z` + `<defunct>` คือ zombie — ยังไม่ต้องตกใจ ไปดู 4.6 ว่าทำไมมันฆ่าไม่ได้

```sh
$ top -b -n1 | head -5
Tasks: 42 total,   2 running,   9 sleeping,   0 stopped,  31 zombie
  Mem:    11424M total,   10953M used,      471M free,        2M buffers
 Swap:    16383M total,    6164M used,    10219M free,     2569M cached
800%cpu   0%user   0%nice   0%sys 800%idle   0%iow   0%irq   0%sirq   0%host
   PID USER         PR  NI VIRT  RES  SHR S[%CPU] %MEM     TIME+ ARGS
```

process group กับ session ใช้เวลาแก้ปัญหา "kill แล้วไม่ตายทั้งกลุ่ม" — shell ของคุณเป็นทั้ง session leader และ group leader:

```sh
$ ps -o pid,ppid,pgid,sid,stat,comm -p $$
  PID  PPID  PGID   SID STAT COMMAND
20759 12343 20759 12343 S    /usr/bin/bash
```

> ⚠️ caveat ของเครื่องนี้ (ทดสอบแล้ว)
>
> - proot แสดงเฉพาะ process ใน subtree ของ container — ไม่เห็น process ทั้งหมดของ Android
> - PPID รากคือ `12343` (proot) ไม่ใช่ PID 1 จริง
> - `top` รายงาน RAM ของ host (~11 GB) ไม่ใช่ limit ของ container
> - มี **31 zombie** ค้างจาก session ที่รันมานาน — เกิดจาก proot ไม่ reap ลูกทั้งหมด ดู 4.6

---

## 4.2 /proc/<pid> — แหล่งความจริงที่ไม่ต้องมี strace

`/proc` เป็น filesystem ปลอมที่ kernel สร้างให้อ่านสถานะของ process ได้โดยไม่ต้องติดตั้งอะไรเพิ่ม — บนเครื่องนี้ `strace` ไม่มี แต่ `/proc` ใช้ได้ครบ

```sh
$ bash -c 'echo "shell pid=$$"; sleep 30 & echo "child=$!"; grep -E "^(Name|Pid|PPid|State|Threads|VmRSS):" /proc/$$/status'
shell pid=22751
child=22758
Name:	bash
State:	S (sleeping)
Pid:	22751
PPid:	12343
VmRSS:	    2748 kB
Threads:	1
```

อ่าน command line เต็ม (argument คั่นด้วย NUL ไม่ใช่ช่องว่าง):

```sh
$ p=$(pgrep -x sleep | head -1)
$ tr '\0' ' ' < /proc/$p/cmdline; echo
sleep 30
```

อ่าน working directory และ file descriptor ที่ process เปิดอยู่:

```sh
$ readlink /proc/$p/cwd
/root

$ ls -l /proc/$p/fd | head -4
total 0
lr-x------. 1 root root 64 Nov 15 00:57 0 -> /dev/null
l-wx------. 1 root root 64 Nov 15 00:57 1 -> pipe:[4738692]
l-wx------. 1 root root 64 Nov 15 00:57 2 -> pipe:[4738692]
```

| ไฟล์ | ใช้อ่านอะไร |
|---|---|
| `/proc/<pid>/status` | state, PPID, memory, threads |
| `/proc/<pid>/cmdline` | คำสั่ง + argument จริง |
| `/proc/<pid>/cwd` | โฟลเดอร์ที่ process ทำงานอยู่ |
| `/proc/<pid>/fd/` | ไฟล์/socket ที่เปิดอยู่ |
| `/proc/<pid>/environ` | environment variables (ระวัง secret) |

> ⚠️ caveat: `/proc` บางไฟล์ถูก proot แปลงค่า เช่น `PPid` ของ root process, `/proc/1/comm` ที่อ่านไม่ได้ — ถ้าค่าแปลก ให้เทียบกับ `ps` อีกครั้ง

---

## 4.3 signal — kill ไม่ได้แปลว่าฆ่า

`kill` จริง ๆ คือ "ส่ง signal" ไม่ได้บังคับให้ตาย ชื่อคำสั่งทำให้เข้าใจผิดตั้งแต่แรก

```sh
$ kill -l | head -3
 1) SIGHUP	 2) SIGINT	 3) SIGQUIT	 4) SIGILL	 5) SIGTRAP
 6) SIGABRT	 7) SIGBUS	 8) SIGFPE	 9) SIGKILL	10) SIGUSR1
11) SIGSEGV	12) SIGUSR2	13) SIGPIPE	14) SIGALRM	15) SIGTERM
```

| signal | เลข | ใช้เมื่อ |
|---|---|---|
| `SIGHUP` | 1 | terminal ปิด / ขอให้ reload config |
| `SIGINT` | 2 | `Ctrl-C` — ยกเลิกงาน |
| `SIGKILL` | 9 | ฆ่าแน่นอน **จับไม่ได้** |
| `SIGTERM` | 15 | ขอให้จบอย่างสุภาพ (ค่าเริ่มต้นของ `kill`) |
| `SIGSTOP` / `SIGCONT` | 19 / 18 | หยุด / ทำต่อ |
| `SIGCHLD` | 17 | ลูกตาย (พ่อแม่ใช้ `wait`) |
| `SIGTSTP` | 20 | `Ctrl-Z` — ขอหยุดชั่วคราว |

### trap: จับ signal แล้วทำความสะอาด

```sh
$ bash -c 'trap "echo CAUGHT-TERM; exit 0" TERM; sleep 30' &
$ p=$!
$ sleep 0.3
$ kill -TERM $p
CAUGHT-TERM
$ wait $p; echo "exit=$?"
exit=0
```

`SIGKILL` จับไม่ได้ — ไม่มี trap ใด ๆ หยุดมันได้:

```sh
$ bash -c 'trap "echo NEVER" TERM; sleep 30' &
$ p=$!; sleep 0.2; kill -KILL $p
Killed
$ wait $p; echo "exit=$?"
exit=137
```

### exit code 128+N

เมื่อ process ตายเพราะ signal shell จะรายงาน `128 + เลข signal`:

| ออกเพราะ | exit code |
|---|---|
| สำเร็จ | `0` |
| `SIGINT` (2) | `130` |
| `SIGKILL` (9) | `137` |
| `SIGTERM` (15) | `143` |

```sh
$ sleep 30 & p=$!; kill -TERM $p; wait $p; echo "wait=$?"
wait=143
```

`timeout` ใช้เลข `124` เมื่อหมดเวลา (ไม่ใช่ 143 เพราะ `timeout` เป็นคนรายงาน):

```sh
$ timeout 1 sleep 5
$ echo $?
124
```

### ตรวจว่า process ยังมีชีวิตด้วย `kill -0`

`kill -0` ไม่ส่ง signal จริง แต่คืน 0 ถ้า process ยังมีอยู่ — ใช้เช็กก่อน/หลังฆ่า:

```sh
$ sleep 5 & p=$!
$ kill -0 $p && echo alive
alive
$ kill $p; wait $p; echo "exit=$?"
exit=143
```

> ⚠️ caveat: `kill -STOP` / `kill -CONT` ที่ส่งจาก script **ไม่ suspend จริง** บน proot นี้ (ทดสอบแล้ว: `ps` ยังเป็น `S`, `jobs` ยังเป็น `Running`) แต่ `Ctrl-Z` ใน terminal ทำงาน — ดู 4.4 และบน Linux ปกติทั้งสองทางใช้ได้

---

## 4.4 job control — `&`, `jobs`, `fg`, `bg`, `wait`

job คือ process group ที่ shell ดูแลอยู่ มี foreground job ได้ทีละหนึ่งตัว ที่เหลือเป็น background

transcript นี้มาจาก terminal จริง (PTY) บนเครื่องนี้:

```sh
$ sleep 60
^Z
[1]+  Stopped                 sleep 60
$ jobs
[1]+  Stopped                 sleep 60
$ bg %1
[1]+ sleep 60 &
$ jobs
[1]+  Running                 sleep 60 &
$ fg %1
sleep 60
^C
$ jobs; echo DONE
DONE
```

| คำสั่ง | ทำอะไร |
|---|---|
| `cmd &` | เริ่มเป็น background job |
| `jobs` / `jobs -l` | ดู job ( `-l` เพิ่ม PID) |
| `Ctrl-Z` | หยุด foreground job (ส่ง `SIGTSTP`) |
| `bg %1` | ให้ job 1 ทำงานต่อใน background |
| `fg %1` | ดึง job 1 กลับมา foreground |
| `kill %1` | ส่ง signal ให้ job 1 (ค่าเริ่มต้น TERM) |
| `wait` | รอ job ทั้งหมด / `wait $pid` รอตัวเดียว |
| `disown %1` | เอา job ออกจากตารางของ shell |

`jobs -l` แสดง PID ด้วย ซึ่งจำเป็นเมื่อต้องส่ง signal ตรง ๆ:

```sh
$ sleep 60 &
[1] 23053
$ jobs -l
[1]+ 23053 Running                 sleep 60 &
```

`wait` คืน exit code ของ job นั้น — มีประโยชน์เวลาตรวจว่าลูกตายเพราะอะไร:

```sh
$ sleep 30 & p=$!; kill -TERM $p; wait $p; echo "wait=$?"
wait=143
```

`disown` ทำให้ shell ลืม job — ตอน shell ออกจะไม่ส่ง `SIGHUP` ให้:

```sh
$ bash -c 'sleep 5 & disown %1; jobs; echo "jobs empty above"'
jobs empty above
```

> ⚠️ caveat: ปิด terminal ตามปกติจะส่ง `SIGHUP` ให้ job ที่ยังผูกกับ terminal — งานจะตายถ้าไม่ได้ `nohup`/`setsid`/`disown` (ดู 4.5). ส่วน `Ctrl-Z` ใช้ได้จริงบน proot นี้ แต่ `kill -STOP` จาก script ไม่หยุด process (ทดสอบแล้ว)

---

## 4.5 งานยาว — nohup, setsid, timeout, pgrep/pkill

| วิธี | ทำอะไร | อยู่รอดหลังปิด shell |
|---|---|---|
| `cmd &` | background ใน shell ปัจจุบัน | ❌ (รับ `SIGHUP`) |
| `nohup cmd &` | ignore `SIGHUP` + redirect output | ✅ |
| `setsid cmd` | สร้าง session ใหม่ ไม่มี controlling terminal | ✅ |
| `timeout 30 cmd` | ส่ง `TERM` หลัง 30 วินาที | — |
| `disown %1` | เอาออกจาก job table | ✅ (แต่ยังผูก terminal) |

ทดสอบ orphan จริง — shell ออกแล้ว ลูกยังอยู่และถูก reparent ไป PID 1:

```sh
$ bash -c 'sleep 25 & echo $! > /tmp/orphan.pid'
$ sleep 1; ps -o pid,ppid,stat,comm -p "$(cat /tmp/orphan.pid)"
  PID  PPID STAT COMMAND
19907     1 S    sleep
```

`nohup` ให้ผลแบบเดียวกัน และกัน `SIGHUP` ให้ด้วย:

```sh
$ bash -c 'nohup sleep 25 >/tmp/nohup-test.out 2>&1 & echo $! > /tmp/nohup.pid'
$ sleep 1; ps -o pid,ppid,stat,comm -p "$(cat /tmp/nohup.pid)"
  PID  PPID STAT COMMAND
20038     1 S    sleep
```

`setsid` สร้าง session ของตัวเอง — สังเกตว่า SID เป็น PID ของมันเอง:

```sh
$ setsid sleep 8 >/dev/null 2>&1 &
$ sleep 0.3; ps -o pid,ppid,sid,comm -p $!
  PID  PPID   SID COMMAND
20431 20407 20431 sleep
```

`timeout` คือทางป้องกัน runaway process ที่ดีที่สุด — ใส่ตั้งแต่ตอนสั่ง:

```sh
$ timeout 1 sleep 5
$ echo $?
124
```

`nice` / `renice` ลดความสำคัญของงานยาว (ตัวเลขยิ่งมาก = ยิ่งมีความสำคัญต่ำ):

```sh
$ sleep 5 & p=$!
$ renice 5 -p $p
20437 (process ID) old priority 0, new priority 5
$ ps -o pid,ni,comm -p $p
  PID  NI COMMAND
20437   5 sleep
```

### pgrep / pkill — ตรวจก่อนฆ่าเสมอ

`pgrep -x` จับชื่อโปรแกรมแบบตรงตัว ปลอดภัยกว่า:

```sh
$ sleep 8 & p=$!
$ pgrep -a -x sleep
23055 sleep 8
$ pkill -x sleep
$ wait $p; echo "exit=$?"
exit=143
```

`pkill -f` จับ **command line ทั้งบรรทัด** ซึ่งรวม command line ของ shell ที่รันคำสั่งนั้นด้วย — ทดสอบแล้วว่าฆ่าตัวเองได้:

```sh
$ pgrep -a -f 'sleep 6'
20407 /usr/bin/bash -c echo ... sleep 6 ...
20441 sleep 6
$ pkill -f 'sleep 6'
# shell ที่รันคำสั่งนี้ถูก SIGTERM ไปด้วย (exit 143)
```

กฎที่ควรจำ:

1. `pgrep -a` ดูก่อนทุกครั้ง ว่ามีอะไร match บ้าง
2. ใช้ `-x` เมื่อจับชื่อโปรแกรม หรือ anchor pattern (`^sleep `)
3. อย่าใช้ pattern กว้าง เช่น `pkill -f python` — อาจฆ่า agent/IDE ของตัวเอง

> ⚠️ caveat: เครื่องนี้ไม่มี `systemd`, `tmux`, `screen` — งานยาวจึงต้องใช้ `nohup`/`setsid` + log + `pgrep` ตรวจสอบเอง ตัวอย่าง log:
>
> ```sh
> nohup ./long-task.sh > /tmp/task.log 2>&1 &
> tail -f /tmp/task.log
> ```

---

## 4.6 zombie และ orphan

zombie = process ที่**ตายแล้ว** แต่พ่อแม่ยังไม่เรียก `wait()` เพื่อเก็บ exit status — kernel เก็บ entry ไว้ (ไม่มี CPU/RAM ใช้ แต่กิน PID/Task slot)

สร้าง zombie ดูจริงด้วย `fork()` ใน Python:

```sh
$ python3 - <<'PY' &
import os, time
pid = os.fork()
if pid == 0:
    os._exit(0)
print(pid, flush=True)
time.sleep(15)
PY
20570

$ ps -o pid,ppid,stat,comm -p 20570
  PID  PPID STAT COMMAND
20570 20568 Z    python3 <defunct>
```

zombie ตอบ `kill -0` ว่ายังมีชีวิต และ `kill` ก็ไม่ทำให้หาย:

```sh
$ kill -0 20570 && echo alive
alive

$ kill 20570
$ ps -o pid,ppid,stat,comm -p 20570
  PID  PPID STAT COMMAND
20570 20568 Z    python3 <defunct>
```

เมื่อพ่อแม่จบ (หรือ reap) zombie จะหายเอง:

```sh
$ kill 20568          # พ่อแม่ตาย
$ ps -p 20570 >/dev/null && echo "still there" || echo "gone"
gone
```

หา zombie ทั้งหมดและดูว่าใครเป็นพ่อแม่:

```sh
$ ps -eo stat= | grep -c '^Z'
31

$ ps -eo pid,ppid,stat,comm | awk '$3 ~ /Z/ {print}'
```

| เรื่อง | zombie | orphan |
|---|---|---|
| คือ | ลูกตายแล้ว แต่พ่อแม่ยังไม่ `wait()` | พ่อแม่ตายก่อน ลูกยังรัน |
| ฆ่าได้ไหม | ❌ (ตายแล้ว) | ✅ |
| แก้ที่ไหน | พ่อแม่: `wait()` หรือ restart พ่อแม่ | ไม่ต้องแก้ — reparent ไป PID 1 |
| อันตราย | กิน PID slot ถ้าสะสมมาก | ปกติ |

> ⚠️ caveat: 31 zombie บนเครื่องนี้มีพ่อแม่เป็น proot (PID 12343) — ฆ่าไม่ได้และไม่กิน CPU/RAM แต่ถ้าเขียนโปรแกรมที่ `fork()` เอง ต้อง `wait()` ทุกครั้ง ไม่งั้น zombie จะโตเรื่อย ๆ

---

## 4.7 lab: ตามล่า runaway process

สถานการณ์: มี process กิน CPU และไม่รู้ว่าตัวไหน — ลำดับที่ปลอดภัยคือ **ดู → ยืนยัน → TERM → รอ → KILL → ตรวจ**

```sh
$ yes > /dev/null &
[1] 21270

$ ps -o pid,pcpu,stat,comm -p 21270
  PID %CPU STAT COMMAND
21270  2.3 R    yes

$ ps -eo pid,pcpu,stat,comm --sort=-pcpu | head -3
  PID %CPU STAT COMMAND
21485 28.1 R    ps
11566  5.8 Sl+  opencode
```

ยืนยันว่าตัวถูกก่อนฆ่า — ดู `cmdline` และ `cwd`:

```sh
$ tr '\0' ' ' < /proc/21270/cmdline; echo
yes
$ readlink /proc/21270/cwd
/root
```

ฆ่าอย่างสุภาพก่อน แล้วรอ:

```sh
$ kill -TERM 21270
$ sleep 0.3
$ kill -0 21270 2>/dev/null && echo ALIVE || echo GONE
GONE
$ wait 21270; echo "exit=$?"
exit=143
```

ถ้า `TERM` ไม่พอ (process ignore signal) ค่อยใช้ `KILL`:

```sh
$ yes > /dev/null &
$ p=$!; kill -KILL $p; wait $p; echo "exit=$?"
exit=137
```

ขั้นตอนที่ควรทำทุกครั้ง:

1. `ps -eo pid,pcpu,stat,comm --sort=-pcpu | head` หรือ `top -b -n1`
2. `ps -o pid,ppid,user,stat,etime,cmd -p <pid>` — ดู PPID/owner ก่อน
3. `tr '\0' ' ' < /proc/<pid>/cmdline` — ยืนยันว่าใช่ตัวที่คิด
4. `kill -TERM <pid>` แล้วรอ 2–5 วินาที
5. `kill -0 <pid>` — ยังอยู่ไหม
6. ถ้ายังอยู่ `kill -KILL <pid>` แล้วตรวจซ้ำ
7. ป้องกันรอบหน้า: ใส่ `timeout`, ลด priority ด้วย `nice`, เขียน log

> ⚠️ อย่าฆ่าจาก pattern กว้าง และอย่าฆ่า process ที่มี PPID เป็น 1 หรือ proot/opencode โดยไม่ตรวจก่อน — `pkill -f` อาจ match shell ของตัวเอง (ทดสอบแล้วใน 4.5)

---

## 4.8 แบบฝึกหัด 4 ข้อ + เช็คความเข้าใจ 3 ข้อ

### แบบฝึกหัด

1. เริ่ม `sleep 100` เป็น background, ดู `jobs -l`, แล้วฆ่าด้วย `kill %1` — ตรวจว่า `jobs` ว่างและ `wait` คืน 143
2. เขียนสคริปต์ `/tmp/trap-demo.sh` ที่ `trap 'echo CAUGHT; exit 0' TERM` แล้ว `sleep 30` — รัน, ส่ง TERM, ตรวจว่าได้ `CAUGHT` และ exit 0
3. สร้าง zombie ด้วย snippet ใน 4.6 — ตรวจด้วย `ps` ว่าเป็น `Z`, ลอง `kill -0`, แล้วอธิบายว่าทำไม `kill` ไม่ทำให้หาย
4. รัน `timeout 2 sleep 10; echo $?` (ได้ 124) แล้วรัน `sleep 10` พร้อมกด `Ctrl-C` แล้ว `echo $?` (ได้ 130)

### เช็คความเข้าใจ

1. exit code `143` มาจาก signal อะไร และคำนวณอย่างไร
2. ทำไม zombie ฆ่าไม่ได้ และใครเท่านั้นที่ทำให้มันหาย
3. `nohup` กับ `setsid` ต่างกันอย่างไร และเมื่อไรควรใช้อันไหน

---

## สรุปบทที่ 4

1. **ดู state ก่อนฆ่า** — `R/S/D/T/Z` บอกว่าควรทำอะไรต่อ; `D` รอ I/O อย่าเพิ่ง `kill -9`
2. **signal เป็นบันได** — `TERM` → รอ → `KILL`; exit code คือ `128 + N` และ `timeout` คืน `124`
3. **งานยาวต้องมีหลักประกัน** — `nohup`/`setsid` + log + `timeout`; ตรวจด้วย `pgrep -a` ก่อน `pkill`; zombie แก้ที่พ่อแม่ ไม่ใช่ที่ตัวมัน

> ⚠️ สรุป caveat ของเครื่องนี้: process view ถูกจำกัดโดย proot, `kill -STOP` จาก script ไม่หยุดจริง, ไม่มี `systemd`/`strace`/`ss`/`tmux` — แต่ signal, job control, orphan และ zombie ทำงานเหมือน Linux ปกติพอให้เรียนและใช้งานได้
