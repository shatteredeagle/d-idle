package loot

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/hand"
)

// PickOpts tunes one PickUp call. Zero values mean "use the Tuning default".
type PickOpts struct {
	Center     *input.Point // search around this point instead of the player
	Radius     int          // subtiles
	Max        int          // pickups to make at most
	Strictness int          // -1 or unset (see StrictSet) uses the tuning value
	StrictSet  bool         // true: Strictness is meant, even when 0
	MinPrio    int          // skip rules with a lower prio (fast pickup in a fight: gold, runes, potions)
	// Names are item display names (case-insensitive, e.g. quest items) taken whatever the pickit says, as long as
	// there is room. Ground items only: nothing is ever opened (chests stay shut).
	Names []string
	// Abort is polled before each item and while waiting; true stops the call (an enemy came close).
	Abort func() bool
}

// Picked is one confirmed pickup.
type Picked struct {
	ID      uint32
	Class   int
	Name    string
	Verdict Verdict
	Gold    int // gold gained, for gold piles
}

// Skipped explains an item that was seen but not taken (or failed).
type Skipped struct {
	ID   uint32
	Name string
	Why  string
	Prio int // pickit priority of the verdict (set for "no room")
}

// PickReport is the outcome of a PickUp call.
type PickReport struct {
	Picked     []Picked
	Skipped    []Skipped
	Failed     []Skipped
	GoldGained int
	Seen       int  // pickable-or-skipped ground items in range when the call began
	Full       bool // a wanted item did not fit: a town trip would free room
	Aborted    bool
}

type cand struct {
	u     d2.Unit
	v     Verdict
	dist  float64
	name  string
	gold0 int // player gold when the candidate list was built
}

var errGone = errors.New("loot: item left the ground without arriving in the pack")

// PickUp walks to and picks up ground items that pass the pickit, best first. When anything was in range it logs
// one summary line (seen, picked, skipped by reason, gold gained).
func (l *Looter) PickUp(ctx context.Context, o PickOpts) (PickReport, error) {
	rep, err := l.pickUp(ctx, o)
	if l.recovering.Load() {
		l.recTotals[0].Add(int64(len(rep.Picked)))
		l.recTotals[1].Add(int64(rep.GoldGained))
		for _, sk := range rep.Skipped {
			if sk.Why == "no room" {
				l.recTotals[2].Add(1)
			}
		}
	}
	if rep.Seen > 0 {
		l.e.Log.Info(rep.Summary())
	}
	return rep, err
}

// Summary is the one-line "loot: N seen, picked M, skipped K (reasons), gold +G" text of a PickUp call.
func (r PickReport) Summary() string {
	why := map[string]int{}
	for _, sk := range r.Skipped {
		why[sk.Why]++
	}
	keys := make([]string, 0, len(why))
	for k := range why {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%dx %s", why[k], k)
	}
	return fmt.Sprintf("loot: %d seen, picked %d, skipped %d (%s), gold +%d", r.Seen, len(r.Picked), len(r.Skipped), strings.Join(parts, "; "), r.GoldGained)
}

func (l *Looter) pickUp(ctx context.Context, o PickOpts) (PickReport, error) {
	var rep PickReport
	first := true
	t := &l.e.Tune.Loot
	max := o.Max
	if max <= 0 {
		max = t.MaxPerCall
	}
	failed := map[uint32]int{}
	noRoomLogged := map[uint32]bool{}
	for len(rep.Picked) < max {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		if o.Abort != nil && o.Abort() {
			rep.Aborted = true
			return rep, nil
		}
		s, err := l.hand.Fresh()
		if err != nil {
			return rep, err
		}
		if s.Inventory.Cursor != nil {
			if err := l.hand.ClearCursor(ctx); err != nil {
				return rep, fmt.Errorf("loot: item stuck on the cursor: %w", err)
			}
			continue
		}
		if hand.PanelOpen(s.UI) {
			if err := l.hand.CloseAll(ctx); err != nil {
				return rep, err
			}
			continue
		}
		cands, skipped, full := l.candidates(s, o, failed)
		l.e.Log.Debug("loot: ground", "units", len(s.Ground), "cands", len(cands), "skipped", len(skipped), "radius", o.Radius)
		rep.Skipped = skipped
		if first {
			first = false
			rep.Seen = len(cands) + len(skipped)
		}
		for _, sk := range skipped {
			if sk.Why == "no room" && !noRoomLogged[sk.ID] { // one line per item and call, not per loop pass
				noRoomLogged[sk.ID] = true
				l.e.Log.Info("loot: pack full, item left on the ground", "item", sk.Name, "id", sk.ID)
			}
		}
		rep.Full = rep.Full || full
		if len(cands) == 0 {
			l.noteLeftBehind(skipped, o)
			return rep, nil
		}
		c := cands[0]
		goldBefore := s.Player.Gold
		err = l.pickOne(ctx, c, o)
		switch {
		case err == nil:
			after, _ := l.hand.Fresh()
			g := 0
			if after != nil && after.Player.Gold > goldBefore {
				g = after.Player.Gold - goldBefore
				rep.GoldGained += g
			}
			if err := l.unwear(ctx, c.u); err != nil {
				return rep, err
			}
			rep.Picked = append(rep.Picked, Picked{ID: c.u.ID, Class: c.u.Class, Name: c.name, Verdict: c.v, Gold: g})
			l.emitDrop(c)
			l.e.Clock.Sleep(time.Duration(t.GapMs) * time.Millisecond)
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return rep, err
		case errors.Is(err, errAborted):
			rep.Aborted = true
			return rep, nil
		default:
			failed[c.u.ID]++
			l.e.Log.Debug("pickup failed", "item", c.name, "id", c.u.ID, "err", err)
			if failed[c.u.ID] >= 2 { // pickOne already retried PickTries times; a second round is the limit
				rep.Failed = append(rep.Failed, Skipped{ID: c.u.ID, Name: c.name, Why: err.Error()})
			}
		}
	}
	return rep, nil
}

// noWear lists the base codes of quest items the game wears on pickup when the slot is free (V live, docs/act2.md
// P1: Shaft msf in the weapon slot, Top vip in the amulet slot). They must stay in the pack: the Cube needs them
// and a worn Shaft replaces the weapon. Codes from items.txt: Cube box, Staff hst, Scroll tr1, Book ass are not
// wearable and listed only so a future auto-equip change cannot wear them (U).
//
// Acts 3 to 5 add QuestItemCodes (docs/act3.md X6, codes S from Arreat/Kolbot, not yet seen live): Khalim's Flail qf1,
// Will qf2 and the Hell Forge Hammer hfh are weapons and would be worn by the same rule (U).
var noWear = map[string]bool{"msf": true, "vip": true, "hst": true, "box": true, "tr1": true, "ass": true}

// QuestItemCodes are the quest items of Acts 3 to 5 the bot must always pick up and never wear. The starter pickit
// lists the same codes (a test keeps the two equal).
var QuestItemCodes = []string{"j34", "g34", "d33", "g33", "bbb", "qey", "qhr", "qbr", "qf1", "qf2", "mss", "hfh", "ice", "tr2", "xyz", "luv"}

func init() {
	for _, c := range QuestItemCodes {
		noWear[c] = true
	}
}

// unwear moves a just-picked quest item from the body to the pack when the game wore it.
func (l *Looter) unwear(ctx context.Context, u d2.Unit) error {
	if b := l.base(u.Class); b == nil || !noWear[b.Code] {
		return nil
	}
	s, err := l.hand.Fresh()
	if err != nil {
		return err
	}
	if w, ok := hand.Find(s.Inventory, u.ID); !ok || w.Item == nil || w.Item.Where != d2.ContEquipment {
		return nil
	}
	if err := l.hand.Unequip(ctx, u.ID); err != nil {
		return fmt.Errorf("loot: %s was worn on pickup and could not be taken off: %w", l.itemName(u), err)
	}
	return l.hand.CloseAll(ctx)
}

var errAborted = errors.New("loot: aborted")

func (l *Looter) emitDrop(c cand) {
	if l.e.Emit == nil {
		return
	}
	l.e.Emit(goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: c.name, Quality: QualityName(c.u.Item.Quality)}})
}

// itemName is a log/UI name: base display name plus quality.
func (l *Looter) itemName(u d2.Unit) string {
	b := l.base(u.Class)
	n := fmt.Sprintf("class%d", u.Class)
	if b != nil {
		n = b.Display
		if n == "" {
			n = b.Name
		}
	}
	if u.Item != nil && u.Item.Quality != 2 && u.Item.Quality != 0 {
		n = QualityName(u.Item.Quality) + " " + n
	}
	return n
}

// sellKeepExtra is how many more free cells a Sell-verdict pickup leaves than FreeCellsKeep asks for.
const sellKeepExtra = 4

// candidates lists the pickable ground items, best first, plus why the others were skipped.
func (l *Looter) candidates(s *d2.Snapshot, o PickOpts, failed map[uint32]int) (out []cand, skipped []Skipped, full bool) {
	t := &l.e.Tune.Loot
	radius := o.Radius
	if radius <= 0 {
		radius = t.PickRadius
	}
	strict := t.PickStrictness
	if o.StrictSet {
		strict = o.Strictness
	}
	cx, cy := float64(s.Player.X), float64(s.Player.Y)
	if o.Center != nil {
		cx, cy = float64(o.Center.X), float64(o.Center.Y)
	}
	ctx := l.ctxFrom(s)
	lay, _ := l.hand.Layout(s)
	var occ interface{ FreeCells() int }
	var spot func(w, h int) bool
	if lay != nil {
		o := l.hand.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory)
		occ = o
		spot = func(w, h int) bool { _, _, ok := o.FindSpot(w, h); return ok }
	}
	beltSt := belt.Read(l.e.Data, s.Inventory)
	lowGold := s.Player.Gold < t.LowGoldBelow
	for _, u := range s.Ground {
		if u.Item == nil || u.Mode != d2.ItemModeGround {
			continue
		}
		name := l.itemName(u)
		skip := func(why string) { skipped = append(skipped, Skipped{ID: u.ID, Name: name, Why: why}) }
		if failed[u.ID] >= 2 {
			skip("failed before")
			continue
		}
		d := math.Hypot(float64(u.X)-cx, float64(u.Y)-cy)
		if d > float64(radius) {
			continue // out of range is not worth a report line
		}
		b := l.base(u.Class)
		c := ctx
		if l.stats != nil && (b == nil || b.Type == "gold") {
			if st, err := l.stats(u.Addr); err == nil {
				c.Stats = st
			}
		}
		v := l.EvaluateWith(u, c)
		if namedItem(o.Names, name, b) {
			v = Verdict{Action: Keep, Tier: "S", Prio: 100, Rule: "named", Reason: "asked for by name"}
		}
		if !passes(v, strict, lowGold) {
			skip("filtered: " + v.String())
			continue
		}
		if v.Prio < o.MinPrio {
			skip("below min prio")
			continue
		}
		if v.Action == Sell && strict <= 1 {
			cells := 1
			if b != nil {
				cells = max(1, b.Width*b.Height)
			}
			// recovering: anything worth the PoorMinSell line is gold, the per-cell bar is for a character that can afford to wait
			if l.recovering.Load() {
				if strict == 1 && v.Rule != poorRule && SellValue(b, u.Item.Quality) < t.PoorMinSell {
					skip("not worth selling")
					continue
				}
			} else if strict == 1 && v.Rule != poorRule && SellValue(b, u.Item.Quality)/cells < t.MinValuePerCell {
				skip("not worth selling")
				continue
			}
		}
		if b != nil && b.Type == "gold" {
			// A broke character takes every pile: small piles are most of what Normal drops, and it needs the gold.
			if amt := c.Stats.Get(d2.StatGold); c.Stats != nil && int(amt) < t.GoldMinStack && s.Player.Gold >= l.poorBelow() {
				skip("small gold pile")
				continue
			}
			out = append(out, cand{u, v, d, name, s.Player.Gold})
			continue
		}
		// Potions the belt can take do not need pack space.
		if k := belt.PotionKind(b); k != belt.None && beltSt.Free() > 0 {
			out = append(out, cand{u, v, d, name, s.Player.Gold})
			continue
		}
		w, h := 1, 1
		if b != nil {
			w, h = max(1, b.Width), max(1, b.Height)
		}
		if spot != nil {
			keep := t.FreeCellsKeep
			if v.Prio >= 70 {
				keep = 0 // gold, runes, gems and uniques always take the last cells
			} else if v.Action == Sell && !l.recovering.Load() {
				keep += sellKeepExtra // junk to be sold never eats into the room the stash-worthy finds need
			}
			if !spot(w, h) || occ.FreeCells()-w*h < keep {
				skipped = append(skipped, Skipped{ID: u.ID, Name: name, Why: "no room", Prio: v.Prio})
				full = true
				continue
			}
		}
		out = append(out, cand{u, v, d, name, s.Player.Gold})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].v.Prio != out[j].v.Prio {
			return out[i].v.Prio > out[j].v.Prio
		}
		return out[i].dist < out[j].dist
	})
	return out, skipped, full
}

// namedItem reports whether the item's name (or its base name) is one of names.
func namedItem(names []string, shown string, b *data.ItemBase) bool {
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		if strings.EqualFold(shown, n) || b != nil && (strings.EqualFold(b.Display, n) || strings.EqualFold(b.Name, n)) {
			return true
		}
	}
	return false
}

// passes applies the strictness knob to a verdict: 0 loose S/A/B, 1 normal S/A (+B when poor), 2 S/A, 3 S.
func passes(v Verdict, strict int, lowGold bool) bool {
	if v.Action == Drop {
		return false
	}
	r := tierRank(v.Tier)
	switch {
	case strict <= 0:
		return r >= 1
	case strict == 1:
		return r >= 2 || r == 1 && lowGold
	case strict == 2:
		return r >= 2
	}
	return r >= 3
}

// pickOne walks next to the item, hover-confirms and clicks it, then proves the pickup in memory.
func (l *Looter) pickOne(ctx context.Context, c cand, o PickOpts) error {
	t := &l.e.Tune.Loot
	var last error
	for try := 0; try < t.PickTries; try++ {
		if o.Abort != nil && o.Abort() {
			return errAborted
		}
		s, err := l.hand.Fresh()
		if err != nil {
			return err
		}
		u, on := groundItem(s, c.u.ID)
		if !on {
			if ok, _ := l.arrived(s, c); ok {
				return nil
			}
			return errGone
		}
		d := math.Hypot(float64(u.X-s.Player.X), float64(u.Y-s.Player.Y))
		if d > float64(t.WalkDist) {
			if err := l.mv.GoTo(ctx, input.Point{X: u.X, Y: u.Y}); err != nil {
				last = fmt.Errorf("walk: %w", err)
				continue
			}
		}
		if hand.PanelOpen(s.UI) {
			if err := l.hand.CloseAll(ctx); err != nil {
				return err
			}
		}
		err = l.e.In.Interact(input.Target{Type: d2.TypeItem, ID: u.ID, Pos: input.Point{X: u.X, Y: u.Y}})
		if err != nil {
			last = fmt.Errorf("click: %w", err)
			if errors.Is(err, input.ErrOffscreen) || errors.Is(err, input.ErrNotHovered) {
				continue
			}
			return err
		}
		walk := time.Duration(d) * 90 * time.Millisecond
		timeout := time.Duration(t.ConfirmMs)*time.Millisecond + walk
		ns, ok, werr := l.hand.Wait(ctx, timeout, func(s *d2.Snapshot) bool {
			if _, on := groundItem(s, c.u.ID); !on {
				return true
			}
			return o.Abort != nil && o.Abort()
		})
		if werr != nil {
			return werr
		}
		if !ok {
			last = fmt.Errorf("item still on the ground after %v", timeout)
			continue
		}
		if _, on := groundItem(ns, c.u.ID); on { // abort fired
			return errAborted
		}
		if ns.Inventory.Cursor != nil && ns.Inventory.Cursor.ID == c.u.ID {
			if err := l.hand.ClearCursor(ctx); err != nil {
				return fmt.Errorf("loot: picked %s but could not place it: %w", c.name, err)
			}
			return nil
		}
		if ok, why := l.arrived(ns, c); ok {
			return nil
		} else {
			last = fmt.Errorf("%s: %s", c.name, why)
		}
	}
	if last == nil {
		last = errGone
	}
	return last
}

func groundItem(s *d2.Snapshot, id uint32) (d2.Unit, bool) {
	for _, u := range s.Ground {
		if u.ID == id && u.Mode == d2.ItemModeGround {
			return u, true
		}
	}
	return d2.Unit{}, false
}

// arrived says whether the item is now with the player: in a pack/belt list, merged into a stack, or (for
// gold) reflected in the gold stat. Stacked items (scrolls into a tome, keys) disappear without a unit of
// their own; that case is accepted for base types that stack.
func (l *Looter) arrived(s *d2.Snapshot, c cand) (bool, string) {
	if _, ok := hand.Find(s.Inventory, c.u.ID); ok {
		return true, ""
	}
	b := l.base(c.u.Class)
	if b != nil && b.Type == "gold" {
		return s.Player.Gold > c.gold0, "gold did not increase"
	}
	if b != nil && (b.Stackable || b.Type == "scro") {
		return true, ""
	}
	return false, "gone from the ground but not in the pack"
}

// LeftBehindMinPrio is the pickit priority from which an item left on the ground for lack of room counts as worth a
// town trip (stash-worthy bases, jewelry, gems, runes, uniques; plain junk and sell-only magic items stay below).
const LeftBehindMinPrio = 40

// noteLeftBehind remembers how many valuable items the last full-radius pickup left for lack of room. A pickup that
// leaves none (or a named one-item pickup, which says nothing about the rest) resets or keeps the count.
func (l *Looter) noteLeftBehind(skipped []Skipped, o PickOpts) {
	if len(o.Names) > 0 || o.MinPrio > 0 {
		return
	}
	n := 0
	for _, sk := range skipped {
		if sk.Why == "no room" && sk.Prio >= LeftBehindMinPrio {
			n++
		}
	}
	l.leftBehind.Store(int64(n))
}

// LeftBehind is the number of valuable items the last pickup could not take for lack of room. A town visit that
// made room calls ClearLeftBehind; the bot then goes back to pick them up.
func (l *Looter) LeftBehind() int { return int(l.leftBehind.Load()) }

// ClearLeftBehind forgets the count (after a town trip).
func (l *Looter) ClearLeftBehind() { l.leftBehind.Store(0) }
