package town

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/town/hand"
)

// gearSlot ties an equipment slot to the body location it fills, the inventory.txt slot name and the
// minimum RebuyGearTier at which it is bought. Slot names: rArm is the weapon slot, lArm the shield slot
// (inventory.txt calls the ring slots rHand/lHand).
type gearSlot struct {
	name string
	body int
	loc  string // itemtypes BodyLoc code
	slot string
	tier int
	pick func(db *data.DB, b *data.ItemBase) bool
}

// slotWanted says whether the rebuy may buy for slot g: below the early-gear limit only the weapon of a class that needs
// one (docs/sme/early-gear-policy.md, knob resource.earlyGearMax; 0 = old behaviour), else every slot up to the tier.
func (t *Town) slotWanted(p *d2.Player, g gearSlot) bool {
	if resource.Early(p.Level, t.e.Tune.Resource.EarlyGearMax) {
		if g.slot == "belt" && t.tune().BeltBuyLevel > 0 && p.Level >= t.tune().BeltBuyLevel {
			return true // the one armor slot worth gold early
		}
		return g.slot == "rArm" && resource.NeedsWeapon(p.Class)
	}
	return true
}

// slotFilled: the body slot holds gear in the active weapon set or, for weapon and shield slots, in the swapped-away set.
func slotFilled(inv *d2.Inventory, body int) bool { return hand.FilledEither(inv, body) }

func isType(db *data.DB, b *data.ItemBase, t string) bool { return db.IsA(b.Type, t) }

var gearSlots = []gearSlot{
	{"weapon", d2.BodyRHand, "rarm", "rArm", 1, func(db *data.DB, b *data.ItemBase) bool {
		return b.Kind == data.KindWeapon && !isType(db, b, "miss") && !isType(db, b, "thro")
	}},
	{"body armor", d2.BodyTorso, "tors", "torso", 1, nil},
	{"helm", d2.BodyHead, "head", "head", 2, nil},
	{"boots", d2.BodyFeet, "feet", "feet", 2, nil},
	{"gloves", d2.BodyGloves, "glov", "gloves", 2, nil},
	{"belt", d2.BodyBelt, "belt", "belt", 2, nil},
	{"shield", d2.BodyLHand, "larm", "lArm", 3, func(db *data.DB, b *data.ItemBase) bool { return isType(db, b, "shld") }},
}

// RebuyReport is what poverty mode did.
type RebuyReport struct {
	Equipped []string
	Spent    int
	Missing  []string // slots still empty (no gold, no viable item, or a failed click)
	// Blocked: slots whose stock holds an affordable item of the right type that the character cannot wear yet
	// (strength, dexterity or level). No gold fixes these, only a level or stat point.
	Blocked []string
	// NeedStr and NeedDex: what the cheapest affordable blocked item of any blocked slot needs (0 = nothing known);
	// the leveler spends the next stat points there.
	NeedStr, NeedDex int
	Errors           []string
	Gold             int // gold left
}

// viable says whether the sorceress-or-any-class character can wear/use base b in the slot.
func (t *Town) viable(s *d2.Snapshot, g gearSlot, b *data.ItemBase) bool {
	return t.reject(s, g, b) == ""
}

// Rejection classes of reject, also the keys of the "rebuy: stock seen" log line.
const (
	rejNotGear = "not gear (no base or not a weapon/armor)"
	rejSlot    = "wrong body slot"
	rejBuild   = "build rule (shield build, shield in weapon slot, or weapon kind)"
	rejClass   = "other class only"
	rejStr     = "needs more strength"
	rejDex     = "needs more dexterity"
	rejLevel   = "needs a higher level"
	rejPrice   = "over the price cap"
	rejNoBase  = "unknown item base"
)

// reject names why base b cannot fill slot g for the character, "" when it can.
func (t *Town) reject(s *d2.Snapshot, g gearSlot, b *data.ItemBase) string {
	db := t.e.Data
	if b == nil {
		return rejNoBase
	}
	if b.Kind != data.KindWeapon && b.Kind != data.KindArmor {
		return rejNotGear
	}
	ty := db.ItemType(b.Type)
	if ty == nil || (ty.BodyLocs[0] != g.loc && ty.BodyLocs[1] != g.loc) {
		return rejSlot
	}
	if prefs := leveler.PrefsFor(s.Player.Class.String()); prefs.WantsShield && g.slot == "rArm" && b.TwoHanded {
		return rejBuild // a shield build never rebuys a two-hander (it would block the shield slot)
	}
	if g.slot == "rArm" && isType(db, b, "shld") || g.slot == "lArm" && !isType(db, b, "shld") {
		return rejSlot
	}
	if ty.Class != "" {
		cls := db.ClassByName(s.Player.Class.String())
		if cls == nil || ty.Class != cls.Code {
			return rejClass
		}
	}
	if g.pick != nil && !g.pick(db, b) {
		if !(g.slot == "rArm" && s.Player.Class == d2.Amazon && b.Kind == data.KindWeapon) {
			return rejBuild
		}
	}
	// The Amazon's skills need a bow, crossbow or javelins (docs/sme/early-gear-policy.md); a melee weapon is no kit.
	if g.slot == "rArm" && s.Player.Class == d2.Amazon && !(isType(db, b, "miss") || isType(db, b, "thro") || isType(db, b, "jave")) {
		return rejBuild
	}
	p := s.Player
	switch {
	case b.ReqStr > p.Strength:
		return rejStr
	case b.ReqDex > p.Dexterity:
		return rejDex
	case b.LevelReq > p.Level:
		return rejLevel
	}
	return ""
}

func isRequirement(why string) bool { return why == rejStr || why == rejDex || why == rejLevel }

// Rebuy is poverty mode after a death: buy the potions the belt plan wants first, then the cheapest viable
// item for each empty equipment slot up to RebuyGearTier, within budget gold (and one item at most
// RebuyItemMaxPct of it), and equip each by clicking it into its slot. It reports what is still missing.
// Vendors: the potion shop for potions, the act's repair NPC (weapon/armor seller) for gear.
func (t *Town) Rebuy(ctx context.Context, budget int) (RebuyReport, error) {
	var out RebuyReport
	s, err := t.current()
	if err != nil {
		return out, err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return out, ErrNotInTown
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return out, err
	}
	// Standing on an empty weapon set while the other holds the gear: swap back instead of buying (hand/weaponset.go).
	if sw, err := t.h.EnsureMainSet(ctx); err != nil {
		out.Errors = append(out.Errors, "weapon set: "+err.Error())
	} else if sw {
		if s, err = t.current(); err != nil {
			return out, err
		}
	}
	startGold := s.Player.Gold
	budget = min(budget, startGold)
	rep := ChoreReport{Bought: map[string]int{}}
	tier := t.tune().RebuyGearTier
	missing := func(s *d2.Snapshot, maxTier int) []gearSlot {
		var todo []gearSlot
		for _, g := range gearSlots {
			if g.tier <= min(tier, maxTier) && t.slotWanted(s.Player, g) && !slotFilled(s.Inventory, g.body) {
				todo = append(todo, g)
			}
		}
		return todo
	}
	// Weapon and body armor come before any potion: a character without them dies in the first fight, and the cheapest
	// of each costs a fraction of one potion round (DEATHSPIRAL, V: Soaky died four times with neither). Potions follow,
	// then the lesser slots.
	if todo := missing(s, 1); len(todo) > 0 {
		t.rebuyGear(ctx, a, todo, startGold-budget, &out)
		if s, err = t.current(); err != nil {
			return out, err
		}
	}
	if t.needPotions(s) {
		t.potionCap = max(budget-(startGold-s.Player.Gold), 0) * t.tune().RebuyPotionPct / 100
		defer func() { t.potionCap = 0 }()
		if err := t.shopVisit(ctx, a, ChorePlan{Potions: true}, &rep); err != nil {
			out.Errors = append(out.Errors, "potions: "+err.Error())
		}
		if s, err = t.current(); err != nil {
			return out, err
		}
	}
	if todo := missing(s, 3); len(todo) > 0 {
		t.rebuyGear(ctx, a, todo, startGold-budget, &out)
	}
	slices.Sort(out.Blocked)
	out.Blocked = slices.Compact(out.Blocked)
	if s, err = t.current(); err == nil {
		out.Gold = s.Player.Gold
		out.Spent = startGold - s.Player.Gold
		for _, g := range gearSlots {
			if g.tier <= tier && t.slotWanted(s.Player, g) && !slotFilled(s.Inventory, g.body) {
				out.Missing = append(out.Missing, g.name)
			}
		}
	}
	return out, t.h.CloseAll(ctx)
}

// rebuyGear buys and equips; floor is the gold that must remain (startGold - budget).
func (t *Town) rebuyGear(ctx context.Context, a *Act, todo []gearSlot, floor int, out *RebuyReport) {
	fail := func(g gearSlot, err error) { out.Errors = append(out.Errors, g.name+": "+err.Error()) }
	npc, err := t.openShop(ctx, a.Repair, t.npcPreset(a.Repair...))
	if err != nil {
		out.Errors = append(out.Errors, "gear vendor: "+err.Error())
		return
	}
	stock, err := t.Stock(npc)
	if err != nil || len(stock) == 0 {
		out.Errors = append(out.Errors, fmt.Sprintf("gear stock unreadable: %v", err))
		_ = t.h.CloseAll(ctx)
		return
	}
	var bought []struct {
		g  gearSlot
		it d2.Unit
	}
	for _, g := range todo {
		s, err := t.current()
		if err != nil {
			return
		}
		lay, err := t.layout(s)
		if err != nil {
			fail(g, err)
			continue
		}
		spendable := s.Player.Gold - floor
		capCost := (s.Player.Gold - floor) * 100 / 100
		if c := t.tune().RebuyItemMaxPct; c > 0 {
			capCost = min(capCost, spendable*c/100)
		}
		var c []d2.Unit
		rejected := map[string]int{}
		blocked := ""
		needStr, needDex, needCost := 0, 0, 0
		for _, u := range stock {
			b := t.base(u)
			why := t.reject(s, g, b)
			if why == "" {
				why, _, _ = t.unitReject(s.Player, u) // affix level, unidentified (require.go)
			}
			if why == "" && b.Cost > max(capCost, 0) {
				why = rejPrice
			}
			if why != "" {
				rejected[why]++
				if isRequirement(why) && blocked == "" {
					blocked = fmt.Sprintf("%s (%s)", t.nameOf(u), why)
				}
				// Stats can fix strength and dexterity, not level or price: remember the cheapest such item.
				if (why == rejStr || why == rejDex) && b.LevelReq <= s.Player.Level && b.Cost <= max(capCost, 0) &&
					(needCost == 0 || b.Cost < needCost) {
					needStr, needDex, needCost = b.ReqStr, b.ReqDex, b.Cost
				}
				continue
			}
			c = append(c, u)
		}
		t.e.Log.Info("rebuy: stock seen", "slot", g.name, "items", len(stock), "viable", len(c), "rejected", rejected,
			"str", s.Player.Strength, "dex", s.Player.Dexterity, "clvl", s.Player.Level, "spendable", spendable, "cap", capCost)
		sort.SliceStable(c, func(i, j int) bool { return t.base(c[i]).Cost < t.base(c[j]).Cost })
		done := false
		for _, it := range c {
			if _, err := t.buyUnit(ctx, lay, it); err != nil {
				continue
			}
			bought = append(bought, struct {
				g  gearSlot
				it d2.Unit
			}{g, it})
			done = true
			break
		}
		if !done {
			if blocked != "" && len(c) == 0 {
				out.Blocked = append(out.Blocked, g.name)
				out.NeedStr, out.NeedDex = max(out.NeedStr, needStr), max(out.NeedDex, needDex)
				fail(g, fmt.Errorf("nothing wearable in stock yet, e.g. %s", blocked))
			} else {
				fail(g, fmt.Errorf("nothing affordable and viable in stock"))
			}
		}
	}
	if len(bought) == 0 {
		return // the vendor window stays open for the next chore of this NPC (Chores closes it before a non-shop step)
	}
	if err := t.h.CloseAll(ctx); err != nil {
		out.Errors = append(out.Errors, err.Error())
		return
	}
	for _, b := range bought {
		if err := t.equip(ctx, b.g, b.it.Class); err != nil {
			fail(b.g, err)
			continue
		}
		out.Equipped = append(out.Equipped, t.nameOf(b.it))
	}
	_ = t.h.CloseAll(ctx)
}

// equip finds a pack item of class, picks it up and clicks it into the slot, confirming it is equipped.
func (t *Town) equip(ctx context.Context, g gearSlot, class int) error {
	return t.equipUnit(ctx, g, d2.Unit{Class: class})
}

// equipUnit is equip for a specific item: with a nonzero ID that very unit is taken, else the first pack item of u.Class.
func (t *Town) equipUnit(ctx context.Context, g gearSlot, want d2.Unit) error {
	class := want.Class
	// A right-click purchase into an empty matching slot equips at once (VERIFIED LIVE, dagger and quilted armor).
	if s, err := t.current(); err == nil {
		if u := s.Inventory.Equipped(g.body); u != nil && u.Class == class && (want.ID == 0 || u.ID == want.ID) {
			return nil
		}
	}
	s, err := t.h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	rect, ok := lay.Slots[g.slot]
	if !ok {
		return fmt.Errorf("no screen slot %q", g.slot)
	}
	var it d2.Unit
	found := false
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.Class == class && (want.ID == 0 || u.ID == want.ID) {
			it, found = u, true
			break
		}
	}
	if !found {
		return fmt.Errorf("bought item not in the pack")
	}
	st := stampOf(s.Player)
	if isRefused(it.ID, st) {
		return fmt.Errorf("equip: game refused %s at this level and stats before", t.nameOf(it))
	}
	if why, need, have := t.unitReject(s.Player, it); why != "" {
		t.logSkip(it, st.clvl, reasonKey(why), need, have)
		return fmt.Errorf("equip: %s needs %s (need %d, have %d)", t.nameOf(it), reasonKey(why), need, have)
	}
	start := t.e.Clock.Now()
	worn := func(s *d2.Snapshot) bool {
		u := s.Inventory.Equipped(g.body)
		return u != nil && u.ID == it.ID
	}
	confirmed := func() {
		t.e.Log.Info(fmt.Sprintf("equip: confirmed after %d ms", t.e.Clock.Now().Sub(start).Milliseconds()), "slot", g.name)
	}
	if err := t.h.Take(ctx, it, lay.Inventory); err != nil {
		return err
	}
	// The game leaves an item it refuses on the cursor (live 2026-10-30: "slot click refused ... stillOnCursor", the
	// same click that works for other items), so a click that left it there is repeated at other points of the box.
	var last *d2.Snapshot
	for try := 0; try < equipClicks; try++ {
		c := hand.SlotPoint(rect, try)
		t.e.Log.Info(fmt.Sprintf("equip: slot click at (%d,%d)", c.X, c.Y), "slot", g.name, "cursor", it.ID, "try", try, "rect", rect)
		if err := t.e.In.MoveMouse(c.X, c.Y); err != nil {
			return err
		}
		t.e.Clock.Sleep(equipHover)
		if err := t.e.In.Click(0, c.X, c.Y); err != nil {
			return err
		}
		s2, ok, err := t.h.Wait(ctx, t.uiTimeout(), worn)
		if err != nil {
			return err
		}
		if ok {
			confirmed()
			// Wearing over an occupied slot swaps: the old item is on the cursor now, put it back for the sell chore.
			return t.h.ClearCursor(ctx)
		}
		last = s2
		if s2.Inventory.Cursor == nil || s2.Inventory.Cursor.ID != it.ID {
			break // the item left the cursor without being worn: another click would hit something else
		}
	}
	reason := t.equipFailReason(last, it, g) // before the cursor is emptied, or the refusal is not visible
	// Never leave the item held: it goes back to a free pack cell (hand.ClearCursor).
	if err := t.h.ClearCursor(ctx); err != nil {
		t.e.Log.Warn("equip: could not put the refused item back", "item", it.ID, "err", err)
	}
	// Right-click does not equip armor in 1.14d (EQUIPREQ, d2-sme), so there is no right-click retry unless switched on.
	if equipRightClickFallback && last != nil && t.equipRightClick(ctx, it, worn) {
		confirmed()
		return t.h.ClearCursor(ctx)
	}
	if strings.Contains(reason, "still on the cursor") {
		markRefused(it, st)
		t.logSkip(it, st.clvl, "refused", 0, 0)
	}
	t.e.Log.Warn("equip: not confirmed", "slot", g.name, "item", it.ID, "because", reason, "clicks", equipClicks)
	return fmt.Errorf("equip not confirmed in memory: %s", reason)
}

// refused remembers pack items the game would not let the character wear, by unit ID, with the character's level,
// strength and dexterity at the time (EXP: the pack item is a magic/rare piece whose affixes need more level than the
// base row says: red tile). Retrying the same item with the same character only burns clicks and a town trip; it is
// tried again after a level-up or a stat point.
var (
	refusedMu sync.Mutex
	refused   = map[uint32]charStamp{}
)

type charStamp struct{ clvl, str, dex int }

func stampOf(p *d2.Player) charStamp {
	if p == nil {
		return charStamp{}
	}
	return charStamp{p.Level, p.Strength, p.Dexterity}
}

func markRefused(it d2.Unit, st charStamp) {
	refusedMu.Lock()
	refused[it.ID] = st
	refusedMu.Unlock()
}

func isRefused(id uint32, st charStamp) bool {
	refusedMu.Lock()
	defer refusedMu.Unlock()
	l, ok := refused[id]
	return ok && l == st
}

func (t *Town) clvl() int {
	if s, err := t.current(); err == nil && s.Player != nil {
		return s.Player.Level
	}
	return 0
}

// equipRightClickFallback re-enables the right-click retry of EQUIPCONFIRM. Off: 1.14d equips by click only (S, d2-sme).
const equipRightClickFallback = false

const (
	equipClicks = 3                      // slot clicks per equip, at different points of the slot box
	equipHover  = 150 * time.Millisecond // pointer rests on the slot before the click
)

// equipRightClick right-clicks it in the pack and reports whether it is then worn. The pack must be showing.
func (t *Town) equipRightClick(ctx context.Context, it d2.Unit, worn func(*d2.Snapshot) bool) bool {
	s, err := t.h.OpenInventory(ctx)
	if err != nil {
		return false
	}
	lay, err := t.layout(s)
	if err != nil {
		return false
	}
	cur, ok := hand.Find(s.Inventory, it.ID)
	if !ok || cur.Item == nil || cur.Item.Where != d2.ContInventory {
		return false
	}
	if t.h.RightClick(cur, lay.Inventory) != nil {
		return false
	}
	_, ok, err = t.h.Wait(ctx, t.uiTimeout(), worn)
	return err == nil && ok
}

// equipFailReason says where the item is after a failed equip, so the log tells a refused click from a lost item.
func (t *Town) equipFailReason(s *d2.Snapshot, it d2.Unit, g gearSlot) string {
	fresh, err := t.h.Fresh()
	if err != nil {
		fresh = s
	}
	if fresh == nil || fresh.Inventory == nil {
		return "no snapshot"
	}
	if c := fresh.Inventory.Cursor; c != nil && c.ID == it.ID {
		return "item still on the cursor (the game refused the slot click)"
	}
	u, ok := hand.Find(fresh.Inventory, it.ID)
	switch {
	case !ok || u.Item == nil:
		return "item no longer in memory (id changed or item gone)"
	case u.Item.Where == d2.ContEquipment:
		return fmt.Sprintf("item worn at body %d, not %d", u.Item.Body, g.body)
	}
	return fmt.Sprintf("item back in container %v", u.Item.Where)
}

// foundGear lists the pack items worth wearing: an empty weapon slot takes any wearable weapon (higher quality first,
// then the pricier base), every armor slot takes the wearable piece with the best loot.WearScore when it beats what is
// worn (an empty slot is beaten by any piece). Requirements are those of viable. Unlike the rebuy it covers all slots at
// any level: wearing what dropped costs nothing (docs/sme/early-gear-policy.md).
func (t *Town) foundGear(s *d2.Snapshot) (slots []gearSlot, units []d2.Unit) {
	score := func(u d2.Unit) int {
		var st d2.Stats
		if u.Addr != 0 {
			st, _ = t.unitStats(u.Addr)
		}
		return loot.WearScore(t.base(u), st)
	}
	twoHander := false
	if w := s.Inventory.Equipped(d2.BodyRHand); w != nil {
		twoHander = t.base(*w) != nil && t.base(*w).TwoHanded
	}
	for _, g := range gearSlots {
		weapon := g.slot == "rArm"
		cur, filled := 0, slotFilled(s.Inventory, g.body)
		if filled && weapon {
			continue
		}
		if filled {
			if u := s.Inventory.Equipped(g.body); u != nil {
				cur = score(*u)
			}
		}
		if g.slot == "lArm" && twoHander {
			continue
		}
		var best *d2.Unit
		bestScore := 0
		for _, u := range s.Inventory.Container(d2.ContInventory) {
			if u.Item == nil || u.Item.Flags&d2.FlagStartItem != 0 || isRefused(u.ID, stampOf(s.Player)) || !t.viable(s, g, t.base(u)) {
				continue
			}
			if why, need, have := t.unitReject(s.Player, u); why != "" {
				t.logSkip(u, s.Player.Level, reasonKey(why), need, have)
				continue
			}
			if weapon {
				if best == nil || u.Item.Quality > best.Item.Quality ||
					u.Item.Quality == best.Item.Quality && t.base(u).Cost > t.base(*best).Cost {
					c := u
					best = &c
				}
				continue
			}
			sc := score(u)
			if filled && sc > cur && sc < cur+minEquipGain {
				t.logSkip(u, s.Player.Level, "small gain", cur+minEquipGain, sc)
				continue
			}
			if (!filled || sc >= cur+minEquipGain) && (best == nil || sc > bestScore) {
				c := u
				best, bestScore = &c, sc
			}
		}
		if best != nil {
			slots, units = append(slots, g), append(units, *best)
		}
	}
	return slots, units
}

// EquipFound wears gear that was picked up for an empty weapon or body armor slot (the loot pickit keeps it for that,
// loot.nakedRule). It returns the names equipped.
func (t *Town) EquipFound(ctx context.Context) ([]string, error) {
	s, err := t.current()
	if err != nil {
		return nil, err
	}
	slots, units := t.foundGear(s)
	var done []string
	var first error
	for i, g := range slots {
		name := t.nameOf(units[i])
		if err := t.equipUnit(ctx, g, units[i]); err != nil {
			t.e.Log.Warn("equip: failed", "slot", g.name, "item", name, "err", err)
			if first == nil {
				first = fmt.Errorf("equip %s: %w", name, err)
			}
			continue
		}
		t.e.Log.Info("equip: found gear worn", "slot", g.name, "item", name)
		done = append(done, name)
	}
	return done, errors.Join(first, t.h.CloseAll(ctx))
}

func (t *Town) needEquipFound(s *d2.Snapshot) bool {
	slots, _ := t.foundGear(s)
	return len(slots) > 0
}

// needWeaponFound is the only found-gear reason that justifies a trip of its own: a class that needs a weapon has none
// on and one lies in the pack. Armor is worn when the character is in town anyway.
func (t *Town) needWeaponFound(s *d2.Snapshot) bool {
	if !resource.NeedsWeapon(s.Player.Class) {
		return false
	}
	slots, _ := t.foundGear(s)
	return slices.ContainsFunc(slots, func(g gearSlot) bool { return g.slot == "rArm" })
}
