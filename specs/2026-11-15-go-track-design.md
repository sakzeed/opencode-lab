# Go track (CLI tools) — design

วันที่: 2026-11-15
สถานะ: อนุมัติแนวทาง A1 แล้ว (ผู้ใช้: "เอา") — รออนุมัติ design doc

## ทำไม

track ที่มีอยู่ (OpenCode, Linux, Git) สอนเครื่องมือและระบบ แต่ยังไม่มี track ที่ลงมือเขียนโปรแกรมจริง ผู้ใช้เลือกทาง **A: CLI tools** เพราะต่อยอด Linux บทที่ 4 (process/`/proc`/signal) และได้ tool ที่ใช้บนเครื่องนี้ได้จริง โดยไม่ต้องพึ่ง dependency หรือ DB

## เป้าหมาย

- track ใหม่ 5 บทใน `docs/go/` ที่พาคนอ่านจาก "ไม่เคยเขียน Go" ถึง "มี CLI tool ที่มี test และ build binary ได้"
- โปรเจกต์ `labps` ตัวเดียวโตทั้ง track: อ่าน `/proc` แล้วรายงาน process เป็นตาราง/JSON
- โค้ดจริงใน `labs/labps/` (module แยก) พร้อม test ที่รันผ่านบนเครื่องนี้และบน CI
- CI ตรวจ Go code ด้วย (`gofmt`, `go vet`, `go test`) นอกเหนือจาก validator/build เดิม
- publish ผ่าน workflow เดิม: validator 13 → 18 หน้า, deploy ไม่เปลี่ยน

## สิ่งที่ไม่ทำ

- HTTP server / database / UI (ทาง B)
- generics เชิงลึก, reflection, cgo, plugin
- third-party dependency — stdlib เท่านั้น
- reimplement `ps` ครบทุก field (ทาง A3)
- แก้ deploy pipeline (`pages.yml` ไม่ต้องรู้จัก Go)

## ข้อจำกัดจากเครื่องนี้ (ทดสอบแล้ว)

- Go **1.27.4 linux/arm64** อยู่ที่ `/root/.local/opt/go-v1.27.4`; `go` ยังไม่อยู่ใน `PATH`; `~/.bashrc` ยังไม่มี entry (137 บรรทัด)
- `go run`, `go build`, `go test`, `go vet`, `CGO_ENABLED=0 go build` ผ่าน; cgo program รันได้
- **`go test -race` ใช้ไม่ได้**: linker ของ Termux gcc ฟ้อง `gcc: error: linker command failed` — บทที่ 3 ต้องแสดงหลักฐานและ caveat นี้ ห้ามอ้างว่าใช้ได้
- `file` ไม่มีในเครื่อง — บทที่ 5 ใช้ `ls -lh` แทน
- `/proc` ของ proot เห็นเฉพาะ subtree (~41–44 entries) และมี zombie ค้าง — tool ต้องทนต่อ entry ที่หายระหว่างอ่าน
- `setup-go@v7` มีจริง (latest v7.0.0) และ `go-version-file` ใช้ได้

## ทางเลือก

| ทาง | เนื้อหา | trade-off | สรุป |
|---|---|---|---|
| **A1. tool เดียวโตทั้ง track (เลือก)** | `labps` วิวัฒน์ทุกบท + test สะสม | เห็นการออกแบบจริงและจบด้วยของใช้ได้; บทหลังอ้างโค้ดเดิม | **เลือก** |
| A2. มินิโปรเจกต์แยกบท | โปรแกรมสั้นบทละตัว | สั้น แต่ไม่ได้ฝึก refactor และไม่มีของจริงชิ้นเดียว | ไม่เลือก |
| A3. reimplement `ps` เต็ม | โปรเจกต์เดียวใหญ่ | ลึกแต่ชันเกินสำหรับบทต้น | ไม่เลือก |

## Decisions (ฝ่ายเราตัดสิน)

### ตำแหน่งโค้ดและ module

```
labs/labps/
  go.mod                 module github.com/sakzeed/opencode-lab/labs/labps, go 1.27
  main.go                flag parsing, exit codes, เรียก internal packages
  internal/proc/         อ่าน /proc, struct Process, parse, List, ListConcurrent
  internal/render/       ตาราง + JSON ลง io.Writer
  internal/proc/testdata/proc/<pid>/status   fixture
```

- `Reader{Root string}` รับ path ของ procfs เพื่อให้ test ชี้ fixture ได้ (ไม่ผูกกับ `/proc` จริง)
- `Process{PID int; PPID int; State byte; RSSKB int64; Name string}`
- `List(ctx) ([]Process, error)` (บท 2) และ `ListConcurrent(ctx, workers int)` (บท 5)
- ข้าม entry ที่หายระหว่างอ่าน (process ตาย) ไม่ทำให้ทั้ง list fail; error อื่นรายงานพร้อม pid
- อ่านเฉพาะ `/proc/<pid>/status` (Name/State/PPid/VmRSS) — เลี่ยง parsing `/proc/<pid>/stat` ที่มีชื่อในวงเล็บซับซ้อน
- `State` เก็บตัวแรกของ `State:` (`R/S/D/T/Z`)

### CLI

```
labps --version
labps list [--limit N] [--sort pid|rss] [--json] [--workers N]
```

- default: `--limit 0` (ทั้งหมด), `--sort pid`, ตารางคอลัมน์ `PID PPID STATE RSS_KB NAME`
- `--json` → JSON array ของ object field `pid`, `ppid`, `state`, `rss_kb`, `name`
- `--workers` ใช้เฉพาะบท 5; ค่า 0/1 = sequential
- exit code: `0` สำเร็จ, `1` runtime error, `2` usage error (flag ผิด)
- `main` บาง; logic อยู่ใน internal เพื่อ test ได้

### บทเรียน 5 บท (`docs/go/`)

| บท | ไฟล์ | เนื้อหา/ผลลัพธ์ |
|---|---|---|
| 1 | `01-toolchain-cli.md` | PATH, `go mod init`, `main`, `flag`, exit code, `--version`; build แรก |
| 2 | `02-reading-proc.md` | struct, method, error, อ่าน `/proc`, sort/limit, `labps list` |
| 3 | `03-packages-tests.md` | แยก `internal`, table test, fixture, golden output, `go vet`; caveat `-race` |
| 4 | `04-json-pipes.md` | struct tag, `encoding/json`, `io.Writer`, `--json`, pipe กับ `python3 -m json.tool`/`head` |
| 5 | `05-concurrency-ship.md` | goroutine, worker pool, channel, `context`, `--workers`, `signal.NotifyContext`, static build |

แต่ละบท: H1 เดียว, 3 แบบฝึกหัด, 2 คำถามเช็คความเข้าใจ, สรุป, caveat proot, output จากเครื่องจริง

### PATH

เพิ่ม block แบบ idempotent ท้าย `~/.bashrc` (ไม่แตะบรรทัดเดิม):

```sh
# Go toolchain (opencode-lab)
case ":$PATH:" in *":$HOME/.local/opt/go/bin:"*) ;; *) export PATH="$HOME/.local/opt/go/bin:$PATH" ;; esac
```

ตรวจด้วย `bash -lc 'command -v go && go version'` และ `bash -lc 'echo $PATH' | grep -c ...` (ต้องได้ 1 หลังรันซ้ำ)

### CI

เพิ่ม path `labs/**` ใน `ci.yml` และ job `go`:

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

ไม่รัน `-race` บน CI ใน track นี้ (เครื่อง dev รันไม่ได้; ถ้าอนาคตต้องการ ให้เพิ่มภายหลังพร้อม runner ที่รองรับ)

### การต่อเข้าเว็บ

- `mkdocs.yml`: track `Go` หลัง Linux, 5 entries
- `docs/index.md` + `README.md`: เพิ่ม Go (5 บท) และคำอธิบาย
- `tools/check_docs.py`: EXPECTED 13 → 18
- deploy ใช้ `pages.yml` เดิม; ไม่เพิ่ม Go ใน Pages build

## Anti-goals

- บทที่ทำให้ `go test -race` ดูผ่านด้วยการซ่อน error
- ตัวอย่างที่ต้องใช้ `jq`, `file`, `htop` หรือ tool ที่เครื่องนี้ไม่มี
- การอ่าน `/proc` ทั้งเครื่องแบบ hardcode — ต้องรับ root path เพื่อ test
- การเพิ่ม dependency แม้แต่ตัวเดียว

## Left to builder

- ถ้อยคำและลำดับย่อยในแต่ละบท, ขนาดตัวอย่างโค้ดที่แสดง
- จะแสดง benchmark บทที่ 5 หรือไม่ (แนะนำไม่ — เครื่อง proot noisy)
- จำนวน fixture ที่ต้องมี (แนะนำ 3 process: ปกติ, state Z, VmRSS หาย)
- ถ้า `--workers` ให้ผลไม่เร็วขึ้นจริงบนเครื่องนี้ ให้บันทึกตามจริงและอธิบายว่าทำไม (Nproc น้อย/งานเล็ก) ไม่แต่งตัวเลข

## Verification

- `cd labs/labps && gofmt -l . | wc -l` → 0; `go vet ./...` → 0; `go test ./...` → ok
- `go build -o labps . && ./labps list --limit 3` → ตารางจริง; `./labps list --json | python3 -m json.tool` → JSON ถูก
- `python3 tools/check_docs.py` → `ok: 18 pages`; `mkdocs build --strict` → exit 0; `site/go/05-concurrency-ship/index.html` มีจริง
- CI run: job `check` (สอง matrix) + `go` success; deploy success
- หน้าใหม่ 5 หน้า + 18/18 URL = 200; `docs/index.md` และ nav แสดง Go 5 บท
