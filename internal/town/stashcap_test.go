package town_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/town"
)

// capRig is an audit rig whose log lines are captured.
func capRig(t *testing.T, gold, clvl, stash int) (*rig, *bytes.Buffer) {
	r := audit(t, gold, clvl, true)
	var buf bytes.Buffer
	r.Env.Log = slog.New(slog.NewTextHandler(&buf, nil))
	r.W.Player.StashGold = stash
	return r, &buf
}

// The stash step never deposits above the cap: with 100 gold of room only the difference goes in.
func TestStashGoldCapDeposit(t *testing.T) {
	r, buf := capRig(t, 2000, 5, town.StashGoldCap-100)
	r.W.StashCap = town.StashGoldCap
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Gold: true})
	if err != nil || rep.Errors != 0 {
		t.Fatalf("%v %+v", err, rep.Steps)
	}
	if r.W.Player.StashGold != town.StashGoldCap {
		t.Fatalf("bank %d, want the cap %d", r.W.Player.StashGold, town.StashGoldCap)
	}
	// Full now: the next visit skips, logs once, and does not fail.
	for i := 0; i < 2; i++ {
		rep, err = r.T.Chores(context.Background(), town.ChorePlan{Gold: true})
		if err != nil || rep.Errors != 0 {
			t.Fatalf("visit %d: %v %+v", i, err, rep.Steps)
		}
	}
	if n := strings.Count(buf.String(), "stash gold is full"); n != 1 {
		t.Fatalf("%d full-stash log lines, want 1:\n%s", n, buf)
	}
	if r.W.Player.Gold < 1500 {
		t.Fatalf("pack gold %d: it was spent or lost while the stash was full", r.W.Player.Gold)
	}
}

// A stash that refuses below the constant (a wrong cap) costs one failed deposit: it is learned, not an error.
func TestStashGoldCapLearnedFromRefusal(t *testing.T) {
	r, buf := capRig(t, 2000, 5, 200_000)
	r.W.StashCap = 200_000 // the real cap is lower than StashGoldCap
	for i := 0; i < 3; i++ {
		rep, err := r.T.Chores(context.Background(), town.ChorePlan{Gold: true})
		if err != nil || rep.Errors != 0 {
			t.Fatalf("visit %d: %v %+v", i, err, rep.Steps)
		}
	}
	if r.W.Player.StashGold != 200_000 || strings.Count(buf.String(), "taking its gold as the cap") != 1 {
		t.Fatalf("bank %d log:\n%s", r.W.Player.StashGold, buf)
	}
}
