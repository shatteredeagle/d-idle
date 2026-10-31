package leveler

import (
	"d2lod-bot/internal/gear/geartest"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/gear"
)

func cand(id uint32, name string, score float64, usable bool, slots ...int) Cand {
	return Cand{Unit: d2.Unit{ID: id}, Name: name, Score: score, Usable: usable, Slots: slots}
}

func TestChooseGearFillsEmptySlotsAndRespectsMargin(t *testing.T) {
	eq := map[int]Cand{d2.BodyHead: cand(1, "cap", 10, true, d2.BodyHead)}
	pack := []Cand{
		cand(2, "helm", 10.5, true, d2.BodyHead), // inside the margin: no churn
		cand(3, "gloves", 4, true, d2.BodyGloves),
		cand(4, "boots", 3, false, d2.BodyFeet), // not usable
	}
	sw := ChooseGear(eq, pack)
	if len(sw) != 1 || sw[0].Slot != d2.BodyGloves {
		t.Fatalf("swaps %v", sw)
	}
	pack[0].Score = 20
	sw = ChooseGear(eq, pack)
	if len(sw) != 2 || sw[0].Slot != d2.BodyHead || sw[0].Item.ID() != 2 {
		t.Fatalf("swaps %v", sw)
	}
}

func (c Cand) ID() uint32 { return c.Unit.ID }

func TestChooseGearTwoHandersAndShields(t *testing.T) {
	staff := Cand{Unit: d2.Unit{ID: 5}, Name: "staff", Score: 30, Usable: true, Slots: []int{d2.BodyRHand}, TwoHanded: true}
	sword := cand(6, "sword", 12, true, d2.BodyRHand)
	shield := cand(7, "shield", 15, true, d2.BodyLHand)
	// nothing worn: the staff wins the hand, the shield is skipped because the staff is two-handed
	sw := ChooseGear(map[int]Cand{}, []Cand{staff, sword, shield})
	if len(sw) != 1 || sw[0].Item.ID() != 5 {
		t.Fatalf("swaps %v", sw)
	}
	// a shield is worn: the two-hander is never planned (it would need an unequip), the sword is
	sw = ChooseGear(map[int]Cand{d2.BodyLHand: shield}, []Cand{staff, sword})
	if len(sw) != 1 || sw[0].Item.ID() != 6 {
		t.Fatalf("swaps %v", sw)
	}
}

func TestChooseGearRingsFillBothSlotsThenReplaceTheWeaker(t *testing.T) {
	r := func(id uint32, s float64) Cand {
		return cand(id, "ring", s, true, d2.BodyRRing, d2.BodyLRing)
	}
	sw := ChooseGear(map[int]Cand{}, []Cand{r(1, 10), r(2, 8), r(3, 6)})
	if len(sw) != 2 || sw[0].Slot == sw[1].Slot {
		t.Fatalf("empty slots: %v", sw)
	}
	eq := map[int]Cand{d2.BodyRRing: r(8, 5), d2.BodyLRing: r(9, 9)}
	sw = ChooseGear(eq, []Cand{r(1, 10)})
	if len(sw) != 1 || sw[0].Slot != d2.BodyRRing { // the 5 gives way, not the 9
		t.Fatalf("replace the weaker: %v", sw)
	}
}

func TestPlanBuysPicksBestValueWithinBudgetAndCountsPackAsWorn(t *testing.T) {
	armor := cand(1, "armor", 20, true, d2.BodyTorso)
	armor.Price = 400
	cap1 := cand(2, "cap", 6, true, d2.BodyHead)
	cap1.Price = 40
	cap2 := cand(3, "helm", 8, true, d2.BodyHead)
	cap2.Price = 300
	got := PlanBuys(map[int]Cand{}, nil, []Cand{armor, cap1, cap2}, 500)
	// armor: gain 20 for 400 (0.05/gold), cap: 6 for 40 (0.15/gold) first; then the budget (460) cannot also fit armor+helm
	// better than... cap first, armor second (360 left cannot afford armor at 400 -> helm 300 is not better than the cap by the margin)
	if len(got) == 0 || got[0].ID() != 2 {
		t.Fatalf("buys %v", got)
	}
	for _, g := range got {
		if g.ID() == 3 {
			t.Fatalf("the helm replaces the cap bought a moment ago for 2 points: %v", got)
		}
	}
	// a better cap already in the pack: no head purchase
	pack := []Cand{cand(9, "pack cap", 9, true, d2.BodyHead)}
	got = PlanBuys(map[int]Cand{}, pack, []Cand{cap1, cap2}, 500)
	if len(got) != 0 {
		t.Fatalf("bought %v although the pack holds a better cap", got)
	}
}

func TestStatValueWeights(t *testing.T) {
	st := d2.Stats{uint32(stAllSkills): 1, uint32(stFireRes): 20, uint32(stMaxLife): 20 * 256}
	if got := statValue(st); got != 40+20*1.2+20*0.6 {
		t.Fatalf("value %.2f", got)
	}
}

func TestResistValueUsesWornTotalAndDifficulty(t *testing.T) {
	st := d2.Stats{uint32(stFireRes): 30}
	if got := statValueCtx(st, scoreCtx{res: map[int]int{stFireRes: 0}}); got != 30*wRes {
		t.Fatalf("empty: %.2f", got)
	}
	capped := statValueCtx(st, scoreCtx{res: map[int]int{stFireRes: 80}})
	if capped >= 30*wRes/2 {
		t.Fatalf("capped resist still worth %.2f", capped)
	}
	// Hell has a -100 penalty: 80 raw is far from the cap, so the same points count in full.
	hell := statValueCtx(st, scoreCtx{res: map[int]int{stFireRes: 80}, diff: 2})
	if hell != 30*wRes {
		t.Fatalf("hell %.2f", hell)
	}
}

func TestFCRBreakpointValue(t *testing.T) {
	st := d2.Stats{uint32(stFCR): 10}
	ctx := scoreCtx{class: "Sorceress", hasCtx: true}
	cross := statValueCtx(st, ctx) // 0 -> 10 crosses the 9 breakpoint
	ctx.fcr = 10
	miss := statValueCtx(st, ctx) // 10 -> 20 crosses 20
	ctx.fcr = 21
	none := statValueCtx(st, ctx) // 21 -> 31 crosses nothing
	if !(cross > none && miss > none) || none != 10*wFCR {
		t.Fatalf("cross %.1f miss %.1f none %.1f", cross, miss, none)
	}
}

func TestEtherealUsesGearFactor(t *testing.T) {
	b := &data.ItemBase{Kind: data.KindArmor, MinAC: 100, MaxAC: 100}
	plain := scoreOf(nil, Prefs{}, b, &d2.Item{}, nil, scoreCtx{})
	eth := scoreOf(nil, Prefs{}, b, &d2.Item{Flags: d2.FlagEthereal}, nil, scoreCtx{})
	if eth != plain*gear.EtherealFactor() {
		t.Fatalf("eth %.1f plain %.1f", eth, plain)
	}
}

func TestRunewordUnreadGetsFlatScore(t *testing.T) {
	b := &data.ItemBase{Kind: data.KindArmor}
	base := scoreOf(nil, Prefs{}, b, &d2.Item{}, nil, scoreCtx{})
	rw := scoreOf(nil, Prefs{}, b, &d2.Item{Flags: d2.FlagRuneword}, nil, scoreCtx{})
	if rw-base != wRuneUnread {
		t.Fatalf("rw %.1f base %.1f", rw, base)
	}
}

// Str and Dex requirements gate armor, shields and claws (V: armor.txt reqstr and weapons.txt reqdex of the real
// install, the d2tables armor.json); needs /tmp/d2mpq, run on the worker.
func TestCandidateRefusesUnmetStrDex(t *testing.T) {
	geartest.Need(t)
	db := realDB(t)
	idx := map[string]int{}
	for i, b := range db.Items {
		if _, dup := idx[b.Code]; !dup {
			idx[b.Code] = i
		}
	}
	check := func(code string, str, dex int, want bool, why string) {
		t.Helper()
		p := &d2.Player{Class: d2.Paladin, Level: 99, Strength: str, Dexterity: dex}
		u := d2.Unit{Class: idx[code], Item: &d2.Item{Quality: d2.QualityNormal, Where: d2.ContInventory}}
		if c := Candidate(db, p, u, nil); c.Usable != want {
			t.Errorf("%s str %d dex %d: usable %v (%s), want %v", code, str, dex, c.Usable, c.Why, want)
		}
	}
	for _, code := range gear.ArmorCodes() {
		a, _ := gear.ArmorByCode(code)
		if _, ok := idx[code]; !ok || a.ReqStr == 0 || a.Type == "pelt" || a.Type == "head" || a.Type == "phlm" || a.Type == "ashd" || a.Type == "circ" {
			continue // class-restricted rows are refused for a Paladin whatever the stats
		}
		check(code, a.ReqStr-1, 999, false, "str")
		check(code, a.ReqStr, 0, true, "str")
	}
	check("pab", 86, 0, true, "Sacred Targe is a Paladin item")
	check("pab", 85, 0, false, "Sacred Targe")
	check("xtp", 54, 0, false, "Mage Plate")
	check("xtp", 55, 0, true, "Mage Plate")
}
