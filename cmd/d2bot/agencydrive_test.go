package main

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"d2lod-bot/internal/agency"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
)

// driveCase is one hero for the mode 2 versus mode 0 comparison: a class and the hard points its player bought.
type driveCase struct {
	class d2.Class
	own   map[string]int
}

// The offline drill of agency.planner 2 (TASK-984 step 3): the same hero under mode 0, 1 and 2, a Paladin and a
// Sorceress. Modes 0 and 1 never hand the fighter anything; mode 2 hands it exactly agency.DriveOf of the plan.
// Shipped capability table: nothing the planner could pick is proven for the attack or aura yet, so mode 2 keeps the
// profile (no change). With the planner's picks marked supported, mode 2 overrides where the plan differs from the
// profile's own pick and nowhere else.
func TestAgencyDriveModesAgainstProfile(t *testing.T) {
	cases := map[string][]driveCase{
		"paladin": {
			{d2.Paladin, map[string]int{"Zeal": 1, "Might": 1, "Blessed Hammer": 12, "Concentration": 8}},
			{d2.Paladin, map[string]int{"Zeal": 1, "Vengeance": 20, "Might": 1, "Concentration": 1}},
			{d2.Paladin, map[string]int{"Zeal": 1, "Holy Bolt": 20, "Sanctuary": 1}},
			{d2.Paladin, map[string]int{"Zeal": 8, "Might": 6}},
		},
		"sorceress": {
			{d2.Sorceress, map[string]int{"Fire Bolt": 1, "Blizzard": 12, "Frozen Orb": 8, "Cold Mastery": 3}},
			{d2.Sorceress, map[string]int{"Charged Bolt": 1, "Fire Ball": 20}},
			{d2.Sorceress, map[string]int{"Charged Bolt": 6, "Fire Bolt": 1, "Ice Bolt": 1, "Blizzard": 12}},
		},
	}
	for name, hs := range cases {
		for _, supportAll := range []bool{false, true} {
			overrides := 0
			for i, c := range hs {
				var plans [3]agency.Drive
				for mode := 0; mode <= 2; mode++ {
					w, pl := driveWire(t, c, mode, supportAll)
					at, au := w.Drive()
					plans[mode] = agency.Drive{Attack: at, Aura: au}
					want := agency.Drive{}
					if mode == agency.ModeDrive {
						want = agency.DriveOf(pl)
					}
					if at != want.Attack || au != want.Aura {
						t.Fatalf("%s #%d supportAll=%v mode %d: drive (%q,%q), DriveOf says (%q,%q)", name, i, supportAll, mode, at, au, want.Attack, want.Aura)
					}
					if mode < 2 && (at != "" || au != "") {
						t.Fatalf("%s #%d mode %d changed the hero: %q %q", name, i, mode, at, au)
					}
					if mode == 2 && (at != "" || au != "") {
						overrides++
						if at != "" && (pl.Attack == nil || pl.Attack.Skill != at || pl.Attack.Skill == pl.Prior) {
							t.Fatalf("%s #%d: override %q is not the plan's pick over the profile's %q", name, i, at, pl.Prior)
						}
					}
				}
				if plans[0] != plans[1] {
					t.Fatalf("%s #%d: mode 0 and 1 differ", name, i)
				}
			}
			if !supportAll && overrides != 0 {
				t.Errorf("%s: the shipped table overrode the profile %d times (the planner's picks are not proven)", name, overrides)
			}
			if supportAll && overrides == 0 {
				t.Errorf("%s: with every pick supported mode 2 never differed from the profile in %d heroes; the test heroes no longer exercise the wire", name, len(hs))
			}
		}
	}
}

// Must-fail: a plan pick that is not supported is never driven, however much better the estimate; this is the line that
// keeps mode 2 from playing a skill the engine cannot cast.
func TestAgencyDriveIgnoresUnprovenPicks(t *testing.T) {
	for _, c := range []driveCase{
		{d2.Paladin, map[string]int{"Zeal": 1, "Might": 1, "Blessed Hammer": 12, "Concentration": 8}},
		{d2.Sorceress, map[string]int{"Fire Bolt": 1, "Blizzard": 12, "Frozen Orb": 8}},
	} {
		w, pl := driveWire(t, c, agency.ModeDrive, false)
		if pl.Attack != nil && pl.Attack.Tier == agency.Supported {
			t.Skipf("%v: the plan's attack is now supported, the premise of this test is gone", c.class)
		}
		if at, au := w.Drive(); at != "" || au != "" {
			t.Fatalf("%v: unproven plan driven: %q %q (plan attack %v)", c.class, at, au, pl.Attack)
		}
	}
}

// driveWire builds the wire at a knob mode, feeds it one snapshot at clvl 24 and returns it with the plan it used.
// supportAll marks every attack and aura row of the hero's class supported (a stand-in for a future proven.json).
func driveWire(t *testing.T, c driveCase, mode int, supportAll bool) (*agencyWire, agency.Plan) {
	t.Helper()
	e := &env.Env{Tune: &env.Tuning{}}
	e.Tune.Agency.Planner = mode
	db := &data.DB{Skills: []*data.Skill{{Key: "Zeal"}}, Classes: []*data.CharClass{{}, {}, {}, {Int: 15}}}
	now := time.Unix(9000, 0)
	w := wireAgency(db, e, func() string { return "Testy" }, func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if w == nil {
		t.Fatal("no wire")
	}
	if supportAll {
		for n, cp := range w.pl.Env.Caps.Skills {
			if cp.Role == agency.RoleAttack || cp.Role == agency.RoleAura {
				cp.Tier = agency.Supported
				w.pl.Env.Caps.Skills[n] = cp
			}
		}
	}
	w.owned = func(*d2.Player) map[string]int { return c.own }
	snap := &d2.Snapshot{Player: &d2.Player{Class: c.class, Level: 24, Energy: 40, MaxLife: 300, Area: 2}}
	w.OnSnapshot(snap)
	_, pl, err := w.eval(snap)
	if err != nil {
		t.Fatal(err)
	}
	return w, pl
}
