package belt

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/resource"
)

func TestPlan(t *testing.T) {
	if b := PlanFor(1); b.Cols != [Columns]Kind{HP, HP, MP, HP} || b.MaxHPTier != 2 {
		t.Fatalf("%+v", b)
	}
	if b := PlanFor(40); b.Cols[3] != RV || b.MaxHPTier != 5 {
		t.Fatalf("%+v", b)
	}
	if PlanFor(120).ClvlMax != 99 || PlanFor(0).ClvlMin != 1 {
		t.Fatal("clamp")
	}
}

func TestWant(t *testing.T) {
	b := PlanFor(1)
	st := State{Slots: 4, Count: map[Kind]int{}}
	if w := b.Want(st, 100); w[HP] != 3 || w[MP] != 1 {
		t.Fatalf("empty 4-slot belt: %v", w)
	}
	st.Count[HP] = 3
	if w := b.Want(st, 100); w[HP] != 0 || w[MP] != 1 {
		t.Fatalf("%v", w)
	}
	st = State{Slots: 8, Count: map[Kind]int{HP: 2, MP: 2}}
	if w := b.Want(st, 50); w[HP] != 1 || w[MP] != 0 {
		t.Fatalf("%v", w)
	}
	st = State{Slots: 4, Count: map[Kind]int{HP: 1}, Other: 2}
	if w := b.Want(st, 100); w[HP]+w[MP] != 1 {
		t.Fatalf("%v", w)
	}
	if b.FillPct(State{Slots: 4, Count: map[Kind]int{HP: 1}}, HP) != 33 {
		t.Fatal("FillPct")
	}
}

func TestPlanForClassFollowsResourcePlan(t *testing.T) {
	count := func(b Band, k Kind) int {
		n := 0
		for _, c := range b.Cols {
			if c == k {
				n++
			}
		}
		return n
	}
	if b := PlanForClass(d2.Sorceress, 5); count(b, MP) != 2 || count(b, HP) != 2 {
		t.Errorf("sorceress clvl 5: %v", b.Cols)
	}
	if b := PlanForClass(d2.Sorceress, 35); count(b, MP) != 1 || count(b, RV) != 1 {
		t.Errorf("sorceress clvl 35: %v", b.Cols)
	}
	if b := PlanForClass(d2.Barbarian, 5); count(b, MP) != 0 {
		t.Errorf("barbarian: %v", b.Cols)
	}
	if b := PlanForClass(d2.Sorceress, 5); b.MaxHPTier != 2 {
		t.Errorf("tier cap lost: %+v", b)
	}
}

func TestPlanForBuildNormalUnchanged(t *testing.T) {
	for _, c := range []d2.Class{d2.Sorceress, d2.Barbarian, d2.Paladin, d2.Necromancer} {
		for lvl := 1; lvl <= 99; lvl++ {
			for _, b := range []string{"", "fallback", "paladin-normal30"} {
				want := PlanFor(lvl)
				for i, k := range resource.PlanFor(c, lvl).Cols {
					want.Cols[i] = map[resource.Potion]Kind{resource.PotionLife: HP, resource.PotionMana: MP, resource.PotionRejuv: RV}[k]
				}
				if PlanForBuild(c, lvl, d2.Normal, b) != want || PlanForClass(c, lvl) != want {
					t.Fatalf("%v clvl %d build %q: Normal plan differs", c, lvl, b)
				}
			}
		}
	}
}

func TestPlanForBuildHellFollowsResource(t *testing.T) {
	for _, d := range []d2.Difficulty{d2.Nightmare, d2.Hell} {
		for lvl := 20; lvl <= 90; lvl += 5 {
			b := PlanForBuild(d2.Sorceress, lvl, d, "sorceress_blizzard_endgame90")
			rp := resource.PlanForBuild(d2.Sorceress, lvl, d, "sorceress_blizzard_endgame90")
			for i, k := range rp.Cols {
				want := map[resource.Potion]Kind{resource.PotionLife: HP, resource.PotionMana: MP, resource.PotionRejuv: RV}[k]
				if b.Cols[i] != want {
					t.Fatalf("diff %v clvl %d col %d: %v want %v", d, lvl, i, b.Cols[i], want)
				}
			}
		}
	}
}
