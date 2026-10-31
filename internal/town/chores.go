package town

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/panel"
)

// tripReason is one thing that makes a town trip advisable. gold marks reasons a visit can only fix by spending.
type tripReason struct {
	why  string
	key  string // stable across visits (the text carries percentages)
	gold bool
}

// futileMark remembers when and with how much gold a visit left a reason unfixed.
type futileMark struct {
	gold int
	at   time.Time
}

// NeedsTrip says whether the bot should go to town now, and why. Pure: reads only the snapshot and the memory of
// visits that could not fix a reason. Reasons that cost gold are not raised when the character cannot afford the
// cheapest potion; a reason a visit left unfixed stays quiet for FutileMinutes and, if it costs gold, until gold has
// grown by FutileGoldStep.
func (t *Town) NeedsTrip(s *d2.Snapshot) (bool, string) {
	for _, r := range t.tripReasons(s) {
		if r.gold && s.Player.Gold < t.e.Tune.Resource.TripMinGold {
			continue
		}
		if t.isFutile(r, s.Player.Gold) {
			continue
		}
		return true, r.why
	}
	return false, ""
}

func (t *Town) isFutile(r tripReason, gold int) bool {
	m, seen := t.futile[r.key]
	if !seen {
		return false
	}
	if t.e.Clock.Now().Sub(m.at) < time.Duration(t.tune().FutileMinutes)*time.Minute {
		return true
	}
	return r.gold && gold < m.gold+t.tune().FutileGoldStep
}

// tripReasons lists every current reason, most urgent first.
func (t *Town) tripReasons(s *d2.Snapshot) []tripReason {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return nil
	}
	tn := t.tune()
	p, inv := s.Player, s.Inventory
	band := t.plan(p)
	st := belt.Read(t.e.Data, inv)
	var out []tripReason
	add := func(gold bool, key, f string, a ...any) {
		out = append(out, tripReason{fmt.Sprintf(f, a...), key, gold})
	}
	if band.Capacity(belt.HP, st.Slots) > 0 && st.Count[belt.HP]+belt.PackCount(t.e.Data, inv, belt.HP) < tn.TripMinHPPotions {
		add(true, "hp", "low on healing potions")
	}
	if band.Capacity(belt.MP, st.Slots) > 0 && tn.TripMinMPPotions > 0 && st.Count[belt.MP]+belt.PackCount(t.e.Data, inv, belt.MP) < tn.TripMinMPPotions {
		add(true, "mp", "low on mana potions")
	}
	for _, k := range []belt.Kind{belt.HP, belt.MP} {
		if band.Capacity(k, st.Slots) > 0 && band.FillPct(st, k) < tn.TripBeltColumnPct && belt.PackCount(t.e.Data, inv, k) == 0 {
			add(true, "belt"+k.String(), "%s belt columns below %d%%", k, tn.TripBeltColumnPct)
		}
	}
	if t.wantsBow(s) {
		add(false, "bow", "bow-path Amazon without a bow and gold for one")
	}
	if t.needWeaponFound(s) {
		add(false, "wear", "found weapon to wear")
	}
	if n, _ := t.scrollCount(s, "tsc", "tbk"); n == 0 && !t.tpTooPoor(s) {
		add(true, "tsc", "no town portal scroll")
	}
	if e, ok := t.gearOf(s); ok && e.Launcher() && len(gear.Restock(e)) > 0 {
		add(true, "ammo", "ammo low")
	}
	if lay, err := t.layout(s); err == nil {
		if pct := t.h.Occupancy(inv, d2.ContInventory, lay.Inventory).UsedPct(); pct >= tn.TripPackUsedPct {
			add(false, "pack", "pack %d%% full", pct)
		}
	}
	if pct, ok := t.lowestDurability(s); ok && pct < tn.TripRepairPct {
		add(true, "durability", "equipment at %d%% durability", pct)
	}
	return out
}

// lowestDurability scans equipped items.
func (t *Town) lowestDurability(s *d2.Snapshot) (int, bool) {
	low, any := 101, false
	for _, u := range s.Inventory.Container(d2.ContEquipment) {
		if pct, ok := t.durabilityPct(u); ok {
			any = true
			if pct < low {
				low = pct
			}
		}
	}
	return low, any
}

func (t *Town) needHeal(s *d2.Snapshot) bool {
	p := s.Player
	if p.LifePct() < t.tune().HealBelowLifePct {
		return true
	}
	if m := t.tune().HealBelowManaPct; m > 0 && p.MaxMana > 0 && p.Mana*100/p.MaxMana < m {
		return true
	}
	return false
}

func (t *Town) needIdentify(s *d2.Snapshot) []d2.Unit {
	var out []d2.Unit
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.Item.Identified() {
			continue
		}
		if v := t.judge.EvaluateUnit(u); v.NeedID {
			out = append(out, u)
		}
	}
	return out
}

func (t *Town) needPotions(s *d2.Snapshot) bool {
	return len(t.potionsWanted(s)) > 0
}

// potionsWanted is the belt plan's shortfall after counting the potions the pack holds (a belt fill that failed or has
// not run yet must not make the shop buy what is already carried).
func (t *Town) potionsWanted(s *d2.Snapshot) map[belt.Kind]int {
	band := t.plan(s.Player)
	bs := belt.Read(t.e.Data, s.Inventory)
	want := band.Want(bs, t.tune().BeltFillPct)
	for k, n := range want {
		if n -= belt.PackCount(t.e.Data, s.Inventory, k); n > 0 {
			want[k] = n
		} else {
			delete(want, k)
		}
	}
	return want
}

func (t *Town) needScrolls(s *d2.Snapshot) bool {
	if n, hasTome := t.scrollCount(s, "tsc", "tbk"); n < t.tune().TPScrollTarget || !hasTome && s.Player.Gold >= t.tune().TPTomeMinGold {
		return true
	}
	if t.tune().KeyTarget > 0 && t.keyCount(s) < t.tune().KeyTarget {
		return true
	}
	if t.tune().IDScrollTarget > 0 {
		if n, _ := t.scrollCount(s, "isc", "ibk"); n < t.tune().IDScrollTarget {
			return true
		}
	}
	return false
}

func (t *Town) needRepair(s *d2.Snapshot) bool {
	pct, ok := t.lowestDurability(s)
	return ok && pct < t.tune().RepairBelowPct
}

// vendorStep names the chores that work inside a vendor's trade window. One window serves consecutive chores of the
// same NPC; any other chore closes it first.
var vendorStep = map[string]bool{"sell": true, "ammo": true, "shop": true, "beltbuy": true, "repair": true}

// Chores runs the town chores in the order: identify, equip, sell, ammo, belt, repair (all at the repair vendor,
// one visit), belt fill, potions and scrolls (the shop NPC, who also heals on opening: one visit), heal, stash, gold.
// Each step has a need check and its failure does not stop the later steps. Only "not in town", "not in
// game" and context errors abort. Every panel is closed at the end.
func (t *Town) Chores(ctx context.Context, plan ChorePlan) (rep ChoreReport, _ error) {
	rep = ChoreReport{Bought: map[string]int{}}
	s, err := t.current()
	if err != nil {
		return rep, err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return rep, ErrNotInTown
	}
	rep.GoldBefore = s.Player.Gold
	lowAtStart := plan.Heal && t.needHeal(s)
	deadline := t.e.Clock.Now().Add(time.Duration(t.tune().ChoreBudgetS) * time.Second)
	defer func() {
		if s, err := t.current(); err == nil {
			rep.GoldAfter = s.Player.Gold
		}
	}()
	if err := t.h.CloseAll(ctx); err != nil {
		return rep, err
	}

	// step runs one chore if its need check says so (or Force).
	step := func(name string, enabled bool, need func(*d2.Snapshot) bool, run func() error) error {
		if !enabled {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		cur, err := t.current()
		if err != nil {
			return err
		}
		if !plan.Force && !need(cur) {
			rep.add(name, false, "not needed", nil)
			return nil
		}
		if !t.e.Clock.Now().Before(deadline) {
			rep.add(name, false, "chore budget exhausted", fmt.Errorf("town: chore budget exhausted before %s", name))
			return nil
		}
		if !vendorStep[name] && cur.UI.Open(d2.UIShop) {
			if err := t.h.CloseAll(ctx); err != nil {
				return err
			}
		}
		err = run()
		if ce := ctx.Err(); ce != nil {
			return ce
		}
		rep.add(name, true, "", err)
		if err != nil {
			t.e.Log.Warn("town chore failed", "chore", name, "err", err)
			_ = t.h.CloseAll(ctx)
		}
		return nil
	}

	steps := []struct {
		name    string
		enabled bool
		need    func(*d2.Snapshot) bool
		run     func() error
	}{
		{"identify", plan.Identify, func(s *d2.Snapshot) bool { return len(t.needIdentify(s)) > 0 }, func() error { return t.identify(ctx, a, &rep) }},
		// before sell: a naked character wears what it found instead of selling it (G1 attempt 6)
		{"equip", true, t.needEquipFound, func() error { _, err := t.EquipFound(ctx); return err }},
		{"sell", plan.Sell && t.tune().SellJunk != 0, t.needSell, func() error { return t.sell(ctx, a, &rep) }},
		// before shop: the bow is the Amazon's only attack, potions are cheap to skip (live i3: shop spent 450 of 500 gold first)
		{"ammo", plan.Ammo, t.ammoTodo, func() error { return t.buyAmmo(ctx, a, &rep) }},
		// the repair vendor's chores run back to back so his window opens once per trip. The belt purchase keeps its own
		// gold reserve for the potion kit (beltBuyReserve), so it may come before the shop.
		{"beltbuy", plan.Potions, t.needBelt, func() error { return t.buyBelt(ctx, a, &rep) }},
		{"repair", plan.Repair, t.needRepair, func() error { return t.repair(ctx, a, &rep) }},
		// before shop: potions already in the pack go to the belt first, so the shop buys only the shortfall
		{"belt", plan.Potions, t.NeedsBeltFill, func() error { _, err := t.FillBelt(ctx); return err }},
		{"shop", plan.Potions || plan.Scrolls || plan.Antidotes > 0, func(s *d2.Snapshot) bool {
			return plan.Potions && t.needPotions(s) || plan.Scrolls && t.needScrolls(s) || t.needAntidotes(plan, s)
		}, func() error { return t.shopVisit(ctx, a, plan, &rep) }},
		// last of the NPC chores: a healer opened for the shop has healed already, so this normally finds nothing to do
		{"heal", plan.Heal, t.needHeal, func() error { return t.heal(ctx, a, &rep) }},
		{"cube", plan.Cube, func(s *d2.Snapshot) bool { return t.needCube(s, plan) }, func() error { return t.cubeJobs(ctx, a, plan, &rep) }},
		{"stash", plan.Stash || plan.Gold, func(s *d2.Snapshot) bool { return t.needStash(s, plan) }, func() error { return t.doStash(ctx, a, plan, &rep) }},
	}
	for _, st := range steps {
		if err := step(st.name, st.enabled, st.need, st.run); err != nil {
			return rep, err
		}
	}
	if err := t.h.CloseAll(ctx); err != nil {
		rep.add("close", true, "", err)
	}
	if s, err := t.current(); err == nil {
		if lowAtStart && !rep.Healed && s.Player.Life >= s.Player.MaxLife && s.Player.Mana >= s.Player.MaxMana {
			rep.Healed = true // the vendor visit healed on opening
		}
		t.noteFutile(s)
		if lay, err := t.layout(s); err == nil {
			pack := t.h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory)
			stash := t.h.Occupancy(s.Inventory, d2.ContStash, lay.Stash)
			rep.PackFull = pack.UsedPct() >= t.tune().TripPackUsedPct
			rep.PackPct, rep.StashPct = pack.UsedPct(), stash.UsedPct()
			// stash cells read 0 when the stash was not opened this visit and the snapshot lists none
			t.e.Log.Info("inventory", "pct", rep.PackPct, "cells", fmt.Sprintf("%d/%d", lay.Inventory.Cols*lay.Inventory.Rows-pack.FreeCells(), lay.Inventory.Cols*lay.Inventory.Rows),
				"stashPct", rep.StashPct, "stashCells", fmt.Sprintf("%d/%d", lay.Stash.Cols*lay.Stash.Rows-stash.FreeCells(), lay.Stash.Cols*lay.Stash.Rows))
			t.e.Log.Info("inventory contents", "pack", t.contentsOf(s, d2.ContInventory), "stash", t.contentsOf(s, d2.ContStash))
		}
	}
	return rep, nil
}

func (t *Town) needSell(s *d2.Snapshot) bool {
	lay, err := t.layout(s)
	return err == nil && len(t.sellable(s, lay)) > 0
}

func (t *Town) needStash(s *d2.Snapshot, plan ChorePlan) bool {
	if plan.Gold && s.Player.Gold > t.goldKeep(s.Player.Level) && s.Player.Gold > 0 {
		return true
	}
	if !plan.Stash {
		return false
	}
	if vj, ok := t.judge.(interface {
		Valuables(*d2.Snapshot) loot.ValuablesReport
	}); ok && plan.Insure && len(vj.Valuables(s).Stash) > 0 {
		return true
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if t.stashRefused[u.ID] {
			continue
		}
		if v := t.judge.EvaluateUnit(u); t.wantsStash(s, u, v) {
			return true
		}
	}
	return false
}

func (t *Town) heal(ctx context.Context, a *Act, rep *ChoreReport) error {
	// The healer sometimes does not answer the first click (LIVEBOT soak: "healer did not restore"): open it again.
	var last error
	for try := 0; try < healTries; try++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := t.openNPC(ctx, a.Heal, t.npcPreset(a.Heal...))
		if err != nil {
			last = err
			_ = t.h.CloseAll(ctx)
			continue
		}
		_, ok, werr := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool {
			return s.Player.Life >= s.Player.MaxLife && s.Player.Mana >= s.Player.MaxMana
		})
		if cerr := t.h.CloseAll(ctx); cerr != nil && werr == nil {
			werr = cerr
		}
		if werr != nil {
			return werr
		}
		if ok {
			rep.Healed = true
			return nil
		}
		last = fmt.Errorf("town: healer did not restore life and mana")
		t.e.Log.Info("town healer did not restore life and mana; reopening", "try", try+1)
	}
	return last
}

// sellCap is the SellMaxS limit (60 s for a tuning that never set it; a negative value means already expired, for tests).
func (t *Town) sellCap() time.Duration {
	n := t.tune().SellMaxS
	if n == 0 {
		n = 60
	}
	return time.Duration(n) * time.Second
}

// healTries is how often the healer is opened before the chore gives up.
const healTries = 3

// shopVisit buys the first town portal scroll, then potions and scrolls at the shop NPC in one visit.
func (t *Town) shopVisit(ctx context.Context, a *Act, plan ChorePlan, rep *ChoreReport) error {
	npc, err := t.openShop(ctx, a.Shop, t.npcPreset(a.Shop...))
	if err != nil {
		return err
	}
	// town portal scroll first: without one, potions eat the scroll money and the character stays portal-less
	first := t.buyFirstTP(ctx, npc, rep)
	if plan.Potions {
		if err := t.buyPotions(ctx, npc, plan.PotionFloor, rep); err != nil && first == nil {
			first = err
		}
	}
	if plan.Scrolls {
		if err := t.buyScrolls(ctx, npc, rep); err != nil && first == nil {
			first = err
		}
	}
	if plan.Antidotes > 0 {
		if err := t.buyAntidotes(ctx, npc, plan.Antidotes, rep); err != nil && first == nil {
			first = err
		}
	}
	t.logKit(rep, first)
	return first
}

// logKit writes one summary line per shop visit, kit state included, for the soak harness (belt criterion): what was
// bought and what the pack and belt hold afterwards.
func (t *Town) logKit(rep *ChoreReport, err error) {
	s, serr := t.current()
	if serr != nil || s.Inventory == nil {
		t.e.Log.Info("shop: visit", "bought", rep.Bought, "spent", rep.Spent, "broke", rep.Broke, "err", err)
		return
	}
	st := belt.Read(t.e.Data, s.Inventory)
	tp, tome := t.scrollCount(s, "tsc", "tbk")
	t.e.Log.Info("shop: visit", "bought", rep.Bought, "spent", rep.Spent, "broke", rep.Broke, "err", err,
		"gold", s.Player.Gold, "beltHP", st.Count[belt.HP], "beltMP", st.Count[belt.MP], "beltRV", st.Count[belt.RV],
		"beltSlots", st.Slots, "tp", tp, "tome", tome)
}

// sell visits the act's vendors in order until nothing sellable is left.
func (t *Town) sell(parent context.Context, a *Act, rep *ChoreReport) error {
	// One sell chore never takes longer than SellMaxS (a vendor click loop that does not converge must not eat the
	// whole chore budget); the panels are closed with the caller's context.
	ctx, cancel := context.WithTimeout(parent, t.sellCap())
	defer cancel()
	var first error
	tried := map[int]bool{}
	for _, class := range a.Sell {
		if tried[class] {
			continue
		}
		tried[class] = true
		s, err := t.current()
		if err != nil {
			return err
		}
		lay, err := t.layout(s)
		if err != nil {
			return err
		}
		list := t.sellable(s, lay)
		if len(list) == 0 {
			break
		}
		if _, err := t.openShop(ctx, []int{class}, t.npcPreset(class)); err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		_, err = t.sellTo(ctx, list, rep)
		if err != nil && first == nil {
			first = err
		}
	}
	if err := ctx.Err(); err != nil && parent.Err() == nil && first == nil {
		first = fmt.Errorf("town: sell chore capped at %d s", t.tune().SellMaxS)
	}
	if s, err := t.current(); err == nil && parent.Err() == nil {
		if lay, err := t.layout(s); err == nil {
			if left := t.sellable(s, lay); len(left) > 0 && first == nil {
				var names []string
				for _, u := range left {
					names = append(names, t.nameOf(u))
				}
				t.e.Log.Info("sell: left unsold", "items", names)
				first = fmt.Errorf("town: %d item(s) no vendor would buy", len(left))
			}
		}
	}
	return first
}

// logDurability writes the durability of every worn item (diagnosis of repairs that change nothing).
func (t *Town) logDurability(msg string, s *d2.Snapshot, g0 int) {
	var rows []string
	for _, u := range s.Inventory.Container(d2.ContEquipment) {
		if pct, ok := t.durabilityPct(u); ok {
			rows = append(rows, fmt.Sprintf("%s=%d%%", t.nameOf(u), pct))
		}
	}
	t.e.Log.Info(msg, "worn", rows, "goldBefore", g0, "gold", s.Player.Gold)
}

// repair opens the repair NPC's shop and presses Repair All, then checks durability rose.
func (t *Town) repair(ctx context.Context, a *Act, rep *ChoreReport) error {
	s0, err := t.current()
	if err != nil {
		return err
	}
	g0 := s0.Player.Gold
	low0, _ := t.lowestDurability(s0)
	if _, err := t.openShop(ctx, a.Repair, t.npcPreset(a.Repair...)); err != nil {
		return err
	}
	if err := t.e.In.Click(input.Left, RepairAll.X, RepairAll.Y); err != nil {
		_ = t.h.CloseAll(ctx)
		return err
	}
	s1, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool {
		low, _ := t.lowestDurability(s)
		return low > low0
	})
	if err != nil {
		return err
	}
	if !ok {
		t.logDurability("repair: not confirmed", s1, g0)
		return fmt.Errorf("town: repair not confirmed (durability %d%% unchanged)", low0)
	}
	rep.Repaired = true
	rep.Spent += g0 - s1.Player.Gold
	return nil
}

var _ = panel.Pt{}

// noteFutile remembers every trip reason a finished visit left standing (nothing sold, cannot afford, no stock), so
// NeedsTrip stops asking for them. Reasons that went away are forgotten. Percentages in a reason's text change
// between visits, so the key is the text up to the first digit.
func (t *Town) noteFutile(s *d2.Snapshot) {
	if t.futile == nil {
		t.futile = map[string]futileMark{}
	}
	now := t.e.Clock.Now()
	left := map[string]bool{}
	for _, r := range t.tripReasons(s) {
		left[r.key] = true
		if _, seen := t.futile[r.key]; !seen || !t.isFutile(r, s.Player.Gold) {
			t.futile[r.key] = futileMark{s.Player.Gold, now}
		}
	}
	for k := range t.futile {
		if !left[k] {
			delete(t.futile, k)
		}
	}
}

// tpTooPoor is the tpMinGold knob: a character under that gold buys potions before its first portal scroll.
func (t *Town) tpTooPoor(s *d2.Snapshot) bool {
	m := t.tune().TPMinGold
	return m > 0 && s.Player.Gold < m
}
