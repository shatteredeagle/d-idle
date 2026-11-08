package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/watch"
)

// fakeEnableFirsts is fakeEnable on the catalogue with the Codex point, as Runner.SetLadderFirsts makes it.
func fakeEnableFirsts(t *testing.T, clk *watch.SimClock) enableFunc {
	return func(dir, profile string, wire bot.LadderWire) (*ladder.Ladder, error) {
		cat, err := decide.Shipped()
		if err != nil {
			return nil, err
		}
		tbl, err := decide.ShippedFirsts()
		if err != nil {
			return nil, err
		}
		if cat, err = cat.WithFirsts(tbl); err != nil {
			return nil, err
		}
		st, err := decide.New(cat, profile, clk)
		if err != nil {
			return nil, err
		}
		sink, src, err := wire(st, cat, clk)
		if err != nil {
			return nil, err
		}
		l, err := ladder.New(ladder.Config{Cat: cat, Store: st, Clock: clk, Sink: sink, Answers: src})
		if err != nil {
			return nil, err
		}
		ladder.Install(l)
		t.Cleanup(func() { ladder.Install(nil) })
		return l, nil
	}
}

// The manager's events reach the return screen, the tally strip and the Codex through feedHub, in d.idle mode, with the
// class unknown until the first status poll (the firsts wait for it).
func TestFeedHubFunLoop(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			ov := t.TempDir()
			if err := os.MkdirAll(filepath.Join(ov, "excel"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(ov, "excel", "Misc.txt"), []byte("code\tname\nzmk\tMark\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			m := goalManager(t)
			lo := ladderOptions{on: true, watch: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0",
				overlayDir: ov, currencyDir: filepath.Join(t.TempDir(), "currency")}
			status := func() goals.Status {
				return goals.Status{State: goals.RunnerRunning, Character: goals.Character{Class: class, Level: 5, Area: "Blood Moor", AreaID: 2}}
			}
			rig, err := startLadder(ctx, lo, "", t.TempDir(), fakeEnableFirsts(t, &watch.SimClock{}), m, status, func() string { return "Blood Moor" }, nil)
			if err != nil || rig == nil || rig.Fun == nil {
				t.Fatalf("%v %v", rig, err)
			}
			view := "http://" + rig.Served.View.String()
			m.Handle(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2})
			for i := 0; i < 25; i++ {
				m.Handle(goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 10})
			}
			m.Handle(goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1})
			var tv watch.TallyView
			var cv watch.CodexView
			deadline := time.Now().Add(8 * time.Second)
			for time.Now().Before(deadline) {
				httpDo(t, "GET", view+"/api/tally", nil, &tv)
				httpDo(t, "GET", view+"/api/codex", nil, &cv)
				if tv.Balance == 13 && cv.Filled >= 2 {
					break
				}
				time.Sleep(200 * time.Millisecond)
			}
			// 25 kills 1, Fallen first 5, Blood Moor first 5, expedition 2
			if !tv.Enabled || tv.Balance != 13 || cv.Filled < 2 {
				t.Fatalf("tally %+v, codex filled %d (%q), wiring errors %v", tv, cv.Filled, cv.Line, rig.Fun.Errors())
			}
			var d watch.ReturnData
			if code := httpDo(t, "GET", view+"/api/return?hours=1", nil, &d); code != 200 || len(d.Trips) != 1 || d.Trips[0].Kills != 25 {
				t.Errorf("return screen: %d %+v", code, d.Trips)
			}
			if code := httpDo(t, "GET", view+"/watch/return", nil, nil); code != 200 {
				t.Errorf("/watch/return: %d", code)
			}
		})
	}
}

// Without -watch, with a scripted player, or in vanilla mode the parts stay off and the catalogue stays the shipped one.
func TestUseFirstsOnlyForAPerson(t *testing.T) {
	if err := useFirsts(ladderOptions{on: true, watch: true, profile: "cautious"}, nil); err != nil {
		t.Fatal(err) // a nil runner would panic if it were used
	}
	if err := useFirsts(ladderOptions{on: true}, nil); err != nil {
		t.Fatal(err)
	}
	useFun(nil, nil, "run")
	useFun(&ladderRig{}, nil, "run")
	if len(townNames()) != 5 || !townNames()["Rogue Encampment"] {
		t.Errorf("towns: %v", townNames())
	}
}
