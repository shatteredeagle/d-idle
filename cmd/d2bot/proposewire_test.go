package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/watch"
)

// The zone proposer on the real wiring (TASK-131): startLadder + feedHub with a facts reader, two classes. Fixtures only
// (a fake runner, a simulated clock); nothing here ran against the live game.
func TestProposalsOnTheRealWiring(t *testing.T) {
	for _, c := range []struct {
		class string
		level int
	}{{"Sorceress", 7}, {"Paladin", 6}} {
		t.Run(c.class, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			m := goalManager(t)
			state := t.TempDir()
			// the player looked 10 hours ago: away
			old, _ := json.Marshal(map[string]time.Time{"lastSeen": time.Now().Add(-10 * time.Hour)})
			if err := os.WriteFile(filepath.Join(state, "visit.json"), old, 0o600); err != nil {
				t.Fatal(err)
			}
			clk := &watch.SimClock{}
			facts := watch.ProposalFacts{Kit: "OK", Potions: 4, Ok: true}
			lo := ladderOptions{on: true, watch: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0",
				currencyDir: filepath.Join(t.TempDir(), "currency"), proposalFacts: func() watch.ProposalFacts { return facts },
				awayMin: 20, repeatCooldownMin: 30, quietAfter: 3}
			status := func() goals.Status {
				return goals.Status{State: goals.RunnerRunning, Character: goals.Character{Class: c.class, Level: c.level, Area: "Blood Moor", AreaID: 2}}
			}
			rig, err := startLadder(ctx, lo, "", state, fakeEnableFirsts(t, clk), m, status, func() string { return "Blood Moor" }, nil)
			if err != nil || rig == nil || rig.Fun == nil || rig.Fun.Proposals == nil {
				t.Fatalf("%v %v", rig, err)
			}
			pd := rig.Fun.Proposals
			view, answer := "http://"+rig.Served.View.String(), "http://"+rig.Served.Answer.String()
			waitFor := func(what string, ok func() bool) {
				t.Helper()
				for end := time.Now().Add(8 * time.Second); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
					if ok() {
						return
					}
				}
				t.Fatalf("timeout: %s (errors %v %v)", what, rig.Fun.Errors(), pd.Errors())
			}
			orders := func() int {
				var ov watch.OrdersView
				httpDo(t, "GET", view+"/api/orders", nil, &ov)
				return len(ov.Orders)
			}

			// away: the proposal appears with the default (stay) and nothing moves
			clk.Advance(10 * time.Hour)
			waitFor("a proposal", func() bool { return len(pd.View().Pending) == 1 })
			pv := pd.View()
			if !pv.Away || pv.Pending[0].To != "Cold Plains" || pv.Pending[0].Options[1] != "stay" {
				t.Fatalf("view %+v", pv)
			}
			if n := orders(); n != 0 {
				t.Fatalf("%d orders placed while away", n)
			}
			id := pv.Pending[0].ID

			// stay: the pair is quiet for the cooldown, then asked again
			if code := httpDo(t, "POST", answer+"/api/proposals/answer", map[string]any{"id": id, "answer": "stay"}, nil); code != 200 {
				t.Fatalf("stay: %d", code)
			}
			clk.Advance(10 * time.Minute)
			time.Sleep(2500 * time.Millisecond)
			if n := len(pd.View().Pending); n != 0 {
				t.Fatalf("asked again inside the cooldown: %d", n)
			}
			clk.Advance(31 * time.Minute)
			waitFor("the proposal again after the cooldown", func() bool { return len(pd.View().Pending) == 1 })
			id = pd.View().Pending[0].ID

			// go places the order through the orders path
			if code := httpDo(t, "POST", answer+"/api/proposals/answer", map[string]any{"id": id, "answer": "go"}, nil); code != 200 {
				t.Fatalf("go: %d", code)
			}
			if n := orders(); n != 1 {
				t.Fatalf("orders after go: %d", n)
			}

			// the proposal state is a file next to the other ladder state
			if _, err := os.Stat(filepath.Join(state, "proposals-human.json")); err != nil {
				t.Errorf("no state file: %v", err)
			}

			// AC2: an order from the map above the range asks for a charter and the hero says so once
			var jv watch.JournalView
			code := httpDo(t, "POST", answer+"/api/orders", map[string]any{"kind": "go", "area": "Burial Grounds", "until": "forever"}, nil)
			if code != 402 {
				t.Fatalf("above-range map order: %d", code)
			}
			httpDo(t, "GET", view+"/api/journal?limit=50", nil, &jv)
			var said bool
			for _, l := range jv.Lines {
				if strings.HasPrefix(l.Text, "That is above my range (recommended L8-L14, I am level ") && strings.Contains(l.Text, "tally; go?") {
					said = true
				}
			}
			if !said {
				t.Errorf("no hero warning in the journal: %+v", jv.Lines)
			}
			if n := orders(); n != 1 {
				t.Errorf("a refused charter changed the orders: %d", n)
			}
		})
	}
}
