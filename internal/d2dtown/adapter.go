// Package d2dtown runs internal/town on the d2d backend. It supplies the pieces the town code reads through a
// town.Source (unit table with vendor stock, item stats, dialog line points) and an input wrapper that hovers
// before it clicks. Nothing here edits the town package's behaviour for Game.exe; the seams are the optional
// fields of town.Source. Findings and breaks: docs/d2d-town.md.
package d2dtown

import (
	"context"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
)

// Stat ids the town code reads from a unit's stat list (same numbers as town/shop.go).
const (
	statQuantity      = 70
	statDurability    = 72
	statMaxDurability = 73
)

// Address spaces. d2d has no memory addresses, the town code compares them, so we mint them: NPC units use their
// unit id, carried items their item id, vendor stock a number of its own (stock items have no id in d2d, V).
const (
	vendorBase = 0x50000000
	playerAddr = 0x7fffffff // owner of every carried item
)

// Adapter is the bridge for one d2d.
type Adapter struct {
	C   *d2dctl.Client
	Src *d2dsrc.Source
	Tab d2dsrc.Tables // item rows and monster classes; without it item classes are -1 and the shop finds nothing

	// Timeout bounds the channel calls the seams make (default 8 s).
	Timeout time.Duration

	mu   sync.Mutex
	last *d2dsrc.Result
}

// New returns an Adapter.
func New(c *d2dctl.Client, src *d2dsrc.Source, tables d2dsrc.Tables) *Adapter {
	return &Adapter{C: c, Src: src, Tab: tables, Timeout: 8 * time.Second}
}

func (a *Adapter) ctx() (context.Context, context.CancelFunc) {
	d := a.Timeout
	if d == 0 {
		d = 8 * time.Second
	}
	return context.WithTimeout(context.Background(), d)
}

// Source is what town.Town.UseSource takes.
func (a *Adapter) Source() town.Source {
	return town.Source{
		Snapshot:  a.snapshot,
		Items:     a.items,
		Owner:     a.owner,
		Stats:     a.stats,
		MenuEntry: a.menuEntry,
		MenuIndex: a.menuIndex,
		FixedTabs: true,
	}
}

func (a *Adapter) read() (*d2dsrc.Result, error) {
	ctx, cancel := a.ctx()
	defer cancel()
	r, err := a.Src.Read(ctx)
	if err != nil {
		return nil, err
	}
	for i := range r.Snap.Monsters { // the town compares NPC addresses (owner of a vendor item)
		r.Snap.Monsters[i].Addr = r.Snap.Monsters[i].ID
	}
	if inv := r.Snap.Inventory; inv != nil {
		for i := range inv.Items {
			inv.Items[i].Addr = inv.Items[i].ID
		}
	}
	a.classObjects(r)
	a.mu.Lock()
	a.last = r
	a.mu.Unlock()
	return r, nil
}

// classObjects gives the named "npcs" objects the objects.txt class the town code searches for: d2dsrc leaves
// Class -1 (npcs only has a name), which is why the town reported "npc not found" for the stash chest (V:
// docs/d2d-town.md "Gaps closed").
func (a *Adapter) classObjects(r *d2dsrc.Result) {
	for i := range r.Snap.Objects {
		o := &r.Snap.Objects[i]
		if o.Class != -1 {
			continue
		}
		switch r.Extras.Names[o.ID] {
		case "Your Private Stash":
			o.Class = town.ObjStash
		case "Waypoint":
			o.Class = waypointClass(r.Snap.Player.Area)
		}
	}
}

// waypointClass is the town waypoint's objects.txt class by act of the town (first area of each act).
func waypointClass(area int) int {
	switch {
	case area >= 109:
		return town.ObjWPAct5
	case area >= 103:
		return town.ObjWPAct4
	case area >= 75:
		return town.ObjWPAct3
	case area >= 40:
		return town.ObjWPAct2
	}
	return town.ObjWPAct1
}

func (a *Adapter) snapshot() (*d2.Snapshot, error) {
	r, err := a.read()
	if err != nil {
		return nil, err
	}
	return r.Snap, nil
}

// items is every item unit: the carried ones plus, while a store window is open, its stock (Mode stored, Page = tab,
// X/Y = grid cell).
func (a *Adapter) items() ([]d2.Unit, error) {
	r, err := a.read()
	if err != nil {
		return nil, err
	}
	var out []d2.Unit
	if r.Snap.Inventory != nil {
		out = append(out, r.Snap.Inventory.Items...)
	}
	if !r.Snap.UI.Open(d2.UIShop) {
		return out, nil
	}
	ctx, cancel := a.ctx()
	defer cancel()
	v, err := a.C.Vendor(ctx)
	if err != nil {
		if err == d2dctl.ErrNoVendorDump {
			return out, nil // stock d2d: no stock readable, the shop code logs "no vendor stock"
		}
		return nil, err
	}
	for _, it := range v.Items {
		out = append(out, a.vendorUnit(it))
	}
	return out, nil
}

func (a *Adapter) vendorUnit(it d2dctl.VendorItem) d2.Unit {
	class := -1
	if a.Tab != nil {
		if n, ok := a.Tab.ItemRow(it.Code); ok {
			class = n
		}
	}
	addr := uint32(vendorBase | it.Tab<<16 | it.Y<<8 | it.X)
	return d2.Unit{
		Addr: addr, ID: addr, Type: d2.TypeItem, Class: class, Mode: d2.ItemModeStored,
		X: it.X, Y: it.Y,
		Item: &d2.Item{Quality: it.Quality, ILvl: it.ILvl, Page: it.Tab, GX: it.X, GY: it.Y, NodePage: 1},
	}
}

// owner: vendor stock belongs to the NPC whose window is open, everything else to the player.
func (a *Adapter) owner(it d2.Unit) (uint32, error) {
	if it.Addr < vendorBase {
		return playerAddr, nil
	}
	ctx, cancel := a.ctx()
	defer cancel()
	v, err := a.C.Vendor(ctx)
	if err != nil {
		return 0, err
	}
	return uint32(d2dsrc.NPCIDBase + v.NPC), nil
}

// stats synthesises the stat entries the town reads: quantity, durability. Stock quantity comes from the vendor
// dump on the next read; here the carried item extras are enough (V that dump items has qty and dur).
func (a *Adapter) stats(addr uint32) (d2.Stats, error) {
	a.mu.Lock()
	r := a.last
	a.mu.Unlock()
	if r == nil {
		return nil, d2.ErrUnsupported
	}
	ex, ok := r.Extras.Items[addr]
	if !ok {
		return nil, d2.ErrUnsupported
	}
	st := d2.Stats{}
	if ex.Qty > 0 {
		st[statQuantity] = int32(ex.Qty)
	}
	if ex.MaxDur > 0 {
		st[statDurability] = int32(ex.Dur)
		st[statMaxDurability] = int32(ex.MaxDur)
	}
	return st, nil
}

var entryText = map[town.Entry]string{
	town.EntryTalk: "talk", town.EntryTrade: "trade", town.EntryGamble: "gamble",
	town.EntryCancel: "cancel", town.EntryResurrect: "resurrect",
}

func (a *Adapter) menu() []d2dctl.MenuLine {
	ctx, cancel := a.ctx()
	defer cancel()
	m, err := a.C.Menu(ctx)
	if err != nil {
		return nil
	}
	return m.Items
}

func (a *Adapter) menuEntry(e town.Entry) (input.Point, bool) {
	text, ok := entryText[e]
	if !ok {
		return input.Point{}, false
	}
	l, ok := d2dctl.FindLine(a.menu(), text)
	return input.Point{X: l.X, Y: l.Y}, ok
}

func (a *Adapter) menuIndex(idx int) (input.Point, bool) {
	m := a.menu()
	if idx < 0 || idx+1 >= len(m) {
		return input.Point{}, false
	}
	return input.Point{X: m[idx+1].X, Y: m[idx+1].Y}, true
}

// HoverFirst wraps an input.Controller so every unmodified left or right click is preceded by the two-step hover
// d2d needs (V 2026-10-30: a click without a prior hover on a dialog line or vendor cell is ignored).
func HoverFirst(in input.Controller) input.Controller { return hoverFirst{in} }

type hoverFirst struct{ input.Controller }

// HoverFirstBelt is HoverFirst plus the d2d belt move (town.beltPutter), for a town that is built on the controller
// directly. D2DLOOP: the wiring used plain HoverFirst, so BeltPut was never offered and the belt fill kept failing.
func HoverFirstBelt(in input.Controller, c *d2dctl.Client) input.Controller {
	return hoverBelt{hoverFirst{in}, c}
}

type hoverBelt struct {
	hoverFirst
	c *d2dctl.Client
}

func (h hoverBelt) BeltPut(id uint32, box int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return h.c.BeltPut(ctx, int(id), box)
}

// wait uses the controller's clock when it has one (lockstep: stepped ticks, not wall time).
func (h hoverFirst) wait(d time.Duration) {
	if w, ok := h.Controller.(interface{ Wait(time.Duration) }); ok {
		w.Wait(d)
		return
	}
	time.Sleep(d)
}

func (h hoverFirst) Click(b input.Button, x, y int, mods ...input.Mod) error {
	if len(mods) == 0 {
		if err := h.MoveMouse(x-3, y); err != nil {
			return err
		}
		h.wait(120 * time.Millisecond)
		if err := h.MoveMouse(x, y); err != nil {
			return err
		}
		h.wait(350 * time.Millisecond)
	}
	return h.Controller.Click(b, x, y, mods...)
}

// Input wraps a d2d input.Controller for the town layer: hover before click (HoverFirst), and the interactions
// d2dinput does one-shot become the walks that actually arrive: an NPC, the stash chest or the waypoint through
// ApproachNPC (side-steps around the camp's obstacles), a portal through EnterPortal (a click on its sprite, V).
func Input(in input.Controller, c *d2dctl.Client) input.Controller {
	return townInput{HoverFirst(in), c}
}

type townInput struct {
	input.Controller
	c *d2dctl.Client
}

// BeltPut is the d2d way to put a pack potion into the belt (town.beltPutter).
func (t townInput) BeltPut(id uint32, box int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return t.c.BeltPut(ctx, int(id), box)
}

func (t townInput) Interact(tg input.Target) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	switch id := tg.ID; {
	case id >= d2dsrc.PortalIDBase:
		return t.c.EnterPortal(ctx)
	case id >= d2dsrc.NPCIDBase && id != d2dsrc.MercID:
		return t.c.ApproachNPC(ctx, int(id-d2dsrc.NPCIDBase), 15*time.Second)
	}
	return t.Controller.Interact(tg)
}
