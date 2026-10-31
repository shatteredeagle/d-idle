package goals

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var stubChain = &Chain{
	Next: func(cur string, baal bool) string {
		switch {
		case !baal:
			return ""
		case cur == "normal":
			return "nightmare"
		case cur == "nightmare":
			return "hell"
		}
		return ""
	},
	Ready: func(next string, lvl int) (bool, string) {
		if f := map[string]int{"nightmare": 40, "hell": 70}[next]; lvl < f {
			return false, "below floor"
		}
		return true, ""
	},
	Floor: func(next string) int { return map[string]int{"nightmare": 40, "hell": 70}[next] },
}

func chainMgr(t *testing.T, path string, level int) (*Manager, *fakeRunner, *clock) {
	t.Helper()
	f := newFake()
	f.level = level
	c := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	m, err := New(Options{Runner: f, Path: path, Clock: c.now, Chain: stubChain})
	if err != nil {
		t.Fatal(err)
	}
	return m, f, c
}

func finish(m *Manager, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finishLocked(m.find(id), StateDone, "", m.opt.Clock())
}

func TestChainQueuesNextCampaignWhenReady(t *testing.T) {
	m, _, _ := chainMgr(t, "", 45)
	g := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignNormalAll, Chain: true}})
	finish(m, g.ID)
	gs := m.Goals()
	if len(gs) != 2 || gs[0].Params.Template != CampaignNightmareAll || !gs[0].Params.Chain || gs[0].Params.Difficulty != "nightmare" {
		t.Fatalf("%+v", gs)
	}
	if got, _ := m.Goal(g.ID); len(got.NextIDs) != 1 || !strings.Contains(got.Note, "nightmare_all") {
		t.Fatalf("%+v", got)
	}
}

func TestChainQueuesLevelGoalFirstWhenBelowFloor(t *testing.T) {
	m, _, _ := chainMgr(t, "", 30)
	g := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignNightmareAll, Chain: true}})
	finish(m, g.ID)
	var live []Goal
	for _, x := range m.Goals() {
		if !x.State.Finished() {
			live = append(live, x)
		}
	}
	if len(live) != 2 || live[0].Type != ReachLevel || live[0].Params.Level != 70 || live[1].Params.Template != CampaignHellAll {
		t.Fatalf("%+v", live)
	}
	if got, _ := m.Goal(g.ID); len(got.NextIDs) != 2 || !strings.Contains(got.Note, "below floor") {
		t.Fatalf("%+v", got)
	}
}

func TestChainOffAndEndOfChain(t *testing.T) {
	m, _, _ := chainMgr(t, "", 99)
	a := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignNormalAll}})
	b := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignHellAll, Chain: true}})
	c := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignNormalAct2, Chain: true}})
	finish(m, a.ID)
	finish(m, b.ID)
	finish(m, c.ID)
	if n := len(m.Goals()); n != 3 {
		t.Fatalf("%d goals", n)
	}
}

func TestChainIdempotentAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	m, _, _ := chainMgr(t, path, 45)
	g := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignNormalAll, Chain: true}})
	finish(m, g.ID)
	m.Close()
	m2, _, _ := chainMgr(t, path, 45)
	if n := len(m2.Goals()); n != 2 {
		t.Fatalf("restart duplicated the chain: %d goals", n)
	}
	// even after the user removes the follow-up it is not recreated
	got, _ := m2.Goal(g.ID)
	if err := m2.Remove(got.NextIDs[0]); err != nil {
		t.Fatal(err)
	}
	m2.Close()
	m3, _, _ := chainMgr(t, path, 45)
	if n := len(m3.Goals()); n != 1 {
		t.Fatalf("%d goals", n)
	}
}

// A campaign finished but the follow-up never saved (crash): the next start queues it.
func TestChainResumedAfterCrash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	m, _, _ := chainMgr(t, path, 45)
	m.opt.Chain = nil
	g := mustAdd(t, m, Goal{Type: Campaign, Params: Params{Template: CampaignNormalAll, Chain: true}})
	finish(m, g.ID)
	m.Close()
	m2, _, _ := chainMgr(t, path, 45)
	if n := len(m2.Goals()); n != 2 {
		t.Fatalf("%d goals", n)
	}
}
