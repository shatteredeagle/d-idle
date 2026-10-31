package combat

import (
	"context"
	"fmt"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// LeftPopupGeometry places the icons of the LEFT-skill popup on the 800x600 client. VERIFIED LIVE (CLASSPAL, Paladin
// Palli, docs/classes/smoke-paladin.md): the button is at (140,578); the grid grows to the right and up from x=104;
// the bottom row (y=480) holds Attack, the row above (y=433) the Paladin combat skills in tree order Sacrifice 104,
// Smite 153, Holy Bolt 202, Zeal 251 (pitch 49). Only skills with leftskill=1 are listed, so rows are counted among
// those. The row spacing of 47 above row 0 is measured once; every pick is verified by reading LeftSkill back.
type LeftPopupGeometry struct {
	BtnX, BtnY int
	Col0X      int // x of the first (leftmost) icon column
	ColDX      int // column pitch, to the right
	Row0Y      int // y of the bottom row
	RowDY      int // row pitch, upward
}

// LeftPopup is the geometry SetLeft uses.
var LeftPopup = LeftPopupGeometry{BtnX: 140, BtnY: 578, Col0X: 104, ColDX: 49, Row0Y: 480, RowDY: 47}

// leftCells returns the candidate screen cells of skill id in the left popup, most likely first: the cell the
// tree position predicts (row, then column order within the ListRow), then the rest of its row.
func (f *Fighter) leftCells(p *d2.Player, id int) []input.Screen {
	row, ok := f.listRows[id]
	if !ok || row < 0 {
		return nil
	}
	type ent struct{ id, row, tr, tc int }
	byRow := map[int][]ent{}
	for _, sk := range p.Skills {
		def := f.cat.Skill(sk.ID)
		r, ok := f.listRows[sk.ID]
		if def == nil || !ok || r < 0 || !(def.LeftSkill || sk.ID == 0) {
			continue
		}
		byRow[r] = append(byRow[r], ent{sk.ID, r, def.Row, def.Column})
	}
	if _, ok := byRow[row]; !ok {
		return nil
	}
	var rows []int
	for r := range byRow {
		rows = append(rows, r)
	}
	sort.Ints(rows)
	idx := sort.SearchInts(rows, row)
	es := byRow[row]
	sort.Slice(es, func(i, j int) bool {
		if es[i].tr != es[j].tr {
			return es[i].tr < es[j].tr
		}
		if es[i].tc != es[j].tc {
			return es[i].tc < es[j].tc
		}
		return es[i].id < es[j].id
	})
	g := LeftPopup
	y := g.Row0Y - idx*g.RowDY
	var first, rest []input.Screen
	for c, e := range es {
		cell := input.Screen{X: g.Col0X + c*g.ColDX, Y: y}
		if e.id == id {
			first = append(first, cell)
		} else {
			rest = append(rest, cell)
		}
	}
	return append(first, rest...)
}

func (f *Fighter) openLeftPopup(ctx context.Context) error {
	if err := f.e.In.Click(input.Left, LeftPopup.BtnX, LeftPopup.BtnY); err != nil {
		return err
	}
	return f.sleep(ctx, 400*time.Millisecond)
}

// readHands returns the current left and right skill ids after the game had time to apply a pick.
func (f *Fighter) readHands(ctx context.Context) (left, right int, err error) {
	if err = f.sleep(ctx, 200*time.Millisecond); err != nil {
		return
	}
	s, err := f.read(ctx)
	if err != nil {
		return
	}
	return s.Player.LeftSkill, s.Player.RightSkill, nil
}

// SetLeft makes skill id the left-hand skill through the left popup and verifies it by reading LeftSkill back. A cell
// that gave another skill is not an error, the next candidate is tried. The right hand (the aura) is never touched.
func (f *Fighter) SetLeft(ctx context.Context, id int) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	p := snap.Player
	if p.LeftSkill == id {
		return nil
	}
	if len(f.listRows) == 0 {
		return fmt.Errorf("%w: popup rows not loaded (SetListRows)", ErrBindFailed)
	}
	cells := f.leftCells(p, id)
	if c, ok := f.leftCell[id]; ok { // a cell that worked before goes first
		cells = append([]input.Screen{c}, cells...)
	}
	if len(cells) == 0 {
		return fmt.Errorf("%w: skill %d has no left popup cell", ErrBindFailed, id)
	}
	tried := map[input.Screen]bool{}
	for _, c := range cells {
		if tried[c] {
			continue
		}
		tried[c] = true
		if err := f.openLeftPopup(ctx); err != nil {
			return err
		}
		if err := f.e.In.Click(input.Left, c.X, c.Y); err != nil { // picking an icon closes the popup
			return err
		}
		left, _, err := f.readHands(ctx)
		if err != nil {
			return err
		}
		if left == id {
			f.leftCell[id] = c
			f.info("left skill set", "skill", id, "cell", c)
			return nil
		}
	}
	return fmt.Errorf("%w: left skill %d not found in %d popup cells", ErrBindFailed, id, len(tried))
}

// SetRight makes skill id the right-hand skill: by its verified hotkey when it has one (about 200 ms, no mana), else
// through the right popup. The left hand is not touched.
func (f *Fighter) SetRight(ctx context.Context, id int) error {
	snap, err := f.read(ctx)
	if err != nil {
		return err
	}
	p := snap.Player
	if p.RightSkill == id {
		return nil
	}
	if key := f.keys[id]; key != "" {
		if err := f.e.In.Key(key); err != nil {
			return err
		}
		if _, r, err := f.readHands(ctx); err != nil || r == id {
			return err
		}
	}
	for _, c := range f.popupCells(p, id) {
		if err := f.openPopup(ctx); err != nil {
			return err
		}
		if err := f.e.In.Click(input.Left, c.X, c.Y); err != nil {
			return err
		}
		if _, r, err := f.readHands(ctx); err != nil || r == id {
			return err
		}
	}
	return fmt.Errorf("%w: right skill %d", ErrBindFailed, id)
}
