// Command labps reports processes by reading procfs.
package main

import (
	"fmt"
	"io"
	"os"
)

const version = "0.1.0"

const usage = `usage:
  labps --version
  labps list [--limit N] [--sort pid|rss] [--json] [--workers N]
`

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
