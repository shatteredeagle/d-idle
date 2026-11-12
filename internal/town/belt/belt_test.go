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

func TestPlace(t *testing.T) {
	P := func(box int, k Kind) Placed { return Placed{box, k} }
	cases := []struct {
		name  string
		slots int
		items []Placed
		k     Kind
		box   int
		ok    bool
	}{
		{"empty belt takes first column", 4, nil, HP, 0, true},
		{"none-belt full", 4, []Placed{P(0, HP), P(1, HP), P(2, HP), P(3, MP)}, HP, 0, false},
		{"matching column lowest free box", 8, []Placed{P(0, MP), P(1, HP)}, HP, 5, true},
		{"matching column full goes to empty column", 8, []Placed{P(0, MP), P(1, HP), P(5, HP)}, HP, 2, true},
		{"mixed column full, empty column used", 4, []Placed{P(0, HP)}, HP, 1, true},
		{"mixed column full, no empty column", 4, []Placed{P(0, HP), P(1, MP), P(2, MP), P(3, MP)}, HP, 0, false},
		{"first matching column wins", 12, []Placed{P(0, MP), P(1, HP), P(2, HP), P(6, HP)}, HP, 5, true},
		{"rejuvenation column", 8, []Placed{P(0, HP), P(1, RV)}, RV, 5, true},
		{"column kind is its bottom item", 8, []Placed{P(0, None), P(1, HP)}, HP, 5, true},
		{"sash 8 boxes full for kind", 8, []Placed{P(0, HP), P(4, HP), P(1, HP), P(5, HP), P(2, HP), P(6, HP), P(3, HP), P(7, HP)}, HP, 0, false},
		{"plated 16 boxes, fourth row", 16, []Placed{P(0, HP), P(4, HP), P(8, HP)}, HP, 12, true},
		{"not a potion", 16, nil, None, 0, false},
	}
	for _, c := range cases {
		box, ok := Place(c.slots, c.items, c.k)
		if ok != c.ok || ok && box != c.box {
			t.Errorf("%s: got (%d,%v) want (%d,%v)", c.name, box, ok, c.box, c.ok)
		}
	}
}

// FIXBELT2: PlaceFor keeps an empty column back for each planned kind that has no column yet.
func TestPlaceFor(t *testing.T) {
	P := func(box int, k Kind) Placed { return Placed{box, k} }
	band := Band{Cols: [Columns]Kind{HP, HP, MP, MP}}
	cases := []struct {
		name  string
		slots int
		items []Placed
		k     Kind
		box   int
		ok    bool
	}{
		{"empty belt, first healing", 8, nil, HP, 0, true},
		{"mana never has a column: it may open one", 8, []Placed{P(0, HP), P(4, HP), P(1, HP), P(5, HP), P(2, HP)}, MP, 3, true},
		{"third healing column is held back for mana", 8, []Placed{P(0, HP), P(4, HP), P(1, HP), P(5, HP), P(2, HP), P(6, HP)}, HP, 0, false},
		{"healing joins its own column", 8, []Placed{P(0, HP), P(1, HP), P(5, HP)}, HP, 4, true},
		{"mana column exists but is not yet there: the last column is kept for it (layout rule)", 8, []Placed{P(0, HP), P(4, HP), P(1, HP), P(5, HP), P(2, MP)}, HP, 0, false},
		{"plan columns full: a third mana column is refused (fixes 6 mana / 0 healing)", 8, []Placed{P(0, MP), P(4, MP), P(1, MP), P(5, MP)}, MP, 0, false},
		{"mana columns full, healing still gets the empty one", 8, []Placed{P(0, MP), P(4, MP), P(1, MP), P(5, MP)}, HP, 2, true},
		{"LOG SIGNATURE: sash, hp x3 columns, mana column full, nothing fits", 8, []Placed{P(0, HP), P(4, HP), P(1, HP), P(5, HP), P(3, MP), P(7, MP), P(2, HP)}, MP, 0, false},
	}
	for _, c := range cases {
		box, ok := PlaceFor(c.slots, c.items, c.k, band)
		if ok != c.ok || ok && box != c.box {
			t.Errorf("%s: got (%d,%v) want (%d,%v)", c.name, box, ok, c.box, c.ok)
		}
	}
}

// docs/beltfix.md: the sandbox state, six mana potions in boxes 0,1,2,4,5,6 and column 3 empty, with a Paladin
// (three healing columns, one mana) and a Sorceress (two and two). Slots must not read 4 and the surplus mana column
// is offered to the healing potion, top first.
func TestSurplusMana(t *testing.T) {
	P := func(box int, k Kind) Placed { return Placed{box, k} }
	items := []Placed{P(0, MP), P(1, MP), P(2, MP), P(4, MP), P(5, MP), P(6, MP)}
	for _, tc := range []struct {
		name string
		band Band
		fits bool // healing placeable without repair
	}{
		{"paladin", Band{Cols: [Columns]Kind{HP, HP, HP, MP}}, true},
		{"sorceress", Band{Cols: [Columns]Kind{HP, HP, MP, MP}}, true},
	} {
		if _, ok := PlaceFor(8, items, HP, tc.band); ok != tc.fits {
			t.Errorf("%s: PlaceFor %v", tc.name, ok)
		}
	}
	// column 3 taken by one healing potion: the second healing column needs a mana column freed
	full := append(append([]Placed{}, items...), P(3, HP), P(7, HP))
	pal := Band{Cols: [Columns]Kind{HP, HP, HP, MP}}
	boxes, from, ok := Surplus(8, full, HP, pal)
	if !ok || from != MP || len(boxes) != 2 || boxes[0] != 6 || boxes[1] != 2 {
		t.Fatalf("paladin surplus: %v %v %v", boxes, from, ok)
	}
	// the Sorceress plans two mana columns: exactly one is surplus
	sor := Band{Cols: [Columns]Kind{HP, HP, MP, MP}}
	if boxes, from, ok = Surplus(8, full, HP, sor); !ok || from != MP || len(boxes) != 2 {
		t.Fatalf("sorceress surplus: %v %v %v", boxes, from, ok)
	}
	// after the repair the layout is stable: mana may not come back, healing fits
	left := []Placed{P(0, MP), P(4, MP), P(1, MP), P(5, MP), P(3, HP), P(7, HP)}
	if _, ok := PlaceFor(8, left, MP, sor); ok {
		t.Error("mana re-entered the freed column")
	}
	if _, _, ok := Surplus(8, left, HP, sor); ok {
		t.Error("a belt at its plan offers a surplus column")
	}
	// all-mana by the player's choice: nothing to repair
	allMana := Band{Cols: [Columns]Kind{MP, MP, MP, MP}}
	if _, _, ok := Surplus(4, []Placed{P(0, MP), P(1, MP), P(2, MP), P(3, MP)}, HP, allMana); ok {
		t.Error("an all-mana plan was repaired")
	}
}
