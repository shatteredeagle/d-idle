package leveler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

// EquipReport is what one Equip call did.
type EquipReport struct {
	Swaps  []string
	Failed []string
	Err    error
}

// gearState splits the snapshot's items into worn items and pack candidates.
func (l *Leveler) gearState(s *d2.Snapshot) (eq map[int]Cand, pack []Cand) {
	eq = map[int]Cand{}
	alt := map[int]Cand{}
	// On an empty weapon set (the `w` swap, hand/weaponset.go) the gear sits at body 11/12: judge candidates against it
	// and never plan a weapon or shield into the empty active slots. Equip swaps back first.
	defer func() {
		if s != nil && hand.SetsOf(s.Inventory).OnEmptySet() {
			if c, ok := alt[d2.BodyAltR]; ok {
				eq[d2.BodyRHand] = c
			}
			if c, ok := alt[d2.BodyAltL]; ok {
				eq[d2.BodyLHand] = c
			}
		}
	}()
	if s == nil || s.Player == nil || s.Inventory == nil || l.e.Data == nil {
		return
	}
	for _, u := range s.Inventory.Items {
		if u.Item == nil {
			continue
		}
		switch u.Item.Where {
		case d2.ContEquipment:
			c := l.cand(s.Player, u)
			c.Usable = true // worn is by definition usable; its score is what matters
			eq[u.Item.Body] = c
			if u.Item.Body == d2.BodyAltR || u.Item.Body == d2.BodyAltL {
				alt[u.Item.Body] = c
			}
		case d2.ContInventory:
			if c := l.cand(s.Player, u); c.Slots != nil && !l.badEverywhere(u.ID, c.Slots, s.Player.Level) {
				pack = append(pack, c)
			}
		}
	}
	return
}

func (l *Leveler) baseCost(u d2.Unit) int {
	if l.e.Data != nil && u.Class >= 0 && u.Class < len(l.e.Data.Items) {
		return l.e.Data.Items[u.Class].Cost
	}
	return 0
}

// Candidate rates one item unit for p with its stat list when identified.
func (l *Leveler) Candidate(p *d2.Player, u d2.Unit) Cand { return l.cand(p, u) }

func (l *Leveler) cand(p *d2.Player, u d2.Unit) Cand {
	var st d2.Stats
	if u.Item != nil && u.Item.Identified() && l.e.Read != nil {
		st, _ = l.e.Read.UnitStats(u.Addr)
	}
	return Candidate(l.e.Data, p, u, st)
}

// SwapsFor returns what Equip would do for a snapshot (no clicks).
func (l *Leveler) SwapsFor(s *d2.Snapshot) []Swap {
	eq, pack := l.gearState(s)
	for {
		sws := ChooseGear(eq, pack)
		bad := -1
		for _, sw := range sws {
			if l.isBadSlot(sw.Item.Unit.ID, sw.Slot, s.Player.Level) {
				bad = packIndex(pack, sw.Item.Unit.ID)
				break
			}
		}
		if bad < 0 {
			return sws
		}
		pack = append(pack[:bad:bad], pack[bad+1:]...) // drop the item that failed here and plan again without it
	}
}

func packIndex(pack []Cand, id uint32) int {
	for i, c := range pack {
		if c.Unit.ID == id {
			return i
		}
	}
	return -1
}

// DumpItems prints every carried and worn item with its score, for tuning and live checks.
func (l *Leveler) DumpItems(w io.Writer, s *d2.Snapshot) {
	if s == nil || s.Inventory == nil {
		fmt.Fprintln(w, "no inventory")
		return
	}
	for _, u := range s.Inventory.Items {
		if u.Item == nil {
			continue
		}
		c := l.cand(s.Player, u)
		fmt.Fprintf(w, "%-9s body=%d cell=(%d,%d) class=%d %-22q q=%d id=%v flags=%#x ilvl=%d score=%.1f usable=%v slots=%v cost=%d %s\n",
			u.Item.Where, u.Item.Body, u.X, u.Y, u.Class, c.Name, u.Item.Quality, u.Item.Identified(), u.Item.Flags, u.Item.ILvl, c.Score, c.Usable, c.Slots, l.baseCost(u), c.Why)
	}
	if s.Inventory.Cursor != nil {
		fmt.Fprintln(w, "cursor holds item class", s.Inventory.Cursor.Class)
	}
}

// Equip equips every better item from the pack, one verified swap at a time, re-reading memory between swaps.
// Safe to call anywhere calm (town); it refuses while a fight is on.
func (l *Leveler) Equip(ctx context.Context) (rep EquipReport) {
	h := l.newHand()
	if !l.o.Dry {
		if sw, err := h.EnsureMainSet(ctx); err != nil {
			rep.Err = err
			return
		} else if sw {
			l.logf("weapon set swapped back to the gear set")
		}
	}
	for round := 0; round < 12; round++ {
		s, err := h.Fresh()
		if err != nil {
			rep.Err = err
			return
		}
		if l.busy(s) {
			rep.Err = ErrBusy
			return
		}
		sw := l.SwapsFor(s)
		if s.Player != nil {
			kept := sw[:0]
			for _, x := range sw {
				if !l.slotBlocked(x.Slot, s.Player.Level) {
					kept = append(kept, x)
				}
			}
			sw = kept
		}
		if len(sw) == 0 {
			break
		}
		if l.o.Dry {
			for _, x := range sw {
				rep.Swaps = append(rep.Swaps, "dry: "+x.String())
			}
			return
		}
		if err := l.doSwapRetry(ctx, h, s, sw[0]); err != nil {
			l.e.Log.Warn("leveler: equip failed", "swap", sw[0].String(), "err", err)
			rep.Failed = append(rep.Failed, sw[0].String()+": "+err.Error())
			l.markBadSwap(sw[0])
			if len(rep.Failed) >= 2 {
				break
			}
			continue
		}
		l.logf("equipped", "swap", sw[0].String())
		rep.Swaps = append(rep.Swaps, sw[0].String())
		l.mu.Lock()
		l.stats.EquipMoves++
		l.mu.Unlock()
	}
	if err := h.CloseAll(ctx); err != nil {
		l.e.Log.Warn("leveler: closing panels after equip failed", "err", err)
	}
	return
}

// badMark remembers failures at a character level. The thing is skipped until the character has gained as many
// levels as it failed times (capped at maxBackoff): a refused item is not worth a panel dance on every town visit.
type badMark struct{ level, fails int }

const maxBackoff = 4

func (b badMark) blocked(level int) bool {
	return b.fails > 0 && level < b.level+min(b.fails, maxBackoff)
}

// markBadSwap stops the same swap from being retried forever: the game refused the item at this character level
// (hidden affix requirements, strength), so it is skipped until the character gains a level (two after a second
// failure, and so on). Two different items failing into one slot at the same level block the slot for the same time,
// so a pile of unwearable amulets does not cost a retry each (live: 4 amulets in a row, 2026-10-29).
func (l *Leveler) markBadSwap(sw Swap) {
	lvl := l.curLevel()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.bad == nil {
		l.bad = map[badKey]badMark{}
		l.badSlot = map[int]badMark{}
	}
	k := badKey{sw.Item.Unit.ID, sw.Slot}
	m := l.bad[k]
	l.bad[k] = badMark{lvl, m.fails + 1}
	if sw.Slot > 0 {
		sm := l.badSlot[sw.Slot]
		if sm.level != lvl {
			sm = badMark{level: lvl}
		}
		sm.fails++
		l.badSlot[sw.Slot] = sm
	}
}

func (l *Leveler) curLevel() int {
	if l.o.Snapshot != nil {
		if s, err := l.o.Snapshot(); err == nil && s.Player != nil {
			return s.Player.Level
		}
	}
	if l.e.Read != nil {
		if p, err := l.e.Read.Player(); err == nil {
			return p.Level
		}
	}
	return 0
}

// badKey is one failed (item, body slot). Slot -1 is a failed purchase, which is not tied to a slot.
type badKey struct {
	id   uint32
	slot int
}

// badEverywhere: the item failed purchase, or every slot it fits, at this character level (a ring that missed one
// slot may still go to the other).
func (l *Leveler) badEverywhere(id uint32, slots []int, level int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.bad[badKey{id, -1}].blocked(level) {
		return true
	}
	for _, sl := range slots {
		if !l.bad[badKey{id, sl}].blocked(level) {
			return false
		}
	}
	return len(slots) > 0
}

// isBadSlot: this item failed to reach this slot recently (backoff by failure count; a level up or a new id
// clears it, so the amulet that never reached `neck` is not retried on every town visit, soak R7).
func (l *Leveler) isBadSlot(id uint32, slot, level int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.bad[badKey{id, slot}].blocked(level)
}

// slotBlocked: two items already failed into the slot at this level.
func (l *Leveler) slotBlocked(slot, level int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	m := l.badSlot[slot]
	return m.fails >= 2 && m.blocked(level)
}

// swapAttempts is how often one swap is tried before it is marked bad: a ring click once missed its slot live
// ("item did not reach slot rHand") although the same click works the next time.
const swapAttempts = 2

// doSwapRetry runs doSwap, and after a failure re-reads memory: the item may have landed late (then the swap is
// done), otherwise the panels are closed and the swap is tried again from a fresh snapshot.
func (l *Leveler) doSwapRetry(ctx context.Context, h *hand.Hand, s *d2.Snapshot, sw Swap) (err error) {
	for try := 0; try < swapAttempts; try++ {
		if err = l.doSwap(ctx, h, s, sw, try); err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return err
		}
		if err := h.CloseAll(ctx); err != nil {
			l.e.Log.Warn("leveler: closing panels before an equip retry failed", "err", err)
		}
		fresh, ferr := h.Fresh()
		if ferr != nil {
			return ferr
		}
		if swapLanded(fresh.Inventory, sw.Item.Unit.ID, sw.Slot) {
			l.e.Log.Info("leveler: item reached its slot after all", "swap", sw.String())
			return nil
		}
		l.e.Log.Warn("leveler: equip retry", "swap", sw.String(), "try", try+1, "err", err)
		s = fresh
	}
	return err
}

// swapLanded reports whether the item is worn: at slot, or at the other ring slot when a ring was swapped.
func swapLanded(inv *d2.Inventory, id uint32, slot int) bool {
	it, ok := hand.Find(inv, id)
	if !ok || it.Item == nil || it.Item.Where != d2.ContEquipment {
		return false
	}
	ring := func(b int) bool { return b == d2.BodyRRing || b == d2.BodyLRing }
	return it.Item.Body == slot || ring(slot) && ring(it.Item.Body)
}

var errSlot = errors.New("leveler: no screen position for that equipment slot")

// doSwap moves one pack item into its slot: pick it up, click the slot, and prove (cursor empty, item worn at the
// slot). A displaced item lands on the cursor and goes back into the pack.
func (l *Leveler) doSwap(ctx context.Context, h *hand.Hand, s *d2.Snapshot, sw Swap, try int) error {
	var err error
	if s, err = h.OpenInventory(ctx); err != nil {
		return err
	}
	lay, err := h.Layout(s)
	if err != nil {
		return err
	}
	r, ok := lay.Slots[slotName[sw.Slot]]
	if !ok {
		return fmt.Errorf("%w: %s", errSlot, slotName[sw.Slot])
	}
	if s.Inventory.Cursor != nil {
		if err := h.ClearCursor(ctx); err != nil {
			return fmt.Errorf("cursor not empty: %w", err)
		}
	}
	u := sw.Item.Unit
	if err := h.Take(ctx, u, lay.Inventory); err != nil {
		return err
	}
	c := hand.SlotPoint(r, try)
	if err := l.hoverClick(Pt{c.X, c.Y}); err != nil {
		return err
	}
	id, slot := u.ID, sw.Slot
	s2, done, err := h.Wait(ctx, 2*time.Second, func(s *d2.Snapshot) bool {
		it, found := hand.Find(s.Inventory, id)
		return found && it.Item != nil && it.Item.Where == d2.ContEquipment && it.Item.Body == slot
	})
	if err != nil {
		return err
	}
	if !done {
		if l.o.DebugHold > 0 {
			l.sleep(l.o.DebugHold)
		}
		return l.giveBack(ctx, h, u, slot, c, try, s2)
	}
	if s2.Inventory.Cursor != nil { // the displaced item
		if err := h.ClearCursor(ctx); err != nil {
			return fmt.Errorf("displaced item stays on the cursor: %w", err)
		}
	}
	return nil
}

// giveBack handles a refused slot click: the item is on the cursor (or already back), so it returns to the pack, and
// the error says what the cursor and the item looked like so a live run shows whether the click was refused (item
// still on the cursor: requirement or click point) or went elsewhere. The pack is the only place a failed equip
// may leave the item, with an empty cursor; if that cannot be proven the error says so.
func (l *Leveler) giveBack(ctx context.Context, h *hand.Hand, u d2.Unit, slot int, c panel.Pt, try int, s2 *d2.Snapshot) error {
	onCursor := s2 != nil && s2.Inventory != nil && s2.Inventory.Cursor != nil && s2.Inventory.Cursor.ID == u.ID
	lvl := 0
	if s2 != nil && s2.Player != nil {
		lvl = s2.Player.Level
	}
	l.e.Log.Warn("leveler: slot click refused", "item", u.ID, "slot", slotName[slot], "at", c, "try", try,
		"stillOnCursor", onCursor, "clvl", lvl)
	if err := h.ClearCursor(ctx); err != nil {
		return fmt.Errorf("item %d did not reach slot %s and the cursor could not be emptied: %w", u.ID, slotName[slot], err)
	}
	if fresh, err := h.Fresh(); err == nil {
		if fresh.Inventory.Cursor != nil {
			return fmt.Errorf("item %d did not reach slot %s and the cursor is not empty", u.ID, slotName[slot])
		}
		if it, ok := hand.Find(fresh.Inventory, u.ID); ok && it.Item != nil && it.Item.Where != d2.ContInventory && it.Item.Where != d2.ContEquipment {
			return fmt.Errorf("item %d did not reach slot %s and is not back in the pack (%v)", u.ID, slotName[slot], it.Item.Where)
		}
	}
	return fmt.Errorf("item %d did not reach slot %s", u.ID, slotName[slot])
}
