package ladder_test

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/task"
)

// The T0 grids (docs/t0-slice.md): each calls the legacy function directly and asserts the ladder answer equals it
// (V by construction of AskWithDefault, kept as the proof), and counts where the untrained catalogue default differs.

// legacyDrinkAnswer is the legacy action at a state: PotionPolicy first, then AbortFight. ok is false for RetreatOut,
// which has no answer of the point (the call site does not ask there).
func legacyDrinkAnswer(in resource.Input, t resource.Tuning) (string, bool) {
	if resource.PotionPolicy(in, t).Drink != resource.PotionNone {
		return "drink", true
	}
	switch resource.AbortFight(in, t).Kind {
	case resource.Continue:
		return "fight", true
	case resource.PortalOut:
		return "portal", true
	case resource.LeaveGame:
		return "leave", true
	case resource.Flee:
		return "run", true
	}
	return "", false
}

func TestParityRetreatDrink(t *testing.T) {
	l, _, _ := newLadder(t, nil)
	n, skipped, split := 0, 0, 0
	for _, belt0 := range []int{0, -1} { // -1: the shipped retreatBelt
		tn := resource.DefaultTuning()
		if belt0 == 0 {
			tn.RetreatBelt = 0
		}
		for _, c := range classes {
			for _, diff := range []d2.Difficulty{d2.Normal, d2.Nightmare} {
				for _, hc := range []bool{false, true} {
					for life := 1; life <= 100; life++ {
						for _, held := range []string{ladder.HeldNone, ladder.HeldHP, ladder.HeldRV} {
							for _, tp := range []bool{false, true} {
								for _, champ := range []bool{false, true} {
									in := resource.Input{Life: life, MaxLife: 100, Hostiles: 1, HasTP: tp, Diff: diff, Level: 10, ChampionPack: champ, Class: c}
									switch held {
									case ladder.HeldHP:
										in.Belt.HP = 3
									case ladder.HeldRV:
										in.Belt.RV = 3
									}
									want, ok := legacyDrinkAnswer(in, tn)
									if !ok {
										skipped++
										continue
									}
									h := hero(c, 100)
									h.Diff, h.Hardcore, h.AreaClass = diff, hc, "field"
									kit := ladder.KitFor(tp, in.Belt.HP+in.Belt.RV)
									got, err := l.DrinkChoice(h, life, held, kit, resource.ChampionEarly(in), want)
									if err != nil {
										t.Fatal(err)
									}
									if got != want {
										t.Fatalf("%v %v hc %v life %d held %s tp %v champ %v: ladder %q, legacy %q", c, diff, hc, life, held, tp, champ, got, want)
									}
									o := decide.Obs{Num: map[string]float64{"life": float64(life)}, Cat: map[string]string{"held": held, "kit": kit}}
									if resource.ChampionEarly(in) {
										o.Cat["champbonus"] = "early"
									}
									if def, err := l.CatalogueDefault(ladder.PointDrink, h, o); err != nil {
										t.Fatal(err)
									} else if def != want {
										split++
									}
									n++
								}
							}
						}
					}
				}
			}
		}
	}
	if n < 19000 {
		t.Errorf("only %d states", n)
	}
	t.Logf("t0.retreat.drink parity grid: %d states (%d RetreatOut states skipped), catalogue default differs at %d", n, skipped, split)
}

func TestParityUniqueFight(t *testing.T) {
	l, _, _ := newLadder(t, nil)
	rt := resource.DefaultTuning()
	n, avoids, split := 0, 0, 0
	for _, c := range classes {
		for _, diff := range []d2.Difficulty{d2.Normal, d2.Nightmare} {
			for life := 1; life <= 100; life++ {
				for _, tp := range []bool{false, true} {
					for _, hp := range []int{0, 1, 5} {
						for danger := 0; danger <= 3; danger++ {
							in := resource.Input{Life: life, MaxLife: 100, PackSeen: 3, PackDanger: danger, HasTP: tp, Diff: diff, Level: 10, Class: c,
								Mana: 100, MaxMana: 100, Belt: resource.Belt{HP: hp}}
							want := "fight"
							if resource.PreEngage(in, rt).Action == resource.Avoid || in.LifePct() < rt.MinEngageLifePct {
								want = "avoid"
								avoids++
							}
							h := hero(c, 100)
							h.Diff, h.AreaClass = diff, "field"
							kit := ladder.KitFor(tp, hp)
							got, err := l.UniqueFight(h, life, danger, kit, want)
							if err != nil {
								t.Fatal(err)
							}
							if got != want {
								t.Fatalf("%v %v life %d tp %v hp %d danger %d: ladder %q, legacy %q", c, diff, life, tp, hp, danger, got, want)
							}
							o := decide.Obs{Num: map[string]float64{"life": float64(life)}, Cat: map[string]string{"kit": kit, "packdanger": string(rune('0' + max(danger, 1)))}}
							if def, err := l.CatalogueDefault(ladder.PointUnique, h, o); err != nil {
								t.Fatal(err)
							} else if def != want {
								split++
							}
							n++
						}
					}
				}
			}
		}
	}
	t.Logf("t0.unique.fight parity grid: %d states, %d legacy avoids, catalogue default differs at %d", n, avoids, split)
}

// towerLegacy is the legacy verdict of the tall dungeon guard, as bot.towerWhy computes it.
func towerLegacy(w task.World, kit string) string {
	switch {
	case task.TowerLevelWhy(w) == "":
		return "go"
	case kit == ladder.KitNone:
		return "lower"
	}
	return "skip"
}

func TestParityTooHard(t *testing.T) {
	l, _, _ := newLadder(t, nil)
	rt := env.DefaultTuning().Route
	n, blocked, split := 0, 0, 0
	for _, c := range classes {
		for _, diff := range []string{"normal", "nightmare"} {
			for clvl := 1; clvl <= 40; clvl++ {
				for _, merc := range []string{"alive", "dead", "none"} {
					for _, kit := range []string{ladder.KitOK, ladder.KitThin, ladder.KitNone} {
						w := task.World{Level: clvl, MaxLife: 10000, Difficulty: diff,
							TowerMinLevel: rt.TowerMinLevel, TowerMinLevelMerc: rt.TowerMinLevelMerc, TowerNeedsMerc: rt.TowerNeedsMerc != 0,
							TowerMinLife: rt.TowerMinLife, TowerMinLevelNoLife: rt.TowerMinLevelNoLife}
						switch merc {
						case "alive":
							w.MercKnown, w.MercAlive = true, true
						case "dead":
							w.MercKnown = true
						}
						if task.MercState(w) != merc {
							t.Fatalf("merc state %q != %q", task.MercState(w), merc)
						}
						want := towerLegacy(w, kit)
						h := hero(c, 100)
						h.Diff = d2.Normal
						if diff == "nightmare" {
							h.Diff = d2.Nightmare
						}
						got, err := l.TooHard(h, clvl, task.MercState(w), kit, want)
						if err != nil {
							t.Fatal(err)
						}
						if got != want {
							t.Fatalf("%v %s clvl %d merc %s kit %s: ladder %q, legacy %q", c, diff, clvl, merc, kit, got, want)
						}
						if want != "go" {
							blocked++
						}
						o := decide.Obs{Num: map[string]float64{"clvl": float64(clvl)}, Cat: map[string]string{"merc": merc, "kit": kit, "areaclass": "field"}}
						if def, err := l.CatalogueDefault(ladder.PointTooHard, h, o); err != nil {
							t.Fatal(err)
						} else if def != want {
							if diff == "normal" {
								t.Errorf("%v clvl %d merc %s kit %s: legacy %s, default %s", c, clvl, merc, kit, want, def)
							}
							split++
						}
						n++
					}
				}
			}
		}
	}
	t.Logf("t0.area.toohard parity grid: %d states, %d guarded, catalogue default differs at %d, all in Nightmare (the guard is off there and the call site does not ask)", n, blocked, split)
}

func TestParityMercenary(t *testing.T) {
	l, _, _ := newLadder(t, nil)
	floor := env.DefaultTuning().Bot.MercHireMinGold
	n, split := 0, 0
	for _, c := range classes {
		for _, merc := range []string{"alive", "dead", "none"} {
			for _, kit := range []string{ladder.KitOK, ladder.KitThin, ladder.KitNone} {
				for gold := 0; gold <= 3000; gold++ {
					want := merc != "alive" && gold >= floor
					h := hero(c, gold)
					got, err := l.MercHire(h, merc, kit, want)
					if err != nil {
						t.Fatal(err)
					}
					if got != want {
						t.Fatalf("%v merc %s kit %s gold %d: ladder %v, legacy %v", c, merc, kit, gold, got, want)
					}
					o := decide.Obs{Cat: map[string]string{"merc": merc, "kit": kit}}
					if defIs(t, l, ladder.PointMerc, h, o, "hire") != want {
						split++
					}
					n++
				}
			}
		}
	}
	if split != 0 {
		t.Errorf("catalogue default differs from the hire gate at %d states", split)
	}
	t.Logf("t2.mercenary parity grid: %d states", n)
}

// TestLearnedRuleOverridesLegacyT0: a scripted player teaches the opposite answer at each T0 point, for two builds.
func TestLearnedRuleOverridesLegacyT0(t *testing.T) {
	pts := []struct {
		point, keyHas, teach, legacy string
		ask                          func(l *ladder.Ladder, h ladder.Hero, legacy string) (string, error)
	}{
		{ladder.PointDrink, "held=hp", "portal", "fight", func(l *ladder.Ladder, h ladder.Hero, lg string) (string, error) {
			return l.DrinkChoice(h, 40, ladder.HeldHP, ladder.KitOK, false, lg)
		}},
		{ladder.PointUnique, "kit=OK", "avoid", "fight", func(l *ladder.Ladder, h ladder.Hero, lg string) (string, error) {
			return l.UniqueFight(h, 60, 2, ladder.KitOK, lg)
		}},
		{ladder.PointTooHard, "merc=dead", "go", "skip", func(l *ladder.Ladder, h ladder.Hero, lg string) (string, error) {
			return l.TooHard(h, 10, "dead", ladder.KitOK, lg)
		}},
		{ladder.PointMerc, "merc=none", "wait", "hire", func(l *ladder.Ladder, h ladder.Hero, lg string) (string, error) {
			ok, err := l.MercHire(h, "none", ladder.KitOK, lg == "hire")
			if ok {
				return "hire", err
			}
			return "wait", err
		}},
	}
	for _, p := range pts {
		for _, c := range classes {
			src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"` + p.point + `","keyHas":"` + p.keyHas + `","answer":"` + p.teach + `","confirm":"yes","delaySec":1}]}`))
			if err != nil {
				t.Fatal(err)
			}
			l, clk, _ := newLadder(t, src)
			h := hero(c, 500)
			for i := 0; i < 6; i++ {
				if _, err := p.ask(l, h, p.legacy); err != nil {
					t.Fatalf("%s %v: %v", p.point, c, err)
				}
				clk.Advance(5 * time.Second)
				l.Pump()
			}
			got, err := p.ask(l, h, p.legacy)
			if err != nil {
				t.Fatal(err)
			}
			if got != p.teach {
				t.Errorf("%s %v: got %q, want the taught %q over legacy %q (stats %+v)", p.point, c, got, p.teach, p.legacy, l.Stats())
			}
			if l.Stats().ByRule == 0 {
				t.Errorf("%s %v: no decision came from a rule", p.point, c)
			}
		}
	}
}
