package questspec

import (
	"strings"
	"testing"

	questspecdata "d2lod-bot/data/questspec"
)

// TestShippedCoverage is the CI gate of roadmap row R13: every quest on the board has a chain or a pending row that
// says who owns it and why it has none. Acts 1 and 2 are pinned to their chains so a deleted chain cannot hide behind
// a new pending row.
func TestShippedCoverage(t *testing.T) {
	chains, _, _, board := shipped(t)
	raw, err := questspecdata.FS.ReadFile("pending.json")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := LoadPending(raw)
	if err != nil {
		t.Fatal(err)
	}
	acts, bad := Coverage(board, chains, pending)
	for _, f := range bad {
		t.Error(f)
	}
	if len(acts) != 5 {
		t.Fatalf("acts: %+v", acts)
	}
	total := 0
	for _, a := range acts {
		total += len(a.Chained) + len(a.Pending)
		t.Logf("act %d: %d chained, %d pending %v", a.Act, len(a.Chained), len(a.Pending), a.Pending)
	}
	if total != len(board.Quests()) {
		t.Errorf("coverage names %d quests, the board has %d", total, len(board.Quests()))
	}
	if len(acts[0].Pending) != 0 {
		t.Errorf("act 1 was fully chained; pending now: %v", acts[0].Pending)
	}
	if got := strings.Join(acts[1].Pending, ","); got != "arcane_sanctuary" {
		t.Errorf("act 2 pending %q: only arcane_sanctuary had no chain", got)
	}
}

func TestCoverageFindings(t *testing.T) {
	chains, _, _, board := shipped(t)
	all := func() *PendingFile {
		p := &PendingFile{Schema: PendingSchema}
		chained := map[string]bool{}
		for _, c := range chains {
			chained[c.Quest] = true
		}
		for _, q := range board.Quests() {
			if !chained[q.ID] {
				p.Pending = append(p.Pending, Pending{Quest: q.ID, Owner: "o", Why: "w"})
			}
		}
		return p
	}
	if _, bad := Coverage(board, chains, all()); len(bad) != 0 {
		t.Fatalf("a complete list has findings: %v", bad)
	}
	for _, c := range []struct {
		name string
		edit func(p *PendingFile) []*Chain
		want string
	}{
		{"a quest with neither", func(p *PendingFile) []*Chain { p.Pending = p.Pending[1:]; return chains }, "no chain in data/questspec and no pending row"},
		{"a chain was deleted", func(p *PendingFile) []*Chain { return chains[1:] }, "quest " + chains[0].Quest + ": no chain"},
		{"stale row", func(p *PendingFile) []*Chain {
			p.Pending = append(p.Pending, Pending{Quest: chains[0].Quest, Owner: "o", Why: "w"})
			return chains
		}, "has a chain now"},
		{"unknown quest", func(p *PendingFile) []*Chain {
			p.Pending = append(p.Pending, Pending{Quest: "no_such_quest", Owner: "o", Why: "w"})
			return chains
		}, `unknown quest "no_such_quest"`},
		{"duplicate", func(p *PendingFile) []*Chain { p.Pending = append(p.Pending, p.Pending[0]); return chains }, "listed twice"},
		{"no reason", func(p *PendingFile) []*Chain { p.Pending[0].Why = ""; return chains }, "owner and why are required"},
		{"no owner", func(p *PendingFile) []*Chain { p.Pending[0].Owner = ""; return chains }, "owner and why are required"},
		{"chain off the board", func(p *PendingFile) []*Chain {
			return append(append([]*Chain(nil), chains...), &Chain{ID: "x", Quest: "gone"})
		}, `chain x: questboard: unknown quest "gone"`},
	} {
		p := all()
		_, bad := Coverage(board, c.edit(p), p)
		if len(bad) != 1 || !strings.Contains(bad[0], c.want) {
			t.Errorf("%s: findings %v, want one with %q", c.name, bad, c.want)
		}
	}
}

func TestLoadPendingStrict(t *testing.T) {
	for _, in := range []string{`{"schema":"questpending/0","pending":[]}`, `{"schema":"questpending/1","pending":[],"x":1}`, `{"schema":"questpending/1","pending":[]} {}`} {
		if _, err := LoadPending([]byte(in)); err == nil {
			t.Errorf("%s: accepted", in)
		}
	}
}
