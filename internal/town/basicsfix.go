package town

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town/belt"
)

// BASICSFIX (docs/basics-audit.md). Policy numbers below are the d2-sme's EXP rule of thumb (docs/sme/play-economy.md),
// not measured on d2d; each sits behind a town.* knob.

// idScrollWanted is the idScrollNeed knob: an identify scroll is bought only while an unidentified set, rare, unique or
// crafted item sits in the pack (the SME: identify only likely valuables; a plain magic item under ~150 gold is sold as
// it is). 0 = the old rule, always up to IDScrollTarget.
func (t *Town) idScrollWanted(s *d2.Snapshot) bool {
	if t.tune().IDScrollNeed == 0 {
		return true
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if likelyValuable(u) {
			return true
		}
	}
	return false
}

// likelyValuable: an unidentified set, rare, unique or crafted item.
func likelyValuable(u d2.Unit) bool {
	if u.Item.Identified() {
		return false
	}
	switch u.Item.Quality {
	case d2.QualitySet, d2.QualityRare, d2.QualityUnique, d2.QualityCrafted:
		return true
	}
	return false
}

// sellUnidentified: with idScrollNeed on and no identify tool in the pack, an unidentified plain item is sold as it is
// (the SME: not worth a scroll) instead of waiting for a full pack.
func (t *Town) sellUnidentified(s *d2.Snapshot, u d2.Unit) bool {
	if t.tune().IDScrollNeed == 0 || likelyValuable(u) {
		return false
	}
	_, have := t.idTool(s)
	return !have
}

// identifyNeeded is the identify step's need check: items to identify, and a scroll in the pack or a reason to buy one.
func (t *Town) identifyNeeded(s *d2.Snapshot) bool {
	if len(t.needIdentify(s)) == 0 {
		return false
	}
	if _, have := t.idTool(s); have {
		return true
	}
	if t.tune().IdentifyTome == 0 {
		t.logNoIDTool()
		return false
	}
	return t.idScrollWanted(s)
}

// repairLatch remembers a repair that did not change durability.
type repairLatch struct {
	set       bool
	gold, low int
}

// repairLatched is the repairLatch knob: after a failed repair the chore stays quiet until the lowest durability changed
// or gold grew by FutileGoldStep (the SME: "it did not change, do not retry"; d2d logged 5 failed repairs in one run).
func (t *Town) repairLatched(s *d2.Snapshot, low int) bool {
	l := t.repairFail
	if !l.set || t.tune().RepairLatch == 0 {
		return false
	}
	if low != l.low || s.Player.Gold >= l.gold+max(t.tune().FutileGoldStep, 1) {
		t.repairFail = repairLatch{}
		return false
	}
	return true
}

// kitGold is the gold of the minimal kit: one portal scroll and the first healing potions (resource knobs).
func (t *Town) kitGold() int {
	r := &t.e.Tune.Resource
	return r.KitTPGold + r.HPFirst*r.KitPotionGold
}

// bankKeep is the gold the pack keeps while banking before the shop (goldKeep, never below the minimal kit).
func (t *Town) bankKeep(level int) int { return t.goldKeep(level) }

// needBank is the bankFirst knob: the surplus above the keep line goes to the stash before the shop, so the shop
// (tome, spare scrolls, extra potions) cannot spend it (audit row 11: 40 of 1500 gold banked at clvl 5).
func (t *Town) needBank(s *d2.Snapshot, plan ChorePlan) bool {
	return plan.Gold && t.bankFirst() && s.Player.Gold > max(t.bankKeep(s.Player.Level), plan.BossPurse)
}

// needBossPurse: a boss trip leaves with the boss purse when the stash has it. The kit withdrawal only covers the shopping list
// (a purse of 859 paid it, "withdraw not needed"), and the purse fell to 188 over five trips of Andariel's lair while the
// stash held 1.1 million (V live 2026-11-24): potions, a belt and the repair want gold between the trips too.
func (t *Town) needBossPurse(s *d2.Snapshot, plan ChorePlan) bool {
	if plan.BossPurse <= 0 || !plan.Gold || s.Player.Gold >= plan.BossPurse || s.Player.StashGold <= 0 {
		return false
	}
	_, can := t.e.In.(goldMover)
	return can
}

func (t *Town) bossPurse(ctx context.Context, a *Act, plan ChorePlan) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	amount := min(plan.BossPurse-s.Player.Gold, s.Player.StashGold)
	if amount <= 0 {
		return nil
	}
	return t.withdrawAmount(ctx, a, s, amount, "for the boss kit")
}

func (t *Town) bank(ctx context.Context, a *Act, plan ChorePlan) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	var first error
	if keep := max(t.bankKeep(s.Player.Level), plan.BossPurse); s.Player.Gold > keep {
		first = t.depositCapped(ctx, s, s.Player.Gold-keep)
	}
	if err := t.h.CloseAll(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

// tpAllowed is the potionsFirst knob (d2-sme order, user decision D3): the first town portal scroll waits for the kit
// potions and is bought only when the gold left covers the scroll plus TPScrollPotions potions. TPFirstLevel stays
// an extra clvl floor (default 0 = none). A held scroll or tome is never affected. Prices are the kit tuning's.
func (t *Town) tpAllowed(s *d2.Snapshot) bool {
	r := t.e.Tune.Resource
	return t.tpAllowedAt(s, r.KitTPGold, r.KitPotionGold)
}

// tpAllowedAt is tpAllowed with the scroll and cheapest healing potion prices of the vendor in front of us.
func (t *Town) tpAllowedAt(s *d2.Snapshot, scroll, potion int) bool {
	tu := t.tune()
	// goldfloor: potions first, the scroll waits until 2 healing potions are carried (not with potionsFirst 0, the old order)
	return tu.PotionsFirst == 0 || !t.poorKit(s) && s.Player.Level >= tu.TPFirstLevel && s.Player.Gold >= scroll+tu.TPScrollPotions*potion
}

// kitPotions is the healing potion count of the kit pass (resource HPFirst, at most a recovery floor).
func (t *Town) kitPotions(plan ChorePlan) int {
	n := t.e.Tune.Resource.HPFirst
	if plan.PotionFloor > 0 {
		n = min(n, plan.PotionFloor)
	}
	return max(n, 1)
}

// healingCarried is the healing and rejuvenation potions in belt and pack (the bot side counts the same set, codefix67 F6).
func (t *Town) healingCarried(s *d2.Snapshot) int {
	return t.supplyCount(s).Healing()
}

// supplyCount is the town side of the one supply counter (internal/supply): belt and pack potions of the snapshot.
func (t *Town) supplyCount(s *d2.Snapshot) supply.Count {
	var ps []supply.Potion
	if s != nil && s.Inventory != nil {
		for _, w := range []d2.Container{d2.ContBelt, d2.ContInventory} {
			for _, u := range s.Inventory.Container(w) {
				if u.Class < 0 || u.Class >= len(t.e.Data.Items) {
					continue
				}
				b := t.e.Data.Items[u.Class]
				var k supply.Kind
				switch belt.PotionKind(b) {
				case belt.HP:
					k = supply.HP
				case belt.MP:
					k = supply.MP
				case belt.RV:
					k = supply.RV
				default:
					continue
				}
				ps = append(ps, supply.Potion{Kind: k, Tier: belt.Tier(b), InBelt: w == d2.ContBelt})
			}
		}
	}
	return supply.CountOf(ps, supply.Floors{})
}

// needWithdraw is the BELTFIX5 knob-free rule: a character with no healing potion anywhere and less gold in the pack
// than the keep line, with gold in the stash, takes the difference out before the shop. Banking keeps the pack at
// the keep line, so the line is also where a withdrawal stops. Gate 1 20261103-0338: 1418 gold in the stash, 6 in the
// pack, 0 potions at clvl 11; the town package had no withdraw chore.
func (t *Town) needWithdraw(s *d2.Snapshot) bool {
	_, can := t.e.In.(goldMover)
	return can && s.Player.StashGold > 0 && s.Player.Gold < t.kitCost() && s.Player.Gold < t.bankKeep(s.Player.Level) && t.healingCarried(s) == 0
}

// kitCost is the minimal kit: one portal scroll and HPFirst healing potions at the kit tuning prices. A purse that
// pays it needs no stash round (codefix67 F6; the withdrawal still tops up to the keep line when it does run).
func (t *Town) kitCost() int {
	r := t.e.Tune.Resource
	return r.KitTPGold + max(r.HPFirst, 1)*r.KitPotionGold
}

// NeedWithdrawForTest exposes needWithdraw on the current snapshot.
func (t *Town) NeedWithdrawForTest() (bool, error) {
	s, err := t.current()
	if err != nil {
		return false, err
	}
	return t.needWithdraw(s), nil
}

// shopCost is the gold the shop pass of this plan would spend: missing town portal scrolls, a portal tome when none
// is owned (TPTomeMinGold is its price, S), the identify tome when the chore keeps one, and the belt's potion shortfall at
// the kit prices. The potion mix comes from the belt plan, so it follows the class.
func (t *Town) shopCost(s *d2.Snapshot, plan ChorePlan) int {
	r, tu := &t.e.Tune.Resource, t.tune()
	cost := 0
	if plan.Scrolls {
		n, hasTome := t.scrollCount(s, "tsc", "tbk")
		cost += max(tu.TPScrollTarget-n, 0) * r.KitTPGold
		if !hasTome {
			cost += tu.TPTomeMinGold
		}
		if tu.IdentifyTome != 0 && t.idScrollWanted(s) {
			if _, has := t.scrollCount(s, "isc", "ibk"); !has {
				cost += tu.TPTomeMinGold
			}
		}
	}
	if plan.Potions {
		for _, n := range t.potionsWanted(s) {
			cost += n * r.KitPotionGold
		}
	}
	return cost
}

// shopTarget is the pack gold a withdrawal for the shop aims at: the shopping list, never below the minimal kit.
func (t *Town) shopTarget(s *d2.Snapshot, plan ChorePlan) int {
	if t.PlanStage() >= 3 {
		return max(t.shopCost(s, plan), t.kitGold(), t.goldKeep(s.Player.Level)) // the purse target: withdraw the gap
	}
	if !t.withdrawShop() {
		return 0
	}
	return max(t.shopCost(s, plan), t.kitGold())
}

// needShopWithdraw: the stash has gold, the pack cannot pay the shopping list, and the input can move gold. The old
// rule only looked at an empty belt and ignored scrolls and tomes: a hero with 8 gold and 128k in the stash never
// withdrew (tpfix diagnosis 2026-11-18). A stash that reads 0 is logged, since it may be unreadable.
func (t *Town) needShopWithdraw(s *d2.Snapshot, plan ChorePlan) bool {
	stage3 := t.PlanStage() >= 3
	if !t.withdrawShop() || !plan.Gold || !plan.Potions && !plan.Scrolls {
		return false
	}
	if _, can := t.e.In.(goldMover); !can {
		return false
	}
	cost := t.shopCost(s, plan)
	if stage3 {
		cost = t.shopTarget(s, plan)
	}
	if cost == 0 || s.Player.Gold >= max(cost, t.kitGold()) {
		return false
	}
	if s.Player.StashGold <= 0 {
		t.e.Log.Info("town: no withdrawal for the shop, stash gold reads 0 (empty or unknown)", "gold", s.Player.Gold, "need", cost)
		return false
	}
	return true
}

func (t *Town) withdraw(ctx context.Context, a *Act, plan ChorePlan) error {
	if _, can := t.e.In.(goldMover); !can {
		return nil // a forced round runs every chore; this input cannot move gold
	}
	s, err := t.current()
	if err != nil {
		return err
	}
	// A withdrawal for the shop tops the purse up to the carry line (the line banking keeps), not only to the list: a
	// stash of 500000 gold behind a purse of 150 paid one portal scroll and three Minor potions per visit and the hero
	// walked home again (V live and lab, 2026-11-23). Banking leaves the pack at the same line, so nothing flips.
	target := max(t.shopTarget(s, plan), t.bankKeep(s.Player.Level)*b2i(t.needWithdraw(s) || t.needShopWithdraw(s, plan)))
	amount := min(target-s.Player.Gold, s.Player.StashGold)
	if amount <= 0 {
		return nil
	}
	return t.withdrawAmount(ctx, a, s, amount, "for the kit")
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ErrNoGoldMover: the input cannot move gold through the game's own command, so the stash cannot be emptied.
var ErrNoGoldMover = errors.New("town: no gold mover on this input")

// WithdrawGold takes up to amount gold out of the stash into the pack (the merc revive pays from the pack only: g2f seed
// 17 held the price in the stash and pack plus stash passed the bar, while banking kept the pack at the keep line).
// Nothing happens for amount <= 0 or an empty stash.
func (t *Town) WithdrawGold(ctx context.Context, amount int) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return ErrNotInTown
	}
	amount = min(amount, s.Player.StashGold)
	if amount <= 0 {
		return nil
	}
	if _, ok := t.e.In.(goldMover); !ok {
		return ErrNoGoldMover
	}
	return t.withdrawAmount(ctx, a, s, amount, "for the merc")
}

func (t *Town) withdrawAmount(ctx context.Context, a *Act, s *d2.Snapshot, amount int, why string) error {
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	first := t.e.In.(goldMover).MoveGold(amount, false)
	if first == nil {
		g0 := s.Player.Gold
		_, ok, werr := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return s.Player.Gold >= g0+amount })
		if werr != nil || !ok {
			first = fmt.Errorf("town: gold withdrawal of %d not confirmed (%v)", amount, werr)
		}
	} else {
		_ = t.e.In.Key("esc")
	}
	if first == nil {
		t.withdrew = true
		t.e.Log.Info("town: withdrew gold "+why, "amount", amount, "stash", s.Player.StashGold-amount)
	}
	if err := t.h.CloseAll(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

// logNoIDTool is the reason when the identify chore has no tool and may not buy one (town.identifyTome 0); once per visit.
func (t *Town) logNoIDTool() {
	if !t.idTomeLogged {
		t.idTomeLogged = true
		t.e.Log.Info("town: identify skipped", "reason", "no scroll/tome (buy on request)")
	}
}
