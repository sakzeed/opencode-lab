// Package proc reads process information from a procfs directory.
package proc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

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

// ParseStatus parses the content of /proc/<pid>/status.
// A missing VmRSS (kernel threads, zombies) leaves RSSKB at 0.
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

// pids returns the numeric entries of Root in ascending order.
func (rd Reader) pids() ([]int, error) {
	entries, err := os.ReadDir(rd.Root)
	if err != nil {
		return nil, err
	}
	var out []int
	for _, e := range entries {
		if n, err := strconv.Atoi(e.Name()); err == nil && e.IsDir() {
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out, nil
}

// read loads one process. ok is false when the process vanished
// (exited between listing Root and opening its status file).
func (rd Reader) read(pid int) (p Process, ok bool, err error) {
	f, err := os.Open(filepath.Join(rd.Root, strconv.Itoa(pid), "status"))
	if errors.Is(err, fs.ErrNotExist) {
		return Process{}, false, nil
	}
	if err != nil {
		return Process{}, false, fmt.Errorf("pid %d: %w", pid, err)
	}
	defer f.Close()
	p, err = ParseStatus(pid, f)
	if err != nil {
		return Process{}, false, fmt.Errorf("pid %d: %w", pid, err)
	}
	return p, true, nil
}

// List reads every process under Root, sorted by PID.
func (rd Reader) List(ctx context.Context) ([]Process, error) {
	pids, err := rd.pids()
	if err != nil {
		return nil, err
	}
	out := make([]Process, 0, len(pids))
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p, ok, err := rd.read(pid)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, p)
		}
	}
	return out, nil
}

// Sort orders ps in place by key: "pid" ascending or "rss" descending.
func Sort(ps []Process, key string) error {
	switch key {
	case "pid":
		sort.SliceStable(ps, func(i, j int) bool { return ps[i].PID < ps[j].PID })
	case "rss":
		sort.SliceStable(ps, func(i, j int) bool { return ps[i].RSSKB > ps[j].RSSKB })
	default:
		return fmt.Errorf("unknown sort key %q (want pid or rss)", key)
	}
	return nil
}
