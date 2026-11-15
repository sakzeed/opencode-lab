# Go บทที่ 1: toolchain, module และ CLI แรก

track Go พาจาก "ไม่เคยเขียน Go" ไปจนมี CLI tool ชื่อ **`labps`** ที่อ่าน `/proc` แล้วรายงาน process เป็นตารางหรือ JSON มี test และ build เป็น binary ได้ ทั้ง track ใช้ **stdlib อย่างเดียว** ไม่มี dependency ภายนอก

โค้ดฉบับเต็มอยู่ใน repo ที่ [`labs/labps/`](https://github.com/sakzeed/opencode-lab/tree/main/labs/labps) แต่ละบทจะเพิ่มความสามารถทีละชั้น บทนี้ทำแค่โครงกับ `--version`

ทุก output ในบท Go รันจริงบน **Ubuntu 24.04 (proot-distro บน Android, aarch64)** ด้วย Go 1.27.4 ส่วนเลข PID ขนาดไฟล์ และเวลา จะต่างไปตาม session

---

## 1.1 ติดตั้ง toolchain และตั้ง PATH

เครื่องนี้มี `go` จาก apt อยู่แล้ว แต่เป็นเวอร์ชันเก่า:

```sh
$ command -v go
/usr/bin/go
$ go version
go version go1.22.2 linux/arm64
```

Go 1.27.4 ถูกแตกไว้ที่ `~/.local/opt/go-v1.27.4` และมี symlink `~/.local/opt/go` ชี้ไปหา ทำให้ตอนอัปเกรดแค่เปลี่ยน symlink โดยไม่ต้องแก้ PATH เอา `bin` ของมันขึ้นหน้า PATH:

```sh
$ export PATH="$HOME/.local/opt/go/bin:$PATH"
$ command -v go
~/.local/opt/go/bin/go
$ go version
go version go1.27.4 linux/arm64
$ go env GOROOT GOPATH GOOS GOARCH
~/.local/opt/go-v1.27.4
~/go
linux
arm64
```

ให้ทุก shell ใหม่ได้ PATH นี้ด้วยการเพิ่ม block ท้าย `~/.bashrc` เขียนเป็น `case` เพื่อให้ **รันซ้ำแล้วไม่เพิ่มซ้ำ** (idempotent):

```sh
# Go toolchain (opencode-lab)
case ":$PATH:" in *":$HOME/.local/opt/go/bin:"*) ;; *) export PATH="$HOME/.local/opt/go/bin:$PATH" ;; esac
```

ตรวจด้วย `bash -ic 'command -v go'` ต้องได้ `~/.local/opt/go/bin/go` และ `echo "$PATH" | tr : '\n' | grep -c .local/opt/go/bin` ต้องได้ `1` แม้จะ `source ~/.bashrc` ซ้ำหลายรอบ

| ตัวแปร | ความหมาย |
|---|---|
| `GOROOT` | ที่อยู่ของ toolchain (compiler, stdlib) |
| `GOPATH` | ที่เก็บ module cache (`~/go/pkg/mod`) และ binary จาก `go install` (`~/go/bin`) |
| `GOOS` / `GOARCH` | OS และ CPU ที่ binary จะไปรัน เปลี่ยนเพื่อ cross-compile ได้ (บทที่ 5) |

!!! warning "caveat: proot กับ toolchain ที่ Go ดาวน์โหลดเอง"
    Go 1.21 ขึ้นไปมี `GOTOOLCHAIN=auto` ถ้า `go.mod` ต้องการเวอร์ชันใหม่กว่าตัวที่มี มันจะดาวน์โหลด toolchain มาเก็บใน module cache เอง แต่บน proot เครื่องนี้ไฟล์ใน cache ถูกตั้งเป็น `-r--r--r--` (ไม่มีสิทธิ์ execute) จึงรันไม่ได้ ส่วน `go1.22` ตัวเก่าก็อ่าน `go 1.27` ใน `go.mod` เป็นชื่อ toolchain ที่ไม่มีอยู่จริง:

    ```sh
    $ cd labs/labps && /usr/bin/go version
    go: downloading go1.27 (linux/arm64)
    go: download go1.27 for linux/arm64: toolchain not available
    ```

    ทางแก้คือติดตั้ง toolchain เองแล้วตั้ง PATH ตามข้างบน อย่าพึ่งการดาวน์โหลดอัตโนมัติ

## 1.2 module คืออะไร

**module** คือหน่วยของโค้ด Go ที่มี `go.mod` อยู่ที่ราก บอกชื่อ module (ใช้เป็น prefix ตอน import) และเวอร์ชันภาษาขั้นต่ำ:

```sh
$ go mod init example.com/hello
go: creating new go.mod: module example.com/hello
$ cat go.mod
module example.com/hello

go 1.27.4
```

`labps` ใช้ชื่อ module ตาม path ใน GitHub เพราะเป็นแบบแผนที่ `go install` และ import path เข้าใจ:

```text
module github.com/sakzeed/opencode-lab/labs/labps

go 1.27
```

`labps` อยู่ใน subfolder ของ repo ที่เป็นเว็บ docs จึงแยกเป็น module ของตัวเอง เพื่อให้ `go test ./...` เห็นแค่โค้ด Go และไม่ไปยุ่งกับส่วนอื่น

## 1.3 โปรแกรมแรก: `flag` และ exit code

```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	name := flag.String("name", "world", "who to greet")
	flag.Parse()
	if *name == "" {
		fmt.Fprintln(os.Stderr, "hello: --name must not be empty")
		os.Exit(2)
	}
	fmt.Printf("hello, %s\n", *name)
}
```

- `package main` + `func main()` = โปรแกรมที่รันได้ (package อื่นเป็น library)
- `flag.String` คืน **pointer** (`*string`) เพราะค่าจริงจะถูกเติมตอน `flag.Parse()`
- ข้อความ error ไปที่ `os.Stderr` ส่วนผลลัพธ์ไป stdout เพื่อให้ pipe ต่อได้โดยไม่ปนกัน (Linux บทที่ 2)

```sh
$ go run . --name proot
hello, proot
$ go build -o hello . && ./hello
hello, world
$ ./hello --name ""; echo "exit=$?"
hello: --name must not be empty
exit=2
$ ./hello --nme x; echo "exit=$?"
flag provided but not defined: -nme
Usage of ./hello:
  -name string
    	who to greet (default "world")
exit=2
$ ./hello -h
Usage of ./hello:
  -name string
    	who to greet (default "world")
exit=0
```

`go run` compile ไว้ในโฟลเดอร์ชั่วคราวแล้วรัน ส่วน `go build -o` ได้ไฟล์ binary ที่เก็บไว้ใช้ได้

exit code เป็นสัญญากับคนที่เรียกโปรแกรม (shell, script, CI) — `labps` ใช้แบบแผนเดียวกับเครื่องมือ Unix ส่วนใหญ่:

| exit | ความหมาย | ตัวอย่าง |
|---|---|---|
| `0` | สำเร็จ | `labps --version` |
| `1` | runtime error | อ่าน `/proc` ไม่ได้ |
| `2` | usage error | flag ผิด, ค่าไม่ถูกต้อง, คำสั่งที่ไม่รู้จัก |

## 1.4 `labps` เวอร์ชันแรก: `main` ต้องบาง

ถ้าเรียก `os.Exit` ใน `main` ตรง ๆ จะเขียน test ไม่ได้ เพราะ test จะตายไปพร้อมกัน `labps` จึงแยก logic ไปไว้ใน `run` ที่ **คืน** exit code และรับ `io.Writer` แทนการเขียนลง `os.Stdout` ตรง ๆ:

```go
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main without os.Exit so tests can call it.
// Exit codes: 0 ok, 1 runtime error, 2 usage error.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "--version", "-version":
		fmt.Fprintf(stdout, "labps %s\n", version)
		return 0
	default:
		fmt.Fprintf(stderr, "labps: unknown command %q\n%s", args[0], usage)
		return 2
	}
}
```

test เรียก `run` แล้วเก็บ output ไว้ใน `bytes.Buffer` (ซึ่งเป็น `io.Writer` ตัวหนึ่ง):

```go
func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--version"}, &out, &errb); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr %q)", code, errb.String())
	}
	if got, want := out.String(), "labps 0.1.0\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}
```

build แล้วลองบนเครื่องจริง (ข้อความ error ด้านล่างมาจากโค้ดฉบับเต็มที่ห่อ usage error ไว้ ซึ่งจะพูดถึงในบทที่ 2):

```sh
$ go build -o labps . && ./labps --version
labps 0.1.0
$ ./labps; echo "exit=$?"
usage:
  labps --version
  labps list [--limit N] [--sort pid|rss] [--json] [--workers N]
exit=2
$ ./labps ps; echo "exit=$?"
labps: usage error: unknown command "ps"
usage:
  labps --version
  labps list [--limit N] [--sort pid|rss] [--json] [--workers N]
exit=2
```

binary `labs/labps/labps` อยู่ใน `.gitignore` เพราะเป็นผลจากการ build ไม่ใช่ source

## 1.5 แบบฝึกหัด 3 ข้อ + เช็คความเข้าใจ 2 ข้อ

### แบบฝึกหัด

1. เพิ่ม flag `--shout` (bool) ให้ `hello` ที่พิมพ์ชื่อเป็นตัวพิมพ์ใหญ่ (ใช้ `strings.ToUpper`) แล้วตรวจว่า `./hello --shout --name go` ได้ `hello, GO`
2. รัน `./labps --help; echo $?` แล้วอธิบายว่าทำไมได้ exit `2` ไม่ใช่ `0` (ดูว่า `run` เช็ค `args[0]` กับค่าอะไรบ้าง)
3. เปิด shell ใหม่ที่ไม่มี block PATH ใน `~/.bashrc` แล้วรัน `go version` ในโฟลเดอร์ `labs/labps` จดว่าได้ error แบบไหน แล้วแก้ด้วย `export PATH=...`

### เช็คความเข้าใจ

1. ทำไม `flag.String` ถึงคืน `*string` แทนที่จะคืน `string`
2. ถ้า `main` เรียก `os.Exit(2)` ตรง ๆ จะเกิดอะไรขึ้นกับ test ที่อยากตรวจ usage error

## สรุปบทที่ 1

- ติดตั้ง Go เองไว้ที่ `~/.local/opt/go` แล้วตั้ง PATH แบบ idempotent ใน `~/.bashrc` บน proot อย่าพึ่ง `GOTOOLCHAIN=auto`
- `go.mod` บอกชื่อ module และเวอร์ชันภาษา ส่วน `labps` เป็น module แยกอยู่ใน `labs/labps/`
- `flag` ให้ parse flag กับข้อความ usage มาฟรี และ exit code คือ `0` สำเร็จ / `1` runtime / `2` usage
- `main` ทำแค่ `os.Exit(run(...))` เพื่อให้ logic ทั้งหมด test ได้ผ่าน `io.Writer`
