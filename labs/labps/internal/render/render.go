// Package render writes processes as a text table or JSON.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/sakzeed/opencode-lab/labs/labps/internal/proc"
)

// Table writes aligned columns PID PPID STATE RSS_KB NAME.
// NAME is last so names with spaces never shift other columns.
func Table(w io.Writer, ps []proc.Process) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', tabwriter.AlignRight)
	fmt.Fprintln(tw, "PID\tPPID\tSTATE\tRSS_KB\t NAME")
	for _, p := range ps {
		fmt.Fprintf(tw, "%d\t%d\t%s\t%d\t %s\n", p.PID, p.PPID, state(p.State), p.RSSKB, p.Name)
	}
	return tw.Flush()
}

type row struct {
	PID   int    `json:"pid"`
	PPID  int    `json:"ppid"`
	State string `json:"state"`
	RSSKB int64  `json:"rss_kb"`
	Name  string `json:"name"`
}

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

// state renders the one-letter state; "?" when status had no State line.
func state(b byte) string {
	if b == 0 {
		return "?"
	}
	return string(b)
}
