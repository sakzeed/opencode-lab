package main

import (
	"bytes"
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
