package proc

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const fixture = "testdata/proc"

func pids(ps []Process) []int {
	out := make([]int, len(ps))
	for i, p := range ps {
		out[i] = p.PID
	}
	return out
}

func TestParseStatus(t *testing.T) {
	f, err := os.Open(filepath.Join(fixture, "1", "status"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := ParseStatus(1, f)
	if err != nil {
		t.Fatal(err)
	}
	want := Process{PID: 1, PPID: 0, State: 'S', RSSKB: 1024, Name: "systemd"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseStatusNoVmRSS(t *testing.T) {
	got, err := ParseStatus(42, strings.NewReader("Name:\tzombie\nState:\tZ (zombie)\nPPid:\t1\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.RSSKB != 0 || got.State != 'Z' || got.PPID != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestParseStatusMissingName(t *testing.T) {
	if _, err := ParseStatus(5, strings.NewReader("State:\tS (sleeping)\n")); err == nil {
		t.Fatal("want error for status without Name")
	}
}

func TestListFixture(t *testing.T) {
	ps, err := Reader{Root: fixture}.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pids(ps), []int{1, 42, 300}; !reflect.DeepEqual(got, want) {
		t.Errorf("pids = %v, want %v", got, want)
	}
	if ps[2].Name != "Web Content" {
		t.Errorf("name with space = %q", ps[2].Name)
	}
}

// pid 77 is a directory without status: the process exited between
// ReadDir and ReadFile. It must be skipped, not fail the whole list.
func TestListSkipsVanished(t *testing.T) {
	ps, err := Reader{Root: fixture}.List(context.Background())
	if err != nil {
		t.Fatalf("vanished pid should be skipped, got %v", err)
	}
	for _, p := range ps {
		if p.PID == 77 {
			t.Error("pid 77 should be skipped")
		}
	}
}

func TestListMissingRoot(t *testing.T) {
	if _, err := (Reader{Root: "testdata/nope"}).List(context.Background()); err == nil {
		t.Fatal("want error for missing root")
	}
}

func TestSortRSS(t *testing.T) {
	ps, _ := Reader{Root: fixture}.List(context.Background())
	if err := Sort(ps, "rss"); err != nil {
		t.Fatal(err)
	}
	if got, want := pids(ps), []int{300, 1, 42}; !reflect.DeepEqual(got, want) {
		t.Errorf("pids = %v, want %v", got, want)
	}
}

func TestSortUnknown(t *testing.T) {
	if err := Sort(nil, "cpu"); err == nil {
		t.Fatal("want error for unknown sort key")
	}
}
