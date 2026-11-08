package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/watch"
)

// A synthesized Andariel kill through feedHub raises the Hall question; keep leaves the hero running, retire books the
// ledger entry and calls the shutdown path once. The routes are token gated. Both classes.
func TestHallWiredThroughFeedHub(t *testing.T) {
	for _, c := range []struct{ class, name string }{{"Sorceress", "Sorra"}, {"Paladin", "Paula"}} {
		for _, answer := range []string{"keep", "retire"} {
			t.Run(c.class+"/"+answer, func(t *testing.T) {
				var stops atomic.Int32
				old := stopSelf
				stopSelf = func() error { stops.Add(1); return nil }
				t.Cleanup(func() { stopSelf = old })
				ctx, cancel := context.WithCancel(context.Background())
				t.Cleanup(cancel)
				m := goalManager(t)
				const tok = "s3cret-watch-token"
				lo := ladderOptions{on: true, watch: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0", watchToken: tok,
					currencyDir: filepath.Join(t.TempDir(), "currency")}
				status := func() goals.Status {
					return goals.Status{State: goals.RunnerRunning, Character: goals.Character{Name: c.name, Class: c.class, Level: 30, Area: "Catacombs Level 4"}}
				}
				rig, err := startLadder(ctx, lo, "", t.TempDir(), fakeEnableFirsts(t, &watch.SimClock{}), m, status, func() string { return "x" }, nil)
				if err != nil || rig == nil || rig.Fun == nil || rig.Fun.Hall == nil {
					t.Fatalf("Hall not mounted: %v %v", rig, err)
				}
				view, ans := "http://"+rig.Served.View.String(), "http://"+rig.Served.Answer.String()
				hdr := map[string]string{"Authorization": "Bearer " + tok}
				for _, p := range []string{"/watch/hall", "/api/hall"} {
					if code := httpDo(t, "GET", view+p, nil, nil); code != 401 {
						t.Errorf("%s without a token: %d", p, code)
					}
					if code := httpDoH(t, "GET", view+p, nil, hdr); code != 200 {
						t.Errorf("%s with the token: %d", p, code)
					}
				}
				if code := httpDoH(t, "POST", ans+"/api/hall/answer", map[string]string{"answer": answer}, nil); code != 401 {
					t.Errorf("answer without a token: %d", code)
				}
				m.Handle(goals.Event{Type: goals.EvKill, Monster: "Andariel", XP: 100})
				deadline := time.Now().Add(5 * time.Second)
				for rig.Fun.Hall.Pending() == nil && time.Now().Before(deadline) {
					time.Sleep(50 * time.Millisecond)
				}
				q := rig.Fun.Hall.Pending()
				if q == nil || !strings.Contains(q.Text, "Retire "+c.name+" into the Hall of Fame") {
					t.Fatalf("question: %+v", q)
				}
				// no auto-retire: ticks pass, the question stays open and the bot is never stopped
				for i := 0; i < 5; i++ {
					rig.Fun.Settle()
				}
				if stops.Load() != 0 || rig.Fun.Hall.Pending() == nil {
					t.Fatalf("auto action: stops %d", stops.Load())
				}
				if code := httpDoH(t, "POST", ans+"/api/hall/answer", map[string]string{"answer": answer}, hdr); code != 200 {
					t.Fatalf("answer %s: %d", answer, code)
				}
				var hv watch.HallView
				httpDoH(t, "GET", view+"/api/hall", nil, hdr, &hv)
				switch answer {
				case "keep":
					if stops.Load() != 0 || len(hv.Heroes) != 0 || rig.Fun.Hall.Pending() != nil {
						t.Errorf("keep: stops %d heroes %v", stops.Load(), hv.Heroes)
					}
				case "retire":
					if stops.Load() != 1 || len(hv.Heroes) != 1 || hv.Heroes[0].Status != "retired" || hv.Heroes[0].Name != c.name {
						t.Errorf("retire: stops %d heroes %+v", stops.Load(), hv.Heroes)
					}
					httpDoH(t, "POST", ans+"/api/hall/answer", map[string]string{"answer": answer}, hdr)
					if stops.Load() != 1 {
						t.Errorf("a replay stopped again: %d", stops.Load())
					}
				}
				// a death is not permanent without a signal: nothing enters the Hall
				m.Handle(goals.Event{Type: goals.EvDeath})
				time.Sleep(200 * time.Millisecond)
				httpDoH(t, "GET", view+"/api/hall", nil, hdr, &hv)
				if want := map[string]int{"keep": 0, "retire": 1}[answer]; len(hv.Heroes) != want {
					t.Errorf("a recoverable death changed the Hall: %d, want %d (%s)", len(hv.Heroes), want, fmt.Sprint(hv.Heroes))
				}
			})
		}
	}
}

// Without an account ledger there is no Hall and nothing else breaks.
func TestHallConfigNeedsLedger(t *testing.T) {
	if hc, err := newHallConfig(nil, "human", func() goals.Status { return goals.Status{} }); hc != nil || err != nil {
		t.Errorf("%v %v", hc, err)
	}
}
