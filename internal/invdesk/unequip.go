package invdesk

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

// Take a worn item off (API schema 7, docs/gearsets.md): the drag from a doll slot to the backpack, and a step of a gear
// set apply. The request names the slot; its ID is the item worn there when it was queued. Like every player command it
// runs at a safe moment (docs/inventory-actions.md priority ladder) and the bot never queues one on its own.
const KindUnequip = "unequip"

// PackCols and PackRows are the backpack grid (status.InventoryData: 10 x 4).
const (
	PackCols = 10
	PackRows = 4
)

// Unequipper performs an unequip: take the worn item off and put it in the backpack (leveler.Leveler).
type Unequipper interface {
	PlayerUnequip(ctx context.Context, id uint32) error
}

// HandsInfo says whether an item (in any container of the snapshot) takes both hands (leveler.Leveler).
type HandsInfo interface {
	IsTwoHanded(s *d2.Snapshot, id uint32) bool
}

// SetUnequipper installs the executor of unequip requests; nil leaves them refused with code "off".
func (d *Desk) SetUnequipper(u Unequipper) { d.mu.Lock(); d.un = u; d.mu.Unlock() }

// SetHands installs the two-hand rule used to order a gear set apply.
func (d *Desk) SetHands(h HandsInfo) { d.mu.Lock(); d.hands = h; d.mu.Unlock() }

// packGrid is the occupancy of the backpack as the view shows it.
type packGrid [PackRows][PackCols]bool

func newPackGrid(items []status.Item) *packGrid {
	var g packGrid
	for _, it := range items {
		g.set(it.X, it.Y, it.W, it.H, true)
	}
	return &g
}

func (g *packGrid) set(x, y, w, h int, v bool) {
	for j := y; j < y+max(h, 1); j++ {
		for i := x; i < x+max(w, 1); i++ {
			if i >= 0 && j >= 0 && i < PackCols && j < PackRows {
				g[j][i] = v
			}
		}
	}
}

// place puts a w x h item at the first free spot (row by row, as the click path does) and says whether there was one.
func (g *packGrid) place(w, h int) bool {
	w, h = max(w, 1), max(h, 1)
	for y := 0; y+h <= PackRows; y++ {
	next:
		for x := 0; x+w <= PackCols; x++ {
			for j := y; j < y+h; j++ {
				for i := x; i < x+w; i++ {
					if g[j][i] {
						continue next
					}
				}
			}
			g.set(x, y, w, h, true)
			return true
		}
	}
	return false
}

// unequipWhy says why the worn item cannot be taken off now: the pack has no free spot of its size.
func unequipWhy(data status.InventoryData, it status.Item) string {
	if !newPackGrid(data.Inventory).place(it.W, it.H) {
		return "the backpack has no free spot for it: make room first"
	}
	return ""
}

// RequestUnequip queues taking the item worn in slot off. Refused when the slot is empty or unknown, when the pack has
// no room, or when the same slot already has an unequip waiting. The hero need not be in town.
func (d *Desk) RequestUnequip(slot string) (Request, error) {
	d.mu.Lock()
	un := d.un
	d.mu.Unlock()
	if un == nil {
		return Request{}, refuse("off", "taking items off is not available on this bot")
	}
	if _, ok := SlotBody[slot]; !ok {
		return Request{}, refuse("slot", "unknown slot %q", slot)
	}
	if d.c.Inventory == nil {
		return Request{}, refuse("off", "inventory is not available")
	}
	data := d.c.Inventory()
	it := data.Equipment[slot]
	if it == nil {
		return Request{}, refuse("item", "nothing is worn in that slot")
	}
	if why := unequipWhy(data, *it); why != "" {
		return Request{}, refuse("room", "%s", why)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, x := range d.st.Queue {
		if x.kind() == KindUnequip && x.Slot == slot {
			return Request{}, refuse("queued", "that slot already has an unequip waiting")
		}
	}
	if d.queuedLocked(it.ID, KindTrash) {
		return Request{}, refuse("trash", "that item is queued to be trashed: cancel that first")
	}
	return d.enqueueLocked(Request{ID: it.ID, Kind: KindUnequip, Slot: slot, Name: itemName(*it), Print: fmt.Sprintf("%s/%s/%d", it.Code, it.Quality, it.Level)})
}

func itemName(it status.Item) string {
	if it.Name != "" {
		return it.Name
	}
	return it.Code
}

// dropPinLocked removes the pin of a slot (the player took the item off: the pin has nothing to protect).
func (d *Desk) dropPinLocked(slot string) {
	kept := d.st.Pins[:0]
	for _, p := range d.st.Pins {
		if p.Slot != slot {
			kept = append(kept, p)
		}
	}
	d.st.Pins = kept
}
