package leveler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town/hand"
)

// Vendor opens a gear vendor's trade window and lists what it sells. The Townsman knows how to walk to NPCs and
// pick the Trade line, so cmd/d2bot adapts it (adapters.go); the leveler does the buying itself so it can choose
// the exact item.
type Vendor interface {
	// GearVendors lists NPC classes that sell weapons and armor in the given act (1..5), best first.
	GearVendors(act int) []int
	// OpenTrade walks to the NPC, talks and opens its trade window (UI flag d2.UIShop).
	OpenTrade(ctx context.Context, npcClass int) (d2.Unit, error)
	// Stock lists the items the open vendor offers.
	Stock(npc d2.Unit) ([]d2.Unit, error)
}

// RoomMaker is the optional part of a Vendor that frees pack space (sell junk, then stash). The vendor window is open
// when it is called; it may close it (stashing), after which the leveler reopens the trade. junk lists pack gear the
// leveler knows it does not need.
type RoomMaker interface {
	PackFits(s *d2.Snapshot, w, h int) (bool, error)
	MakeRoom(ctx context.Context, w, h int, junk []d2.Unit) (bool, error)
}

// ErrNoRoom: the bought item would not fit in the pack even after making room.
var ErrNoRoom = errors.New("leveler: no room in the pack for the item")

// ShopTabs are the vendor panel tab click points at 800x600 keyed by the items' Page (0 armor, 1 weapons, 3 misc).
// Measured by TOWN (docs/town.md); armor and weapons VERIFIED LIVE there, misc UNVERIFIED.
var ShopTabs = map[int]Pt{0: {X: 118, Y: 75}, 1: {X: 200, Y: 75}, 3: {X: 361, Y: 75}}

// PlanBuys picks vendor items worth buying with budget gold, best gain per gold first. Items already in the pack are
// assumed worn first (Equip will do that), and each pick is assumed worn afterwards, so a second item for the same slot
// must beat the first. Pure.
func PlanBuys(eq map[int]Cand, pack, stock []Cand, budget int) []Cand {
	eq2 := map[int]Cand{}
	for k, v := range eq {
		eq2[k] = v
	}
	for _, sw := range ChooseGear(eq, pack) {
		eq2[sw.Slot] = sw.Item
	}
	var buys []Cand
	left := append([]Cand(nil), stock...)
	for {
		var affordable []Cand
		for _, c := range left {
			if c.Usable && !c.OffClass && c.Price > 0 && c.Price <= budget {
				affordable = append(affordable, c)
			}
		}
		best, bestRatio := -1, 0.0
		var bestSwap Swap
		for _, c := range affordable { // each item on its own: value is gain per gold, not the best score of a slot
			for _, sw := range ChooseGear(eq2, []Cand{c}) {
				if ratio := sw.Gain / float64(sw.Item.Price); ratio > bestRatio {
					best, bestRatio, bestSwap = 1, ratio, sw
				}
			}
		}
		if best < 0 {
			return buys
		}
		buys = append(buys, bestSwap.Item)
		budget -= bestSwap.Item.Price
		eq2[bestSwap.Slot] = bestSwap.Item
		for i, c := range left {
			if c.Unit.ID == bestSwap.Item.Unit.ID {
				left = append(left[:i], left[i+1:]...)
				break
			}
		}
	}
}

// ShopReport is what one Shop call did.
type ShopReport struct {
	Bought []string
	Spent  int
	Err    error
}

// Shop visits the act's gear vendors and buys items that beat what the character wears or carries, within budget
// gold. It does not equip: call Equip afterwards (the bot does). Verified per purchase by the gold drop.
func (l *Leveler) Shop(ctx context.Context, v Vendor, budget int) (rep ShopReport) {
	if v == nil || budget <= 0 {
		return
	}
	h := l.newHand()
	s, err := h.Fresh()
	if err != nil {
		rep.Err = err
		return
	}
	if !s.Player.InTown() {
		rep.Err = errors.New("leveler: shopping needs the town")
		return
	}
	if r := l.kitReserve(s); r > 0 {
		capped := resource.GearBudget(budget, s.Player.Gold, r)
		l.logf("kit reserve", "reserve", r, "gold", s.Player.Gold, "budget", budget, "capped", capped)
		budget = capped
		if budget <= 0 {
			return
		}
	}
	for _, class := range v.GearVendors(s.Player.Act + 1) {
		if budget <= 0 {
			break
		}
		if err := l.shopAt(ctx, h, v, class, &budget, &rep); err != nil {
			l.e.Log.Warn("leveler: vendor skipped", "npc", class, "err", err)
			rep.Err = err
		}
	}
	if err := h.CloseAll(ctx); err != nil {
		l.e.Log.Warn("leveler: closing the vendor failed", "err", err)
	}
	return
}

// kitReserve is the gold the minimal kit (TP scroll, belt potions) still lacks, read from the snapshot.
func (l *Leveler) kitReserve(s *d2.Snapshot) int {
	if l.e.Tune == nil || l.e.Data == nil || s.Inventory == nil {
		return 0
	}
	hasTP, hp, mp := false, 0, 0
	for _, u := range s.Inventory.Items {
		if u.Item == nil || u.Class < 0 || u.Class >= len(l.e.Data.Items) {
			continue
		}
		b := l.e.Data.Items[u.Class]
		switch {
		case u.Item.Where == d2.ContInventory && (b.Code == "tsc" || b.Code == "tbk"):
			hasTP = true
		case u.Item.Where == d2.ContBelt && (b.Type == "hpot" || b.Type == "rpot"):
			hp++
		case u.Item.Where == d2.ContBelt && b.Type == "mpot":
			mp++
		}
	}
	return resource.KitShortfall(hasTP, hp, mp, resource.FromEnv(*l.e.Tune))
}

func (l *Leveler) shopAt(ctx context.Context, h *hand.Hand, v Vendor, class int, budget *int, rep *ShopReport) error {
	npc, err := v.OpenTrade(ctx, class)
	if err != nil {
		return fmt.Errorf("open trade: %w", err)
	}
	fails := 0
	for round := 0; round < 6; round++ {
		s, err := h.Fresh()
		if err != nil {
			return err
		}
		items, err := v.Stock(npc)
		if err != nil {
			return err
		}
		eq, pack := l.gearState(s)
		var stock []Cand
		for _, u := range items {
			if c := l.cand(s.Player, u); c.Slots != nil && !l.badEverywhere(u.ID, c.Slots, s.Player.Level) {
				stock = append(stock, c)
			}
		}
		buys := PlanBuys(eq, pack, stock, *budget)
		if len(buys) == 0 {
			return nil
		}
		b := buys[0] // one at a time: the stock list and the pack change with every purchase
		l.logf("buying", "item", b.Name, "score", fmt.Sprintf("%.1f", b.Score), "estimate", b.Price, "budget", *budget)
		paid, err := l.buyWithRoom(ctx, h, v, class, &npc, b)
		if err != nil {
			l.e.Log.Warn("leveler: purchase failed", "item", b.Name, "err", err)
			l.markBadSwap(Swap{Item: b, Slot: -1})
			if fails++; fails >= 2 { // LIVE (i4): every purchase at a vendor failed the same way, 4 s each; stop after two
				return fmt.Errorf("purchases keep failing at this vendor: %w", err)
			}
			continue
		}
		*budget -= paid
		l.logf("bought", "item", b.Name, "price", paid, "estimate", b.Price, "budget left", *budget)
		rep.Spent += paid
		rep.Bought = append(rep.Bought, fmt.Sprintf("%s (%d gold)", b.Name, paid))
		l.mu.Lock()
		l.stats.Purchases++
		l.mu.Unlock()
	}
	return nil
}

// needsPackRoom: a bought item lands in the pack unless it goes straight into an empty matching slot.
func needsPackRoom(eq map[int]Cand, c Cand) bool {
	for _, sl := range c.Slots {
		if _, worn := eq[sl]; !worn {
			return false
		}
	}
	return true
}

// buyWithRoom checks that the item fits the pack before the click (the game refuses silently and the proof wait
// costs seconds), makes room through the vendor when it does not, and retries a failed purchase once after making room.
func (l *Leveler) buyWithRoom(ctx context.Context, h *hand.Hand, v Vendor, class int, npc *d2.Unit, c Cand) (int, error) {
	rm, canMake := v.(RoomMaker)
	room := func() error {
		if !canMake {
			return ErrNoRoom
		}
		s, err := h.Fresh()
		if err != nil {
			return err
		}
		eq, pack := l.gearState(s)
		w, hh := h.Size(c.Unit)
		if ok, err := rm.PackFits(s, w, hh); err != nil || ok {
			return err
		}
		keep := map[uint32]bool{}
		for _, sw := range ChooseGear(eq, pack) {
			keep[sw.Item.Unit.ID] = true
		}
		var junk []d2.Unit
		for _, p := range pack {
			if !keep[p.Unit.ID] {
				junk = append(junk, p.Unit)
			}
		}
		l.logf("pack has no room, making some", "item", c.Name, "w", w, "h", hh, "junk", len(junk))
		ok, err := rm.MakeRoom(ctx, w, hh, junk)
		if err != nil {
			if ctx.Err() != nil {
				return err
			}
			l.e.Log.Warn("leveler: making room failed", "err", err)
			if s, ferr := h.Fresh(); ferr == nil && !s.UI.Open(d2.UIShop) {
				if *npc, ferr = v.OpenTrade(ctx, class); ferr != nil {
					return fmt.Errorf("reopen trade: %w", ferr)
				}
			}
			return fmt.Errorf("%w: %v", ErrNoRoom, err)
		}
		if s, err = h.Fresh(); err != nil {
			return err
		}
		if !s.UI.Open(d2.UIShop) { // stashing closed the vendor
			if *npc, err = v.OpenTrade(ctx, class); err != nil {
				return fmt.Errorf("reopen trade: %w", err)
			}
		}
		if !ok {
			return ErrNoRoom
		}
		return nil
	}
	s, err := h.Fresh()
	if err != nil {
		return 0, err
	}
	eq, _ := l.gearState(s)
	if needsPackRoom(eq, c) {
		if err := room(); err != nil {
			return 0, err
		}
	}
	paid, err := l.buy(ctx, h, c)
	if err == nil || !canMake || errors.Is(err, ErrNoRoom) {
		return paid, err
	}
	// The pre-check can be wrong (sizes, a rounded grid): make room once more and retry once.
	if rerr := room(); rerr != nil {
		return 0, err
	}
	return l.buy(ctx, h, c)
}

// buy right-clicks the item in the vendor grid and proves the purchase: gold fell and an item appeared in the pack.
func (l *Leveler) buy(ctx context.Context, h *hand.Hand, c Cand) (int, error) {
	s, err := h.Fresh()
	if err != nil {
		return 0, err
	}
	lay, err := h.Layout(s)
	if err != nil {
		return 0, err
	}
	if tab, ok := ShopTabs[c.Unit.Item.Page]; ok {
		if err := l.hoverClick(tab); err != nil {
			return 0, err
		}
	}
	g0, itemsBefore := s.Player.Gold, len(s.Inventory.Items)
	w, hh := h.Size(c.Unit)
	at := lay.Vendor.ItemCenter(c.Unit.X, c.Unit.Y, w, hh)
	if err := l.e.In.MoveMouse(at.X, at.Y); err != nil { // hover first: the game ignores a click that comes right after the move
		return 0, err
	}
	l.sleep(350)
	if err := h.RightClick(c.Unit, lay.Vendor); err != nil {
		return 0, err
	}
	s2, ok, err := h.Wait(ctx, 2*time.Second, func(s *d2.Snapshot) bool {
		// any container: right-clicking gear into an EMPTY matching slot equips it at once (docs/town.md round 2)
		return s.Player.Gold < g0 && len(s.Inventory.Items) > itemsBefore
	})
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("gold or pack did not change (no room, not enough gold, or the click missed)")
	}
	return g0 - s2.Player.Gold, nil
}

// hoverClick moves onto a button, lets the game register the hover for a few frames, then clicks. The vendor tabs
// ignore a click that follows the move too closely (VERIFIED LIVE on Akara: move, 600 ms, press 250 ms switches the
// tab; the stock Click, move then a short settle, does not).
func (l *Leveler) hoverClick(p Pt) error {
	if err := l.e.In.MoveMouse(p.X, p.Y); err != nil {
		return err
	}
	l.sleep(450)
	if err := l.e.In.Click(input.Left, p.X, p.Y); err != nil {
		return err
	}
	l.sleep(350)
	return nil
}

// HoverClick is exported for the live tool.
func (l *Leveler) HoverClick(x, y int) error { return l.hoverClick(Pt{x, y}) }

// BuyUnit buys one stock item of the open vendor (live tool and retries); the proof is the same as Shop's.
func (l *Leveler) BuyUnit(ctx context.Context, v Vendor, class int, npc d2.Unit, c Cand) (int, error) {
	return l.buyWithRoom(ctx, l.newHand(), v, class, &npc, c)
}
