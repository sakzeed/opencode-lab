package render

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/sakzeed/opencode-lab/labs/labps/internal/proc"
)

var update = flag.Bool("update", false, "rewrite golden files")

var sample = []proc.Process{
	{PID: 1, PPID: 0, State: 'S', RSSKB: 1024, Name: "systemd"},
	{PID: 42, PPID: 1, State: 'Z', RSSKB: 0, Name: "zombie"},
	{PID: 300, PPID: 1, State: 'R', RSSKB: 204800, Name: "Web Content"},
}

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
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != string(want) {
		t.Errorf("table mismatch\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestJSONFields(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sample); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	for _, key := range []string{"pid", "ppid", "state", "rss_kb", "name"} {
		if _, ok := rows[0][key]; !ok {
			t.Errorf("missing key %q in %v", key, rows[0])
		}
	}
	if rows[1]["state"] != "Z" {
		t.Errorf("state = %v, want \"Z\"", rows[1]["state"])
	}
	if rows[2]["name"] != "Web Content" {
		t.Errorf("name = %v", rows[2]["name"])
	}
}

func TestJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "[]\n" {
		t.Errorf("got %q, want \"[]\\n\"", got)
	}
}

// A status file without State leaves State at 0; never print a NUL byte.
func TestUnknownState(t *testing.T) {
	ps := []proc.Process{{PID: 9, Name: "odd"}}
	var tbl, js bytes.Buffer
	if err := Table(&tbl, ps); err != nil {
		t.Fatal(err)
	}
	if err := JSON(&js, ps); err != nil {
		t.Fatal(err)
	}
	if bytes.IndexByte(tbl.Bytes(), 0) >= 0 || !bytes.Contains(tbl.Bytes(), []byte(" ? ")) {
		t.Errorf("table = %q, want ? for unknown state", tbl.String())
	}
	if !bytes.Contains(js.Bytes(), []byte(`"state": "?"`)) {
		t.Errorf("json = %s, want \"state\": \"?\"", js.String())
	}
}
