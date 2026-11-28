package survival

import (
	"testing"
	"time"

	"d2lod-bot/internal/power"
)

// Card zone5: the Sorceress arm of the A/B lost because the model's first pick was a graveyard it cannot farm for a
// stay (docs/probes/survival-holdout5-2026-11-21.md section 4). Units are the measured means of
// internal/danger/areaunits.json (V); the priors are the ones the margin logs printed (holdout 5); truth is what the
// zones realized there. All five candidates share one margin, so only the value decides.
var act1Supply = []struct {
	area         int
	name         string
	prior, units float64
	truth        float64
	graveyard    bool
}{
	{3, "Cold Plains", 54, 198.1, 60, false},
	{4, "Stony Field", 120, 152.5, 300, false},
	{17, "Burial Grounds", 152, 38.2, 7, true},
	{18, "Crypt", 165, 29.1, 7, true},
	{19, "Mausoleum", 160, 25.8, 7, true},
}

func supplyScene(e env, h power.Hero, arch string, cur ZoneKey, now time.Time) Input {
	in := activeIn(e, h, arch, now)
	base := in.Cands[0]
	in.Cands, in.Current = nil, cur
	for _, z := range act1Supply {
		c := base
		c.Key, c.Name, c.Units = ZoneKey{z.area, "normal"}, z.name, z.units
		c.Eco.KillsPerMin, c.Eco.XPPerMin = 9, z.prior
		if c.Key == cur {
			c.TravelMin = 0
		}
		in.Cands = append(in.Cands, c)
	}
	return in
}

// farm plays the choice for the given minutes against the truth table and returns the XP earned, the minutes spent in
// the graveyards and the first pick.
func farm(t *testing.T, e env, h power.Hero, arch string, minutes int) (xp float64, graveMin int, first string) {
	t.Helper()
	l := NewLedger(e.p)
	cur := ZoneKey{}
	for i := 0; i < minutes; i++ {
		now := t0.Add(time.Duration(i) * time.Minute)
		in := supplyScene(e, h, arch, cur, now)
		d, err := l.Choose(in)
		if err != nil {
			t.Fatal(err)
		}
		if d.Hole {
			t.Fatalf("minute %d: premise: every zone is safe for the fixture hero, got a hole", i)
		}
		if i == 0 {
			first = d.Name
		}
		cur = ZoneKey{d.Area, d.Diff}
		for _, z := range act1Supply {
			if z.area != d.Area {
				continue
			}
			xp += z.truth
			if z.graveyard {
				graveMin++
			}
			l.Observe(cur, now, Event{Kind: EvXP, XP: z.truth, Seconds: 60}, in.Power)
		}
	}
	return xp, graveMin, first
}

func supplyOff(p *Params) {
	p.SupplyMinutes = 0
	for id, a := range p.Archetypes {
		a.SupplyMinutes = 0
		p.Archetypes[id] = a
	}
}

// The caster with the shipped rows opens in a field the SME route names for these levels (docs/sme/play-act-route.md:
// Cold Plains, then Stony Field, S) and never farms a graveyard; without the supply rows it opens in one, as the A/B did.
func TestCasterDoesNotOpenInAGraveyard(t *testing.T) {
	e := newEnv(t, "caster")
	xp, grave, first := farm(t, e, sorceress(12), "caster", 30)
	if first != "Stony Field" && first != "Cold Plains" {
		t.Errorf("first pick %q, want a field with supply", first)
	}
	if grave != 0 {
		t.Errorf("%d minutes in graveyards, want none", grave)
	}
	old := newEnv(t, "caster")
	supplyOff(old.p)
	oldXP, oldGrave, oldFirst := farm(t, old, sorceress(12), "caster", 30)
	if oldGrave == 0 || oldFirst == first {
		t.Fatalf("premise: without the supply rows the caster opens in a graveyard, got %q (%d graveyard minutes)", oldFirst, oldGrave)
	}
	if xp <= oldXP {
		t.Errorf("xp %.0f with the supply rows, %.0f without: must be higher", xp, oldXP)
	}
}

// The bruiser row is left as measured (Paladin PASS, holdout 4 and 5): its run is the same with and without the rows.
func TestBruiserChoiceIsUnchangedBySupplyRows(t *testing.T) {
	e := newEnv(t, "bruiser")
	xp, grave, first := farm(t, e, paladin(12), "bruiser", 30)
	old := newEnv(t, "bruiser")
	supplyOff(old.p)
	oldXP, oldGrave, oldFirst := farm(t, old, paladin(12), "bruiser", 30)
	if xp != oldXP || grave != oldGrave || first != oldFirst {
		t.Errorf("bruiser run changed: %.0f/%d/%q against %.0f/%d/%q", xp, grave, first, oldXP, oldGrave, oldFirst)
	}
}

// The cap is the zone's units over the stay, for both builds when their row has it; unknown units and the shadow
// (not Active) leave the prior alone.
func TestSupplyCapIsUnitsOverStay(t *testing.T) {
	for _, b := range twoBuilds {
		t.Run(b.name, func(t *testing.T) {
			e := newEnv(t, b.arch)
			e.p.Archetypes = map[string]ArchTune{b.arch: {SupplyMinutes: 20}}
			in := supplyScene(e, b.hero(12), b.arch, ZoneKey{}, t0)
			l := NewLedger(e.p)
			bg := in.Cands[2]
			want := 152 * (38.2 / 20 / 9)
			if got := l.prior(in, bg); got < want*0.999 || got > want*1.001 {
				t.Errorf("capped prior %.2f, want %.2f", got, want)
			}
			if got := l.prior(in, in.Cands[0]); got != 54 {
				t.Errorf("Cold Plains holds enough for the stay: prior %.2f, want 54", got)
			}
			bg.Units = 0
			if got := l.prior(in, bg); got != 152 {
				t.Errorf("unknown units: prior %.2f, want 152", got)
			}
			in.Active = false
			if got := l.prior(in, in.Cands[2]); got != 152 {
				t.Errorf("shadow: prior %.2f, want 152", got)
			}
		})
	}
}

// Every archetype the power profiles can place a build on has a row here, so no class is judged by the globals by
// accident (Necromancer summoner, Amazon ranged_physical, Hammerdin hybrid_tank_caster; Barbarian is bruiser, Druid and
// Assassin are caster: internal/power TestEveryBuildResolvesArchetype).
func TestEveryArchetypeHasASurvivalRow(t *testing.T) {
	e := newEnv(t, "caster")
	for _, id := range e.m.P.ArchetypeIDs() {
		if _, ok := e.p.Archetypes[id]; !ok {
			t.Errorf("archetype %q has no row in params.json archetypes", id)
		}
	}
}
