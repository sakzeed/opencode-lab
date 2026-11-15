// Command labps reports processes by reading procfs.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/sakzeed/opencode-lab/labs/labps/internal/proc"
	"github.com/sakzeed/opencode-lab/labs/labps/internal/render"
)

const version = "0.1.0"

const usage = `usage:
  labps --version
  labps list [--limit N] [--sort pid|rss] [--json] [--workers N]
`

// errUsage marks errors that should exit 2 instead of 1.
var errUsage = errors.New("usage error")

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
	var err error
	switch args[0] {
	case "--version", "-version":
		fmt.Fprintf(stdout, "labps %s\n", version)
		return 0
	case "list":
		// Ctrl-C or SIGTERM cancels ctx so workers stop instead of
		// the process being killed mid-write.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err = list(ctx, args[1:], stdout, stderr)
		stop()
	default:
		err = fmt.Errorf("%w: unknown command %q", errUsage, args[0])
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errUsage):
		fmt.Fprintf(stderr, "labps: %v\n%s", err, usage)
		return 2
	default:
		fmt.Fprintf(stderr, "labps: %v\n", err)
		return 1
	}
}

func list(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	limit := fs.Int("limit", 0, "show at most N processes (0 = all)")
	sortKey := fs.String("sort", "pid", "sort by pid or rss")
	asJSON := fs.Bool("json", false, "print JSON instead of a table")
	workers := fs.Int("workers", 0, "read status files with N goroutines (0/1 = sequential)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return fmt.Errorf("%w: %v", errUsage, err)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("%w: unexpected argument %q", errUsage, fs.Arg(0))
	}
	if *limit < 0 {
		return fmt.Errorf("%w: --limit must be >= 0, got %d", errUsage, *limit)
	}
	if *workers < 0 {
		return fmt.Errorf("%w: --workers must be >= 0, got %d", errUsage, *workers)
	}
	if *sortKey != "pid" && *sortKey != "rss" {
		return fmt.Errorf("%w: --sort must be pid or rss, got %q", errUsage, *sortKey)
	}

	rd := proc.Reader{Root: procRoot()}
	ps, err := rd.ListConcurrent(ctx, *workers)
	if err != nil {
		return err
	}
	if err := proc.Sort(ps, *sortKey); err != nil {
		return err
	}
	if *limit > 0 && len(ps) > *limit {
		ps = ps[:*limit]
	}
	if *asJSON {
		return render.JSON(stdout, ps)
	}
	return render.Table(stdout, ps)
}

// procRoot is /proc unless LABPS_PROC_ROOT points somewhere else (tests).
func procRoot() string {
	if root := os.Getenv("LABPS_PROC_ROOT"); root != "" {
		return root
	}
	return "/proc"
}
