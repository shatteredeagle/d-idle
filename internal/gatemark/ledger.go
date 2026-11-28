package gatemark

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Line is one flake ledger line (JSONL, append-only). A run line (Test and Package empty) says how a gate run on Commit
// ended; a failure line names one failing test of that run.
type Line struct {
	At      string `json:"at"`
	Commit  string `json:"commit"`
	Job     string `json:"job,omitempty"`
	Green   bool   `json:"green,omitempty"`
	Package string `json:"pkg,omitempty"`
	Test    string `json:"test,omitempty"`
}

// Lines are the ledger lines of one gate run: the run line, then one line per failure.
func (s Summary) Lines(at, commit, job string) []Line {
	out := []Line{{At: at, Commit: commit, Job: job, Green: s.Green()}}
	for _, f := range s.Failures {
		out = append(out, Line{At: at, Commit: commit, Job: job, Package: f.Package, Test: f.Test})
	}
	return out
}

// Append writes lines as JSONL.
func Append(w io.Writer, lines []Line) error {
	enc := json.NewEncoder(w)
	for _, l := range lines {
		if err := enc.Encode(l); err != nil {
			return err
		}
	}
	return nil
}

// ReadLedger parses a ledger; a malformed line is an error that names its number (a ledger is never guessed at).
func ReadLedger(r io.Reader) ([]Line, error) {
	var out []Line
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for n := 1; sc.Scan(); n++ {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var l Line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil || l.Commit == "" {
			return nil, fmt.Errorf("gatemark: ledger line %d: bad line", n)
		}
		out = append(out, l)
	}
	return out, sc.Err()
}

// Flake is a test with failures that did not reproduce: the same commit also has a green gate run.
type Flake struct {
	Package string
	Test    string
	Commits []string // the commits of the non-reproducing failures, sorted
}

// Flakes returns the tests that failed on at least QuarantineAfter commits which also passed the gate unchanged. A
// failure on a commit that never went green is a real failure until proven otherwise and is not counted. A package
// failure without a test name is never a quarantine candidate: there is nothing to skip.
func Flakes(lines []Line) []Flake {
	green := map[string]bool{}
	for _, l := range lines {
		if l.Test == "" && l.Package == "" && l.Green {
			green[l.Commit] = true
		}
	}
	type key struct{ pkg, test string }
	seen := map[key]map[string]bool{}
	for _, l := range lines {
		if l.Test == "" || !green[l.Commit] {
			continue
		}
		k := key{l.Package, l.Test}
		if seen[k] == nil {
			seen[k] = map[string]bool{}
		}
		seen[k][l.Commit] = true
	}
	var out []Flake
	for k, cs := range seen {
		if len(cs) < QuarantineAfter {
			continue
		}
		f := Flake{Package: k.pkg, Test: k.test}
		for c := range cs {
			f.Commits = append(f.Commits, c)
		}
		sort.Strings(f.Commits)
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Package != out[j].Package {
			return out[i].Package < out[j].Package
		}
		return out[i].Test < out[j].Test
	})
	return out
}

// QuarantineRow is the tools/ci/flaky.txt line of a flake (TestName, package, owner, note; tab separated). The top-level
// test is named because `go test -skip` takes it; the owner is left for the coordinator.
func (f Flake) QuarantineRow() string {
	top, _, _ := strings.Cut(f.Test, "/")
	return fmt.Sprintf("%s\t%s\t%s\t%d non-reproducing gate failures (%s)", top, f.Package, "unassigned", len(f.Commits), strings.Join(f.Commits, ", "))
}
