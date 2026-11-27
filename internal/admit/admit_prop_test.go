package admit

import (
	"math/rand/v2"
	"testing"
)

// Property tests on the pure admission function (roadmap T3, docs/design/proptest-t3.md). Fixed seeds, no clock. Two builds as
// input shapes: a melee one (big life, no town portal half the time) and a caster one (small life); the function takes the
// archetype as data, no class.

const propCases = 20000

var propBuilds = []struct {
	name     string
	archetyp string
	lifeMax  int
}{{"pal", "melee", 400}, {"sorc", "caster", 180}}

var allPurposes = []Purpose{QuestWalk, Farm, Level, Boss, CorpseRun, Trip}

func randView(r *rand.Rand, arch string, lifeMax int) View {
	maxLife := 1 + r.IntN(lifeMax)
	v := View{
		Archetype: arch, Normal: r.IntN(5) != 0, Level: 1 + r.IntN(30),
		MaxLife: maxLife, Life: r.IntN(maxLife + 1), Healing: r.IntN(7), HasTP: r.IntN(2) == 0,
		Gold: r.IntN(3000), InTown: r.IntN(2) == 0, Area: r.IntN(12), FirstField: 2, Fields: []int{2, 3, 4, 5, 6},
		Try:        r.IntN(8) == 0,
		IsTown:     func(a int) bool { return a == 1 || a == 40 },
		MLvl:       func(a int) int { return a * 2 },
		Knobs:      map[string]int{"campaign.denMinLevelMelee": 3 + r.IntN(8), "campaign.denMinLevelCaster": 3 + r.IntN(8)},
		PotionGold: 50, TPGold: 100,
	}
	return v
}

func TestPropAdmission(t *testing.T) {
	rs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for bi, b := range propBuilds {
		r := rand.New(rand.NewPCG(310, uint64(bi)))
		for i := 0; i < propCases; i++ {
			v := randView(r, b.archetyp, b.lifeMax)
			area := r.IntN(12)
			p := allPurposes[r.IntN(len(allPurposes))]
			ve, vl := rs.MayEnter(v, area, p), rs.MayLeaveTown(v, p)

			// A reflex is never refused: a corpse run, a trip home, the owner's try anyway, any difficulty but Normal.
			if p == CorpseRun || p == Trip || v.Try || !v.Normal {
				if ve.Refused || vl.Refused {
					t.Fatalf("%s: reflex refused: purpose %s try %v normal %v: %+v %+v", b.name, p, v.Try, v.Normal, ve, vl)
				}
			}
			// A refusal is data: a closed need, its wake, the row and words.
			for _, x := range []Verdict{ve, vl} {
				if x.Refused && (Wake[x.Need] == "" || x.Wake != Wake[x.Need] || x.Row == "" || x.Why == "") {
					t.Fatalf("%s: malformed refusal %+v", b.name, x)
				}
				if !x.Refused && x != Allow {
					t.Fatalf("%s: an open verdict carries data: %+v", b.name, x)
				}
			}
			// Deterministic: the same view answers the same.
			if ve2, vl2 := rs.MayEnter(v, area, p), rs.MayLeaveTown(v, p); ve2 != ve || vl2 != vl {
				t.Fatalf("%s: not deterministic", b.name)
			}
			// The town is always open to enter.
			if a := 1; rs.MayEnter(v, a, p).Refused {
				t.Fatalf("%s: town refused", b.name)
			}
			// One more healing potion, or a portal, never turns an open verdict into a refusal.
			more := v
			more.Healing++
			if !ve.Refused && rs.MayEnter(more, area, p).Refused || !vl.Refused && rs.MayLeaveTown(more, p).Refused {
				t.Fatalf("%s: one more potion closed a door: %+v area %d %s", b.name, v, area, p)
			}
			tp := v
			tp.HasTP = true
			if !ve.Refused && rs.MayEnter(tp, area, p).Refused || !vl.Refused && rs.MayLeaveTown(tp, p).Refused {
				t.Fatalf("%s: a portal closed a door: %+v area %d %s", b.name, v, area, p)
			}
		}
	}
}
