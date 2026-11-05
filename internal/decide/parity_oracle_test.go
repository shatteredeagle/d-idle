package decide_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

// External test package: resource imports env, env imports decide, so the legacy code can be called as the oracle
// only from here.

// legacyDrink asks the real legacy functions (resource.PotionPolicy, then resource.AbortFight) what a hero in a fight
// does at integer life percent v. V: called directly. RetreatBelt is 0 here: the thin-belt retreat is a rule of the
// belt, not of life, and is not in this key (named in the rule file gap list).
func legacyDrink(v int, f map[string]string, diff d2.Difficulty) string {
	t := resource.DefaultTuning()
	t.RetreatBelt = 0
	in := resource.Input{Life: v, MaxLife: 100, Hostiles: 1, HasTP: f["kit"] != "NONE", Diff: diff, Level: 10,
		ChampionPack: f["champbonus"] == "early"}
	switch f["held"] {
	case "hp":
		in.Belt.HP = 3
	case "rv":
		in.Belt.RV = 3
	}
	if resource.PotionPolicy(in, t).Drink != resource.PotionNone {
		return "drink"
	}
	switch k := resource.AbortFight(in, t).Kind; k {
	case resource.Continue:
		return "fight"
	case resource.PortalOut:
		return "portal"
	case resource.LeaveGame:
		return "leave"
	case resource.Flee:
		return "run"
	default:
		return "unexpected " + k.String()
	}
}

// legacyEngage is the unique-pack rule of resource.PreEngage: never engage below minEngageLifePct (V preengage.go,
// attack.go read; the knob value comes from the default tuning).
func legacyEngage(v int) string {
	if v < resource.DefaultTuning().MinEngageLifePct {
		return "avoid"
	}
	return "fight"
}

// legacyMerc is bot's hire gate: a hero without a living merc hires at or above bot.mercHireMinGold (V
// internal/bot/resource.go).
func legacyMerc(v int, f map[string]string) string {
	if f["merc"] != "alive" && v >= env.DefaultTuning().Bot.MercHireMinGold {
		return "hire"
	}
	return "wait"
}

// TestDefaultParityNumericPoints closes the band-interior splits of t0.retreat.drink, t0.unique.fight and
// t2.mercenary: for every context of the point (both hardcore flags, every enum value), every integer value of the
// reported number (life 1..100, gold 0..3000) that falls in the context's band must give the SAME legacy answer
// (0 splits), equal to the catalogue default and the rule file answer (0 mismatches), and Build from the raw number
// must land on that context (the From plumbing). Two builds.
func TestDefaultParityNumericPoints(t *testing.T) {
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	rf, err := decide.ShippedRuleFile()
	if err != nil {
		t.Fatal(err)
	}
	type axis struct {
		field, src string
		lo, hi     int
		legacy     func(v int, f map[string]string, diff d2.Difficulty) string
	}
	axes := map[string]axis{
		"t0.retreat.drink": {"drinklife", "life", 1, 100, legacyDrink},
		"t0.unique.fight":  {"lifeengage", "life", 1, 100, func(v int, _ map[string]string, _ d2.Difficulty) string { return legacyEngage(v) }},
		"t2.mercenary":     {"mercgold", "gold", 0, 3000, func(v int, f map[string]string, _ d2.Difficulty) string { return legacyMerc(v, f) }},
	}
	for id, ax := range axes {
		ctxs, err := cat.Contexts(id)
		if err != nil {
			t.Fatal(err)
		}
		var checked, splits, mismatch int
		for _, class := range []string{"sorceress", "paladin"} {
			for _, diff := range []struct {
				name string
				d    d2.Difficulty
			}{{"normal", d2.Normal}, {"nightmare", d2.Nightmare}} {
				for _, c := range ctxs {
					fields := map[string]string{}
					for k, v := range c.Fields {
						fields[k] = v
					}
					fields["diff"] = diff.name
					var seen map[string]bool = map[string]bool{}
					for v := ax.lo; v <= ax.hi; v++ {
						label, err := cat.BandLabel(class, ax.field, float64(v))
						if err != nil {
							t.Fatal(err)
						}
						if label != c.Fields[ax.field] {
							continue
						}
						want := ax.legacy(v, fields, diff.d)
						seen[want] = true
						o := decide.Obs{Class: class, Num: map[string]float64{ax.src: float64(v)}, Cat: map[string]string{}}
						for k, x := range fields {
							if k != ax.field {
								o.Cat[k] = x
							}
						}
						got, err := cat.Build(id, o, 1, nil)
						if err != nil {
							t.Fatal(err)
						}
						if got.Fields[ax.field] != label {
							t.Fatalf("%s %s=%d built into band %s, want %s", id, ax.src, v, got.Fields[ax.field], label)
						}
						def, _ := cat.Default(got)
						file, ok := rf.Lookup(got)
						if !ok || def != file {
							t.Errorf("%s: default %q and rule file %q (%v) disagree at %s", id, def, file, ok, got.Key)
						}
						if def != want {
							mismatch++
							if mismatch < 10 {
								t.Errorf("%s %s=%d %s: legacy %q, default %q at %s", id, ax.src, v, class, want, def, got.Key)
							}
						}
						checked++
					}
					if len(seen) > 1 {
						splits++
						t.Errorf("%s: band-interior split at %s (%s): legacy answers %v", id, c.Key, diff.name, seen)
					}
				}
			}
		}
		t.Logf("%s: %d contexts, %d grid states, %d mismatch, %d band-interior splits", id, len(ctxs), checked, mismatch, splits)
		if checked == 0 {
			t.Errorf("%s: nothing checked", id)
		}
	}
}
