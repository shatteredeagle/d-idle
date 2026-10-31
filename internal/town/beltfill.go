package town

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

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
	for _, k := range []belt.Kind{belt.HP, belt.MP, belt.RV} {
		room := band.Capacity(k, st.Slots) - st.Count[k]
		var pack []d2.Unit
		for _, u := range s.Inventory.Container(d2.ContInventory) {
			if belt.PotionKind(t.base(u)) == k {
				pack = append(pack, u)
			}
		}
		sort.SliceStable(pack, func(i, j int) bool { return belt.Tier(t.base(pack[i])) > belt.Tier(t.base(pack[j])) })
		for _, u := range pack {
			if room <= 0 || free <= 0 {
				break
			}
			out = append(out, u)
			room--
			free--
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
		if free <= 0 {
			break
		}
		out = append(out, u)
		free--
	}
	return out
}

// NeedsBeltFill is the pure check: the pack holds potions the belt has room and plan for.
func (t *Town) NeedsBeltFill(s *d2.Snapshot) bool { return len(t.beltFillCandidates(s)) > 0 }

// FillBelt moves pack potions to free belt slots by shift-click (drag and drop when shift-click does nothing) and
// re-reads the belt after every move. It opens the backpack itself and closes it again. Safe in town and, with no
// hostile near, in the field. The returned error is the last move failure, if any move failed.
func (t *Town) FillBelt(ctx context.Context) (BeltFillReport, error) {
	var rep BeltFillReport
	s, err := t.current()
	if err != nil {
		return rep, err
	}
	cand := t.beltFillCandidates(s)
	rep.Before = belt.Read(t.e.Data, s.Inventory)
	rep.After = rep.Before
	if len(cand) == 0 {
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
	var last error
	for _, u := range cand {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		dragged, err := t.moveToBelt(ctx, lay, u, &rep)
		if err != nil {
			rep.Failed++
			last = err
			t.e.Log.Warn("belt: move failed", "item", t.nameOf(u), "err", err)
			if cur, e := t.current(); e == nil {
				sig := beltSig(belt.Read(t.e.Data, cur.Inventory))
				t.beltFullAt = &sig // no more drags until a potion enters or leaves the belt
			}
			if rep.Failed >= 2 { // the belt is full or the clicks do not land: stop clicking
				break
			}
			continue
		}
		rep.Moved++
		t.beltFullAt = nil
		if dragged {
			rep.Dragged++
		}
	}
	if s, err := t.current(); err == nil {
		rep.After = belt.Read(t.e.Data, s.Inventory)
	}
	t.e.Log.Info(fmt.Sprintf("belt: moved %d potions from pack (hp %d->%d mp %d->%d rv %d->%d)", rep.Moved,
		rep.Before.Count[belt.HP], rep.After.Count[belt.HP], rep.Before.Count[belt.MP], rep.After.Count[belt.MP], rep.Before.Count[belt.RV], rep.After.Count[belt.RV]),
		"rv", rep.After.Count[belt.RV], "dragged", rep.Dragged, "failed", rep.Failed)
	return rep, last
}

// beltPutter is an input that can move a pack item into a belt box directly (d2d: "cmd belt put").
type beltPutter interface {
	BeltPut(id uint32, box int) error
}

// freeBeltBox is the lowest belt box (row*4+column) that holds no item, within the slots the belt has.
func freeBeltBox(inv *d2.Inventory, slots int) (int, bool) {
	used := map[int]bool{}
	for _, u := range inv.Container(d2.ContBelt) {
		if u.Item != nil {
			used[u.Item.GY*belt.Columns+u.Item.GX] = true
		}
	}
	for b := 0; b < slots; b++ {
		if !used[b] {
			return b, true
		}
	}
	return 0, false
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
			if box, free := freeBeltBox(cur.Inventory, belt.Slots(t.e.Data, cur.Inventory)); free {
				if err := bp.BeltPut(u.ID, box); err != nil {
					return false, err
				}
				if _, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return inBelt(s, u.ID) }); err != nil {
					return false, err
				} else if ok {
					return false, nil
				}
				continue
			}
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

// beltRoom is the number of empty belt cells. Free() counts slots minus potions, which overstates the room when the
// belt is smaller than the equipped belt's table row says (live Fresha 2026-10-30: a full 4-slot belt, Free() > 0, twelve
// drags that did not land). The occupied cells are read from the item positions when they are all inside the belt grid, and
// a belt that refused a move is not tried again until its contents change.
func (t *Town) beltRoom(inv *d2.Inventory, st belt.State) int {
	free := st.Free()
	cells := map[[2]int]bool{}
	valid := true
	for _, u := range inv.Container(d2.ContBelt) {
		if u.Item == nil || u.Item.GX < 0 || u.Item.GX >= belt.Columns || u.Item.GY < 0 || u.Item.GY >= st.Rows() {
			valid = false
			break
		}
		cells[[2]int{u.Item.GX, u.Item.GY}] = true
	}
	if valid && len(cells) > 0 {
		free = min(free, st.Slots-len(cells))
	}
	if t.beltFullAt != nil && *t.beltFullAt == beltSig(st) {
		return 0
	}
	return max(free, 0)
}

type beltSignature struct{ slots, total int }

func beltSig(st belt.State) beltSignature { return beltSignature{st.Slots, st.Total()} }

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
