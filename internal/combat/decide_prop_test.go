package combat

import (
	"math/rand/v2"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

// Property tests on the pure survival decider (roadmap T3, docs/design/proptest-t3.md). Fixed seeds, no clock.

const propCases = 20000

// propBuild shapes the random inputs like one build: its life and mana pools and whether its portal guard reads
// damage terms (bot.portalGuard 1). Labels only; Decide itself takes no class.
type propBuild struct {
	name               string
	lifeMax, manaMax   int
	guardDamage        bool
	crowdAbandonQuestN int
}

var propBuilds = []propBuild{
	{name: "sorc", lifeMax: 400, manaMax: 600},
	{name: "pal", lifeMax: 1200, manaMax: 120, guardDamage: true, crowdAbandonQuestN: 40},
}

func randTune(r *rand.Rand, b propBuild) env.CombatTuning {
	t := env.DefaultCombatTuning()
	// ranges are the knob min..max of internal/env/tuning_combat.go
	t.ChickenPct = 15 + r.IntN(31)
	t.ChickenHellPct = 20 + r.IntN(31)
	t.DrinkRejuvPct = 20 + r.IntN(41)
	t.DrinkLifePct = 35 + r.IntN(46)
	t.DrinkManaPct = 5 + r.IntN(56)
	t.TownPotionMin = r.IntN(5)
	t.CrowdN = r.IntN(21)
	t.CrowdPortalPct = 40 + r.IntN(51)
	t.CrowdDrinkPct = 40 + r.IntN(46)
	t.CrowdAbandonN = r.IntN(31)
	t.CrowdAbandonQuestN = b.crowdAbandonQuestN
	t.CrowdFarN = r.IntN(21)
	t.CrowdFarPct = 30 + r.IntN(61)
	return t
}

func randBelt(r *rand.Rand) Belt {
	var b Belt
	for i := range b.Cols {
		k := PotionKind(r.IntN(4)) // PotionNone, healing, mana, rejuv
		if k == PotionNone {
			continue
		}
		b.Cols[i] = BeltColumn{Kind: k, Tier: 1 + r.IntN(5), Count: 1 + r.IntN(4)}
	}
	return b
}

func randSafety(r *rand.Rand, b propBuild) SafetyInput {
	in := SafetyInput{
		InTown:     r.IntN(10) == 0,
		Max:        1 + r.IntN(b.lifeMax),
		MMax:       r.IntN(b.manaMax + 1),
		Diff:       d2.Difficulty(r.IntN(3)),
		Belt:       randBelt(r),
		InFight:    r.IntN(2) == 0,
		HasPortal:  r.IntN(3) != 0,
		Walking:    r.IntN(2) == 0,
		CanRetreat: r.IntN(2) == 0,
		HealReady:  r.IntN(3) != 0,
		ManaReady:  r.IntN(3) != 0,
		RejuvReady: r.IntN(3) != 0,
		Arrival:    r.IntN(15) == 0,
		Quest:      r.IntN(4) == 0,
		Tune:       randTune(r, b),
	}
	in.Life = r.IntN(in.Max + 1)
	in.Mana = r.IntN(in.MMax + 1)
	in.Adjacent = r.IntN(6)
	in.Crowd = in.Adjacent + r.IntN(30)
	in.CrowdFar = in.Crowd + r.IntN(10)
	if b.guardDamage {
		in.CastGuard = resource.CastGuard{Mode: 1, ExposureSec: 2.5, LifeFrac: 0.5, Incoming: float64(r.IntN(60) * in.Adjacent), DmgKnown: r.IntN(4) != 0}
	}
	return in
}

// forEachInput runs f on propCases random inputs per build, each build with its own fixed seed.
func forEachInput(t *testing.T, f func(t *testing.T, b propBuild, in SafetyInput)) {
	t.Helper()
	for i, b := range propBuilds {
		t.Run(b.name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(uint64(980+i), 3))
			for range propCases {
				f(t, b, randSafety(r, b))
				if t.Failed() {
					return
				}
			}
		})
	}
}

// Every action Decide asks for can be carried out with what the hero has: a drink names a ready potion that is in the
// belt, a portal needs a scroll, a retreat needs a way out, and nothing fires in town or for a dead hero.
func TestPropDecideActionIsFeasible(t *testing.T) {
	forEachInput(t, func(t *testing.T, _ propBuild, in SafetyInput) {
		a := Decide(in)
		ok := true
		switch a {
		case DrinkLife:
			ok = in.Belt.Count(PotionHealing) > 0 && in.HealReady
		case DrinkMana:
			ok = in.Belt.Count(PotionMana) > 0 && in.ManaReady
		case DrinkRejuv:
			ok = in.Belt.Count(PotionRejuv) > 0 && in.RejuvReady
		case TownPortal, CrowdAbandon:
			ok = in.HasPortal
		case CrowdRetreat:
			ok = in.CanRetreat
		}
		if !ok {
			t.Errorf("%v not feasible for %+v", a, in)
		}
		if (in.InTown || in.Life <= 0 || in.Max <= 0) && a != SafetyNone {
			t.Errorf("%v in town or dead: %+v", a, in)
		}
	})
}

// A reflex always wins: at or below the chicken line (and no arrival crowd to leave) the answer is never "nothing", and
// it is a rejuvenation, a healing potion out of a fight, or leaving the game.
func TestPropDecideChickenLineAlwaysActs(t *testing.T) {
	forEachInput(t, func(t *testing.T, _ propBuild, in SafetyInput) {
		if in.InTown || in.Life <= 0 || in.Arrival {
			return
		}
		ch := in.Tune.ChickenPct
		if in.Diff == d2.Hell {
			ch = in.Tune.ChickenHellPct
		}
		if pct(in.Life, in.Max) > ch {
			return
		}
		switch a := Decide(in); a {
		case DrinkRejuv, Chicken:
		case DrinkLife:
			if in.InFight {
				t.Errorf("healing potion at the chicken line in a fight: %+v", in)
			}
		default:
			t.Errorf("%v at the chicken line: %+v", a, in)
		}
	})
}

// A town portal is never cast while it is too exposed in a fight (the cast guard turns it into a drink or a chicken).
func TestPropDecideNoExposedPortalCast(t *testing.T) {
	forEachInput(t, func(t *testing.T, _ propBuild, in SafetyInput) {
		if !in.InFight {
			return
		}
		if blocked, _ := in.castBlocked(); blocked && Decide(in) == TownPortal {
			t.Errorf("exposed portal cast: %+v", in)
		}
	})
}

// More potions never make the verdict worse: with one more potion in the belt (of any kind, readiness unchanged), a
// hero that was not leaving the game does not start leaving it.
func TestPropDecideMorePotionsNeverChicken(t *testing.T) {
	forEachInput(t, func(t *testing.T, _ propBuild, in SafetyInput) {
		before := Decide(in)
		if before == Chicken {
			return
		}
		for col := range in.Belt.Cols {
			for _, k := range []PotionKind{PotionHealing, PotionMana, PotionRejuv} {
				more := in
				c := &more.Belt.Cols[col]
				switch {
				case c.Count == 0:
					*c = BeltColumn{Kind: k, Tier: 1, Count: 1}
				case c.Kind == k:
					c.Count++
				default:
					continue
				}
				if after := Decide(more); after == Chicken {
					t.Errorf("adding %v to column %d turns %v into chicken: %+v", k, col, before, in)
				}
			}
		}
	})
}

// Decide is pure: the same input gives the same answer.
func TestPropDecideDeterministic(t *testing.T) {
	forEachInput(t, func(t *testing.T, _ propBuild, in SafetyInput) {
		if a, b := Decide(in), Decide(in); a != b {
			t.Errorf("Decide gave %v then %v for %+v", a, b, in)
		}
	})
}
