package loot_test

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

func newLooter(t *testing.T) (*loot.Looter, *sim.Rig) {
	r := sim.NewRig(t)
	l := loot.New(r.Env, r.Mv)
	l.UseSource(r.W.Snapshot, r.W.UnitStats)
	return l, r
}

func TestStarterParses(t *testing.T) {
	p := loot.Starter()
	if len(p.Rules) < 15 {
		t.Fatalf("rules %d", len(p.Rules))
	}
	if _, err := loot.ParsePickit(strings.NewReader(`{"rules":[{"name":"x","action":"bogus","tier":"S"}]}`)); err == nil {
		t.Fatal("bad action accepted")
	}
	if _, err := loot.ParsePickit(strings.NewReader(`{"rules":[{"name":"x","action":"keep","tier":"S","typo":1}]}`)); err == nil {
		t.Fatal("unknown field accepted")
	}
}

func TestStarterTakesActQuestItems(t *testing.T) {
	var codes []string
	for _, r := range loot.Starter().Rules {
		if r.Name == "quest items by code" {
			codes = r.Match.Codes
		}
	}
	got := map[string]bool{}
	for _, c := range codes {
		got[c] = true
	}
	if len(codes) != len(loot.QuestItemCodes) {
		t.Fatalf("pickit has %d codes, QuestItemCodes %d", len(codes), len(loot.QuestItemCodes))
	}
	for _, c := range loot.QuestItemCodes {
		if !got[c] {
			t.Errorf("%s missing from the pickit rule", c)
		}
	}
}

func TestEvaluate(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Level = 10
	unit := func(code string, q int, flags uint32) d2.Unit {
		u := w.NewItem(code, q)
		u.Item.Flags = flags
		return u
	}
	cases := []struct {
		name string
		u    d2.Unit
		want loot.Action
		rule string
	}{
		{"rune", unit("r07", 2, d2.FlagIdentified), loot.Stash, "runes"},
		{"perfect gem", unit("gpv", 2, d2.FlagIdentified), loot.Stash, "top gems"},
		{"standard gem kept", unit("gsv", 2, d2.FlagIdentified), loot.Stash, "mid gems"},
		{"unique sword", unit("ssd", 7, d2.FlagIdentified), loot.Stash, "unique and set"},
		{"rejuv", unit("rvs", 2, d2.FlagIdentified), loot.Keep, "rejuvenation"},
		{"unid ring", unit("rin", 4, 0), loot.Keep, "jewelry to id"},
		{"magic sword sells", unit("ssd", 4, d2.FlagIdentified), loot.Sell, "magic and rare"},
		{"plain sword", unit("ssd", 2, d2.FlagIdentified), loot.Drop, "junk"},
		{"jewel", unit("jew", 4, 0), loot.Stash, "jewels"},
	}
	for _, c := range cases {
		v := l.EvaluateWith(c.u, loot.Ctx{CLvl: 10, Gold: 5000})
		if v.Action != c.want || v.Rule != c.rule {
			t.Errorf("%s: got %v rule %q, want %v %q", c.name, v.Action, v.Rule, c.want, c.rule)
		}
	}
	// needs: belt full of potions -> hp potion is not wanted; empty belt -> wanted.
	hp := unit("hp1", 2, d2.FlagIdentified)
	if v := l.EvaluateUnit(hp); !v.Wanted() || v.Rule != "healing and mana" {
		t.Errorf("empty belt: %v", v)
	}
	for i := 0; i < 4; i++ {
		w.Add("hp1", d2.ContBelt, i, 0)
	}
	if v := l.EvaluateUnit(hp); v.Rule == "healing and mana" {
		t.Errorf("full belt: %v", v)
	}
	// Stat conditions: good jewelry needs readable stats.
	ring := unit("rin", 6, d2.FlagIdentified)
	ring.Addr = 0xABCD
	if v := l.EvaluateWith(ring, loot.Ctx{CLvl: 10, Gold: 5000}); v.Rule != "magic and rare" {
		t.Errorf("no stats: %v", v)
	}
	if v := l.EvaluateWith(ring, loot.Ctx{CLvl: 10, Stats: d2.Stats{105: 10}}); v.Rule != "good jewelry" || v.Action != loot.Stash {
		t.Errorf("fcr ring: %v", v)
	}
	// The bare-Item contract entry still answers by quality.
	if v := l.Evaluate(d2.Item{Quality: 7, Flags: d2.FlagIdentified}); v.Action != loot.Stash {
		t.Errorf("Evaluate(Item): %v", v)
	}
}

func TestPickUpOrderAndFilters(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	dress(w)
	w.Player.Gold = 5000                      // not poor: poor mode would take the sword
	near := w.AddGround("hp1", 2, 6022, 4930) // potion, belt empty: wanted, prio 65
	rune := w.AddGround("r03", 2, 6030, 4930) // rune, farther but prio 90: first
	_ = w.AddGround("ssd", 2, 6021, 4931)     // plain sword: dropped by the pickit
	far := w.AddGround("r04", 2, 6100, 4930)  // outside radius
	_ = far
	rep, err := l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Picked) != 2 || rep.Picked[0].ID != rune.ID || rep.Picked[1].ID != near.ID {
		t.Fatalf("picked %+v skipped %+v", rep.Picked, rep.Skipped)
	}
	if len(w.Ground) != 2 {
		t.Fatalf("ground %d", len(w.Ground))
	}
	// The potion went to the belt, the rune to the pack.
	inv, _ := w.Snapshot()
	if len(inv.Inventory.Container(d2.ContBelt)) != 1 || len(inv.Inventory.Container(d2.ContInventory)) != 1 {
		t.Fatalf("belt %d pack %d", len(inv.Inventory.Container(d2.ContBelt)), len(inv.Inventory.Container(d2.ContInventory)))
	}
	if len(r.Events) != 2 {
		t.Fatalf("drop events %d", len(r.Events))
	}
}

func TestPickUpRetriesHoverAndCursor(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.NoHover = 1
	w.PickupToCursor = true
	it := w.AddGround("r05", 2, 6022, 4930)
	rep, err := l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil || len(rep.Picked) != 1 {
		t.Fatalf("%v %+v", err, rep)
	}
	s, _ := w.Snapshot()
	if s.Inventory.Cursor != nil {
		t.Fatal("item left on the cursor")
	}
	if u, ok := findID(s, it.ID); !ok || u.Item.Where != d2.ContInventory {
		t.Fatal("rune not in pack")
	}
	if s.UI.AnyPanelOpen() == false {
		t.Log("inventory closed again or never needed")
	}
}

func findID(s *d2.Snapshot, id uint32) (d2.Unit, bool) {
	for _, u := range s.Inventory.Items {
		if u.ID == id {
			return u, true
		}
	}
	return d2.Unit{}, false
}

func TestPickUpFullPackAndAbort(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			w.Add("gcv", d2.ContInventory, x, y)
		}
	}
	w.AddGround("ssd", 4, 6022, 4930) // magic sword: strictness 1 skips B anyway; use a rune
	w.AddGround("r01", 2, 6023, 4930)
	rep, err := l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Picked) != 0 || !rep.Full {
		t.Fatalf("full pack must pick nothing: %+v", rep)
	}
	calls := 0
	r2, err := func() (loot.PickReport, error) {
		w.Items = nil
		return l.PickUp(context.Background(), loot.PickOpts{Abort: func() bool { calls++; return true }})
	}()
	if err != nil || !r2.Aborted || len(r2.Picked) != 0 {
		t.Fatalf("abort: %+v %v", r2, err)
	}
}

func TestStrictness(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Gold = 10 // low-gold mode lets B tier in at strictness 1
	w.AddGround("amu", 6, 6022, 4930)
	rep, _ := l.PickUp(context.Background(), loot.PickOpts{StrictSet: true, Strictness: 3})
	if len(rep.Picked) != 0 {
		t.Fatalf("strict 3 picked %+v", rep.Picked)
	}
	rep, _ = l.PickUp(context.Background(), loot.PickOpts{StrictSet: true, Strictness: 0})
	if len(rep.Picked) != 1 {
		t.Fatalf("loose picked %+v / %+v", rep.Picked, rep.Skipped)
	}
}

func TestValuables(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Add("r10", d2.ContInventory, 0, 0)
	w.Add("hp1", d2.ContInventory, 1, 0)
	w.Add("tsc", d2.ContInventory, 2, 0)
	w.Add("gpv", d2.ContInventory, 3, 0)
	s, _ := w.Snapshot()
	rep := l.Valuables(s)
	if len(rep.Stash) != 2 || len(rep.Carry) != 0 {
		t.Fatalf("%+v", rep)
	}
	r.Env.Tune.Loot.CarryValuables = 1
	rep = l.Valuables(s)
	if len(rep.Stash) != 1 || len(rep.Carry) != 1 {
		t.Fatalf("carry 1: stash %d carry %d", len(rep.Stash), len(rep.Carry))
	}
}

func TestPickUpByNameTakesWhatThePickitDrops(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	dress(w)
	w.Player.Gold = 5000
	sword := w.AddGround("ssd", 2, 6021, 4931) // plain sword: dropped by the pickit
	rep, err := l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil || len(rep.Picked) != 0 {
		t.Fatalf("without a name: %+v %v", rep, err)
	}
	rep, err = l.PickUp(context.Background(), loot.PickOpts{Names: []string{"Short Sword"}})
	if err != nil || len(rep.Picked) != 1 || rep.Picked[0].ID != sword.ID {
		t.Fatalf("by name: %+v %v", rep, err)
	}
}

func TestPickUpNeverWearsQuestItems(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.AutoEquipQuest = true
	shaft := w.AddGround("msf", 2, 6021, 4931)
	top := w.AddGround("vip", 2, 6022, 4931)
	rep, err := l.PickUp(context.Background(), loot.PickOpts{Names: []string{"Shaft of the Horadric Staff", "Top of the Horadric Staff"}})
	if err != nil || len(rep.Picked) != 2 {
		t.Fatalf("%+v %v", rep, err)
	}
	s, _ := w.Snapshot()
	for _, id := range []uint32{shaft.ID, top.ID} {
		u, ok := findID(s, id)
		if !ok || u.Item.Where != d2.ContInventory {
			t.Fatalf("item %d not in the pack: %+v", id, u.Item)
		}
	}
	if w.UI[d2.UIInventory] != 0 {
		t.Fatal("pack left open")
	}
}

func TestRunewordBasesExactSockets(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Level = 10
	base := func(code string, q int, sockets int) (d2.Unit, loot.Ctx) {
		u := w.NewItem(code, q)
		return u, loot.Ctx{CLvl: 10, Gold: 5000, Stats: d2.Stats{194: int32(sockets)}}
	}
	cases := []struct {
		name    string
		code    string
		sockets int
		want    loot.Action
		rule    string
	}{
		{"quilted 2 (Stealth)", "qui", 2, loot.Stash, "runeword base 2"},
		{"cap 2 (Lore)", "cap", 2, loot.Stash, "runeword base 2"},
		{"short staff 2 (Leaf)", "sst", 2, loot.Stash, "runeword base 2"},
		{"large shield 3 (Pledge)", "lrg", 3, loot.Stash, "runeword base 3"},
		{"broad sword 4 (Spirit)", "bsd", 4, loot.Stash, "runeword base 4"},
		{"broad sword 5 fits nothing early", "bsd", 5, loot.Drop, "junk"},
		{"broad sword 3 is no sword runeword", "bsd", 3, loot.Drop, "junk"},
		{"quilted 1", "qui", 1, loot.Drop, "junk"},
	}
	for _, c := range cases {
		u, ctx := base(c.code, 2, c.sockets)
		v := l.EvaluateWith(u, ctx)
		if v.Action != c.want || v.Rule != c.rule {
			t.Errorf("%s: got %v %q", c.name, v.Action, v.Rule)
		}
	}
	// A magic item can never become a runeword.
	u, ctx := base("qui", 4, 2)
	if v := l.EvaluateWith(u, ctx); v.Rule == "runeword base 2" {
		t.Errorf("magic base kept: %v", v)
	}
}

func TestRunewordKnob(t *testing.T) {
	l, r := newLooter(t)
	r.W.Player.Level = 10
	r.W.Player.Gold = 5000
	u := r.W.NewItem("qui", 2)
	u.Addr = 0xABCE
	r.W.Stats[u.Addr] = d2.Stats{194: 2}
	if v := l.EvaluateUnit(u); v.Rule != "runeword base 2" {
		t.Fatalf("default allows runewords: %v", v)
	}
	l.NoRunewords = true
	if v := l.EvaluateUnit(u); v.Rule != "junk" {
		t.Errorf("knob off: %v", v)
	}
}

func TestCharmStatFilter(t *testing.T) {
	l, r := newLooter(t)
	u := r.W.NewItem("cm1", 4) // small charm, magic, identified
	check := func(name string, st d2.Stats, rule string) {
		t.Helper()
		if v := l.EvaluateWith(u, loot.Ctx{CLvl: 10, Stats: st}); v.Rule != rule {
			t.Errorf("%s: rule %q want %q", name, v.Rule, rule)
		}
	}
	check("life 20", d2.Stats{7: 20 << 8}, "charms")
	check("life 10", d2.Stats{7: 10 << 8}, "magic and rare")
	check("fire res 5", d2.Stats{39: 5}, "charms")
	check("mf only", d2.Stats{80: 7}, "charms")
	check("fcr", d2.Stats{105: 3}, "charms")
	check("damage only", d2.Stats{21: 3}, "magic and rare")
	u.Item.Flags = 0
	if v := l.EvaluateWith(u, loot.Ctx{CLvl: 10, Gold: 5000}); v.Rule != "charms to id" || !v.NeedID {
		t.Errorf("unidentified: %v", v)
	}
}

func TestGemsKeptForCube(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Add("box", d2.ContStash, 9, 0) // the Cube is held: low gems are kept for upgrades (D4)
	chipped := w.NewItem("gcv", 2)
	if v := l.EvaluateUnit(chipped); v.Action != loot.Stash || v.Rule != "cube gems" {
		t.Fatalf("first chipped: %v", v)
	}
	for i := 0; i < loot.GemKeep; i++ {
		w.Add("gcv", d2.ContStash, i, 0)
	}
	if v := l.EvaluateUnit(chipped); v.Action != loot.Sell || v.Rule != "low gems" {
		t.Errorf("surplus chipped: %v", v)
	}
	// Another colour still counts from zero.
	if v := l.EvaluateUnit(w.NewItem("gfv", 2)); v.Rule != "cube gems" {
		t.Errorf("flawed amethyst: %v", v)
	}
}

func TestPotionWantFollowsBeltSize(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Level = 20
	hp := w.NewItem("hp3", 2)
	for i := 0; i < 4; i++ {
		w.Add("hp1", d2.ContBelt, i, 0)
	}
	if v := l.EvaluateUnit(hp); v.Rule == "healing and mana" {
		t.Fatalf("4-slot belt is full: %v", v)
	}
	// A 12-slot belt (Belt, "mbl") leaves room for more.
	bu := w.Add("mbl", d2.ContEquipment, 0, 0)
	for i := range w.Items {
		if w.Items[i].ID == bu.ID {
			w.Items[i].Item.Body = d2.BodyBelt
		}
	}
	if v := l.EvaluateUnit(hp); v.Rule != "healing and mana" {
		t.Errorf("12-slot belt: %v", v)
	}
}

func TestPotionWantByBuildAndDifficulty(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	r.Env.Tune.Loot.PotionBuffer = 0
	w.Player.Class = d2.Barbarian
	w.Player.Level = 40
	w.Add("hp1", d2.ContBelt, 0, 0) // room left on the belt, so only the plan's columns decide
	w.Add("hp1", d2.ContBelt, 3, 0)
	mp := w.NewItem("mp3", 2)
	want := func() bool { return l.EvaluateUnit(mp).Rule == "healing and mana" }

	// Normal is unchanged by the build: the Barbarian has no mana column.
	w.Player.Difficulty = d2.Normal
	base := want()
	l.SetBuild("barbarian_warcry_endgame90")
	if want() != base {
		t.Fatalf("Normal changed with the build: %v", base)
	}
	if base {
		t.Fatal("Normal Barbarian wants mana potions")
	}
	// Nightmare: War Cry carries two mana columns, so a belt with room still takes mana potions.
	w.Player.Difficulty = d2.Nightmare
	if !want() {
		t.Errorf("Nightmare mana-heavy build ignores mana potions: %v", l.EvaluateUnit(mp))
	}
	l.SetBuild("")
	if want() {
		t.Error("Nightmare without the build wants mana potions")
	}
	// Rejuvenation is an emergency heal: a spare of RejuvSpare in the pack, then the rest stays on the ground
	// (INVREJUV: the old unconditional keep filled 2/3 of the pack).
	rv := w.NewItem("rvl", 2)
	for _, d := range []d2.Difficulty{d2.Normal, d2.Nightmare, d2.Hell} {
		w.Player.Difficulty = d
		if v := l.EvaluateUnit(rv); v.Rule != "rejuvenation" {
			t.Errorf("%v rejuv: %v", d, v)
		}
	}
}

// Normal is bit-for-bit the old PlanForClass behaviour: the build changes no verdict.
func TestPotionWantNormalIgnoresBuild(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	r.Env.Tune.Loot.PotionBuffer = 0
	w.Add("hp1", d2.ContBelt, 0, 0)
	w.Player.Difficulty = d2.Normal
	items := []d2.Unit{w.NewItem("hp3", 2), w.NewItem("mp3", 2), w.NewItem("rvl", 2)}
	for _, c := range []d2.Class{d2.Amazon, d2.Sorceress, d2.Barbarian, d2.Paladin} {
		for lvl := 1; lvl <= 99; lvl++ {
			w.Player.Class, w.Player.Level = c, lvl
			for _, it := range items {
				l.SetBuild("")
				a := l.EvaluateUnit(it).Rule
				l.SetBuild("sorceress_blizzard_endgame90")
				if b := l.EvaluateUnit(it).Rule; a != b {
					t.Fatalf("%v clvl %d: %q vs %q", c, lvl, a, b)
				}
			}
		}
	}
}

// fillPack puts 1x1 gems in the pack until free cells remain, from the last cell backwards.
func fillPack(w *sim.World, free int) {
	n := 40 - free
	for i := 0; i < n; i++ {
		w.Add("gcv", d2.ContInventory, i%10, i/10)
	}
}

func TestPickUpGateNearlyFullPack(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	fillPack(w, 5)
	junk := w.AddGround("amu", 4, 6022, 4930) // magic amulet: Sell verdict at strictness 0
	rep, err := l.PickUp(context.Background(), loot.PickOpts{Strictness: 0, StrictSet: true})
	if err != nil || len(rep.Picked) != 0 {
		t.Fatalf("junk taken from a nearly full pack: %+v %v", rep, err)
	}
	_ = junk
	// the same junk is taken while there is room
	w.Items = w.Items[:len(w.Items)-6]
	rep, err = l.PickUp(context.Background(), loot.PickOpts{Strictness: 0, StrictSet: true})
	if err != nil || len(rep.Picked) != 1 {
		t.Fatalf("junk skipped with room: %+v %v", rep, err)
	}
}

func TestPickUpQuestItemWithFullPackIsReportedNotLooped(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			w.Add("gcv", d2.ContInventory, x, y)
		}
	}
	w.AddGround("msf", 2, 6022, 4930)
	rep, err := l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil || len(rep.Picked) != 0 || !rep.Full || len(rep.Failed) != 0 {
		t.Fatalf("%+v %v", rep, err)
	}
	// a free hole is enough for the quest item even though FreeCellsKeep would forbid junk
	kept := w.Items[:0]
	for _, u := range w.Items {
		if u.X < 6 || u.Y < 1 { // free a 4x3 block at the bottom right
			kept = append(kept, u)
		}
	}
	w.Items = kept
	rep, err = l.PickUp(context.Background(), loot.PickOpts{})
	if err != nil || len(rep.Picked) != 1 {
		t.Fatalf("quest item not taken into a free hole: %+v %v", rep, err)
	}
}

// Early gear: armor is kept when its slot is empty or it scores higher than the worn piece, never otherwise.
func TestWearBetterKeepsOnlyUpgrades(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	ctx := loot.Ctx{CLvl: 9, Str: 30, Dex: 30, Class: "sor", WearBetter: true, Worn: map[int]int{}}
	studded := w.NewItem("stu", 1) // gray: nothing else keeps it
	if v := l.EvaluateWith(studded, ctx); v.Action != loot.Keep || v.Rule != "empty slot" {
		t.Errorf("empty torso: %v %q", v.Action, v.Rule)
	}
	quilted := w.NewItem("qui", 1)
	ctx.Worn[d2.BodyTorso] = loot.WearScore(r.W.DB.ItemByCode("stu"), nil)
	if v := l.EvaluateWith(quilted, ctx); v.Action == loot.Keep && v.Rule == "empty slot" {
		t.Errorf("quilted is no upgrade over studded: %v %q", v.Action, v.Rule)
	}
	ctx.Worn[d2.BodyTorso] = loot.WearScore(r.W.DB.ItemByCode("qui"), nil)
	if v := l.EvaluateWith(studded, ctx); v.Action != loot.Keep {
		t.Errorf("studded beats quilted: %v", v.Action)
	}
	ctx.Str = 5
	if v := l.EvaluateWith(studded, ctx); v.Action == loot.Keep && v.Rule == "empty slot" {
		t.Errorf("unwearable (Str 5) kept for wearing: %v %q", v.Action, v.Rule)
	}
}

// REJUVUSE: up to RejuvSpare rejuvenation potions in total (belt plus pack), at any level; at the cap a Full is still
// wanted while a regular one is in the pack, so the sale keeps the Full ones.
func TestRejuvIsCapped(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	r.Env.Tune.Loot.PotionBuffer = 0
	r.Env.Tune.Loot.RejuvSpare = 3
	w.Player.Class = d2.Barbarian
	w.Player.Difficulty = d2.Normal
	rvs, rvl := w.NewItem("rvs", 2), w.NewItem("rvl", 2)

	w.Player.Level = 14
	if v := l.EvaluateUnit(rvs); v.Rule != "rejuvenation" {
		t.Errorf("clvl 14 empty: %v", v)
	}
	w.Add("rvl", d2.ContBelt, 3, 0)
	w.Add("rvs", d2.ContInventory, 0, 0)
	if v := l.EvaluateUnit(rvs); v.Rule != "rejuvenation" {
		t.Errorf("2 of 3 held: %v", v)
	}
	w.Add("rvs", d2.ContInventory, 1, 0)
	if v := l.EvaluateUnit(rvs); v.Rule == "rejuvenation" {
		t.Errorf("at the cap a regular is still wanted: %v", v)
	}
	if v := l.EvaluateUnit(rvl); v.Rule != "rejuvenation" {
		t.Errorf("at the cap a Full replaces a regular: %v", v)
	}
}
