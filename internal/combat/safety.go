package combat

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
)

// Belt columns are 0..3, drunk with the keys "1".."4" (docs/plan/data/hotkeys.json). The front potion of a column
// is the one drunk; which row is the front is UNVERIFIED, so a column is described by its contents.
const beltCols = 4

// BeltColumn describes one belt column.
type BeltColumn struct {
	Kind  PotionKind // the kind of potion in the column (PotionNone when empty)
	Tier  int        // highest tier in the column
	Count int
	Mixed bool // more than one kind in the column
}

// Belt is the potion inventory, per column.
type Belt struct {
	Cols [beltCols]BeltColumn
}

// BeltOf reads the belt contents out of an inventory.
func BeltOf(inv *d2.Inventory, cat Catalog) Belt {
	var b Belt
	if inv == nil {
		return b
	}
	var front [beltCols]int
	for _, u := range inv.Container(d2.ContBelt) {
		c := cat.Item(u.Class)
		// box = row*4+column. Game.exe reports (column,row); d2d reports the box 0..15 in GX with GY 0 (F7, fork
		// put_in_belt/pick_up). One formula serves both; the column is box%4 and the lowest box of a column is its front.
		box := u.Item.GY*beltCols + u.Item.GX
		if c.Potion == PotionNone || box < 0 || box >= 4*beltCols {
			continue
		}
		col := box % beltCols
		bc := &b.Cols[col]
		if bc.Count > 0 && bc.Kind != c.Potion {
			bc.Mixed = true
		}
		if bc.Count == 0 || box < front[col] {
			bc.Kind, front[col] = c.Potion, box
		}
		if c.Tier > bc.Tier {
			bc.Tier = c.Tier
		}
		bc.Count++
	}
	return b
}

// BeltCapacity is the number of potion slots of the equipped belt (4 without one), 0 when the inventory or tables are
// unknown. The belt row of armor.txt indexes belts.txt (numboxes: sash 8, belt 12, ...).
func BeltCapacity(inv *d2.Inventory, db *data.DB) int {
	if inv == nil || db == nil {
		return 0
	}
	slots := 4
	if u := inv.Equipped(d2.BodyBelt); u != nil && u.Class >= 0 && u.Class < len(db.Items) {
		if b := db.BeltByIndex(db.Items[u.Class].BeltIndex); b != nil && b.Slots > 0 {
			slots = b.Slots
		}
	}
	// never fewer than the boxes in use (same floor as town/belt.Slots, docs/beltfix.md)
	for _, u := range inv.Container(d2.ContBelt) {
		if u.Item == nil {
			continue
		}
		if box := u.Item.GY*beltCols + u.Item.GX; box >= slots && box < 4*beltCols {
			slots = (box/beltCols + 1) * beltCols
		}
	}
	return slots
}

// Count returns the number of potions of a kind in the belt.
func (b Belt) Count(k PotionKind) int {
	n := 0
	for _, c := range b.Cols {
		if c.Kind == k {
			n += c.Count
		}
	}
	return n
}

// ColumnFor returns the belt column to drink a kind from: the highest tier when best is true (emergencies),
// the lowest otherwise (do not waste a super potion on a scratch).
func (b Belt) ColumnFor(k PotionKind, best bool) (int, bool) {
	idx := -1
	for i, c := range b.Cols {
		if c.Kind != k || c.Count == 0 {
			continue
		}
		if idx < 0 || (best && c.Tier > b.Cols[idx].Tier) || (!best && c.Tier < b.Cols[idx].Tier) {
			idx = i
		}
	}
	return idx, idx >= 0
}

// SafetyInput is everything Decide looks at; Fighter.Safety fills it from a snapshot.
type SafetyInput struct {
	InTown     bool
	Life, Max  int
	Mana, MMax int
	Diff       d2.Difficulty
	Belt       Belt
	InFight    bool // a hostile monster is close enough to matter
	HasPortal  bool // Town Portal scroll or tome in the inventory
	Crowd      int  // hostiles within 8 subtiles (crowd.go)
	Adjacent   int  // hostiles within 3 subtiles
	Walking    bool // the bot is on a travel or quest walk
	CanRetreat bool // a step away from the crowd exists
	HealReady  bool // healing potion cooldown elapsed
	ManaReady  bool
	RejuvReady bool
	Tune       env.CombatTuning
}

func pct(v, max int) int {
	if max <= 0 {
		return 0
	}
	return v * 100 / max
}

// Decide is the pure survival decision (docs/plan/leveling.md section 5, potions-belt.json thresholds).
//
//	life <= chicken%:    rejuvenation if one is ready, else leave the game (a healing potion does not save a
//	                     character that low; out of a fight a potion is fine)
//	life <  rejuv%:      rejuvenation
//	life <  drink%:      healing potion; with no potion at all open a town portal
//	mana <  mana%:       mana potion
//	hurt, out of a fight and few healing potions left: town portal (free healing, restock)
func Decide(in SafetyInput) SafetyAction {
	return meleeCastGuard(in, decideBase(in))
}

// meleeCastGuard replaces a town portal cast that melee would cut (card fixportalmelee, docs/fixportalmelee.md). A cast
// takes game time and a hit interrupts it, so with resource.CastAdjacentMax or more attackers in reach it is not an
// escape (soak 2 death: scroll spent, character died). Order: drink if a potion is ready, else leave the game. A step
// away is not offered: melee monsters follow at walking speed and the hits continue (INFERRED, UNVERIFIED live).
// Only fights count; quiet town trips and every case below the adjacent count are untouched.
func meleeCastGuard(in SafetyInput, a SafetyAction) SafetyAction {
	if a != TownPortal || !in.InFight || in.Adjacent < crowdAdjacentMax {
		return a
	}
	switch {
	case in.Belt.Count(PotionRejuv) > 0 && in.RejuvReady:
		return DrinkRejuv
	case in.Belt.Count(PotionHealing) > 0 && in.HealReady:
		return DrinkLife
	}
	return Chicken
}

func decideBase(in SafetyInput) SafetyAction {
	if in.InTown || in.Max <= 0 || in.Life <= 0 {
		return SafetyNone
	}
	t := in.Tune
	lp, mp := pct(in.Life, in.Max), pct(in.Mana, in.MMax)
	chicken := t.ChickenPct
	if in.Diff == d2.Hell {
		chicken = t.ChickenHellPct
	}
	rejuv := in.Belt.Count(PotionRejuv) > 0
	heal := in.Belt.Count(PotionHealing) > 0
	if lp <= chicken {
		switch {
		case rejuv && in.RejuvReady:
			return DrinkRejuv
		case !in.InFight && heal && in.HealReady:
			return DrinkLife
		}
		return Chicken
	}
	if a, ok := crowdDecide(in, lp, heal, rejuv); ok {
		return a
	}
	if lp < t.DrinkRejuvPct && rejuv && in.RejuvReady {
		return DrinkRejuv
	}
	if lp < t.DrinkLifePct {
		switch {
		case !in.InFight && in.HasPortal && in.Belt.Count(PotionHealing) <= in.Tune.TownPotionMin:
			return TownPortal // hurt, quiet and short of potions: healing in town is free
		case heal && in.HealReady:
			return DrinkLife
		case rejuv && in.RejuvReady:
			return DrinkRejuv
		case !heal && !rejuv && in.HasPortal:
			return TownPortal
		}
	}
	if in.MMax > 0 && mp < t.DrinkManaPct && in.Belt.Count(PotionMana) > 0 && in.ManaReady {
		return DrinkMana
	}
	return SafetyNone
}
