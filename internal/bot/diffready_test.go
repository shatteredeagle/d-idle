package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

// Opt-in hop: under bot.diffReadyV2 the hero is told ready but hops only after the owner's go, once.
func TestDifficultyV2WaitsForOwnerGo(t *testing.T) {
	g := newRig(t)
	g.r.e.Tune.Bot.DiffReadyV2 = 1
	g.r.d.Catalog = nil // potions unread: listed as unread, not a blocker
	g.w.set(func(w *world) { w.level = 60 })
	s, _ := g.w.snapshot()
	s.Player.MaxLife = 900
	s.Player.Resist = d2.Resists{Fire: 80, Cold: 80, Lightning: 80, Poison: 80}
	if got := g.r.wantedDifficulty(s); got != "" {
		t.Fatalf("no go yet, got %q", got)
	}
	if line := g.r.DifficultyReadiness(); !strings.Contains(line, "ready for nightmare") || !strings.Contains(line, "waiting for your go") {
		t.Fatalf("line %q", line)
	}
	g.r.GrantDifficultyGo("nightmare")
	if got := g.r.wantedDifficulty(s); got != "nightmare" {
		t.Fatalf("with the go: %q", got)
	}
	if got := g.r.wantedDifficulty(s); got != "" {
		t.Fatalf("the go is consumed, got %q", got)
	}
}

func TestDifficultyV2NotReadyShowsMissing(t *testing.T) {
	g := newRig(t)
	g.r.e.Tune.Bot.DiffReadyV2 = 1
	g.w.set(func(w *world) { w.level = 20 })
	s, _ := g.w.snapshot()
	g.r.GrantDifficultyGo("nightmare")
	if got := g.r.wantedDifficulty(s); got != "" {
		t.Fatalf("clvl 20 hopped: %q", got)
	}
	if line := g.r.DifficultyReadiness(); !strings.Contains(line, "not ready for nightmare") || !strings.Contains(line, "clvl 20, need 38") {
		t.Fatalf("line %q", line)
	}
	if ok, why, active := g.r.DifficultyGate("nightmare", 20); ok || !active || why == "" {
		t.Fatalf("gate %v %q %v", ok, why, active)
	}
	g.r.e.Tune.Bot.DiffReadyV2 = 0
	if _, _, active := g.r.DifficultyGate("nightmare", 20); active {
		t.Fatal("knob 0 must leave the old rule")
	}
}

func readyRig(t *testing.T, level int) (*rig, *d2.Snapshot) {
	g := newRig(t)
	g.r.e.Tune.Bot.DiffReadyV2 = 1
	g.r.d.Catalog = nil
	g.w.set(func(w *world) { w.level = level })
	s, _ := g.w.snapshot()
	s.Player.MaxLife = 900
	s.Player.Resist = d2.Resists{Fire: 80, Cold: 80, Lightning: 80, Poison: 80}
	s.State = d2.StateInGame
	g.r.obMu.Lock()
	g.r.ob.lastSnap = s
	g.r.obMu.Unlock()
	return g, s
}

// The player's go is refused with the reason while the hero is not ready, and nothing is recorded.
func TestRequestDifficultyGoRefusedWhenNotReady(t *testing.T) {
	g, _ := readyRig(t, 20)
	msg, err := g.r.RequestDifficultyGo("nightmare")
	if err == nil || !strings.Contains(err.Error(), "clvl 20, need 38") {
		t.Fatalf("msg %q err %v", msg, err)
	}
	if g.r.diffGo.Granted("nightmare") {
		t.Fatal("a refused go was recorded")
	}
	if o := g.r.DifficultyOffer(); o.Ready || len(o.Missing) == 0 || o.Target != "nightmare" {
		t.Fatalf("offer %+v", o)
	}
	g.r.e.Tune.Bot.DiffReadyV2 = 0
	if _, err := g.r.RequestDifficultyGo("nightmare"); err == nil {
		t.Fatal("knob 0 must refuse")
	}
}

// A ready hero takes the go for the next difficulty only, and the hop consumes it once.
func TestRequestDifficultyGoConsumedOnce(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		t.Run(c.String(), func(t *testing.T) { requestGoConsumedOnce(t, c) })
	}
}

func requestGoConsumedOnce(t *testing.T, c d2.Class) {
	g, s := readyRig(t, 60)
	s.Player.Class = c
	if _, err := g.r.RequestDifficultyGo("hell"); err == nil {
		t.Fatal("hell is not the next difficulty")
	}
	if _, err := g.r.RequestDifficultyGo("nightmare"); err != nil {
		t.Fatal(err)
	}
	if o := g.r.DifficultyOffer(); !o.Ready || !o.Pending {
		t.Fatalf("offer %+v", o)
	}
	if got := g.r.wantedDifficulty(s); got != "nightmare" {
		t.Fatalf("got %q", got)
	}
	if got := g.r.wantedDifficulty(s); got != "" {
		t.Fatalf("consumed go hopped again: %q", got)
	}
	if g.r.DifficultyOffer().Pending {
		t.Fatal("still pending")
	}
}

// An unconsumed go survives a restart (a new runner reading the same file); a changed difficulty drops it.
func TestDifficultyGoSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "diffgo.json")
	g, s := readyRig(t, 60)
	if err := g.r.PersistDifficultyGo(path); err != nil {
		t.Fatal(err)
	}
	if _, err := g.r.RequestDifficultyGo("nightmare"); err != nil {
		t.Fatal(err)
	}
	g2, s2 := readyRig(t, 60)
	if err := g2.r.PersistDifficultyGo(path); err != nil {
		t.Fatal(err)
	}
	if !g2.r.diffGo.Granted("nightmare") {
		t.Fatal("restart lost the go")
	}
	if got := g2.r.wantedDifficulty(s2); got != "nightmare" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("consumed go left its file")
	}
	// the difficulty moved on (hero now in Nightmare): a stale nightmare go is dropped, not kept for later
	g.r.diffGo.Grant("nightmare")
	s.Player.Difficulty = d2.Difficulty(1)
	g.r.wantedDifficultyV2(s)
	if g.r.diffGo.Granted("nightmare") {
		t.Fatal("stale go kept")
	}
	if b, _ := os.ReadFile(path); len(b) != 0 {
		t.Fatalf("file %q", b)
	}
}
