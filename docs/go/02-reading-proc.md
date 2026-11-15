# Go บทที่ 2: struct, error และการอ่าน /proc

บทนี้ทำให้ `labps list` ใช้งานได้จริง คือแปลงไฟล์ `/proc/<pid>/status` เป็น struct แล้วเรียงและตัดจำนวนก่อนพิมพ์เป็นตาราง เนื้อหาต่อจาก Linux บทที่ 4 ที่อ่าน `/proc` ด้วย `grep` ส่วนบทนี้ทำสิ่งเดียวกันด้วยโค้ด

---

## 2.1 ข้อมูลที่จะอ่าน

ทุก process มีโฟลเดอร์ `/proc/<pid>/` และไฟล์ `status` ข้างในเป็นข้อความแบบ `Key:<tab>value` บรรทัดละคู่:

```sh
$ bash -c 'grep -E "^(Name|State|PPid|VmRSS):" /proc/$$/status'
Name:	grep
State:	R (running)
PPid:	30868
VmRSS:	    1364 kB
$ grep -c . /proc/self/status
54
```

(`Name` เป็น `grep` เพราะ `exec` ของ bash แทนที่ตัวเองด้วย grep โดยใช้ PID เดิม)

ไฟล์มีราว 54 บรรทัด แต่ `labps` ใช้แค่ 4 key นี้ design ตั้งใจ **ไม่อ่าน** `/proc/<pid>/stat` เพราะชื่อ process ในไฟล์นั้นอยู่ในวงเล็บ และชื่อเองก็มีวงเล็บหรือช่องว่างได้ ทำให้ parse ผิดง่าย

| key | ใช้ทำอะไร | หมายเหตุ |
|---|---|---|
| `Name` | ชื่อ process | มีช่องว่างได้ เช่น `Web Content` |
| `State` | ตัวอักษรตัวแรก (`R/S/D/T/Z/t`) | ตัดคำในวงเล็บทิ้ง |
| `PPid` | parent PID | |
| `VmRSS` | หน่วยความจำจริง (kB) | **ไม่มี** ใน kernel thread และ zombie |

## 2.2 struct และ method

```go
// Process is the subset of /proc/<pid>/status that labps reports.
type Process struct {
	PID   int
	PPID  int
	State byte // first letter of State: R, S, D, T, Z, ...
	RSSKB int64
	Name  string
}

// Reader reads processes from Root, normally "/proc".
// Tests point Root at a fixture directory instead.
type Reader struct {
	Root string
}
```

- ชื่อที่ขึ้นต้นด้วยตัวใหญ่ (`Process`, `PID`) จะ **export** ให้ package อื่นใช้ได้ ส่วนตัวเล็กใช้ได้แค่ใน package เดียวกัน
- `Reader` มี field `Root` แทนการ hardcode `"/proc"` เพื่อให้ test ชี้ไปที่ fixture ได้ (บทที่ 3) นี่คือ anti-goal ข้อหนึ่งใน design: ห้ามอ่าน `/proc` ทั้งเครื่องแบบ hardcode

method คือ function ที่ผูกกับ type โดยเขียน receiver ไว้หน้าชื่อ:

```go
func (rd Reader) List(ctx context.Context) ([]Process, error)
```

`rd` เป็น value receiver (ได้สำเนาของ `Reader`) ซึ่งเหมาะเพราะ `Reader` เล็กและไม่ต้องแก้ตัวเอง

## 2.3 parse ทีละบรรทัด และ error ใน Go

```go
func ParseStatus(pid int, r io.Reader) (Process, error) {
	p := Process{PID: pid}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		key, val, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch key {
		case "Name":
			p.Name = val
		case "State":
			if val != "" {
				p.State = val[0]
			}
		case "PPid":
			n, err := strconv.Atoi(val)
			if err != nil {
				return p, fmt.Errorf("PPid %q: %w", val, err)
			}
			p.PPID = n
		case "VmRSS":
			n, err := strconv.ParseInt(strings.TrimSuffix(val, " kB"), 10, 64)
			if err != nil {
				return p, fmt.Errorf("VmRSS %q: %w", val, err)
			}
			p.RSSKB = n
		}
	}
	if err := sc.Err(); err != nil {
		return p, err
	}
	if p.Name == "" {
		return p, errors.New("status has no Name")
	}
	return p, nil
}
```

จุดที่ควรสังเกต:

- รับ `io.Reader` ไม่ใช่ชื่อไฟล์ จึง test ด้วย `strings.NewReader("...")` ได้โดยไม่ต้องมีไฟล์จริง
- Go **ไม่มี exception** error เป็นค่าธรรมดาที่คืนเป็นตัวสุดท้าย และผู้เรียกต้อง `if err != nil` เอง
- `fmt.Errorf("...: %w", err)` **ห่อ** error พร้อมเติมบริบท โดยยังให้ `errors.Is` มองเห็น error ต้นทางได้
- `VmRSS` ที่ไม่มีไม่ถือเป็น error ค่า `RSSKB` จึงเป็น zero value คือ `0`

## 2.4 process ที่ตายระหว่างอ่าน

`/proc` เปลี่ยนตลอดเวลา process อาจตายในช่วงระหว่างที่เรา `ReadDir` เห็นโฟลเดอร์ของมันกับตอนที่เปิด `status` ถ้าปล่อยให้ error นี้หลุดออกไป `labps list` จะพังแบบสุ่ม design จึงกำหนดว่า **ข้าม entry ที่หาย แต่รายงาน error อื่นพร้อม pid**:

```go
func (rd Reader) read(pid int) (p Process, ok bool, err error) {
	f, err := os.Open(filepath.Join(rd.Root, strconv.Itoa(pid), "status"))
	if errors.Is(err, fs.ErrNotExist) {
		return Process{}, false, nil // vanished: skip, not an error
	}
	if err != nil {
		return Process{}, false, fmt.Errorf("pid %d: %w", pid, err)
	}
	defer f.Close()
	...
}
```

`defer f.Close()` สั่งให้ปิดไฟล์ตอนที่ function จบ ไม่ว่าจะจบด้วย return ทางไหน

`List` อ่านชื่อใน `Root` เก็บเฉพาะชื่อที่เป็นตัวเลข (ข้าม `self`, `cpuinfo`, ...) เรียงตาม PID แล้วเรียก `read` ทีละตัว:

```go
func (rd Reader) List(ctx context.Context) ([]Process, error)
func Sort(ps []Process, key string) error // "pid" ascending, "rss" descending
```

## 2.5 คำสั่ง `list`: sort และ limit

subcommand `list` ใช้ `flag.NewFlagSet` แยกของตัวเอง และตรวจค่าก่อนจะอ่าน `/proc`:

```go
fs := flag.NewFlagSet("list", flag.ContinueOnError)
limit := fs.Int("limit", 0, "show at most N processes (0 = all)")
sortKey := fs.String("sort", "pid", "sort by pid or rss")
...
if *limit < 0 {
	return fmt.Errorf("%w: --limit must be >= 0, got %d", errUsage, *limit)
}
```

`errUsage` เป็น sentinel error ตัวเดียวที่ทุก usage error ห่อเอาไว้ ส่วน `run` ใช้ `errors.Is(err, errUsage)` ตัดสินว่าจะคืน `2` หรือ `1` วิธีนี้ทำให้ logic เรื่อง exit code อยู่ที่เดียว

```sh
$ ./labps list --limit 5
   PID   PPID  STATE  RSS_KB NAME
  8309  30208      R  161832 opencode
  8328   8309      S  182460 opencode
  8857  30113      S    2396 bash
  8879   8857      S    4352 proot
  8884   8879      S     588 bash
$ ./labps list --sort rss --limit 5
    PID   PPID  STATE  RSS_KB NAME
  11674   8915      S  375952 2.1.325
   8328   8309      S  182460 opencode
   8933   8915      S  163744 2.1.325
   8309  30208      S  161780 opencode
   8915   8884      S   58412 claude
$ ./labps list --limit -1; echo "exit=$?"
labps: usage error: --limit must be >= 0, got -1
usage:
  labps --version
  labps list [--limit N] [--sort pid|rss] [--json] [--workers N]
exit=2
$ LABPS_PROC_ROOT=/nope ./labps list; echo "exit=$?"
labps: open /nope: no such file or directory
exit=1
```

`LABPS_PROC_ROOT` เป็นตัวแปรที่ `labps` อ่านเพื่อเปลี่ยน root (ค่าเริ่มต้นคือ `/proc`) ใช้ใน test และใช้ทดลองกับ fixture ได้:

```sh
$ LABPS_PROC_ROOT=internal/proc/testdata/proc ./labps list
  PID  PPID  STATE  RSS_KB NAME
    1     0      S    1024 systemd
   42     1      Z       0 zombie
  300     1      R  204800 Web Content
```

ตารางใช้ `text/tabwriter` จัดคอลัมน์ และวาง `NAME` เป็นคอลัมน์สุดท้าย ชื่อที่มีช่องว่างจึงไม่ทำให้คอลัมน์อื่นเลื่อน

!!! warning "caveat: proot เห็น process แค่บางส่วน และมี state `t`"
    `/proc` ใน proot เห็นเฉพาะ process ใน subtree ของ session (เครื่องนี้มี 19–22 entries ไม่ใช่ทั้งเครื่อง) และ proot ใช้ `ptrace` ควบคุมทุก process ข้างใน บาง process จึงอยู่ใน state **`t` (tracing stop)** ซึ่งเครื่อง Linux ปกติไม่ค่อยเห็น:

    ```sh
    $ ./labps list | awk 'NR>1 {print $3}' | sort | uniq -c
          1 D
         16 S
          3 t
    ```

## 2.6 แบบฝึกหัด 3 ข้อ + เช็คความเข้าใจ 2 ข้อ

### แบบฝึกหัด

1. รัน `./labps list --sort rss --limit 3` สองครั้งห่างกัน 5 วินาที แล้วเทียบว่า `RSS_KB` ของ process ไหนเปลี่ยน และทำไม
2. สร้าง `/tmp/fakeproc/9/status` ที่มี `Name:` อย่างเดียว แล้วรัน `LABPS_PROC_ROOT=/tmp/fakeproc ./labps list` อธิบายค่า `STATE` และ `RSS_KB` ที่ได้
3. สร้าง `/tmp/fakeproc/10/status` ที่มี `PPid:\tabc` แล้วรันอีกครั้ง จด exit code และข้อความ error พร้อมอธิบายว่า pid กับ key มาจากการห่อ error ชั้นไหน

### เช็คความเข้าใจ

1. ทำไม process ที่ไม่มีไฟล์ `status` ถึงถูกข้าม แต่ `status` ที่ parse ไม่ได้กลับทำให้ทั้งคำสั่ง fail
2. ข้อดีของการรับ `io.Reader` ใน `ParseStatus` แทนการรับชื่อไฟล์คืออะไร

## สรุปบทที่ 2

- `labps` อ่านแค่ `Name/State/PPid/VmRSS` จาก `/proc/<pid>/status` และเลี่ยง `/proc/<pid>/stat`
- error เป็นค่าธรรมดา ห่อด้วย `%w` เพื่อเติมบริบทและตรวจภายหลังด้วย `errors.Is`
- process ที่ตายระหว่างอ่าน (`fs.ErrNotExist`) ต้องข้าม ไม่ใช่ fail ส่วน `VmRSS` ที่ไม่มีให้เป็น `0`
- `Reader{Root}` และ `LABPS_PROC_ROOT` ทำให้ไม่ผูกกับ `/proc` จริง ส่วนบน proot จะเห็น process แค่บางส่วน
