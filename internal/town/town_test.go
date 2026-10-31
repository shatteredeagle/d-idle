package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"errors"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/sim"
)

type rig struct {
	*sim.Rig
	T      *town.Town
	Akara  d2.Unit
	Charsi d2.Unit
	Gheed  d2.Unit
}

func newRig(t *testing.T) *rig { return newRigAt(t, 0, 0) }

// newRigAt shifts the whole town by (dx,dy) subtiles: another game seed puts the Rogue Encampment elsewhere.
func newRigAt(t *testing.T, dx, dy int) *rig {
	r := &rig{Rig: sim.NewRig(t)}
	r.Env.Tune.Resource.EarlyGearMax = 0 // the rebuy tests pin the full rebuy; early-gear tests set it back
	w := r.W
	w.Player.Gold = 3000
	w.Player.Strength, w.Player.Dexterity = 30, 30
	w.Player.X, w.Player.Y = w.Player.X+dx, w.Player.Y+dy
	r.Akara = w.AddNPC(148, 6082+dx, 4911+dy)
	r.Charsi = w.AddNPC(154, 5984+dx, 4922+dy)
	r.Gheed = w.AddNPC(147, 5985+dx, 4983+dy)
	w.AddObject(267, 6016+dx, 4934+dy)
	w.Stock(r.Akara, "hp1", "hp2", "hp3", "mp1", "mp2", "tsc", "isc", "tbk", "ibk")
	w.Stock(r.Charsi, "hax", "ssd", "qui", "cap", "lbt", "buc")
	w.Stock(r.Gheed, "hp1")
	if w.Menu != (sim.MenuGeom{DX: town.NPCMenu.DX, Cancel: town.NPCMenu.Cancel, Step2: town.NPCMenu.Step2}) {
		t.Fatal("sim menu geometry differs from town.NPCMenu")
	}
	r.T = town.New(r.Env, r.Mv)
	r.T.UseSource(town.Source{Snapshot: w.Snapshot, Items: w.AllItems, Owner: w.Owner, Stats: w.UnitStats})
	return r
}

func count(w *sim.World, c d2.Container) int {
	n := 0
	for _, u := range w.Items {
		if u.Item.Where == c {
			n++
		}
	}
	return n
}

// dressed equips a dagger and a quilted armor: a naked character keeps and wears found gear instead of selling it.
func dressed(w *sim.World) {
	w.Add("dgr", d2.ContEquipment, 0, 0).Item.Body = d2.BodyRHand
	w.Add("qui", d2.ContEquipment, 0, 0).Item.Body = d2.BodyTorso
}

func TestChoresAct1(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	dressed(w)
	w.Player.Life = 12
	w.Player.Level = 10                  // keep line 800: bankFirst leaves the shop enough
	w.Add("r07", d2.ContInventory, 0, 0) // rune: stash
	sword := w.Add("ssd", d2.ContInventory, 1, 0)
	sword.Item.Quality = 4 // magic: sell
	w.Items[len(w.Items)-1] = sword
	w.Add("tsc", d2.ContInventory, 9, 3)
	rep, err := r.T.Chores(context.Background(), town.FullPlan())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Errors != 0 {
		t.Fatalf("steps %+v", rep.Steps)
	}
	if !rep.Healed || w.Player.Life != w.Player.MaxLife {
		t.Errorf("heal: %+v life %d", rep.Steps, w.Player.Life)
	}
	if count(w, d2.ContBelt) != 4 {
		t.Errorf("belt has %d potions, want 4 (%v)", count(w, d2.ContBelt), rep.Bought)
	}
	if count(w, d2.ContStash) != 1 || rep.Stashed != 1 {
		t.Errorf("stash %d", count(w, d2.ContStash))
	}
	if rep.Sold != 1 || rep.Earned <= 0 {
		t.Errorf("sold %d earned %d", rep.Sold, rep.Earned)
	}
	if rep.Bought["Light Healing Potion"] != 2 || rep.Bought["Light Mana Potion"] != 2 {
		t.Errorf("bought %v", rep.Bought)
	}
	if w.UI.AnyPanelOpen() || w.Cursor != nil {
		t.Errorf("panels open %v cursor %v", w.UI, w.Cursor)
	}
	if rep.GoldAfter >= rep.GoldBefore+rep.Earned || rep.Spent <= 0 {
		t.Errorf("gold %d -> %d spent %d", rep.GoldBefore, rep.GoldAfter, rep.Spent)
	}
	// Second run: nothing to do, every step skipped, no clicks beyond closing.
	rep2, _ := r.T.Chores(context.Background(), town.FullPlan())
	for _, s := range rep2.Steps {
		if s.Ran && s.Name != "close" {
			t.Errorf("second run executed %s", s.Name)
		}
	}
}

func TestChoresNeedsTownAndVendorRefusal(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Player.Area = 2
	if _, err := r.T.Chores(context.Background(), town.FullPlan()); err != town.ErrNotInTown {
		t.Fatalf("want ErrNotInTown, got %v", err)
	}
	w.Player.Area = 1
	// Charsi refuses the amulet; Gheed takes it.
	w.Add("amu", d2.ContInventory, 0, 0)
	w.Items[0].Item.Quality = 4
	w.Refuse[r.Charsi.ID] = map[string]bool{"amu": true}
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if err != nil || rep.Sold != 1 || rep.Errors != 0 {
		t.Fatalf("%v sold %d steps %+v", err, rep.Sold, rep.Steps)
	}
	// Nobody buys: reported, not fatal.
	w.Add("amu", d2.ContInventory, 0, 0)
	w.Items[len(w.Items)-1].Item.Quality = 4
	w.Refuse[r.Gheed.ID] = map[string]bool{"amu": true}
	w.Refuse[r.Akara.ID] = map[string]bool{"amu": true}
	rep, err = r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if err != nil || rep.Errors != 1 || rep.Sold != 0 {
		t.Fatalf("%v %+v", err, rep.Steps)
	}
}

func TestInsureStashesValuables(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Add("hp1", d2.ContInventory, 0, 0)
	w.Add("gpv", d2.ContInventory, 1, 0)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Stash: true, Insure: true})
	if err != nil || rep.Stashed != 1 || count(w, d2.ContStash) != 1 {
		t.Fatalf("%v stashed %d", err, rep.Stashed)
	}
}

func TestStashGold(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 5000
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Gold: true})
	if err != nil || rep.Errors != 0 || r.W.Player.StashGold != 4650 || r.W.Player.Gold != 350 {
		t.Fatalf("%v %+v gold %d bank %d", err, rep.Steps, r.W.Player.Gold, r.W.Player.StashGold)
	}
}

func TestRebuyNaked(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Gold = 600
	rep, err := r.T.Rebuy(context.Background(), 600)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) != 0 {
		t.Fatalf("%+v", rep)
	}
	s, _ := w.Snapshot()
	if s.Inventory.Equipped(d2.BodyRHand) == nil || s.Inventory.Equipped(d2.BodyTorso) == nil {
		t.Fatalf("not equipped: %+v", rep)
	}
	if len(rep.Missing) != 0 || len(rep.Equipped) != 2 {
		t.Fatalf("%+v", rep)
	}
	if count(w, d2.ContBelt) == 0 {
		t.Error("potions first: belt still empty")
	}
	if w.UI.AnyPanelOpen() || w.Cursor != nil {
		t.Error("panels open")
	}
	// Poor: 20 gold buys nothing of the gear.
	r2 := newRig(t)
	r2.W.Player.Gold = 20
	rep2, _ := r2.T.Rebuy(context.Background(), 20)
	if len(rep2.Missing) != 2 || len(rep2.Equipped) != 0 {
		t.Fatalf("poor: %+v", rep2)
	}
	// Tier 0: potions only.
	r3 := newRig(t)
	r3.Env.Tune.Town.RebuyGearTier = 0
	rep3, _ := r3.T.Rebuy(context.Background(), 1000)
	if len(rep3.Equipped) != 0 || len(rep3.Missing) != 0 {
		t.Fatalf("tier 0: %+v", rep3)
	}
}

func TestNeedsTrip(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Add("tsc", d2.ContInventory, 9, 3)
	s, _ := w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); !ok || why != "low on healing potions" {
		t.Fatalf("%v %q", ok, why)
	}
	for i := 0; i < 3; i++ {
		w.Add("hp1", d2.ContBelt, i, 0)
	}
	w.Add("mp1", d2.ContBelt, 3, 0)
	s, _ = w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); ok {
		t.Fatalf("stocked: %q", why)
	}
	for i := len(w.Items) - 1; i >= 0; i-- {
		if w.Items[i].Item.Where == d2.ContInventory {
			w.Items = append(w.Items[:i], w.Items[i+1:]...)
		}
	}
	s, _ = w.Snapshot()
	if ok, why := r.T.NeedsTrip(s); !ok || why != "no town portal scroll" {
		t.Fatalf("%v %q", ok, why)
	}
}

func TestToTownAndBack(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Area = 2
	w.PortalLevel = 1
	w.Add("tsc", d2.ContInventory, 0, 0)
	ctx := context.Background()
	if err := r.T.ToTown(ctx); err != nil {
		t.Fatal(err)
	}
	if w.Player.Area != 1 || w.UI.AnyPanelOpen() {
		t.Fatalf("area %d ui %v", w.Player.Area, w.UI)
	}
	w.PortalLevel = 2
	if err := r.T.BackToField(ctx, 2); err != nil || w.Player.Area != 2 {
		t.Fatalf("%v area %d", err, w.Player.Area)
	}
	// No scroll: ErrNoPortal, except next to the town, where the character walks back.
	w.Items = nil
	w.Objs = nil
	w.Player.Area = 3 // Cold Plains has a waypoint: no scroll, so it takes it to town
	if err := r.T.ToTown(ctx); err != nil || w.Player.Area != 1 || r.Mv.Calls[len(r.Mv.Calls)-1] != "waypoint 1" {
		t.Fatalf("waypoint fallback: %v area %d calls %v", err, w.Player.Area, r.Mv.Calls)
	}
	w.Player.Area = 7 // Tamoe Highland: no waypoint, two hops from town: ErrNoPortal stays
	if err := r.T.ToTown(ctx); !errors.Is(err, town.ErrNoPortal) {
		t.Fatalf("got %v", err)
	}
	w.Player.Area = 8 // Den of Evil: no waypoint, dead end off Blood Moor: walk through Blood Moor
	r.Mv.Calls = nil
	if err := r.T.ToTown(ctx); err != nil || w.Player.Area != 1 || strings.Join(r.Mv.Calls, ",") != "exitto 2,exitto 1" {
		t.Fatalf("den walk: %v area %d calls %v", err, w.Player.Area, r.Mv.Calls)
	}
	w.Player.Area = 2
	if err := r.T.ToTown(ctx); err != nil || w.Player.Area != 1 {
		t.Fatalf("walk fallback: %v area %d", err, w.Player.Area)
	}
	// Town without portal object: falls back to the exit.
	w.Player.Area = 1
	w.Objs = nil
	if err := r.T.BackToField(ctx, 2); err != nil || w.Player.Area != 2 {
		t.Fatalf("exit fallback: %v", err)
	}
}

// The town portal leads to where the last scroll was read: a corpse elsewhere is the caller's walk (ErrWrongLevel).
func TestBackToFieldPortalToOtherLevel(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Area = 2
	w.PortalLevel = 1
	w.Add("tsc", d2.ContInventory, 0, 0)
	ctx := context.Background()
	if err := r.T.ToTown(ctx); err != nil {
		t.Fatal(err)
	}
	w.PortalLevel = 3
	r.Mv.Calls = nil
	err := r.T.BackToField(ctx, 26)
	if !errors.Is(err, town.ErrWrongLevel) || w.Player.Area != 3 {
		t.Fatalf("got %v area %d", err, w.Player.Area)
	}
	for _, c := range r.Mv.Calls {
		if strings.HasPrefix(c, "exitto") {
			t.Fatalf("single-hop exit tried from the wrong level: %v", r.Mv.Calls)
		}
	}
}

func TestActTablesMatchData(t *testing.T) {
	db := sim.RealDB(t)
	for _, a := range town.Acts {
		for _, list := range [][]int{a.Heal, a.Shop, a.Repair, a.Gamble, a.Merc, a.Keys, a.Cain, a.Sell} {
			for _, c := range list {
				m := db.MonsterByClassID(c)
				if m == nil || !m.NPC {
					t.Errorf("act %d: class %d is not an NPC", a.No, c)
				}
			}
		}
		if !a.IsTown() {
			t.Errorf("act %d town level %d", a.No, a.TownLevel)
		}
	}
	if o := db.ObjectByID(town.ObjStash); o == nil || o.Name != "bank" {
		t.Errorf("stash object %+v", o)
	}
}

func TestIdentifyWithScroll(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Add("rin", d2.ContInventory, 0, 0)
	w.Items[0].Item.Quality, w.Items[0].Item.Flags = 4, 0
	w.Add("isc", d2.ContInventory, 1, 0)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Identify: true})
	if err != nil || rep.Identified != 1 || rep.Errors != 0 {
		t.Fatalf("%v %+v", err, rep)
	}
	if !w.Items[0].Item.Identified() {
		t.Fatal("not identified in world")
	}
	// Without a scroll one is bought first.
	r2 := newRig(t)
	r2.W.Add("rin", d2.ContInventory, 0, 0)
	r2.W.Items[0].Item.Quality, r2.W.Items[0].Item.Flags = 6, 0 // BASICSFIX: a plain magic ring is sold as it is; a rare is worth a scroll
	rep, err = r2.T.Chores(context.Background(), town.ChorePlan{Identify: true})
	if err != nil || rep.Identified != 1 || rep.Bought["Scroll of Identify"] != 1 {
		t.Fatalf("%v %+v", err, rep)
	}
}

// The healer that ignores the first click is opened again (LIVEBOT: "healer did not restore life and mana").
func TestHealRetriesWhenTheHealerIgnoresAClick(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Life = 10
	r.W.HealSkip = 1
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Heal: true})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Healed || r.W.Player.Life != r.W.Player.MaxLife {
		t.Fatalf("not healed: %+v life %d", rep.Steps, r.W.Player.Life)
	}
}

// A sell chore that runs out of its cap fails with a clear message and leaves no panel open.
func TestSellChoreIsCapped(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	sword := w.Add("ssd", d2.ContInventory, 1, 0)
	sword.Item.Quality = 4
	w.Items[len(w.Items)-1] = sword
	tu := r.Env.Tune.Town
	tu.SellMaxS = -1 // already expired
	r.Env.Tune.Town = tu
	rep, _ := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if rep.Sold != 0 {
		t.Fatalf("sold despite the cap: %+v", rep)
	}
	if w.UI.AnyPanelOpen() {
		t.Fatal("panel left open")
	}
}

// A character standing on an empty weapon set (the `w` key, VERIFIED LIVE by CLASSPAL) owns its weapon at body 11:
// rebuy must not buy another one into the active slot.
func TestRebuyNeverBuysIntoSwappedSet(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Gold = 600
	u := w.Add("ssd", d2.ContEquipment, 0, 0)
	u.Item.Body = d2.BodyAltR
	w.Items[len(w.Items)-1] = u
	rep, err := r.T.Rebuy(context.Background(), 600)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := w.Snapshot()
	if s.Inventory.Equipped(d2.BodyRHand) != nil {
		t.Fatalf("bought a weapon into the active set: %+v", rep)
	}
	for _, m := range rep.Missing {
		if m == "weapon" {
			t.Fatalf("weapon reported missing: %+v", rep)
		}
	}
	if s.Inventory.Equipped(d2.BodyTorso) == nil {
		t.Fatalf("body armor should still be bought: %+v", rep)
	}
}

// BASICSFIX: with potionsFirst 0 (the old order) the first portal scroll comes before the potions.
func TestShopBuysTownPortalBeforePotionsWhenPotionsFirstIsOff(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 120
	r.Env.Tune.Town.PotionsFirst = 0
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Scroll of Town Portal"] != 1 {
		t.Fatalf("no town portal scroll first: %v spent %d", rep.Bought, rep.Spent)
	}
	if rep.Spent > 120 {
		t.Errorf("overspent %d", rep.Spent)
	}
}

// Between the trip floor and twice the floor the pack gets at most two spare scrolls, however large the target.
func TestShopCapsSpareScrollsWhenNotRich(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 5000
	r.Env.Tune.Resource.TripGoldFloor = 3000 // 5000 gold is between 1x and 2x the floor
	r.Env.Tune.Town.TPScrollTarget = 6
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Scrolls: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := rep.Bought["Scroll of Town Portal"]; n != 2 {
		t.Fatalf("bought %d scrolls, want 2: %v", n, rep.Bought)
	}
}

// The first-contact greeting is skipped with Esc after a few seconds, not after 20 (found live: Akara took 2.5 min).
func TestGreetingIsSkippedQuickly(t *testing.T) {
	r := newRig(t)
	r.W.Speech = 1
	r.Env.Tune.Town.GreetingWaitS = 90
	start := r.Env.Clock.Now()
	if _, err := r.T.OpenForTest(context.Background(), 148); err != nil {
		t.Fatal(err)
	}
	d := r.Env.Clock.Now().Sub(start)
	if d > 15*time.Second {
		t.Fatalf("greeting took %v of game time", d)
	}
}

func TestChoresWearFoundGearInsteadOfSelling(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	w := r.W
	w.Player.Level, w.Player.Strength, w.Player.Dexterity, w.Player.Gold = 8, 30, 30, 0
	armor := w.Add("lea", d2.ContInventory, 0, 0)
	sword := w.Add("ssd", d2.ContInventory, 3, 0)
	heavy := w.Add("plt", d2.ContInventory, 6, 0) // plate mail: strength 65, not wearable
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if err != nil {
		t.Fatal(err)
	}
	s, _ := w.Snapshot()
	if u := s.Inventory.Equipped(d2.BodyTorso); u == nil || u.ID != armor.ID {
		t.Fatalf("leather armor not worn: %+v", rep.Steps)
	}
	if u := s.Inventory.Equipped(d2.BodyRHand); u == nil || u.ID != sword.ID {
		t.Fatalf("sword not worn: %+v", rep.Steps)
	}
	if u := s.Inventory.Equipped(d2.BodyTorso); u != nil && u.ID == heavy.ID {
		t.Fatal("wore plate mail without the strength")
	}
	if w.UI.AnyPanelOpen() || w.Cursor != nil {
		t.Error("panels open")
	}
}

// G1 attempt 7 (V live): a clvl 9 Sorceress with the starting 10 Str cannot wear Quilted Armor (needs 12), so the
// rebuy found "nothing viable" and the gate never lifted. The report now says the slot is Blocked by requirements.
func TestRebuyBlockedByStrength(t *testing.T) {
	r := newRig(t)
	w := r.W
	w.Player.Level, w.Player.Strength, w.Player.Dexterity, w.Player.Gold = 9, 10, 25, 3000
	w.Add("dgr", d2.ContEquipment, 0, 0).Item.Body = d2.BodyRHand
	w.Stock(r.Charsi, "hax", "ssd", "qui", "lea", "cap", "lbt", "buc")
	rep, err := r.T.Rebuy(context.Background(), 1200)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Blocked) != 1 || rep.Blocked[0] != "body armor" || len(rep.Equipped) != 0 {
		t.Fatalf("%+v", rep)
	}
	if len(rep.Errors) == 0 || !strings.Contains(rep.Errors[0], "nothing wearable") {
		t.Fatalf("errors %v", rep.Errors)
	}
	// Two more Strength and the Quilted Armor goes on.
	r2 := newRig(t)
	r2.W.Player.Level, r2.W.Player.Strength, r2.W.Player.Dexterity, r2.W.Player.Gold = 9, 12, 25, 3000
	r2.W.Add("dgr", d2.ContEquipment, 0, 0).Item.Body = d2.BodyRHand
	r2.W.Stock(r2.Charsi, "hax", "ssd", "qui", "lea", "sst")
	rep2, err := r2.T.Rebuy(context.Background(), 1200)
	s, _ := r2.W.Snapshot()
	if err != nil || len(rep2.Blocked) != 0 || s.Inventory.Equipped(d2.BodyTorso) == nil {
		t.Fatalf("%v %+v", err, rep2)
	}
}

// XPGAP: with tpMinGold set a poor character spends on potions and a missing scroll is no trip reason.
func TestTPMinGoldPotionsFirst(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Player.Gold = 120
	r.Env.Tune.Town.TPMinGold = 250
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Bought["Scroll of Town Portal"] != 0 {
		t.Fatalf("scroll bought below tpMinGold: %v", rep.Bought)
	}
}

// BASICSFIX/XPGAP: "no town portal scroll" is a trip reason only where the shop would sell one (potionsFirst).
func TestNoScrollTripRespectsPotionsFirst(t *testing.T) {
	geartest.Need(t)
	for _, c := range []struct {
		name         string
		level, first int
		want         bool
	}{
		{"below first-scroll level", 2, 1, false},
		{"at first-scroll level", 4, 1, true},
		{"potionsFirst off", 2, 0, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newRig(t)
			r.W.Player.Level = c.level
			r.Env.Tune.Town.PotionsFirst = c.first
			for i := 0; i < 3; i++ {
				r.W.Add("hp1", d2.ContBelt, i, 0)
			}
			r.W.Add("mp1", d2.ContBelt, 3, 0)
			s, _ := r.W.Snapshot()
			ok, why := r.T.NeedsTrip(s)
			if got := ok && why == "no town portal scroll"; got != c.want {
				t.Fatalf("trip %v %q, want scroll reason %v", ok, why, c.want)
			}
		})
	}
}
