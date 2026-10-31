package d2dsrc

import "d2lod-bot/internal/d2"

// cursor fills Inventory.Cursor. d2d's "dump items" does not list the item on the mouse and "dump ui" only gives its
// base code ("held"), so the unit is the one that was in the previous read and is missing now with that base code
// (V live in D2DLOOP: after a click on a pack cell held="tsc" and the tsc was gone from the list; without this the
// hand's Take never saw "cursor holds item N" and every equip failed). The unit stays the cursor item while the same
// code is held and the item is still absent from the lists.
func (b *builder) cursor() {
	s, inv := b.s, b.snap.Inventory
	s.cmu.Lock()
	defer s.cmu.Unlock()
	carried := append([]d2.Unit(nil), inv.Items...)
	defer func() { s.prev = carried }()
	if b.held == "" || inv.Cursor != nil {
		s.cur = nil
		return
	}
	row := b.row(b.held)
	here := map[uint32]bool{}
	for _, u := range inv.Items {
		here[u.ID] = true
	}
	if s.cur != nil && s.cur.Class == row && !here[s.cur.ID] {
		b.putCursor(*s.cur)
		return
	}
	for _, u := range s.prev {
		if u.Class == row && !here[u.ID] && u.Item != nil && u.Item.Where != d2.ContCursor {
			s.cur = &u
			b.putCursor(u)
			return
		}
	}
	s.cur = nil
	b.rp.unknown("inventory.cursor", "held item "+b.held+" was not seen in the previous read")
}

func (b *builder) putCursor(u d2.Unit) {
	it := *u.Item
	it.Where, it.Page, it.NodePage = d2.ContCursor, 0, 0
	u.Item, u.Mode = &it, d2.ItemModeCursor
	inv := b.snap.Inventory
	inv.Cursor = &u
	inv.Items = append(inv.Items, u)
	inv.ItemCount++
	b.rp.derived("inventory.cursor", "unit missing since the previous read with the held base code")
}
