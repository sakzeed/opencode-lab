# Go บทที่ 4: JSON, io.Writer และ pipe

ตารางอ่านง่ายสำหรับคน แต่ถ้าจะส่งให้โปรแกรมอื่นอ่าน JSON ดีกว่า บทนี้เพิ่ม `labps list --json` ด้วย `encoding/json` และ struct tag แล้วต่อ pipe ไปให้ `python3` ซึ่งมีอยู่ในเครื่องแล้ว (เครื่องนี้ไม่มี `jq`) และดูว่า `io.Writer` ทำให้ code ส่วน render ไม่ต้องรู้ว่าปลายทางคือ terminal, ไฟล์ หรือ test

---

## 4.1 `io.Writer`: interface ตัวเดียวที่ทุกอย่างใช้

```go
type Writer interface {
	Write(p []byte) (n int, err error)
}
```

type ไหนมี method `Write` แบบนี้ก็ **เป็น** `io.Writer` อัตโนมัติ ไม่ต้องประกาศว่า implement (Go ใช้ implicit interface):

| ปลายทาง | type |
|---|---|
| terminal / pipe | `os.Stdout`, `os.Stderr` (`*os.File`) |
| ไฟล์ | `*os.File` จาก `os.Create` |
| test | `*bytes.Buffer` |
| ตัวจัดคอลัมน์ | `*tabwriter.Writer` (ห่อ writer อีกตัว) |

ฟังก์ชันใน `render` จึงรับ `io.Writer` ไม่ใช่ `*os.File`:

```go
func Table(w io.Writer, ps []proc.Process) error
func JSON(w io.Writer, ps []proc.Process) error
```

`run` ส่ง `os.Stdout` เข้าไป ส่วน test ส่ง `bytes.Buffer` โค้ด render ชุดเดียวกันใช้ได้ทั้งสองแบบ

## 4.2 struct tag และ `encoding/json`

`encoding/json` ใช้ชื่อ field ของ struct เป็น key ถ้าไม่บอกอะไร (`PID`, `RSSKB`) ซึ่งไม่ตรงกับแบบแผนของ JSON ทั่วไป ให้กำหนดชื่อเองด้วย **struct tag**:

```go
type row struct {
	PID   int    `json:"pid"`
	PPID  int    `json:"ppid"`
	State string `json:"state"`
	RSSKB int64  `json:"rss_kb"`
	Name  string `json:"name"`
}
```

ทำไมต้องมี `row` แยกจาก `proc.Process`:

- `Process.State` เป็น `byte` ถ้า marshal ตรง ๆ จะได้ตัวเลข (`83`) ไม่ใช่ `"S"`
- รูปแบบ JSON เป็น **สัญญากับผู้ใช้** (field ตาม design คือ `pid`, `ppid`, `state`, `rss_kb`, `name`) แยก type ไว้จึงแก้ `Process` ภายในได้โดยไม่ทำให้ JSON เปลี่ยนตาม
- field ต้องขึ้นต้นด้วยตัวใหญ่ (export) `encoding/json` จึงจะเห็น

```go
// JSON writes ps as an indented JSON array; an empty list is "[]".
func JSON(w io.Writer, ps []proc.Process) error {
	rows := make([]row, len(ps))
	for i, p := range ps {
		rows[i] = row{p.PID, p.PPID, state(p.State), p.RSSKB, p.Name}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}
```

`make([]row, len(ps))` สร้าง slice ที่ไม่ใช่ `nil` เสมอ list ว่างจึงออกมาเป็น `[]` ไม่ใช่ `null` (`var rows []row` ที่เป็น nil จะ encode เป็น `null` ซึ่งทำให้ผู้อ่านที่คาดหวัง array พัง) test `TestJSONEmpty` คุมเรื่องนี้ไว้

`json.NewEncoder(w)` เขียนลง writer ตรง ๆ ไม่ต้องสร้าง `[]byte` ทั้งก้อนก่อนแบบ `json.Marshal`

## 4.3 `labps list --json`

```sh
$ ./labps list --json --limit 2
[
  {
    "pid": 8309,
    "ppid": 30208,
    "state": "t",
    "rss_kb": 171476,
    "name": "opencode"
  },
  {
    "pid": 8857,
    "ppid": 30113,
    "state": "S",
    "rss_kb": 2392,
    "name": "bash"
  }
]
```

flag อื่นใช้ร่วมกับ `--json` ได้ทุกตัว เพราะการ sort และ limit เกิดขึ้นก่อน render

```go
if *asJSON {
	return render.JSON(stdout, ps)
}
return render.Table(stdout, ps)
```

## 4.4 ต่อ pipe

ตรวจว่าเป็น JSON ที่ถูกต้อง และย่อให้เหลือบรรทัดเดียว:

```sh
$ ./labps list --json | python3 -m json.tool --compact | head -c 120; echo
[{"pid":8309,"ppid":30208,"state":"t","rss_kb":171476,"name":"opencode"},{"pid":8857,"ppid":30113,"state":"S","rss_kb":2
```

กรองด้วย Python เพียงไม่กี่บรรทัด (ใช้แทน `jq` ได้):

```sh
$ ./labps list --json | python3 -c 'import json,sys
rows = json.load(sys.stdin)
big = [r for r in rows if r["rss_kb"] > 100000]
print(len(rows), "processes,", len(big), "over 100 MB:", sorted({r["name"] for r in big}))'
19 processes, 4 over 100 MB: ['2.1.325', 'opencode']
```

ตัดแค่หัว:

```sh
$ ./labps list --json | head -3; echo "exit=${PIPESTATUS[0]}"
[
  {
    "pid": 8309,
exit=0
```

`labps` ยังได้ exit `0` แม้ `head` จะปิด pipe ไปก่อน เพราะ output ทั้งหมด (ไม่กี่ KB) เขียนลง pipe buffer (64 KiB บน Linux) เสร็จตั้งแต่ก่อน `head` จะออก ถ้า output ใหญ่กว่า buffer โปรแกรมจะได้ `SIGPIPE` แล้วจบด้วย exit `141` (128+13) ตามที่เห็นใน Linux บทที่ 4

error ไปที่ stderr เสมอ stdout จึงไม่มี JSON ครึ่ง ๆ กลาง ๆ ปนข้อความ error:

```sh
$ LABPS_PROC_ROOT=/nope ./labps list --json 2>/dev/null | wc -c
0
```

!!! warning "caveat: ชื่อ process บน proot"
    ชื่อบางตัวดูแปลก เช่น `2.1.325` เพราะ `Name:` ใน `status` คือชื่อไฟล์ executable แบบย่อ (ไม่เกิน 15 ตัวอักษร) ไม่ใช่ command line เต็ม ถ้าต้องการ argument ให้อ่าน `/proc/<pid>/cmdline` (คั่นด้วย NUL) ซึ่งอยู่นอกขอบเขตของ track นี้

## 4.5 test ของ JSON

assert ทั้ง string เปราะเพราะลำดับ key และการเว้นวรรคเปลี่ยนได้ test จึง **decode กลับ** แล้วตรวจโครงสร้าง:

```go
func TestJSONFields(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sample); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pid", "ppid", "state", "rss_kb", "name"} {
		if _, ok := rows[0][key]; !ok {
			t.Errorf("missing key %q in %v", key, rows[0])
		}
	}
	if rows[1]["state"] != "Z" {
		t.Errorf("state = %v, want \"Z\"", rows[1]["state"])
	}
}
```

`map[string]any` decode JSON ได้ทุกแบบ ตัวเลขจะกลายเป็น `float64` จึงเหมาะกับการตรวจว่ามี key ครบ ส่วนการตรวจค่าตัวเลขแบบละเอียด ให้ decode เข้า struct แทน

## 4.6 แบบฝึกหัด 3 ข้อ + เช็คความเข้าใจ 2 ข้อ

### แบบฝึกหัด

1. ใช้ `labps list --json` + Python หาว่า PPID ไหนมีลูกมากที่สุด (นับด้วย `collections.Counter`)
2. ลองเปลี่ยน `rows := make([]row, len(ps))` เป็น `var rows []row` แล้ว `append` ทีละตัว รัน `TestJSONEmpty` ให้เห็นว่า fail เพราะได้ `null` แล้ว revert
3. เขียน `./labps list --json > /tmp/ps.json` แล้วใช้ `python3 -m json.tool /tmp/ps.json > /dev/null && echo valid` ตรวจไฟล์ จากนั้นทำให้ไฟล์เสียด้วย `truncate -s -5 /tmp/ps.json` แล้วดูว่า exit code เปลี่ยนเป็นเท่าไร

### เช็คความเข้าใจ

1. ทำไม `render.JSON` ถึงต้องแปลง `Process` เป็น `row` ก่อน แทนที่จะใส่ tag ลงใน `Process` ตรง ๆ (คิดถึง `State byte` และการแยกหน้าที่)
2. ถ้า `labps` พิมพ์ข้อความ error ลง stdout แทน stderr คำสั่ง `labps list --json | python3 -m json.tool` จะเห็นอะไร

## สรุปบทที่ 4

- `io.Writer` ทำให้ render ไม่ต้องรู้ปลายทาง (terminal, ไฟล์, buffer ใน test)
- struct tag `json:"..."` กำหนดชื่อ key และ type `row` แยกรูปแบบ JSON ออกจาก struct ภายใน
- slice ที่ได้จาก `make` encode เป็น `[]` ส่วน slice ที่เป็น nil encode เป็น `null`
- เครื่องนี้ไม่มี `jq` จึงใช้ `python3 -m json.tool` และ `python3 -c` ต่อ pipe แทน และ error ต้องไป stderr เพื่อไม่ให้ JSON เสีย
