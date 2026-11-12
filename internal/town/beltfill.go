package town

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

// BeltFillReport is what one FillBelt call did.
type BeltFillReport struct {
	Moved    int
	Dragged  int // of Moved, how many needed the drag-and-drop fallback
	Failed   int
	Repaired int // belt potions moved out to the pack to free a column (beltrepair.go)
	Before   belt.State
	After    belt.State
	Attempts int
}

// beltFillCandidates returns the pack potions worth moving to the belt: per kind at most the room the plan still has
// in its columns, best tier first, and never more than the free slots.
func (t *Town) beltFillCandidates(s *d2.Snapshot) []d2.Unit {
	if s == nil || s.Inventory == nil || s.Player == nil {
		return nil
	}
	band := t.plan(s.Player)
	st := belt.Read(t.e.Data, s.Inventory)
	free := t.beltRoom(s.Inventory, st)
	var out []d2.Unit
	// Each candidate is placed on a copy of the belt with belt.PlaceFor, so a kind with no room (every free box sits in a column
	// of another kind) yields no candidate instead of a failed move, and kinds without a column go first.
	placed := belt.PlacedItems(t.e.Data, s.Inventory)
	hasCol := map[belt.Kind]bool{}
	for _, p := range placed {
		if p.Box < belt.Columns {
			hasCol[p.Kind] = true
		}
	}
	kinds := []belt.Kind{belt.HP, belt.MP, belt.RV}
	sort.SliceStable(kinds, func(i, j int) bool { return !hasCol[kinds[i]] && hasCol[kinds[j]] })
	try := func(u d2.Unit, k belt.Kind) bool {
		box, ok := belt.PlaceFor(st.Slots, placed, k, band)
		if ok {
			placed = append(placed, belt.Placed{Box: box, Kind: k})
			out = append(out, u)
			free--
		}
		return ok
	}
	for _, k := range kinds {
		room := band.Capacity(k, st.Slots) - st.Count[k]
		var pack []d2.Unit
		for _, u := range s.Inventory.Container(d2.ContInventory) {
			if belt.PotionKind(t.base(u)) == k {
				pack = append(pack, u)
			}
		}
		sort.SliceStable(pack, func(i, j int) bool { return belt.Tier(t.base(pack[i])) > belt.Tier(t.base(pack[j])) })
		for _, u := range pack {
			if room <= 0 || free <= 0 || !try(u, k) {
				break
			}
			room--
		}
	}
	// REJUVUSE: pack rejuvenation potions beyond the plan's RV column (none below clvl 31) still go to a free slot, last,
	// so the belt hotkey drinks them in an emergency (a pack right-click mid-fight costs frames and needs the panel).
	var extra []d2.Unit
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if belt.PotionKind(t.base(u)) == belt.RV && !slices.ContainsFunc(out, func(o d2.Unit) bool { return o.ID == u.ID }) {
			extra = append(extra, u)
		}
	}
	sort.SliceStable(extra, func(i, j int) bool { return belt.Tier(t.base(extra[i])) > belt.Tier(t.base(extra[j])) })
	for _, u := range extra {
		if free <= 0 || !try(u, belt.RV) {
			break
		}
	}
	return out
}

// NeedsBeltFill is the pure check: the pack holds potions the belt has room and plan for.
func (t *Town) NeedsBeltFill(s *d2.Snapshot) bool {
	if len(t.beltFillCandidates(s)) > 0 {
		return true
	}
	_, _, _, repair := t.beltRepair(s) // BELTFIX: a belt with no column for a planned kind (docs/beltfix.md)
	return repair
}

// FillBelt moves pack potions to free belt slots by shift-click (drag and drop when shift-click does nothing) and
// re-reads the belt after every move. It opens the backpack itself and closes it again. Safe in town and, with no
// hostile near, in the field. The returned error is the last move failure, if any move failed.
func (t *Town) FillBelt(ctx context.Context) (BeltFillReport, error) {
	rep, err := t.fillBelt(ctx)
	if err != nil || rep.Moved == 0 {
		return rep, err
	}
	// BELTFIX: the first pass may have taken the last free column; if a planned kind still has pack potions and no column,
	// free a surplus one and fill again (once: the second pass does not loop back).
	if s, e := t.current(); e == nil {
		if _, _, _, ok := t.beltRepair(s); ok {
			r2, err2 := t.fillBelt(ctx)
			rep.Moved += r2.Moved
			rep.Dragged += r2.Dragged
			rep.Failed += r2.Failed
			rep.Repaired += r2.Repaired
			rep.Attempts += r2.Attempts
			rep.After = r2.After
			return rep, err2
		}
	}
	return rep, err
}

func (t *Town) fillBelt(ctx context.Context) (BeltFillReport, error) {
	var rep BeltFillReport
	s, err := t.current()
	if err != nil {
		return rep, err
	}
	cand := t.beltFillCandidates(s)
	rep.Before = belt.Read(t.e.Data, s.Inventory)
	rep.After = rep.Before
	_, _, _, repair := t.beltRepair(s)
	if len(cand) == 0 && !repair {
		return rep, nil
	}
	s, err = t.h.OpenInventory(ctx)
	if err != nil {
		return rep, err
	}
	defer func() { _ = t.h.CloseAll(ctx) }()
	lay, err := t.layout(s)
	if err != nil {
		return rep, err
	}
	if repair {
		t.repairBelt(ctx, lay, &rep)
		if s, err = t.current(); err != nil {
			return rep, err
		}
	}
	// F9 (BELTFIX): the candidate ids come from the snapshot read with the pack open, not the one from before the open: right
	// after a re-entry the first read can carry ids the game no longer lists.
	cand = t.beltFillCandidates(s)
	var last error
	refreshed := 0
	for i := 0; i < len(cand); i++ {
		u := cand[i]
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		dragged, err := t.moveToBelt(ctx, lay, u, &rep)
		if err != nil {
			// F1 (BELTFIX): a vanished item id is a stale read, not a full belt: re-read the pack and try the rest again, and
			// never mark the belt full for it.
			if errors.Is(err, hand.ErrNoItem) && refreshed < 2 {
				refreshed++
				t.e.Log.Info("belt: item id gone, re-reading the pack", "item", t.nameOf(u), "id", u.ID, "pass", refreshed)
				if cur, e := t.current(); e == nil {
					cand = t.beltFillCandidates(cur)
					i = -1
				}
				continue
			}
			// FIXBELT2: no column for this kind means no room for it, not a full belt: leave it in the pack, try the others.
			if errors.Is(err, errNoBeltColumn) {
				t.e.Log.Info("belt: no room for this kind, left in the pack", "item", t.nameOf(u), "err", err)
				continue
			}
			rep.Failed++
			last = err
			t.e.Log.Warn("belt: move failed", "item", t.nameOf(u), "err", err)
			if !errors.Is(err, hand.ErrNoItem) && ctx.Err() == nil {
				if cur, e := t.current(); e == nil {
					t.markBeltFull(cur.Inventory, err)
				}
			}
			if rep.Failed >= 2 { // the belt is full or the clicks do not land: stop clicking
				break
			}
			continue
		}
		rep.Moved++
		t.clearBeltFull("a potion moved")
		if dragged {
			rep.Dragged++
		}
	}
	if s, err := t.current(); err == nil {
		rep.After = belt.Read(t.e.Data, s.Inventory)
	}
	t.e.Log.Info(fmt.Sprintf("belt: moved %d potions from pack (hp %d->%d mp %d->%d rv %d->%d)", rep.Moved,
		rep.Before.Count[belt.HP], rep.After.Count[belt.HP], rep.Before.Count[belt.MP], rep.After.Count[belt.MP], rep.Before.Count[belt.RV], rep.After.Count[belt.RV]),
		"rv", rep.After.Count[belt.RV], "dragged", rep.Dragged, "failed", rep.Failed, "repaired", rep.Repaired)
	return rep, last
}

var errNoBeltColumn = errors.New("town: no belt column")

// beltPutter is an input that can move a pack item into a belt box directly (d2d: "cmd belt put").
type beltPutter interface {
	BeltPut(id uint32, box int) error
}

// pickBeltBox is the belt box a potion of kind k goes to (F8): belt.Place, the rule game.exe applies itself (B4: the
// first same-kind column with room at its lowest free box, else the first empty column; never above another kind). The
// plan band no longer steers the choice, because a shift-click on Game.exe lands where the game puts it, so choosing
// another column would make the bot's picture of the belt wrong. Reports false when no column can take k.
func (t *Town) pickBeltBox(inv *d2.Inventory, slots int, k belt.Kind, _ belt.Band) (int, bool) {
	return belt.Place(slots, belt.PlacedItems(t.e.Data, inv), k)
}

func inBelt(s *d2.Snapshot, id uint32) bool {
	it, ok := hand.Find(s.Inventory, id)
	return ok && it.Item != nil && it.Item.Where == d2.ContBelt
}

// moveToBelt shift-clicks u, retrying MoveTries times, then falls back to picking it up and dropping it on the belt
// slot (U: where a potion dropped on the belt slot lands is not verified live). Reports whether the fallback was used.
func (t *Town) moveToBelt(ctx context.Context, lay *panel.Layout, u d2.Unit, rep *BeltFillReport) (bool, error) {
	for try := 0; try < max(t.tune().MoveTries, 1); try++ {
		rep.Attempts++
		cur, err := t.current()
		if err != nil {
			return false, err
		}
		it, ok := hand.Find(cur.Inventory, u.ID)
		if !ok {
			return false, hand.ErrNoItem
		}
		if inBelt(cur, u.ID) {
			return false, nil
		}
		if bp, ok := t.h.E.In.(beltPutter); ok {
			kind := belt.PotionKind(t.base(it))
			box, free := t.pickBeltBox(cur.Inventory, belt.Slots(t.e.Data, cur.Inventory), kind, t.plan(cur.Player))
			if !free {
				// F8: the free boxes sit in columns of another kind; do not put it there and do not drag it either
				return false, fmt.Errorf("%w for %s (kind %s)", errNoBeltColumn, t.nameOf(u), kind)
			}
			perr := bp.BeltPut(u.ID, box)
			var landed bool
			if perr == nil {
				_, landed, perr = t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return inBelt(s, u.ID) })
			}
			t.logBeltPut(u, box, kind, try, landed, perr)
			if perr != nil {
				return false, perr
			}
			if landed {
				return false, nil
			}
			continue
		}
		if err := t.h.ShiftClick(it, lay.Inventory); err != nil {
			if errors.Is(err, input.ErrUnsupported) { // d2d has no click modifiers: go straight to the drag
				break
			}
			return false, err
		}
		if _, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return inBelt(s, u.ID) }); err != nil {
			return false, err
		} else if ok {
			return false, nil
		}
	}
	// A full belt looks the same as a swallowed click: do not start a drag when the belt has no room.
	if cur, err := t.current(); err == nil && t.beltRoom(cur.Inventory, belt.Read(t.e.Data, cur.Inventory)) <= 0 {
		return false, fmt.Errorf("town: belt has no free slot for %s", t.nameOf(u))
	}
	slot, ok := lay.Slots["belt"]
	if !ok {
		return false, errors.New("town: no belt slot in the layout for the drag fallback")
	}
	cur, err := t.current()
	if err != nil {
		return false, err
	}
	before := belt.Read(t.e.Data, cur.Inventory).Total()
	aim := beltCellPoint(slot, cur.Inventory, belt.Slots(t.e.Data, cur.Inventory))
	it, ok := hand.Find(cur.Inventory, u.ID)
	if !ok {
		return false, hand.ErrNoItem
	}
	if err := t.h.Take(ctx, it, lay.Inventory); err != nil {
		return true, err
	}
	err = t.h.E.In.Click(input.Left, aim.X, aim.Y)
	if err == nil {
		_, ok, err = t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool {
			return s.Inventory.Cursor == nil && inBelt(s, u.ID) && belt.Read(t.e.Data, s.Inventory).Total() > before
		})
		if err == nil && !ok {
			err = fmt.Errorf("town: drag of %s onto the belt did not land", t.nameOf(u))
		}
	}
	if err != nil {
		_ = t.h.ClearCursor(ctx)
		return true, err
	}
	return true, nil
}

// logBeltPut is the diagnostic line of one put attempt (primitives-review F9): the id, the box and kind chosen, whether it
// landed, the cursor and the belt boxes and pack potion ids of a fresh read, so the next soak shows a stale id, an id that
// changed on the move or an item left on the cursor.
func (t *Town) logBeltPut(u d2.Unit, box int, kind belt.Kind, try int, landed bool, err error) {
	cursor, boxes, ids := "-", "", ""
	if cur, e := t.current(); e == nil && cur.Inventory != nil {
		if c := cur.Inventory.Cursor; c != nil {
			cursor = fmt.Sprint(c.ID)
		}
		for _, b := range cur.Inventory.Container(d2.ContBelt) {
			if b.Item != nil {
				boxes += fmt.Sprintf(" %d:%s", b.Item.GY*belt.Columns+b.Item.GX, t.nameOf(b))
			}
		}
		for _, p := range cur.Inventory.Container(d2.ContInventory) {
			if belt.PotionKind(t.base(p)) != belt.None {
				ids += fmt.Sprintf(" %d", p.ID)
			}
		}
	}
	t.e.Log.Info(fmt.Sprintf("belt: put id=%d box=%d kind=%s try=%d -> inBelt=%v cursor=%s beltBoxes=[%s] packIDs=[%s] err=%v at=%s",
		u.ID, box, kind, try, landed, cursor, strings.TrimSpace(boxes), strings.TrimSpace(ids), err, t.e.Clock.Now().Format("15:04:05")))
}

// beltRoom is the number of empty belt cells. Free() counts slots minus potions, which overstates the room when the
// belt is smaller than the equipped belt's table row says (live Fresha 2026-10-30: a full 4-slot belt, Free() > 0, twelve
// drags that did not land). The occupied cells are read from the item positions when they are all inside the belt grid, and
// a belt that refused a move is not tried again until its contents change.
func (t *Town) beltRoom(inv *d2.Inventory, st belt.State) int {
	free := st.Free()
	cells := map[[2]int]bool{}
	valid := true
	for _, u := range inv.Container(d2.ContBelt) {
		// box = row*4+column covers both backends: Game.exe reports (column,row), d2d reports the box in GX with GY 0 (F7)
		if u.Item == nil || u.Item.GX < 0 || u.Item.GY < 0 || u.Item.GY*belt.Columns+u.Item.GX >= st.Slots {
			valid = false
			break
		}
		cells[[2]int{u.Item.GY*belt.Columns + u.Item.GX, 0}] = true
	}
	if valid && len(cells) > 0 {
		free = min(free, st.Slots-len(cells))
	}
	if t.beltFullAt != nil {
		if *t.beltFullAt == t.beltSig(inv, st) {
			return 0
		}
		t.clearBeltFull("belt or pack potions changed")
	}
	return max(free, 0)
}

// beltSignature is what the sticky "belt refused" mark compares: the belt's contents and the pack's potion set. A change
// of either means the refusal may no longer hold (F1: it used to watch the belt alone).
type beltSignature struct {
	slots, total int
	packN        int
	packSum      uint64
}

func (t *Town) beltSig(inv *d2.Inventory, st belt.State) beltSignature {
	sig := beltSignature{slots: st.Slots, total: st.Total()}
	if inv == nil {
		return sig
	}
	for _, u := range inv.Container(d2.ContInventory) {
		if belt.PotionKind(t.base(u)) != belt.None {
			sig.packN++
			sig.packSum += uint64(u.ID)
		}
	}
	return sig
}

// markBeltFull sets the sticky flag: no more moves until the belt or the pack's potions change, a new game starts
// (ForgetBelt) or a move succeeds.
func (t *Town) markBeltFull(inv *d2.Inventory, why error) {
	sig := t.beltSig(inv, belt.Read(t.e.Data, inv))
	t.beltFullAt = &sig
	t.e.Log.Info(fmt.Sprintf("belt: marked full sig=%d,%d pack=%d reason=%v", sig.slots, sig.total, sig.packN, why))
}

func (t *Town) clearBeltFull(why string) {
	if t.beltFullAt != nil {
		t.beltFullAt = nil
		t.e.Log.Info("belt: full mark cleared (" + why + ")")
	}
}

// ForgetBelt drops the sticky belt-full mark. The runner calls it when a new game starts: a refusal from the last game
// says nothing about this one (F1: the mark survived every save-and-exit).
func (t *Town) ForgetBelt() { t.clearBeltFull("new game") }

// beltCellPoint is the centre of the first empty belt cell inside the belt slot rectangle: columns left to right, row 0
// at the bottom (U: the geometry of the belt box is not verified live; the whole-box centre used before hit an occupied
// cell and swapped or bounced).
func beltCellPoint(slot panel.Rect, inv *d2.Inventory, slots int) panel.Pt {
	rows := max(slots/belt.Columns, 1)
	taken := map[[2]int]bool{}
	for _, u := range inv.Container(d2.ContBelt) {
		if u.Item != nil {
			taken[[2]int{u.Item.GX, u.Item.GY}] = true
		}
	}
	cw, ch := (slot.R-slot.L)/belt.Columns, (slot.B-slot.T)/rows
	for r := 0; r < rows; r++ {
		for c := 0; c < belt.Columns; c++ {
			if !taken[[2]int{c, r}] {
				return panel.Pt{X: slot.L + c*cw + cw/2, Y: slot.B - r*ch - ch/2}
			}
		}
	}
	return slot.Center()
}
