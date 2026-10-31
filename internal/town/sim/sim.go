// Package sim is a small simulated Diablo II town for tests of internal/town and internal/loot: a World that
// serves d2.Snapshots and reacts to the clicks and keys the code under test sends, using the same panel
// geometry. It is test support (no game involved): what it proves is that the code's decisions, click targets
// and memory checks agree with each other, not that the real game behaves this way.
package sim

import (
	"context"
	"fmt"
	"math"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/panel"
)

// NPC menu geometry the sim answers to (relative to the NPC feet); a town test checks it equals town.NPCMenu.
type MenuGeom struct{ DX, Cancel, Step2 int }

// World is the simulated game state.
type World struct {
	DB  *data.DB
	Lay *panel.Layout

	Player  d2.Player
	Items   []d2.Unit // player inventory lists: backpack, belt, stash, equipment
	Cursor  *d2.Unit
	Ground  []d2.Unit
	Mons    []d2.Unit
	Objs    []d2.Unit
	UI      d2.UIState
	Vendors map[uint32][]d2.Unit // NPC unit id -> stock
	Stats   map[uint32]d2.Stats  // by unit Addr
	// Refuse lists item base codes a vendor id refuses to buy.
	Refuse map[uint32]map[string]bool
	// EquipRefuse lists item ids the game refuses to wear (level requirement): a slot click leaves them on the cursor.
	EquipRefuse map[uint32]bool
	// SlotIgnore is the number of equipment-slot left clicks the game ignores (the item stays on the cursor), the
	// live "slot click refused" pattern of soak 2026-10-30; RightClickEquip lets a right click on a pack item wear it
	// into an empty matching slot (U live).
	SlotIgnore      int
	RightClickEquip bool
	// Heal lists NPC classes that heal when opened.
	Heal map[int]bool
	// HealSkip is the number of NPC clicks the healer ignores before it heals (tests the retry).
	HealSkip int
	// PickupToCursor makes ground clicks put the item on the cursor instead of into the pack.
	PickupToCursor bool
	// AutoEquipQuest: a ground click on a Shaft (msf) or Top of the Staff (vip) wears it (V live, docs/act2.md P1).
	AutoEquipQuest bool
	// ShiftFails and DragFails make that many shift-clicks on a pack potion / drops of a potion on the belt slot do nothing.
	ShiftFails, DragFails int
	// NoHover makes Interact fail with ErrNotHovered this many times first.
	NoHover int
	// Speech makes the next NPC clicks play a first-contact greeting: the menu stays closed (no error) that many clicks.
	Speech int
	// StrictPos makes an NPC click miss (ErrNotHovered) unless its position is within 2 subtiles of the NPC's real one
	// (the caller used a stale unit position). WanderOnClose moves every NPC by this much when the dialog is closed (Esc).
	// NoHoverAfterSpeech: after a greeting click, this many further NPC clicks fail with ErrNotHovered (hover is dead
	// while the speech plays, V live i5).
	NoHoverAfterSpeech int
	// SpeechSkipByClick: while a greeting plays a plain world click on the NPC's tile (within 2 subtiles)
	// ends it and opens the menu, as the user says a left click does. A world click anywhere else walks the character
	// there (WalkedAway counts those).
	SpeechSkipByClick bool
	WalkedAway        int
	speaking          bool
	// GuardMoveAfter: after this many snapshots the NPC GuardClass steps 5 subtiles (the Harem gate guard).
	GuardClass, GuardMoveAfter int
	snaps                      int
	deadHover                  int
	StrictPos                  bool
	WanderOnClose              [2]int
	// Menu geometry and the trade entry index per NPC class (default order Talk, Trade, Cancel).
	Menu        MenuGeom
	TradeIndex  map[int]int
	RepairBtn   panel.Pt
	GoldBtn     panel.Pt
	PortalLevel int // area the portal object leads to
	// Services: NPC class -> dialog line (0 = top) -> "imbue" or "socket". The class then has len(Lines[class]) dialog
	// lines (default 3); a click on a service line closes the menu, shows the inventory and the next left click on a
	// plain pack item applies the service (the item-picker model of town/imbue.go, U live).
	Services map[int]map[int]string
	Lines    map[int]int
	pick     string
	// MenuInert is the number of dialog-line left clicks the game ignores while the menu flag stays set (V live i5/G2
	// 2026-10-30: 18 visits where every click from -30 to +15 px did nothing); 1<<30 is "forever".
	MenuInert int

	talk       uint32
	idMode     bool
	goldDialog bool
	goldBuf    string // digits in the deposit dialog; "" with goldTyped=false means the full amount
	goldTyped  bool
	nextID     uint32
	Clicks     []string
}

// NewWorld returns a world with a level-1 Sorceress standing in the Rogue Encampment at (6020,4930).
func NewWorld(db *data.DB) *World {
	w := &World{DB: db, Vendors: map[uint32][]d2.Unit{}, Stats: map[uint32]d2.Stats{}, Refuse: map[uint32]map[string]bool{}, EquipRefuse: map[uint32]bool{},
		Heal: map[int]bool{148: true}, TradeIndex: map[int]int{}, nextID: 100, PortalLevel: 2}
	w.Player = d2.Player{ID: 1, Class: d2.Sorceress, Name: "Testy", X: 6020, Y: 4930, Area: 1, Level: 1,
		Life: 40, MaxLife: 40, Mana: 35, MaxMana: 35}
	lay, err := panel.NewLayout(db, d2.Sorceress, panel.Pt{X: 800, Y: 600}, true)
	if err != nil {
		panic(err)
	}
	w.Lay = lay
	w.Menu = MenuGeom{0, -115, 31}
	w.RepairBtn = panel.Pt{X: 150, Y: 450}
	w.GoldBtn = panel.Pt{X: 487, Y: 453}
	return w
}

func (w *World) classOf(code string) int {
	for i, b := range w.DB.Items {
		if b.Code == code {
			return i
		}
	}
	panic("sim: unknown item code " + code)
}

func (w *World) id() uint32 { w.nextID++; return w.nextID }

// NewItem makes an item unit (not yet placed anywhere).
func (w *World) NewItem(code string, quality int) d2.Unit {
	id := w.id()
	return d2.Unit{Addr: 0x1000000 + id, Type: d2.TypeItem, Class: w.classOf(code), ID: id,
		Item: &d2.Item{Quality: quality, Flags: d2.FlagIdentified, Where: d2.ContInventory}}
}

// Add places an item in a container at a cell.
func (w *World) Add(code string, where d2.Container, x, y int) d2.Unit {
	u := w.NewItem(code, 2)
	u.Item.Where, u.X, u.Y = where, x, y
	if where == d2.ContBelt {
		u.Mode = d2.ItemModeBelt
	}
	if where == d2.ContEquipment {
		u.Mode = d2.ItemModeEquipped
	}
	w.Items = append(w.Items, u)
	return u
}

// Put places an already-built unit.
func (w *World) Put(u d2.Unit, where d2.Container, x, y int) d2.Unit {
	u.Item.Where, u.X, u.Y = where, x, y
	w.Items = append(w.Items, u)
	return u
}

// AddGround drops an item on the ground at world position (x,y).
func (w *World) AddGround(code string, quality, x, y int) d2.Unit {
	u := w.NewItem(code, quality)
	u.Mode, u.X, u.Y = d2.ItemModeGround, x, y
	u.Item.Where = d2.ContGround
	w.Ground = append(w.Ground, u)
	return u
}

// AddNPC adds an NPC of the class at a position and returns it.
func (w *World) AddNPC(class, x, y int) d2.Unit {
	id := w.id()
	u := d2.Unit{Addr: 0x2000000 + id, Type: d2.TypeMonster, Class: class, ID: id, Mode: d2.MonModeStand, X: x, Y: y,
		Monster: &d2.MonsterInfo{LifePct: 100}}
	w.Mons = append(w.Mons, u)
	return u
}

// AddObject adds an object.
func (w *World) AddObject(class, x, y int) d2.Unit {
	id := w.id()
	u := d2.Unit{Addr: 0x3000000 + id, Type: d2.TypeObject, Class: class, ID: id, X: x, Y: y}
	w.Objs = append(w.Objs, u)
	return u
}

// Stock sets the shop inventory of an NPC from item codes, laid out left to right.
func (w *World) Stock(npc d2.Unit, codes ...string) {
	var out []d2.Unit
	for i, c := range codes {
		u := w.NewItem(c, 2)
		u.Item.Where, u.X, u.Y = d2.ContNone, i%10, i/10
		out = append(out, u)
	}
	w.Vendors[npc.ID] = out
}

// Snapshot implements the snapshot source.
func (w *World) Snapshot() (*d2.Snapshot, error) {
	if w.GuardMoveAfter > 0 {
		if w.snaps++; w.snaps == w.GuardMoveAfter {
			for i := range w.Mons {
				if w.Mons[i].Class == w.GuardClass {
					w.Mons[i].X += 5
				}
			}
		}
	}
	p := w.Player
	s := &d2.Snapshot{State: d2.StateInGame, Player: &p, UI: w.UI, Screen: d2.Point{X: 800, Y: 600},
		Monsters: append([]d2.Unit(nil), w.Mons...), Objects: append([]d2.Unit(nil), w.Objs...), Ground: append([]d2.Unit(nil), w.Ground...)}
	inv := &d2.Inventory{Items: append([]d2.Unit(nil), w.Items...), ItemCount: len(w.Items)}
	if w.Cursor != nil {
		c := *w.Cursor
		c.Item = &d2.Item{}
		*c.Item = *w.Cursor.Item
		c.Item.Where = d2.ContCursor
		inv.Cursor = &c
	}
	s.Inventory = inv
	return s, nil
}

// AllItems lists every item unit incl. vendor stock and ground (the unit table).
func (w *World) AllItems() ([]d2.Unit, error) {
	out := append([]d2.Unit(nil), w.Items...)
	out = append(out, w.Ground...)
	for _, st := range w.Vendors {
		out = append(out, st...)
	}
	return out, nil
}

// Owner is the container owner lookup: the NPC for stock items, 0 otherwise.
func (w *World) Owner(it d2.Unit) (uint32, error) {
	for id, st := range w.Vendors {
		for _, u := range st {
			if u.ID == it.ID {
				for _, m := range w.Mons {
					if m.ID == id {
						return m.Addr, nil
					}
				}
			}
		}
	}
	return w.Player.Addr, nil
}

// UnitStats serves the stats map.
func (w *World) UnitStats(addr uint32) (d2.Stats, error) {
	if st, ok := w.Stats[addr]; ok {
		return st, nil
	}
	return nil, fmt.Errorf("sim: no stats for 0x%x", addr)
}

func (w *World) base(u d2.Unit) *data.ItemBase { return w.DB.Items[u.Class] }

func (w *World) find(id uint32) int {
	for i, u := range w.Items {
		if u.ID == id {
			return i
		}
	}
	return -1
}

func (w *World) size(u d2.Unit) (int, int) {
	b := w.base(u)
	return max(1, b.Width), max(1, b.Height)
}

// topLeft converts a click on the footprint centre into the top-left cell, like the game does for a held item.
func topLeft(g panel.Grid, p panel.Pt, ww, hh int) (int, int) {
	c := int(math.Round(float64(p.X-g.Left-ww*g.CellW/2) / float64(g.CellW)))
	r := int(math.Round(float64(p.Y-g.Top-hh*g.CellH/2) / float64(g.CellH)))
	return c, r
}

func (w *World) occupied(where d2.Container, g panel.Grid) []bool {
	occ := make([]bool, g.Cols*g.Rows)
	for _, u := range w.Items {
		if u.Item.Where != where {
			continue
		}
		ww, hh := w.size(u)
		for y := u.Y; y < u.Y+hh; y++ {
			for x := u.X; x < u.X+ww; x++ {
				if x >= 0 && y >= 0 && x < g.Cols && y < g.Rows {
					occ[y*g.Cols+x] = true
				}
			}
		}
	}
	return occ
}

func fits(occ []bool, g panel.Grid, c, r, ww, hh int) bool {
	if c < 0 || r < 0 || c+ww > g.Cols || r+hh > g.Rows {
		return false
	}
	for y := r; y < r+hh; y++ {
		for x := c; x < c+ww; x++ {
			if occ[y*g.Cols+x] {
				return false
			}
		}
	}
	return true
}

// autoPlace puts a bought or picked item into the belt (potions, if there is room) or the first free pack cell.
func (w *World) autoPlace(u d2.Unit) bool {
	b := w.base(u)
	if b.AutoBelt && w.UI.Open(d2.UIInventory) || b.AutoBelt {
		slots := 4
		if belt := w.beltItems(); len(belt) < slots {
			for s := 0; s < slots; s++ {
				used := false
				for _, bi := range belt {
					if bi.X == s {
						used = true
					}
				}
				if !used {
					u.Item.Where, u.Mode, u.X, u.Y = d2.ContBelt, d2.ItemModeBelt, s, 0
					w.Items = append(w.Items, u)
					return true
				}
			}
		}
	}
	occ := w.occupied(d2.ContInventory, w.Lay.Inventory)
	ww, hh := w.size(u)
	for r := 0; r+hh <= w.Lay.Inventory.Rows; r++ {
		for c := 0; c+ww <= w.Lay.Inventory.Cols; c++ {
			if fits(occ, w.Lay.Inventory, c, r, ww, hh) {
				u.Item.Where, u.Mode, u.X, u.Y = d2.ContInventory, d2.ItemModeStored, c, r
				w.Items = append(w.Items, u)
				return true
			}
		}
	}
	return false
}

func (w *World) beltItems() []d2.Unit {
	var o []d2.Unit
	for _, u := range w.Items {
		if u.Item.Where == d2.ContBelt {
			o = append(o, u)
		}
	}
	return o
}

func (w *World) npcAt(id uint32) (d2.Unit, bool) {
	for _, m := range w.Mons {
		if m.ID == id {
			return m, true
		}
	}
	return d2.Unit{}, false
}

// Mover is a fake navigator: it teleports the player next to the target.
type Mover struct {
	W     *World
	Calls []string
	Err   error
	Short int // the next Short GoToUnit calls stop 8 subtiles short of the unit
}

func (m *Mover) GoTo(_ context.Context, p input.Point) error {
	m.Calls = append(m.Calls, fmt.Sprintf("goto %d,%d", p.X, p.Y))
	if m.Err != nil {
		return m.Err
	}
	m.W.Player.X, m.W.Player.Y = p.X-2, p.Y
	return nil
}

func (m *Mover) GoToUnit(_ context.Context, id uint32) error {
	m.Calls = append(m.Calls, fmt.Sprintf("gotounit %d", id))
	if m.Err != nil {
		return m.Err
	}
	for _, list := range [][]d2.Unit{m.W.Mons, m.W.Objs} {
		for _, u := range list {
			if u.ID == id {
				m.W.Player.X, m.W.Player.Y = u.X-2, u.Y
				if m.Short > 0 { // the walk ends early (blocked path)
					m.Short--
					m.W.Player.X = u.X - 8
				}
				return nil
			}
		}
	}
	return fmt.Errorf("sim: unit %d not found", id)
}

func (m *Mover) ExitTo(_ context.Context, level int) error {
	m.Calls = append(m.Calls, fmt.Sprintf("exitto %d", level))
	m.W.Player.Area = level
	return nil
}

func (m *Mover) TakeWaypoint(_ context.Context, level int) error {
	m.Calls = append(m.Calls, fmt.Sprintf("waypoint %d", level))
	m.W.Player.Area = level
	return nil
}
