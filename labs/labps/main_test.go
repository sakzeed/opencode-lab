package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"--version"}, &out, &errb); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr %q)", code, errb.String())
	}
	if got, want := out.String(), "labps 0.1.0\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestRunNoArgs(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run(nil, &out, &errb); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "usage:") {
		t.Errorf("stderr = %q, want usage", errb.String())
	}
}

func runFixture(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	t.Setenv("LABPS_PROC_ROOT", "internal/proc/testdata/proc")
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRunListLimit(t *testing.T) {
	code, out, errs := runFixture(t, "list", "--limit", "2")
	if code != 0 {
		t.Fatalf("exit = %d (stderr %q)", code, errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3 (header + 2):\n%s", len(lines), out)
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[0]), "PID") {
		t.Errorf("header = %q", lines[0])
	}
}

func TestRunListSortRSS(t *testing.T) {
	code, out, _ := runFixture(t, "list", "--sort", "rss", "--limit", "1")
	if code != 0 || !strings.Contains(out, "Web Content") {
		t.Fatalf("exit = %d, out:\n%s", code, out)
	}
}

func TestRunListJSON(t *testing.T) {
	code, out, _ := runFixture(t, "list", "--json")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 3 {
		t.Fatalf("rows = %d, err = %v\n%s", len(rows), err, out)
	}
}

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

func TestRunRuntimeError(t *testing.T) {
	t.Setenv("LABPS_PROC_ROOT", "does/not/exist")
	var out, errb bytes.Buffer
	if code := run([]string{"list"}, &out, &errb); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.HasPrefix(errb.String(), "labps: ") {
		t.Errorf("stderr = %q", errb.String())
	}
}

func TestRunWorkers(t *testing.T) {
	if code, _, errs := runFixture(t, "list", "--workers", "4"); code != 0 {
		t.Fatalf("--workers 4: exit = %d (%s)", code, errs)
	}
	if code, _, _ := runFixture(t, "list", "--workers", "-1"); code != 2 {
		t.Fatalf("--workers -1: exit = %d, want 2", code)
	}
}
