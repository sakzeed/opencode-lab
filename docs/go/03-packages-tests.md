# Go บทที่ 3: package, test, fixture และ go vet

เมื่อ `labps` เริ่มมีหลายส่วน บทนี้แยกโค้ดเป็น package ภายใน และสร้างตาข่ายกันพลาดสามชั้น คือ table test, fixture ที่จำลอง `/proc` และ golden file สำหรับตาราง จากนั้นใช้ `go vet` ตรวจสิ่งที่ compiler ไม่จับ ปิดท้ายด้วย caveat เรื่อง `-race` ที่ใช้ไม่ได้บนเครื่องนี้

---

## 3.1 แยก package ด้วย `internal/`

```text
labs/labps/
  go.mod
  main.go                 flag, exit code — บางที่สุด
  main_test.go
  internal/proc/          อ่าน /proc: Process, Reader, ParseStatus, List, Sort
  internal/render/        ตาราง + JSON ลง io.Writer
```

```sh
$ go list ./...
github.com/sakzeed/opencode-lab/labs/labps
github.com/sakzeed/opencode-lab/labs/labps/internal/proc
github.com/sakzeed/opencode-lab/labs/labps/internal/render
```

- 1 โฟลเดอร์ = 1 package import ด้วย path เต็ม เช่น `.../labs/labps/internal/proc` แล้วเรียกเป็น `proc.Reader`
- โฟลเดอร์ชื่อ **`internal`** มีกติกาพิเศษ: import ได้เฉพาะโค้ดที่อยู่ใต้โฟลเดอร์แม่ของมัน (`labs/labps/...`) module อื่นจึงเอาไปใช้ไม่ได้ ทำให้เราแก้ API ได้โดยไม่ทำให้ใครพัง
- ทิศทางการ import ไปทางเดียว คือ `main → render → proc` โดยที่ `proc` ไม่รู้จัก `render` (Go ไม่ยอมให้ import วนกัน)

## 3.2 test ใน Go

ไฟล์ test ลงท้ายด้วย `_test.go` อยู่ใน package เดียวกับโค้ด และ function ชื่อ `TestXxx(t *testing.T)`:

```go
func TestParseStatusNoVmRSS(t *testing.T) {
	got, err := ParseStatus(42, strings.NewReader("Name:\tzombie\nState:\tZ (zombie)\nPPid:\t1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.RSSKB != 0 || got.State != 'Z' || got.PPID != 1 {
		t.Errorf("got %+v", got)
	}
}
```

| เรียก | ผล |
|---|---|
| `t.Errorf` | บันทึกว่า fail แล้ว **ทำต่อ** (เห็นหลายจุดในรอบเดียว) |
| `t.Fatalf` / `t.Fatal` | บันทึกแล้ว **หยุด test นี้ทันที** (ใช้เมื่อทำต่อไม่มีความหมาย เช่น err ไม่ใช่ nil) |
| `t.Helper()` | ทำให้บรรทัดที่รายงานชี้ไปที่ผู้เรียก helper |
| `t.Setenv` | ตั้ง env เฉพาะ test นี้ แล้วคืนค่าเดิมให้เองตอนจบ |

```sh
$ go test ./...
ok  	github.com/sakzeed/opencode-lab/labs/labps	0.119s
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/proc	0.147s
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/render	0.042s
$ go test -v -run "TestParse|TestListSkips" ./internal/proc
=== RUN   TestParseStatus
--- PASS: TestParseStatus (0.00s)
=== RUN   TestParseStatusNoVmRSS
--- PASS: TestParseStatusNoVmRSS (0.00s)
=== RUN   TestParseStatusMissingName
--- PASS: TestParseStatusMissingName (0.00s)
=== RUN   TestListSkipsVanished
--- PASS: TestListSkipsVanished (0.01s)
PASS
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/proc	0.099s
```

`-run` รับ regex ของชื่อ test และ `-count=1` ใช้ปิด cache เวลาอยากให้รันจริงทุกครั้ง

### table test

เมื่อหลายกรณีตรวจแบบเดียวกัน ให้เก็บ input ไว้ใน slice แล้ววนลูป เพิ่มกรณีใหม่ได้ในบรรทัดเดียว:

```go
func TestRunUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"list", "--sort", "cpu"},
		{"list", "--limit", "-1"},
		{"list", "--bogus"},
		{"list", "extra"},
		{"frobnicate"},
	} {
		code, _, errs := runFixture(t, args...)
		if code != 2 {
			t.Errorf("%v: exit = %d, want 2", args, code)
		}
		if errs == "" {
			t.Errorf("%v: want message on stderr", args)
		}
	}
}
```

## 3.3 fixture: `/proc` ปลอมใน `testdata/`

`go` จะไม่ build โฟลเดอร์ชื่อ `testdata` ที่นี่จึงเป็นที่เก็บไฟล์ input ของ test ส่วน `labps` จำลอง procfs ไว้แบบนี้:

```text
internal/proc/testdata/proc/
  1/status      systemd, S, PPid 0, VmRSS 1024 kB
  42/status     zombie, Z, PPid 1, ไม่มี VmRSS
  300/status    "Web Content" (มีช่องว่าง), R, VmRSS 204800 kB
  77/           โฟลเดอร์ไม่มี status = process ที่ตายระหว่างอ่าน
  self/status   ชื่อไม่ใช่ตัวเลข ต้องถูกข้าม
```

fixture แต่ละตัวคุมพฤติกรรมหนึ่งอย่างที่ design ระบุไว้:

```go
// pid 77 is a directory without status: the process exited between
// ReadDir and ReadFile. It must be skipped, not fail the whole list.
func TestListSkipsVanished(t *testing.T) {
	ps, err := Reader{Root: fixture}.List(context.Background())
	if err != nil {
		t.Fatalf("vanished pid should be skipped, got %v", err)
	}
	...
}
```

git ไม่เก็บโฟลเดอร์ว่าง `77/` จึงมีไฟล์ `.keep` อยู่ข้างใน ไฟล์นี้ไม่ใช่ `status` ผลจึงยังเป็น "process ที่หายไป"

test ของ `main` ชี้ไปที่ fixture เดียวกันผ่าน `LABPS_PROC_ROOT` ด้วย `t.Setenv` ทำให้ทดสอบทั้งเส้นทาง flag → อ่าน → render → exit code ได้โดยไม่แตะ `/proc` จริง

## 3.4 golden file: ตรวจ output ทั้งก้อน

ตารางมีช่องว่างกับการจัดคอลัมน์เยอะ ถ้าจะ assert ทีละบรรทัดจะเปราะ วิธีที่ดีกว่าคือเก็บ output ที่ถูกต้องไว้ในไฟล์ (`testdata/table.golden`) แล้วเทียบทั้งก้อน และเพิ่ม flag `-update` ไว้ใช้เขียนไฟล์ใหม่เมื่อ **ตั้งใจ** เปลี่ยนรูปแบบ:

```go
var update = flag.Bool("update", false, "rewrite golden files")

func TestTableGolden(t *testing.T) {
	var buf bytes.Buffer
	if err := Table(&buf, sample); err != nil {
		t.Fatal(err)
	}
	const golden = "testdata/table.golden"
	if *update {
		if err := os.WriteFile(golden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, _ := os.ReadFile(golden)
	if got := buf.String(); got != string(want) {
		t.Errorf("table mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}
```

ลองแก้หัวตารางจาก `"PID\t..."` เป็น `"PID \t..."` (เพิ่มช่องว่าง 1 ตัว) ในสำเนาของโค้ด:

```sh
$ go test ./internal/render
--- FAIL: TestTableGolden (0.00s)
    render_test.go:37: table mismatch
         got:
          PID   PPID  STATE  RSS_KB NAME
             1     0      S    1024 systemd
            42     1      Z       0 zombie
           300     1      R  204800 Web Content

        want:
          PID  PPID  STATE  RSS_KB NAME
            1     0      S    1024 systemd
           42     1      Z       0 zombie
          300     1      R  204800 Web Content
FAIL
```

ช่องว่างตัวเดียวทำให้ทุกคอลัมน์เลื่อน ซึ่ง golden จับได้ทันที ถ้าตั้งใจเปลี่ยนจริงให้รัน `go test ./internal/render -update` แล้ว **อ่าน `git diff` ของไฟล์ golden ก่อน commit** เพราะ `-update` ทำให้ test ผ่านเสมอ คนต้องเป็นผู้ตรวจว่า output ใหม่ถูกต้อง

## 3.5 coverage และ `go vet`

```sh
$ go test -cover ./...
ok  	github.com/sakzeed/opencode-lab/labs/labps	0.327s	coverage: 90.2% of statements
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/proc	0.349s	coverage: 88.2% of statements
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/render	0.245s	coverage: 100.0% of statements
```

coverage บอกว่าบรรทัดไหนถูกรันบ้าง แต่ไม่ได้บอกว่าตรวจถูก ใช้หาโค้ดที่ไม่มี test เลยพอ อย่าไล่ตัวเลข 100%

`go vet` ตรวจสิ่งที่ compile ผ่านแต่น่าจะผิด เช่น format string ไม่ตรงกับชนิดของ argument:

```go
func vetDemo(name string) string {
	return fmt.Sprintf("pid %d", name)
}
```

```sh
$ go vet .
vetdemo.go:6:26: fmt.Sprintf format %d has arg name of wrong type string
$ echo $?
1
```

โค้ดจริงของ `labps` ต้อง vet ผ่าน (`go vet ./...` ไม่มี output และ exit `0`) และ CI ก็รันคำสั่งนี้ทุก PR (บทที่ 5)

## 3.6 caveat: `go test -race` ใช้ไม่ได้บนเครื่องนี้

race detector (`-race`) หา data race ตอนรันจริง โดยต้องใช้ cgo และ ThreadSanitizer บนเครื่องนี้มี `gcc` และ `CGO_ENABLED=1` แต่ยังรันไม่ได้:

```sh
$ go env CC CGO_ENABLED
gcc
1
$ go test -race ./internal/proc
FATAL: ThreadSanitizer: unsupported VMA range
FATAL: Found 39 - Supported 48
FAIL	github.com/sakzeed/opencode-lab/labs/labps/internal/proc	0.036s
FAIL
$ echo $?
1
```

kernel ของ Android ใช้ virtual address 39 บิต แต่ ThreadSanitizer บน arm64 ต้องการ 48 บิต ซึ่งเป็นข้อจำกัดของ kernel แก้จากใน proot ไม่ได้

- **อย่า** ซ่อน error นี้ด้วย `|| true` หรือตัด `-race` ออกโดยไม่บอก เพราะจะทำให้ดูเหมือนว่าตรวจแล้ว
- CI ของ track นี้ก็ **ไม่** รัน `-race` ตามที่ design ตัดสิน ถ้าอนาคตต้องการ ให้เพิ่มเป็น job แยกบน runner ที่รองรับ
- บทที่ 5 ออกแบบ `ListConcurrent` ให้ goroutine แต่ละตัวเขียน slice คนละช่อง จึงไม่มี memory ที่แชร์กันโดยไม่มีการป้องกัน ทั้งหมดนี้เป็นเหตุผลเชิงออกแบบ ไม่ได้พิสูจน์ด้วย `-race`

## 3.7 แบบฝึกหัด 3 ข้อ + เช็คความเข้าใจ 2 ข้อ

### แบบฝึกหัด

1. เพิ่ม fixture `testdata/proc/500/status` ที่ `VmRSS:` มีค่าเป็น `abc kB` แล้วเขียน test ที่คาดว่า `List` คืน error ที่มีคำว่า `pid 500` (ระวัง test อื่นที่นับจำนวน PID ไว้ อาจต้องแยก fixture ไปไว้ที่ root ใหม่)
2. แก้ `Table` ให้หัวคอลัมน์เป็น `RSS(kB)` รัน test ให้เห็นว่า fail แล้วอัปเดต golden ด้วย `-update` พร้อมอ่าน diff (เสร็จแล้ว revert)
3. เขียน `TestSortStable`: process สองตัวที่ `RSSKB` เท่ากัน ต้องคงลำดับเดิมหลัง `Sort(ps, "rss")`

### เช็คความเข้าใจ

1. ทำไม `proc` ถึง import `render` ไม่ได้ แต่ `render` import `proc` ได้ และถ้าฝืน import วนจะเกิดอะไรขึ้น
2. golden file กับ `-update` มีความเสี่ยงอะไร และป้องกันอย่างไร

## สรุปบทที่ 3

- 1 โฟลเดอร์ = 1 package ส่วน `internal/` กันไม่ให้ module อื่น import และ dependency ต้องไปทางเดียว
- `TestXxx` + `t.Errorf/Fatalf` + table test ใช้ครอบหลายกรณีได้ในที่เดียว
- `testdata/` เก็บ `/proc` ปลอมที่จำลองทุก edge case (zombie, ชื่อมีช่องว่าง, process หาย)
- golden file ตรวจตารางทั้งก้อน ส่วน `-update` ต้องตามด้วยการอ่าน diff เสมอ
- `go vet` ต้องผ่าน ส่วน `-race` ใช้ไม่ได้บน kernel Android (VMA 39 บิต) และห้ามซ่อนผลนี้
