package fidprobe

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Text renders the stable diff table: one line per non-EQUAL cell (EQUAL cells are counted, not listed,
// unless all is set), sorted by key, then the counts and the verdict line.
func (r *Report) Text(all bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "scenario %s: A=%s B=%s\n", r.Scenario, r.SourceA, r.SourceB)
	fmt.Fprintf(&b, "%-10s %-28s %-18s %-18s %s\n", "result", "field", "A", "B", "detail")
	for _, c := range r.Cells {
		if c.Class == Equal && !all {
			continue
		}
		d := c.Detail
		if c.Row != "" {
			d = strings.TrimSpace(d + " [" + c.Row + "] " + c.Reason)
		} else if c.Reason != "" {
			d = strings.TrimSpace(d + " " + c.Reason)
		}
		fmt.Fprintf(&b, "%-10s %-28s %-18s %-18s %s\n", c.Class, c.Key, c.A, c.B, d)
	}
	for _, g := range r.EmptyGroups {
		fmt.Fprintf(&b, "%-10s %-28s %-18s %-18s %s\n", Unreadable, "group:"+g, "-", "-", "no field in this group on either side")
	}
	fmt.Fprintf(&b, "EQUAL %d KNOWN %d MISMATCH %d UNREADABLE %d\n",
		r.Counts[Equal], r.Counts[KnownDev], r.Counts[Mismatch], r.Counts[Unreadable])
	verdict := "PASS"
	if !r.Pass() {
		verdict = "FAIL"
	}
	fmt.Fprintf(&b, "verdict %s\nwrites after start: 0\nlab verbs: off\n", verdict)
	return b.String()
}

// JSON renders the report as indented JSON.
func (r *Report) JSON() ([]byte, error) { return json.MarshalIndent(r, "", " ") }
