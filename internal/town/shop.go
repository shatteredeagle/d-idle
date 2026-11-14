package town

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/panel"
)

// Stat ids read from item stat lists. Durability 72/73 is VERIFIED LIVE as present on the starting staff
// (docs/mem.md Q4); quantity 70 is the standard ItemStatCost id and UNVERIFIED here.
const (
	statQuantity      = 70
	statDurability    = 72
	statMaxDurability = 73
	statDefence       = 31
)

// RepairAll is the pixel of the "repair all" button in the open vendor panel: the fourth of the four icons under
// the vendor grid (Buy, Sell, Repair, Repair All; the tooltips "Sell" at x 264 and "Repair" at x 317 were
// read off screenshots on Charsi). The click effect is verified live where docs/town.md says so.
var RepairAll = panel.Pt{X: 369, Y: 452}

// Vendor panel tabs at 800x600. A vendor shows one tab per non-empty stock page, left to right in page order
// (0 armor, 1 weapons, 2 more weapons, 3 misc), so the x of a tab depends on which pages the NPC has: Charsi shows
// Armor, Weapons, Weapons, Misc; an NPC without armor starts at the first slot. Tab y 75 and the 81 px pitch are
// read off screenshots (Charsi: tab labels at x 118, 200, 281, 361); whether a click on tab N of an NPC with fewer
// tabs lands is checked live per NPC, see docs/town.md.
const (
	tabSettleMs = 600
	tabY        = 75
	tabX0       = 118
	tabPitch    = 81
)

// tabPoint is the click point of the tab that shows stock page `page` for a vendor whose stock uses pages.
func (t *Town) tabPoint(page int) (panel.Pt, bool) {
	if t.fixedTabs {
		return panel.Pt{X: tabX0 + page*tabPitch, Y: tabY}, page >= 0 && page < 4
	}
	return tabPoint(t.pages, page)
}

func tabPoint(pages []int, page int) (panel.Pt, bool) {
	for i, p := range pages {
		if p == page {
			return panel.Pt{X: tabX0 + i*tabPitch, Y: tabY}, true
		}
	}
	return panel.Pt{}, false
}

// selectTab clicks the vendor tab that shows item it (once per tab change).
func (t *Town) selectTab(it d2.Unit) error {
	if t.tab == it.Item.Page+1 {
		return nil
	}
	p, ok := t.tabPoint(it.Item.Page)
	if !ok {
		return nil
	}
	if err := t.e.In.Click(input.Left, p.X, p.Y); err != nil {
		return err
	}
	t.tab = it.Item.Page + 1
	// VERIFIED LIVE (i3, Charsi): with the 150 ms click settle the buy right-click after a tab click landed on the
	// old tab (Arrows bought a Large Axe); give the vendor grid time to repaint.
	t.sleep(max(t.tune().ClickSettleMs, tabSettleMs))
	return nil
}

// items lists every item unit of the game (all owners). Seam for tests.
func (t *Town) allItems() ([]d2.Unit, error) {
	if t.itemsFn != nil {
		return t.itemsFn()
	}
	return t.e.Read.Units(d2.TypeItem)
}

// Stock returns the items offered by the vendor npc: item units whose container owner is the NPC.
func (t *Town) Stock(npc d2.Unit) ([]d2.Unit, error) {
	all, err := t.allItems()
	if err != nil {
		return nil, err
	}
	var out []d2.Unit
	for _, it := range all {
		if it.Item == nil || it.Mode != d2.ItemModeStored {
			continue
		}
		o, err := t.owner(it)
		if err != nil || o != npc.Addr {
			continue
		}
		out = append(out, it)
	}
	// The tab layout follows the stock actually loaded: right after the trade window opens the stock can still be
	// partial, and a stale page list made the tab click a silent no-op (VERIFIED LIVE i3: bow and arrows buys hit the
	// wrong tab).
	if len(out) > 0 {
		t.pages = t.pages[:0]
		seen := map[int]bool{}
		for _, u := range out {
			if !seen[u.Item.Page] {
				seen[u.Item.Page] = true
				t.pages = append(t.pages, u.Item.Page)
			}
		}
		sort.Ints(t.pages)
	}
	return out, nil
}

func (t *Town) qty(u d2.Unit) int {
	if t.statsFn == nil && t.e.Read == nil {
		return 1
	}
	// A single scroll, key or potion carries no quantity stat (implicit 1), but an EMPTY tome has no stat list at all
	// (LIVE, i4: unitStats failed for it, qty read as 1, and the town portal could not be opened for minutes).
	st, err := t.unitStats(u.Addr)
	if err == nil && st.Has(statQuantity) {
		return int(st.Get(statQuantity))
	}
	if b := t.base(u); b != nil && (b.Code == "tbk" || b.Code == "ibk") {
		return 0
	}
	return 1
}

func (t *Town) unitStats(addr uint32) (d2.Stats, error) {
	if t.statsFn != nil {
		return t.statsFn(addr)
	}
	return t.e.Read.UnitStats(addr)
}

// durabilityPct returns the durability of an item as 0..100; ok=false when the item has none or it cannot be read.
func (t *Town) durabilityPct(u d2.Unit) (int, bool) {
	st, err := t.unitStats(u.Addr)
	if err != nil || !st.Has(statMaxDurability) || st.Get(statMaxDurability) <= 0 {
		return 0, false
	}
	return int(st.Get(statDurability)) * 100 / int(st.Get(statMaxDurability)), true
}

func (t *Town) base(u d2.Unit) *data.ItemBase { return t.h.Base(u) }

// buyUnit right-clicks a vendor item and confirms the purchase by the gold drop. Returns the price paid.
func (t *Town) buyUnit(ctx context.Context, lay *panel.Layout, it d2.Unit) (int, error) {
	s, err := t.current()
	if err != nil {
		return 0, err
	}
	g0 := s.Player.Gold
	if b := t.base(it); b != nil && g0 < b.Cost/2 { // sure to fail: cheap items cost exactly Cost, gear 0.9-1.5x (VERIFIED LIVE)
		return 0, fmt.Errorf("%w: %s costs about %d, have %d", ErrNoMoney, t.nameOf(it), b.Cost, g0)
	}
	if err := t.selectTab(it); err != nil {
		return 0, err
	}
	var ok bool
	for try := 0; try < 2 && !ok; try++ { // a click right after a previous purchase can be swallowed
		if err := t.h.RightClick(it, lay.Vendor); err != nil {
			return 0, err
		}
		if s, ok, err = t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return s.Player.Gold < g0 }); err != nil {
			return 0, err
		}
	}
	if !ok {
		if b := t.base(it); b != nil && s.Player.Gold < 2*b.Cost { // the real price is not readable; be conservative
			return 0, fmt.Errorf("%w: %s (have %d)", ErrNoMoney, t.nameOf(it), s.Player.Gold)
		}
		return 0, fmt.Errorf("town: buying %s: gold did not change (no room, not enough gold, or the click missed)", t.nameOf(it))
	}
	paid := g0 - s.Player.Gold
	t.e.Log.Info("shop: bought", "item", t.nameOf(it), "price", paid, "gold left", s.Player.Gold)
	return paid, nil
}

func (t *Town) nameOf(u d2.Unit) string {
	if b := t.base(u); b != nil {
		if b.Display != "" {
			return b.Display
		}
		return b.Name
	}
	return fmt.Sprintf("class%d", u.Class)
}

// pickPotion chooses the best potion of kind k in stock: the highest tier within maxTier that costs at most budget.
func (t *Town) pickPotion(stock []d2.Unit, k belt.Kind, maxTier, budget int) (d2.Unit, bool) {
	var c []d2.Unit
	for _, u := range stock {
		b := t.base(u)
		if belt.PotionKind(b) == k && belt.Tier(b) <= maxTier && b.Cost <= budget {
			c = append(c, u)
		}
	}
	if len(c) == 0 {
		return d2.Unit{}, false
	}
	sort.SliceStable(c, func(i, j int) bool { return belt.Tier(t.base(c[i])) > belt.Tier(t.base(c[j])) })
	return c[0], true
}

// manaReserve is the gold a mana purchase must leave: one healing potion and one portal scroll (Gate 1 20261102: 180 of
// 300 gold went into mana potions and the next trips found the purse empty).
//
// Only the part of the kit still missing is reserved (codefix F3): with the scroll and the potions in the pack every
// coin may become mana, which the mana-bound Sorceress needs.
func (t *Town) manaReserve(s *d2.Snapshot) int {
	r := t.e.Tune.Resource
	res := 0
	if n, hasTome := t.scrollCount(s, "tsc", "tbk"); n == 0 && !hasTome {
		res += r.KitTPGold
	}
	if t.planHasHealing(s.Player) && t.healingCarried(s) < max(r.KitMinHP, 1) {
		res += r.KitPotionGold
	}
	return res
}

// buyPotions fills the belt to the plan. The shop window of the potion vendor must be open.
func (t *Town) buyPotions(ctx context.Context, npc d2.Unit, floor int, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return err
	}
	if len(stock) == 0 {
		return fmt.Errorf("town: no vendor stock readable for npc %d (owner lookup failed?)", npc.Class)
	}
	band := t.plan(s.Player)
	bs := belt.Read(t.e.Data, s.Inventory)
	want := t.potionsWanted(s)
	if floor > 0 {
		want[belt.HP] = max(min(want[belt.HP]+want[belt.RV], floor-bs.Count[belt.HP]-belt.PackCount(t.e.Data, s.Inventory, belt.HP)), 0)
		// A caster keeps a small mana reserve even while recovering: without it a broke Sorceress had no way to fight
		// back (Fresha soak: 0 mana potions for the whole recovery). Healing still comes first (buy order).
		mpHave := bs.Count[belt.MP] + belt.PackCount(t.e.Data, s.Inventory, belt.MP)
		want[belt.MP] = max(min(want[belt.MP], t.e.Tune.Resource.RecoveryManaFloor-mpHave), 0)
		want[belt.RV] = 0
	}
	if t.mpCapped(s.Player) {
		// per visit: the kit pass and the fill pass both come through here, so count what this visit bought already
		left := t.tune().MPCapEarly
		for name, n := range rep.Bought {
			if strings.Contains(name, "Mana Potion") {
				left -= n
			}
		}
		want[belt.MP] = max(min(want[belt.MP], left), 0)
	}
	t.ensureHealing(s, band, bs, want)
	t.e.Log.Info("shop: potion plan", "wantHP", want[belt.HP], "wantMP", want[belt.MP], "wantRV", want[belt.RV], "beltHP", bs.Count[belt.HP],
		"beltMP", bs.Count[belt.MP], "packHP", belt.PackCount(t.e.Data, s.Inventory, belt.HP), "packMP", belt.PackCount(t.e.Data, s.Inventory, belt.MP), "beltOther", bs.Other, "slots", bs.Slots, "gold", s.Player.Gold)
	// A caster buys its first mana potions before the healing fill: the fill alone ate the whole purse for hours (seed 11
	// of the Gate 2 rerun, belt mp = 0 from clvl 4; docs/research/lvl14plan.md TASK-A). Healing still comes first up to
	// HPFirst. The classes come from resource.ManaNeed, the belt columns from the plan: a build without a mana column has
	// want[MP] == 0 and shops exactly as before.
	r := t.e.Tune.Resource
	manaNeed := resource.ManaNeed(s.Player.Class)
	caster := manaNeed >= 2
	hpHave := bs.Count[belt.HP] + bs.Count[belt.RV] + belt.PackCount(t.e.Data, s.Inventory, belt.HP)
	mpHave := bs.Count[belt.MP] + belt.PackCount(t.e.Data, s.Inventory, belt.MP)
	// The player's own plan (a template or column override) outranks the class default: see ManaBuyRule.ForPlan.
	manaRule := resource.ManaBuy(manaNeed, resource.FromEnv(*t.e.Tune)).ForPlan(resource.SpecOfEnv(r), band.Capacity(belt.HP, bs.Slots)+band.Capacity(belt.RV, bs.Slots))
	// buyKind buys up to limit potions of kind k; it returns true when the visit is over (budget).
	buyKind := func(k belt.Kind, limit int) (bool, error) {
		src := k
		for bought := 0; want[k] > 0 && bought < limit; bought++ {
			if t.budgetHit(rep) {
				return true, nil
			}
			// Rejuvenation the shop does not sell becomes healing.
			if _, ok := t.pickPotion(stock, src, band.MaxTier(src), 1<<30); !ok && src == belt.RV {
				want[belt.HP] += want[belt.RV]
				want[belt.RV] = 0
				break
			}
			cur, err := t.current()
			if err != nil {
				return false, err
			}
			budget := cur.Player.Gold
			if t.potionCap > 0 {
				budget = min(budget, t.potionCap-rep.Spent)
			}
			it, ok := t.pickPotion(stock, src, band.MaxTier(src), budget)
			if ok && k == belt.MP && cur.Player.Gold-t.base(it).Cost < t.manaReserve(cur) {
				break // mana never eats the gold of one healing potion and one portal scroll
			}
			if ok && k == belt.MP {
				rt := resource.FromEnv(*t.e.Tune)
				if !resource.ManaBuyOK(manaRule, rt, cur.Player.Gold, t.base(it).Cost, t.healingCarried(cur), boughtMana(rep.Bought)) {
					t.e.Log.Info("shop: mana potion skipped", "gold", cur.Player.Gold, "price", t.base(it).Cost, "healing", t.healingCarried(cur), "manaNeed", manaNeed)
					break // little mana use: healing and the kit first
				}
			}
			if !ok {
				if _, sold := t.pickPotion(stock, src, band.MaxTier(src), 1<<30); sold {
					rep.Broke = true // sold, but not affordable
				}
				break
			}
			price, err := t.buyUnit(ctx, lay, it)
			if errors.Is(err, ErrNoMoney) {
				rep.Broke = true
				break
			}
			if err != nil {
				return false, err
			}
			rep.Spent += price
			rep.Bought[t.nameOf(it)]++
			want[k]--
		}
		return false, nil
	}
	type pass struct {
		k     belt.Kind
		limit int
	}
	passes := []pass{{belt.HP, 1 << 30}, {belt.MP, 1 << 30}, {belt.RV, 1 << 30}}
	if caster && want[belt.MP] > 0 {
		passes = append([]pass{{belt.HP, max(r.HPFirst-hpHave, 0)}, {belt.MP, max(r.MPFirst-mpHave, 0)}}, passes...)
	}
	boughtBefore := boughtMana(rep.Bought)
	for _, ps := range passes {
		if done, err := buyKind(ps.k, ps.limit); done || err != nil {
			return err
		}
	}
	if caster {
		t.logManaRestock(s, mpHave, boughtMana(rep.Bought)-boughtBefore)
	}
	return nil
}

// boughtMana counts the mana potions in a ChoreReport.Bought map.
func boughtMana(bought map[string]int) int {
	n := 0
	for name, c := range bought {
		if strings.Contains(name, "Mana Potion") {
			n += c
		}
	}
	return n
}

// logManaRestock is the live proof of the caster restock: one line per visit that bought mana, and one line per change of
// gold when a caster leaves the shop without any (the farm-gold recovery path takes over from there).
func (t *Town) logManaRestock(s *d2.Snapshot, mpHave, bought int) {
	if bought > 0 {
		t.noManaGold = 0
		t.e.Log.Info("town: restocked mana potions", "bought", bought, "mpBefore", mpHave, "mpAfter", mpHave+bought, "gold", s.Player.Gold)
		return
	}
	if mpHave == 0 && t.noManaGold != s.Player.Gold+1 {
		t.noManaGold = s.Player.Gold + 1
		t.e.Log.Info("caster: no mana, gold " + strconv.Itoa(s.Player.Gold))
	}
}

// buyFirstTP buys one town portal scroll when the pack holds none and the gold allows (resource.BuyKit, priced from
// the vendor's actual stock), so the potions bought next only get what is left.
//
// beforeKit is the pass ahead of the kit potions, for a character that carries a tome with nothing in it (the poor
// character without a tome keeps the potions-first rule): the scroll goes first
// as long as the gold still covers it plus TPScrollPotions healing potions (NAVEXIT21: after the kit potions the
// 284 gold of Gate 1 ms1 seed 17 no longer covered the scroll, and the trip home had to be walked).
func (t *Town) buyFirstTP(ctx context.Context, npc d2.Unit, rep *ChoreReport, beforeKit bool) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	// An empty tome is no portal source (NAVEXIT21, Gate 1 ms1 seed 17: clvl 9, empty tome, 284 gold, no scroll bought,
	// the next trip had to walk): only a tome or scrolls that count above zero make the first scroll unnecessary.
	n, hasTome := t.scrollCount(s, "tsc", "tbk")
	if n > 0 || t.tomeQty(s) > 0 || t.tpTooPoor(s) || beforeKit && !hasTome {
		return nil
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return err
	}
	var su d2.Unit
	var price, potion int
	for _, u := range stock {
		b := t.base(u)
		if b == nil {
			continue
		}
		if b.Code == "tsc" && price == 0 {
			su, price = u, b.Cost
		}
		if belt.PotionKind(b) == belt.HP && b.Cost > 0 && (potion == 0 || b.Cost < potion) {
			potion = b.Cost
		}
	}
	if potion == 0 {
		potion = t.e.Tune.Resource.KitPotionGold
	}
	if beforeKit {
		if s.Player.Gold < price+t.tune().TPScrollPotions*potion {
			return nil
		}
	} else if !t.tpAllowedAt(s, price, potion) {
		return nil
	}
	if price <= 0 || !resource.BuyKit(resource.Input{Gold: s.Player.Gold}, resource.Tuning{KitTPGold: price}).TP {
		return nil
	}
	if !tpBuyAllowed(s.Player) {
		return nil
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	p, err := t.buyUnit(ctx, lay, su)
	if errors.Is(err, ErrNoMoney) {
		rep.Broke = true
		return nil
	}
	if err != nil {
		return err
	}
	rep.Spent += p
	rep.Bought[t.nameOf(su)]++
	t.tpBought.Add(1)
	return nil
}

// scrollNeeds counts what the pack holds of a scroll code and its tome, and which tome code applies.
func (t *Town) scrollCount(s *d2.Snapshot, scroll, tome string) (have int, hasTome bool) {
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		b := t.base(u)
		switch {
		case b == nil:
		case b.Code == scroll:
			have++
		case b.Code == tome:
			hasTome = true
			if !(t.portalEmpty && tome == "tbk") { // an empty source reported by the game outranks the stat read
				have += t.qty(u)
			}
		}
	}
	return have, hasTome
}

// tomeQty is the town portal tome's stat-read quantity, ignoring portalEmpty.
func (t *Town) tomeQty(s *d2.Snapshot) int {
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if b := t.base(u); b != nil && b.Code == "tbk" {
			return t.qty(u)
		}
	}
	return 0
}

// buyScrolls tops up town portal and identify scrolls (and buys a tome when affordable).
func (t *Town) buyScrolls(ctx context.Context, npc d2.Unit, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return err
	}
	find := func(code string) (d2.Unit, bool) {
		for _, u := range stock {
			if b := t.base(u); b != nil && b.Code == code {
				return u, true
			}
		}
		return d2.Unit{}, false
	}
	type job struct {
		scroll, tome string
		target       int
	}
	for _, j := range []job{{"tsc", "tbk", t.tune().TPScrollTarget}, {"isc", "ibk", t.tune().IDScrollTarget}} {
		if j.target <= 0 {
			continue
		}
		cur, err := t.current()
		if err != nil {
			return err
		}
		have, hasTome := t.scrollCount(cur, j.scroll, j.tome)
		if !hasTome && j.scroll == "tsc" && cur.Player.Gold >= t.tune().TPTomeMinGold && t.aboveReserve(cur.Player.Gold, t.tomeCost(find)) {
			tu, ok := find(j.tome)
			if !ok {
				t.e.Log.Warn("shop: no tome in stock", "gold", cur.Player.Gold, "stock", len(stock))
			} else if price, err := t.buyUnit(ctx, lay, tu); err != nil {
				t.e.Log.Warn("shop: tome not bought", "err", err, "gold", cur.Player.Gold)
			} else {
				rep.Spent += price
				rep.Bought[t.nameOf(tu)]++
				hasTome = true
			}
		}
		if hasTome && j.target < 5 {
			j.target = 5
		}
		// Poor: one scroll gets the character home, the rest of the gold is for potions (G1 attempt 8 burned 100 gold a trip).
		// Under twice the floor at most two spares: the Fresha i5 soak spent 12 scrolls (1200 gold) at clvl 4-6 while the belt starved.
		if floor := t.e.Tune.Resource.TripGoldFloor; j.scroll == "tsc" && floor > 0 {
			switch {
			case !t.restockFree(cur, floor):
				// g2e F2b: a poor character that already carries its potions keeps TPReserve scrolls, so a dive does
				// not start with the only scroll spent on the way back (seed 11 reached area 11 with none). With no
				// potion carried the potions still come first: one scroll.
				j.target = min(j.target, max(1, min(t.tune().TPReserve, 1+carriedPotionsBonus(t, cur))))
			case cur.Player.Gold < 2*floor:
				j.target = min(j.target, 2)
			}
		}
		// BASICSFIX: identify scrolls only for a likely valuable; the first portal scroll waits for potionsFirst
		if j.scroll == "isc" && !t.idScrollWanted(cur) || j.scroll == "tsc" && have == 0 && !hasTome && !t.tpAllowed(cur) {
			continue
		}
		su, ok := find(j.scroll)
		if !ok {
			continue
		}
		if j.scroll == "tsc" && t.portalEmpty && hasTome && t.tomeQty(cur) > 0 {
			q := t.tomeQty(cur) // the stat read counts; only prove the source works
			have, j.target = q, q+emptyRestock
			if q >= tomeCap {
				j.target = have
			}
		}
		for have < j.target {
			if t.budgetHit(rep) {
				return nil
			}
			cur, err := t.current()
			if err != nil {
				return err
			}
			b := t.base(su)
			if cur.Player.Gold-b.Cost < t.tune().MinGoldReserve && have > 0 || have > 0 && !t.aboveReserve(cur.Player.Gold, b.Cost) {
				break
			}
			price, err := t.buyUnit(ctx, lay, su)
			if errors.Is(err, ErrNoMoney) {
				rep.Broke = true
				break
			}
			if err != nil {
				return err
			}
			rep.Spent += price
			rep.Bought[t.nameOf(su)]++
			if j.scroll == "tsc" {
				t.tpBought.Add(1)
			}
			have++
		}
	}
	return t.buyKeys(ctx, lay, find, rep)
}

// keyCount is the number of keys in the pack (stackable: quantity stat per stack).
func (t *Town) keyCount(s *d2.Snapshot) int {
	n := 0
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if b := t.base(u); b != nil && b.Code == "key" {
			n += t.qty(u)
		}
	}
	return n
}

// buyKeys tops the pack up to KeyTarget keys (0 = never). A purchase may add more than one key.
func (t *Town) buyKeys(ctx context.Context, lay *panel.Layout, find func(string) (d2.Unit, bool), rep *ChoreReport) error {
	target := t.tune().KeyTarget
	if target <= 0 {
		return nil
	}
	ku, ok := find("key")
	if !ok {
		return nil
	}
	for i := 0; i < target; i++ {
		cur, err := t.current()
		if err != nil {
			return err
		}
		if t.keyCount(cur) >= target || cur.Player.Gold-t.base(ku).Cost < t.tune().MinGoldReserve {
			return nil
		}
		price, err := t.buyUnit(ctx, lay, ku)
		if errors.Is(err, ErrNoMoney) {
			rep.Broke = true
			return nil
		}
		if err != nil {
			return err
		}
		rep.Spent += price
		rep.Bought[t.nameOf(ku)]++
	}
	return nil
}

// sellable returns the pack items to sell. Items stay out of the list when the pickit wants them. When the
// pack is nearly full, items the pickit would drop are sold too.
func (t *Town) sellable(s *d2.Snapshot, lay *panel.Layout) []d2.Unit {
	full := t.h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory).UsedPct() >= t.tune().TripPackUsedPct
	return t.sellableIf(s, full)
}

// sellableIf is sellable with the "pack is full" decision made by the caller (MakeRoom forces it).
func (t *Town) sellableIf(s *d2.Snapshot, full bool) []d2.Unit {
	var out []d2.Unit
	t.ReconcileMarks(s)
	kept := map[string]int{}
	rvOver := t.rejuvOverCap(s)
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.Item.Flags&d2.FlagStartItem != 0 {
			continue
		}
		// The player's marks come before the pickit: "keep" is never sold, "sell" goes at this visit unless the item is
		// one that must never be sold (quest items, the Cube, keys, tomes).
		switch t.markOf(u) {
		case "keep":
			continue
		case "sell":
			if NeverSellReason(t.base(u), u) == "" {
				out = append(out, u)
				continue
			}
		}
		if t.sellMarksOnly {
			continue
		}
		v := t.judge.EvaluateUnit(u)
		if v.NeedID && !u.Item.Identified() {
			// The identify step ran before this and could not read it (no scroll, no gold): a full pack must not hold
			// it for ever. Unidentified jewelry sells as is (SME: no ID tome early); charms are stashed instead.
			if !((full || t.sellUnidentified(s, u)) && v.Prio < t.e.Tune.Loot.ValuableMinPrio) {
				continue
			}
			out = append(out, u)
			continue
		}
		if rvOver[u.ID] {
			t.e.Log.Info("rejuvenation sold", "reason", "over cap", "cap", t.e.Tune.Loot.RejuvSpare, "code", t.base(u).Code)
			out = append(out, u)
		} else if v.Action == loot.Sell || v.Action == loot.Drop && full && v.Tier != "S" {
			out = append(out, u)
		} else if full && t.spareSurplus(u, v, kept) {
			out = append(out, u)
		}
	}
	return out
}

// rejuvOverCap returns the pack rejuvenation potions beyond Loot.RejuvSpare counting the belt, regular ones first so
// Full Rejuvenation stays. Keyed by unit ID.
func (t *Town) rejuvOverCap(s *d2.Snapshot) map[uint32]bool {
	if s.Inventory == nil {
		return nil
	}
	var pack []d2.Unit
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.Item.Flags&d2.FlagStartItem == 0 && belt.PotionKind(t.base(u)) == belt.RV {
			pack = append(pack, u)
		}
	}
	keep := t.e.Tune.Loot.RejuvSpare - belt.Read(t.e.Data, s.Inventory).Count[belt.RV]
	if len(pack) <= max(keep, 0) {
		return nil
	}
	sort.SliceStable(pack, func(i, j int) bool { return belt.Tier(t.base(pack[i])) > belt.Tier(t.base(pack[j])) })
	over := map[uint32]bool{}
	for _, u := range pack[max(keep, 0):] {
		over[u.ID] = true
	}
	return over
}

// spareSurplus reports whether u is a spare potion or identify scroll beyond SpareKeep of its kind. The starter pickit keeps
// these (tier C, "already in the pack") but nothing belts or reads them, so a pack full of them stays full for good (G1
// attempt 10: 97 percent full, nothing sellable, a superior Cap left on the ground). kept counts what stays per code.
func (t *Town) spareSurplus(u d2.Unit, v loot.Verdict, kept map[string]int) bool {
	b := t.base(u)
	if b == nil || v.Action != loot.Keep || v.Tier != "C" || v.Prio != 0 || !t.isSupply(b) {
		return false
	}
	kept[b.Code]++
	if t.e.Data.IsA(b.Type, "rpot") {
		return false // rejuvOverCap decides, regardless of how full the pack is
	}
	return kept[b.Code] > t.spareAllowance(b.Code)
}

// spareAllowance is how many tier C items of one code a full pack keeps: keys only as many as KeyTarget (0: none, the
// policy buys none, SME play-economy), one tome of each kind (the one we hold reads as tier C too), SpareKeep else.
func (t *Town) spareAllowance(code string) int {
	switch code {
	case "key":
		return t.tune().KeyTarget
	case "tbk", "ibk":
		return 1
	}
	return t.tune().SpareKeep
}

// sellTo sells every item in list the open vendor buys. Each sale is confirmed by the item leaving the pack
// and the gold rising; items the vendor refuses (nothing changes) are returned in refused.
func (t *Town) sellTo(ctx context.Context, list []d2.Unit, rep *ChoreReport) (refused []d2.Unit, err error) {
	s, err := t.current()
	if err != nil {
		return nil, err
	}
	lay, err := t.layout(s)
	if err != nil {
		return nil, err
	}
	t.logWorn("sell: worn before")
	defer t.logWorn("sell: worn after")
	for _, u := range list {
		cur, err := t.current()
		if err != nil {
			return refused, err
		}
		fresh, ok := findIn(cur.Inventory, u.ID)
		if !ok {
			continue
		}
		// Only pack items are sold. A worn item (or the start kit) never reaches the vendor from here: findIn looks at
		// every unit, so a stale list or a reused id must not turn into a click on a worn slot (KITSTALL2).
		if fresh.Item == nil || fresh.Item.Where != d2.ContInventory || fresh.Item.Flags&d2.FlagStartItem != 0 {
			t.e.Log.Warn("sell: refused, not a sellable pack item", "item", t.nameOf(fresh))
			continue
		}
		g0 := cur.Player.Gold
		sold := func(s *d2.Snapshot) bool {
			_, still := findIn(s.Inventory, u.ID)
			return !still && s.Player.Gold > g0
		}
		var after *d2.Snapshot
		ok = false
		if !t.sellByCursor {
			if err := t.h.RightClick(fresh, lay.Inventory); err != nil {
				return refused, err
			}
			if after, ok, err = t.h.Wait(ctx, t.uiTimeout(), sold); err != nil {
				return refused, err
			}
		}
		if !ok {
			// Fallback: pick the item up and drop it on the vendor grid (the classic way to sell).
			if err := t.h.Take(ctx, fresh, lay.Inventory); err == nil {
				c := lay.Vendor.Cell(lay.Vendor.Cols/2, lay.Vendor.Rows-2)
				_ = t.e.In.Click(input.Left, c.X, c.Y)
				if after, ok, err = t.h.Wait(ctx, t.uiTimeout(), sold); err != nil {
					return refused, err
				}
				if ok {
					t.sellByCursor = true
				} else {
					_ = t.h.ClearCursor(ctx)
				}
			}
		}
		if !ok {
			refused = append(refused, u)
			continue
		}
		rep.Sold++
		if k, ok := t.marks.(markKeeper); ok {
			k.Forget(u.ID)
		}
		rep.Earned += after.Player.Gold - g0
		t.e.Log.Info("sell: sold", "item", t.nameOf(fresh), "gold", after.Player.Gold-g0)
	}
	return refused, nil
}

func findIn(inv *d2.Inventory, id uint32) (d2.Unit, bool) {
	for _, u := range inv.Items {
		if u.ID == id {
			return u, true
		}
	}
	return d2.Unit{}, false
}

// isSupply is a potion of any kind (the pickit's "spare supplies" types), a town scroll, a tome or a key.
func (t *Town) isSupply(b *data.ItemBase) bool {
	switch b.Code {
	case "isc", "tsc", "key", "tbk", "ibk":
		return true
	}
	for _, ty := range []string{"hpot", "mpot", "rpot", "spot", "apot", "wpot"} {
		if t.e.Data.IsA(b.Type, ty) {
			return true
		}
	}
	return false
}

// antidoteCode is the antidote potion (V: combat/catalog.go classifyCode "yps").
const antidoteCode = "yps"

// antidoteCount counts antidote potions in belt and pack.
func (t *Town) antidoteCount(s *d2.Snapshot) int {
	n := 0
	for _, c := range []d2.Container{d2.ContBelt, d2.ContInventory} {
		for _, u := range s.Inventory.Container(c) {
			if b := t.base(u); b != nil && b.Code == antidoteCode {
				n++
			}
		}
	}
	return n
}

func (t *Town) needAntidotes(plan ChorePlan, s *d2.Snapshot) bool {
	return plan.Antidotes > 0 && s.Inventory != nil && t.antidoteCount(s) < plan.Antidotes
}

// buyAntidotes tops antidote potions up to target at the potion vendor. It never spends below the gold reserve and a
// vendor without antidotes (or a failed buy) is logged and ends the attempt: the gate then passes without them once
// the character cannot afford them, it does not loop (task.antidoteGold).
func (t *Town) buyAntidotes(ctx context.Context, npc d2.Unit, target int, rep *ChoreReport) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return err
	}
	var it d2.Unit
	found := false
	for _, u := range stock {
		if b := t.base(u); b != nil && b.Code == antidoteCode {
			it, found = u, true
			break
		}
	}
	if !found {
		t.e.Log.Warn("shop: no antidote in stock", "stock", len(stock))
		return nil
	}
	for {
		cur, err := t.current()
		if err != nil {
			return err
		}
		if t.antidoteCount(cur) >= target || cur.Player.Gold-t.base(it).Cost < t.tune().MinGoldReserve {
			return nil
		}
		price, err := t.buyUnit(ctx, lay, it)
		if errors.Is(err, ErrNoMoney) {
			rep.Broke = true
			return nil
		}
		if err != nil {
			return err
		}
		rep.Spent += price
		rep.Bought[t.nameOf(it)]++
	}
}

// mpCapped says the early mana cap applies to this character (goldfloor lever B).
func (t *Town) mpCapped(p *d2.Player) bool {
	return p != nil && p.Level < t.tune().MPCapLevel
}

// mpPriceEstimate is the minor mana potion price used to size a planned purchase (V: about 60 gold, docs/goldfloor.md).
const mpPriceEstimate = 60

// PlannedSpend estimates what a shop visit would spend now: the belt plan's missing potions (mana capped early) and the
// first portal scroll when one is allowed, never more than the gold. It is the trip floor's purchase size (goldfloor).
func (t *Town) PlannedSpend(s *d2.Snapshot) int {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return 0
	}
	r := t.e.Tune.Resource
	want := t.potionsWanted(s)
	if t.mpCapped(s.Player) {
		want[belt.MP] = min(want[belt.MP], t.tune().MPCapEarly)
	}
	spend := (want[belt.HP]+want[belt.RV])*r.KitPotionGold + want[belt.MP]*mpPriceEstimate
	if n, hasTome := t.scrollCount(s, "tsc", "tbk"); n == 0 && !hasTome && t.tpAllowed(s) {
		spend += r.KitTPGold
	}
	return min(spend, s.Player.Gold)
}

// poorKit is goldfloor's "poor" test: gold under the kit line and fewer than 2 healing potions carried.
func (t *Town) poorKit(s *d2.Snapshot) bool {
	if t.tune().TripMinSpend <= 0 || s.Player.Gold >= t.e.Tune.Resource.RecoveryKitGold {
		return false
	}
	bs := belt.Read(t.e.Data, s.Inventory)
	return bs.Count[belt.HP]+bs.Count[belt.RV]+belt.PackCount(t.e.Data, s.Inventory, belt.HP) < 2
}

// logWorn names every worn item. sellTo logs it before and after each vendor so a lost weapon or glove (KITSTALL2, U:
// how it left the body is not in the logs) shows which vendor step took it.
func (t *Town) logWorn(msg string) {
	s, err := t.current()
	if err != nil || s.Inventory == nil {
		return
	}
	var names []string
	for _, u := range s.Inventory.Container(d2.ContEquipment) {
		names = append(names, fmt.Sprintf("%d:%s", u.Item.Body, t.nameOf(u)))
	}
	t.e.Log.Info(msg, "worn", names, "gold", s.Player.Gold)
}

// ensureHealing keeps one healing potion in the plan when belt and pack hold none. Rejuvenation is not a healing potion
// for the BELT gate, so a belt of mana and Rejuvenation columns (no free slot, empty want) still buys one; it stays in
// the pack. Gate 2 g2b seed 13: 3 mana + 1 Rejuvenation, gold 88, shop plan "wantHP:0", "bought:{}" for 10 minutes.
func (t *Town) ensureHealing(s *d2.Snapshot, band belt.Band, bs belt.State, want map[belt.Kind]int) {
	if bs.Count[belt.HP]+belt.PackCount(t.e.Data, s.Inventory, belt.HP) == 0 && band.Capacity(belt.HP, bs.Slots) > 0 && want[belt.HP] < 1 {
		want[belt.HP] = 1
	}
}

// restockFree is point t1.restock (docs/t0-slice.md) at the one place the trip gold floor limits town spending: the
// purse covers the belt potions as well as the scrolls. The legacy answer is gold at or above the floor; with the
// ladder on a learned answer for the context (belt state, gold band) replaces it, in either direction.
func (t *Town) restockFree(s *d2.Snapshot, floor int) bool {
	legacy := s.Player.Gold >= floor
	if l := ladder.Active(); l != nil {
		v, err := l.RestockWanted(ladder.HeroFrom(s.Player).With(t.beltFacts(s)...), t.beltBand(s), s.Player.Gold, legacy)
		if err == nil {
			return v
		}
		l.Fault(err)
	}
	return legacy
}

// beltBand is the t1.town.trigger / t1.restock belt field: healing and rejuvenation potions in the belt against the
// plan's capacity for them (U: the band edges "half" and "low" are this function's choice, not measured).
func (t *Town) beltBand(s *d2.Snapshot) string {
	bs := belt.Read(t.e.Data, s.Inventory)
	return ladder.BeltBand(bs.Count[belt.HP]+bs.Count[belt.RV], t.plan(s.Player).Capacity(belt.HP, bs.Slots))
}

// beltFacts are the belt numbers beltBand classes, for the question card (display only).
func (t *Town) beltFacts(s *d2.Snapshot) []any {
	bs := belt.Read(t.e.Data, s.Inventory)
	return []any{"have", bs.Count[belt.HP] + bs.Count[belt.RV], "want", t.plan(s.Player).Capacity(belt.HP, bs.Slots)}
}

// tpBuyAllowed is point t1.tp.lost (docs/t0-slice.md), asked once BuyKit allows the purchase: with the ladder on a
// learned "walk" vetoes it; with it off, or with nothing learned, this is true.
func tpBuyAllowed(p *d2.Player) bool {
	l := ladder.Active()
	if l == nil {
		return true
	}
	buy, err := l.TPBuyWanted(ladder.HeroFrom(p), true)
	if err != nil {
		l.Fault(err)
		return true
	}
	return buy
}

// carriedPotionsBonus is the number of extra scrolls a poor character may afford: TPReserve-1 once its minimal healing
// kit is carried, 0 before.
func carriedPotionsBonus(t *Town, s *d2.Snapshot) int {
	if t.planHasHealing(s.Player) && t.healingCarried(s) < max(t.e.Tune.Resource.KitMinHP, 1) {
		return 0
	}
	return max(t.tune().TPReserve-1, 0)
}

// TPCount is the number of town portal scrolls the bot counts as in hand (scrolls in the pack plus the tome's
// quantity, the same count the shop and trip decisions use). Minute line field tp.
func (t *Town) TPCount(s *d2.Snapshot) int {
	if s == nil || s.Inventory == nil {
		return 0
	}
	n, _ := t.scrollCount(s, "tsc", "tbk")
	return n
}

// TakeTPBought returns the town portal scrolls bought since the last call (minute line field tpBuy).
func (t *Town) TakeTPBought() int { return int(t.tpBought.Swap(0)) }

// aboveReserve says whether spending cost of gold leaves the merc revive money (ChorePlan.ReserveGold) in the purse. Only
// optional buys ask: the kit potions and the first portal scroll are survival and never wait for the merc.
func (t *Town) aboveReserve(gold, cost int) bool { return gold-cost >= t.reserveGold }

// tomeCost is the vendor price of the town portal tome, 0 when it is not in stock (the caller then logs it).
func (t *Town) tomeCost(find func(string) (d2.Unit, bool)) int {
	if u, ok := find("tbk"); ok {
		if b := t.base(u); b != nil {
			return b.Cost
		}
	}
	return 0
}
