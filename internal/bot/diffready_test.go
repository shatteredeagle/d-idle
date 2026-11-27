package bot

import (
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
