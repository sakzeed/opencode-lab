# Go บทที่ 5: goroutine, context, signal และการ ship binary

บทสุดท้ายเพิ่ม `--workers N` ที่อ่าน `status` หลายไฟล์พร้อมกันด้วย worker pool ใช้ `context` เพื่อหยุดงานกลางทางเมื่อกด Ctrl-C แล้วปิดท้ายด้วยการ build binary แบบ static, ฝังเวอร์ชัน และ cross-compile รวมถึง CI ที่ตรวจโค้ด Go ทุก PR

---

## 5.1 goroutine และ channel แบบสั้นที่สุด

- **goroutine** คือ function ที่รันพร้อมกับตัวอื่น เริ่มด้วย `go f()` ใช้หน่วยความจำน้อยมาก (stack เริ่มที่ไม่กี่ KB) สร้างเป็นพันตัวได้
- **channel** คือท่อที่มีชนิด ใช้ส่งค่าระหว่าง goroutine `ch <- v` คือส่ง และ `v := <-ch` คือรับ ถ้าเป็น channel ที่ไม่มี buffer ฝั่งส่งจะรอจนกว่าจะมีคนรับ
- `close(ch)` บอกว่าจะไม่มีค่าเพิ่มอีก ส่วน `for v := range ch` จะจบเองเมื่อ channel ถูกปิดและค่าที่ค้างอยู่ถูกอ่านหมด
- `sync.WaitGroup` ใช้รอให้ goroutine ทั้งกลุ่มเสร็จ

## 5.2 worker pool: `ListConcurrent`

```go
func (rd Reader) ListConcurrent(ctx context.Context, workers int) ([]Process, error)
```

สัญญาตาม design: ได้ผลเหมือน `List` ทุกประการ (เรียงตาม PID) ถ้า `workers` เป็น 0 หรือ 1 ให้กลับไปใช้ `List` และถ้า ctx ถูกยกเลิกให้คืน `ctx.Err()`

```go
results := make([]result, len(pids))
jobs := make(chan int)
var wg sync.WaitGroup
for range workers {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range jobs {
			p, ok, err := rd.read(pids[i])
			results[i] = result{p, ok, err}
		}
	}()
}
send:
for i := range pids {
	select {
	case jobs <- i:
	case <-ctx.Done():
		break send
	}
}
close(jobs)
wg.Wait()
if err := ctx.Err(); err != nil {
	return nil, err
}
```

ตัดสินใจสามเรื่อง:

1. **ส่ง index ไม่ใช่ pid** และ worker เขียนผลลง `results[i]` ช่องของตัวเอง เนื่องจากไม่มี goroutine สองตัวเขียนช่องเดียวกัน slice จึงไม่ต้องใช้ lock และผลก็เรียงตาม PID ตั้งแต่แรกโดยไม่ต้อง sort ใหม่
2. **`select` ตอนส่งงาน** ถ้า ctx ถูกยกเลิกจะเลิกแจกงานทันที แล้ว `close(jobs)` เพื่อให้ worker ออกจาก `range`
3. **`wg.Wait()` ก่อนอ่าน `results`** เพื่อรับประกันว่า worker ทุกตัวเขียนเสร็จแล้ว ถ้าอ่านก่อนจะเป็น data race

test คุมสัญญาทั้งสองข้อ:

```go
func TestListConcurrentMatchesList(t *testing.T) {
	rd := Reader{Root: fixture}
	want, _ := rd.List(context.Background())
	for _, workers := range []int{0, 1, 2, 8} {
		got, err := rd.ListConcurrent(context.Background(), workers)
		if err != nil {
			t.Fatalf("workers=%d: %v", workers, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("workers=%d: got %v, want %v", workers, pids(got), pids(want))
		}
	}
}

func TestListConcurrentCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, workers := range []int{1, 4} {
		if _, err := (Reader{Root: fixture}).ListConcurrent(ctx, workers); !errors.Is(err, context.Canceled) {
			t.Errorf("workers=%d: err = %v, want context.Canceled", workers, err)
		}
	}
}
```

!!! note "ยืนยันด้วย `-race` ไม่ได้"
    ตามบทที่ 3 race detector ใช้บนเครื่องนี้ไม่ได้ ความถูกต้องเรื่อง concurrency ของ `ListConcurrent` จึงมาจากการออกแบบ (แต่ละ goroutine เขียนช่องของตัวเอง และอ่านหลัง `wg.Wait()`) กับ test ข้างบน ไม่ได้ผ่านการตรวจด้วย `-race`

## 5.3 เร็วขึ้นจริงไหม

```sh
$ nproc
8
$ time (for i in $(seq 200); do ./labps list --workers 0 >/dev/null; done)
real	0m18.982s
user	0m0.717s
sys	0m7.434s
$ time (for i in $(seq 200); do ./labps list --workers 4 >/dev/null; done)
real	0m14.741s
user	0m1.005s
sys	0m8.018s
```

รอบสองได้ `16.985s` กับ `13.791s` ตามลำดับ (คำนวณได้ราว 85–95 ms กับ 69–74 ms ต่อครั้ง)

อ่านตัวเลขอย่างระวัง:

- `--workers 4` เร็วกว่าราว 20% ในสองรอบนี้ แต่เวลาส่วนใหญ่หมดไปกับ **การเริ่ม process และ syscall ที่ต้องผ่าน `ptrace` ของ proot** (ดู `sys` ที่สูงกว่า `user` มาก) ไม่ใช่การ parse
- มีแค่ ~20 process ให้อ่าน งานจึงเล็กเกินกว่าที่ worker pool จะได้เปรียบชัดเจน บนเครื่อง Linux ปกติที่มีหลายร้อย process ผลอาจต่างออกไป
- เครื่อง proot มี noise สูง design จึงไม่ใส่ benchmark เป็นตัวเลขยืนยัน ค่าเริ่มต้นของ `--workers` จึงยังเป็น `0` (sequential) ที่เรียบง่ายและพอเพียง

## 5.4 `context` และ signal

`context.Context` ส่งสัญญาณว่า "เลิกทำได้แล้ว" ผ่านทุกชั้นของการเรียก `signal.NotifyContext` สร้าง ctx ที่ถูกยกเลิกเมื่อได้รับ signal:

```go
case "list":
	// Ctrl-C or SIGTERM cancels ctx so workers stop instead of
	// the process being killed mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err = list(ctx, args[1:], stdout, stderr)
	stop()
```

`labps list` เสร็จในไม่กี่สิบ ms จึงกด Ctrl-C ทันได้ยาก ลองดูกลไกเดียวกันกับโปรแกรมที่ทำงานวนไปเรื่อย ๆ:

```go
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	for n := 1; ; n++ {
		select {
		case <-ctx.Done():
			fmt.Println("cleanup: got", context.Cause(ctx))
			os.Exit(130)
		case <-tick.C:
			fmt.Println("working", n)
		}
	}
}
```

```sh
$ timeout --preserve-status -s INT 1 ./sigdemo; echo "exit=$?"
working 1
working 2
working 3
cleanup: got interrupt signal received
exit=130
$ timeout -s TERM 1 ./sigdemo; echo "exit=$?"
working 1
working 2
working 3
cleanup: got terminated signal received
exit=124
$ timeout -s KILL 1 ./sigdemo; echo "exit=$?"
working 1
working 2
working 3
/bin/bash: line 69: 11590 Killed                  timeout -s KILL 1 ./sigdemo
exit=137
```

- SIGINT และ SIGTERM ถูก **จับได้** โปรแกรมจึงทำ cleanup ก่อนออก ส่วน SIGKILL จับไม่ได้ (Linux บทที่ 4)
- `timeout` คืน `124` เมื่อหมดเวลา ถ้าอยากเห็น exit code ของโปรแกรมเอง ต้องใช้ `--preserve-status`
- `130` = 128 + 2 (SIGINT) เป็นแบบแผนเดียวกับ shell

## 5.5 ship: build แบบ static, ฝังเวอร์ชัน และ cross-compile

```sh
$ go build -o labps . && ls -lh labps
-rwxr-xr-x. 3.9M labps
$ ldd ./labps
	not a dynamic executable
```

`labps` ใช้แค่ stdlib และไม่ได้ใช้ `net` หรือ `os/user` (ซึ่งจะดึง cgo เข้ามา) จึงได้ binary แบบ static แม้ว่า `CGO_ENABLED=1` ก็ตาม แต่ถ้าจะให้แน่ใจตอน release ให้ระบุ `CGO_ENABLED=0` ไว้เลย ส่วน `-ldflags="-s -w"` ใช้ตัด symbol table กับ debug info ออก:

```sh
$ CGO_ENABLED=0 go build -ldflags="-s -w" -o labps-small . && ls -lh labps-small
-rwxr-xr-x. 2.6M labps-small
```

(เครื่องนี้ไม่มีคำสั่ง `file` จึงใช้ `ls -lh` ดูขนาด และ `ldd` ดูว่า link แบบ dynamic หรือไม่)

**ฝังเวอร์ชันตอน build:** `-X` เขียนทับค่าของ **ตัวแปร** string ระดับ package ได้ แต่ใช้กับ `const` ไม่ได้ `labps` จึงประกาศ `version` เป็น `var`:

```go
// version is a var, not a const, so release builds can set it with
// go build -ldflags="-X main.version=...".
var version = "0.1.0"
```

```sh
$ go build -ldflags="-X main.version=0.2.0-dev" -o labps-dev . && ./labps-dev --version
labps 0.2.0-dev
$ go version -m labps | head -6
labps: go1.27.4
	path	github.com/sakzeed/opencode-lab/labs/labps
	mod	github.com/sakzeed/opencode-lab/labs/labps	(devel)
	build	-buildmode=exe
	build	-compiler=gc
	build	CGO_ENABLED=1
```

`go version -m` อ่านข้อมูล build ที่ Go ฝังไว้ใน binary ทุกตัว มีประโยชน์เวลาอยากรู้ว่า binary ที่เจอ build มาด้วย Go เวอร์ชันไหนและตั้งค่าอะไรไว้

**cross-compile:** แค่เปลี่ยน `GOOS`/`GOARCH` ไม่ต้องติดตั้ง toolchain เพิ่ม:

```sh
$ CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o labps-amd64 . && ls -lh labps-amd64
-rwxr-xr-x. 4.1M labps-amd64
$ ./labps-amd64 --version; echo "exit=$?"
/bin/bash: line 17: 13002 Illegal instruction     ./labps-amd64 --version
exit=132
$ head -c 20 labps-amd64 | od -An -tx1
 7f 45 4c 46 02 01 01 00 00 00 00 00 00 00 00 00
 02 00 3e 00
$ head -c 20 labps | od -An -tx1
 7f 45 4c 46 02 01 01 00 00 00 00 00 00 00 00 00
 02 00 b7 00
```

binary สำหรับ amd64 รันบนเครื่อง aarch64 นี้ไม่ได้ (exit `132` = 128 + 4, SIGILL) แต่ดูได้จาก ELF header ว่าเป็นคนละสถาปัตยกรรม: byte ที่ offset 18 (นับจาก 0) คือ `e_machine` ซึ่ง `3e` = x86-64 และ `b7` = AArch64 (เป็นวิธีแทน `file` ที่เครื่องนี้ไม่มี)

## 5.6 CI: ตรวจ Go ทุก PR

`ci.yml` มี path `labs/**` และ job `go` ที่ทำงานคู่กับ job `check` (validator + `mkdocs build`) โดยที่ `pages.yml` ไม่ต้องรู้จัก Go เลย:

```yaml
  go:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: labs/labps
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v7
        with:
          go-version-file: labs/labps/go.mod
      - run: test -z "$(gofmt -l .)" || (gofmt -l .; exit 1)
      - run: go vet ./...
      - run: go test ./...
```

- `go-version-file` อ่านเวอร์ชันจาก `go.mod` ไม่ต้องเขียนซ้ำไว้สองที่
- `gofmt -l` พิมพ์ชื่อไฟล์ที่ยังไม่ได้ format และไม่ได้ fail เอง จึงต้องใช้ `test -z` ครอบ ถ้ามีชื่อไฟล์ออกมาให้พิมพ์ซ้ำแล้ว `exit 1`
- ไม่รัน `-race` ตาม caveat ในบทที่ 3

ก่อน push ให้รันสามคำสั่งเดียวกันในเครื่อง (ใน `labs/labps`):

```sh
$ test -z "$(gofmt -l .)" && go vet ./... && go test ./...
ok  	github.com/sakzeed/opencode-lab/labs/labps	0.124s
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/proc	0.134s
ok  	github.com/sakzeed/opencode-lab/labs/labps/internal/render	0.072s
```

## 5.7 แบบฝึกหัด 3 ข้อ + เช็คความเข้าใจ 2 ข้อ

### แบบฝึกหัด

1. ลบ `wg.Wait()` ออกจาก `ListConcurrent` ชั่วคราวแล้วรัน `go test -count=20 ./internal/proc` จดว่าเกิดอะไรขึ้น และอธิบายว่าทำไมผลถึงไม่แน่นอน (เสร็จแล้ว revert)
2. วัดเวลา `--workers 0`, `2`, `8` อย่างละ 200 ครั้งด้วยลูป `time` แบบในหัวข้อ 5.3 แล้วรันซ้ำอีกรอบ สรุปว่าความต่างมากกว่า noise ระหว่างสองรอบหรือไม่
3. build `labps` สำหรับ `GOARCH=arm GOARM=7` แล้วใช้ `od` ดู byte ที่ offset 4 (`EI_CLASS`: `01` = 32 บิต, `02` = 64 บิต) และ offset 18 เทียบกับ binary ที่ build สำหรับเครื่องนี้

### เช็คความเข้าใจ

1. ทำไม worker ใน `ListConcurrent` ถึงเขียน `results[i]` ได้โดยไม่ต้องมี `sync.Mutex` แล้วเงื่อนไขอะไรที่ถ้าไม่เป็นจริงจะทำให้เกิด race
2. ถ้าจะให้ `labps --version` บน CI แสดงชื่อ commit ต้องแก้ตรงไหนบ้าง (คำใบ้: `-ldflags -X` และ `git rev-parse --short HEAD`)

## สรุปบทที่ 5

- worker pool = channel ของงาน + goroutine N ตัว + `WaitGroup` ถ้าเขียนผลลง index ของตัวเองก็ไม่ต้อง lock และได้ลำดับเดิม
- `signal.NotifyContext` แปลง SIGINT/SIGTERM เป็นการยกเลิก ctx ทำให้หยุดงานอย่างเรียบร้อย ส่วน SIGKILL จับไม่ได้
- บน proot `--workers` เร็วขึ้นเล็กน้อยแต่มี noise สูง จึงรายงานตามที่วัดได้และคงค่าเริ่มต้นเป็น sequential
- binary เป็น static อยู่แล้ว (`ldd`: not a dynamic executable) `-ldflags="-s -w"` ลดขนาด, `-X main.version` ฝังเวอร์ชัน และ `GOOS/GOARCH` ใช้ cross-compile
- CI รัน `gofmt`, `go vet`, `go test` ทุก PR ที่แตะ `labs/**`
