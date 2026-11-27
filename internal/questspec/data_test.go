package questspec

import (
	"io/fs"
	"path"
	"sort"
	"testing"

	questboarddata "d2lod-bot/data/questboard"
	questspecdata "d2lod-bot/data/questspec"
	"d2lod-bot/internal/questboard"
)

func shipped(t *testing.T) (chains []*Chain, caps map[string]*Caps, allow *AllowFile, board *questboard.Board) {
	t.Helper()
	caps = map[string]*Caps{}
	names, err := fs.Glob(questspecdata.FS, "caps/*.json")
	if err != nil || len(names) == 0 {
		t.Fatalf("no capability sets: %v", err)
	}
	for _, n := range names {
		b, _ := questspecdata.FS.ReadFile(n)
		c, err := LoadCaps(b)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if want := path.Base(n[:len(n)-5]); c.Backend != want {
			t.Fatalf("%s: backend %q does not match the file name", n, c.Backend)
		}
		caps[c.Backend] = c
	}
	acts, _ := fs.Glob(questspecdata.FS, "act*.json")
	if len(acts) == 0 {
		t.Fatal("no act files")
	}
	for _, n := range acts {
		b, _ := questspecdata.FS.ReadFile(n)
		cs, err := LoadFile(b)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		chains = append(chains, cs...)
	}
	ab, _ := questspecdata.FS.ReadFile("allow.json")
	if allow, err = LoadAllow(ab); err != nil {
		t.Fatal(err)
	}
	board, err = questboard.Parse(questboarddata.Families, questboarddata.Act1, questboarddata.Act2, questboarddata.Act3, questboarddata.Act4, questboarddata.Act5)
	if err != nil {
		t.Fatal(err)
	}
	return
}

// TestShippedChains is the CI gate: every chain of data/questspec is checked against every capability set and against
// data/questboard. A finding that is not in allow.json fails, and so does an allow entry whose finding is gone.
func TestShippedChains(t *testing.T) {
	chains, caps, allow, board := shipped(t)
	used := map[int]bool{}
	var backends []string
	for b := range caps {
		backends = append(backends, b)
	}
	sort.Strings(backends)
	var all []*Caps
	for _, b := range backends {
		all = append(all, caps[b])
	}
	for _, f := range CheckOrder(board) {
		t.Errorf("board: %s", f)
	}
	for _, c := range chains {
		for _, f := range CheckLatched(c, all) {
			t.Errorf("chain %s: %s", c.ID, f)
		}
		for _, f := range CheckBoard(c, board) {
			t.Errorf("chain %s: %s", c.ID, f)
		}
		for _, b := range backends {
			for _, f := range allow.Filter(c.ID, b, Check(c, caps[b]), used) {
				t.Errorf("chain %s on %s: %s", c.ID, b, f)
			}
		}
	}
	for i, e := range allow.Allowed {
		if !used[i] {
			t.Errorf("allow entry %d (%s/%s/%s rule %d, owner %s) matches no finding: remove it", i, e.Chain, e.Caps, e.Step, e.Rule, e.Owner)
		}
	}
}
