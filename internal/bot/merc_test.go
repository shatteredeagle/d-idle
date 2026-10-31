package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

type mercTownFake struct {
	*fakeTown
	hires, revives int
	hireOK         bool
	unverified     bool
}

func (m *mercTownFake) HireMerc(context.Context, *d2.Snapshot) (bool, error) {
	m.hires++
	return m.hireOK, nil
}

func (m *mercTownFake) ReviveMerc(_ context.Context, u bool) (bool, error) {
	m.revives++
	m.unverified = u
	return true, nil
}

func townSnap(area, gold int, mons ...d2.Unit) *d2.Snapshot {
	return &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Area: area, Gold: gold, Life: 100, MaxLife: 100}, Monsters: mons}
}

func TestMercHireChore(t *testing.T) {
	rg := newRig(t)
	mt := &mercTownFake{fakeTown: rg.town, hireOK: true}
	rg.r.d.Town = mt
	s := townSnap(areaLutGholein, 1000)
	r := rg.r
	if r.mercWanted(s) != "" {
		t.Fatal("hire wanted before the merc was missing long enough")
	}
	r.resInput(s) // first town read starts the absence clock
	r.e.Clock.Sleep(mercGoneAfter)
	if got := r.mercWanted(s); got != "hire" {
		t.Fatalf("want hire, got %q", got)
	}
	if r.mercWanted(townSnap(areaLutGholein, 100)) != "" || r.mercWanted(townSnap(1, 1000)) != "" {
		t.Error("hire wanted without gold or outside act 2")
	}
	r.mercChore(context.Background(), s)
	if mt.hires != 1 || !r.merc.known {
		t.Fatalf("hires %d known %v", mt.hires, r.merc.known)
	}
	if r.mercWanted(s) != "" {
		t.Error("chore repeated inside the backoff")
	}
}

func TestMercReviveKnob(t *testing.T) {
	for _, on := range []bool{false, true} {
		rg := newRig(t, func(_ *Config, _ *Deps, tn *env.Tuning) { tn.Bot.MercReviveUnverified = on })
		mt := &mercTownFake{fakeTown: rg.town}
		rg.r.d.Town = mt
		r := rg.r
		r.merc.known = true // hired earlier this run
		s := townSnap(areaLutGholein, 50000)
		r.resInput(s)
		r.e.Clock.Sleep(mercGoneAfter + time.Second)
		got := r.mercWanted(s)
		if on && got != "revive" || !on && got != "" {
			t.Errorf("knob %v: got %q", on, got)
		}
		if in := r.resInput(s); !in.MercKnown || !in.MercDead() {
			t.Errorf("sticky known merc must read dead: %+v", in)
		}
	}
}

func TestEntryGate(t *testing.T) {
	p, _ := LoadPlan("")
	if d := p.EntryAllowed("nightmare", 35, 2, nil); d.OK {
		t.Error("nightmare Blood Moor at 35 must be refused (doc floor 36)")
	}
	if d := p.EntryAllowed("nightmare", 36, 2, nil); !d.OK {
		t.Errorf("nightmare Blood Moor at 36: %s", d.Why)
	}
	if d := p.EntryAllowed("hell", 70, 9, nil); d.OK {
		t.Error("hell cave at 70 must be refused")
	}
	if d := p.EntryAllowed("normal", 1, 2, nil); !d.OK {
		t.Error("normal is never refused")
	}
	if p.DifficultyFor(35, 0, nil) != "normal" || p.DifficultyFor(36, 0, nil) != "nightmare" {
		t.Errorf("difficulty at 35/36: %s %s", p.DifficultyFor(35, 0, nil), p.DifficultyFor(36, 0, nil))
	}
}

func TestDifficultyForResistGate(t *testing.T) {
	p, _ := LoadPlan("")
	low := d2.Resists{Fire: 20, Cold: 75, Lightning: 75, Poison: 50}
	if got := p.DifficultyFor(36, 0, &low); got != "normal" {
		t.Errorf("low fire: %s", got)
	}
	ok := d2.Resists{Fire: 75, Cold: 75, Lightning: 75, Poison: 50}
	if got := p.DifficultyFor(36, 0, &ok); got != "nightmare" {
		t.Errorf("capped: %s", got)
	}
}
