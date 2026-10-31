package town

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/town/panel"
)

// Item services of the quest NPCs: Charsi's Imbue (Act 1, Tools of the Trade) and Larzuk's sockets (Act 5, Siege on
// Harrogath). ALL of it is U live: no run has opened either window. The flow is modelled as: click the dialog line,
// the inventory shows and the game waits for a left click on one pack item (the item picker), optionally a confirm
// button, and the item changes in memory. Every row index, line count and screen coordinate is in ServiceFlow below
// and nowhere else, so a live run corrects one table (checklist in docs/town.md "Imbue and sockets").

// Line is one click on the NPC dialog: line Idx (0 = top) of a box with N lines.
type Line struct{ Idx, N int }

// ServiceFlow describes one item service.
type ServiceFlow struct {
	Name string
	Act  int
	NPC  int    // monstats class (V: 154 Charsi, 511 Larzuk, docs/act5.md)
	Path []Line // dialog clicks that end in the item picker
	// Confirm is a button to click after the item (screen px at 800x600); zero = the item click is the whole choice.
	Confirm panel.Pt
}

// Flows. U: the Imbue line is assumed between Trade and Cancel (quest.Menus Charsi, itself U); Larzuk's reward is
// assumed to come with the plain Talk line (the way Akara's Den reward does, V live) and to open the picker at once.
var (
	ImbueFlow  = ServiceFlow{Name: "imbue", Act: 1, NPC: 154, Path: []Line{{Idx: 2, N: 4}}}
	SocketFlow = ServiceFlow{Name: "socket", Act: 5, NPC: 511, Path: []Line{{Idx: 0, N: 3}}}
)

var (
	ErrNoTarget       = errors.New("town: no suitable item to imbue or socket in the pack")
	ErrServiceUnshown = errors.New("town: the item picker never appeared")
	ErrNotApplied     = errors.New("town: the item did not change after the pick")
)

// ItemImbuer is Charsi's reward for the bot. Implemented by *Town.
type ItemImbuer interface {
	Imbue(ctx context.Context, w ImbueWant) (ServiceReport, error)
}

// ItemSocketer is Larzuk's reward for the bot. Implemented by *Town.
type ItemSocketer interface {
	Socket(ctx context.Context, w SocketWant) (ServiceReport, error)
}

var (
	_ ItemImbuer   = (*Town)(nil)
	_ ItemSocketer = (*Town)(nil)
)

// ImbueWant: Item 0 lets the gear score choose.
type ImbueWant struct{ Item uint32 }

// SocketWant: Item 0 lets the runeword base rule choose; Runewords limits the target runewords (empty: every runeword
// gear.RunewordScore knows).
type SocketWant struct {
	Item      uint32
	Runewords []string
}

// ServiceReport is what a service run did.
type ServiceReport struct {
	Item    d2.Unit
	Name    string
	Why     string // the reason the item was chosen
	WasWorn bool   // it was taken off first; the leveler's gear step wears the result
	Steps   []string
}

// Pick is a chosen item and why.
type Pick struct {
	Unit  d2.Unit
	Name  string
	Score float64 // imbue: gain over the worn item of its slot; socket: runeword score, else base score
	Why   string
	Worn  bool
}

// StatsFn reads the stat list of an item (nil result is fine).
type StatsFn func(d2.Unit) d2.Stats

// plainBase is true for a white or superior item without sockets: the only kind both services work on usefully
// (S: imbue needs a normal or superior item; runewords need white sockets).
func plainBase(u d2.Unit, st d2.Stats) bool {
	it := u.Item
	if it == nil || it.Ethereal() || it.Socketed() || st.Get(statSockets) > 0 {
		return false
	}
	return it.Quality == d2.QualityNormal || it.Quality == d2.QualitySuperior
}

// statSockets is the socket count stat (V docs/items.md: the loot rule reads stat 194).
const statSockets = 194

// candidates scores every plain pack or worn item of the main weapon set. Worn items (not the alternate set) are
// included: the service picks from the pack, so a worn winner is taken off first.
func candidates(db *data.DB, p *d2.Player, inv *d2.Inventory, stats StatsFn) (cands []leveler.Cand, worn map[int]float64) {
	worn = map[int]float64{}
	for _, u := range inv.Items {
		if u.Item == nil {
			continue
		}
		in := u.Item.Where == d2.ContInventory
		eq := u.Item.Where == d2.ContEquipment && u.Item.Body < d2.BodyAltR
		if !in && !eq {
			continue
		}
		st := stats(u)
		c := leveler.Candidate(db, p, u, st)
		if eq {
			worn[u.Item.Body] = c.Score
		}
		if c.Slots == nil || !c.Usable || c.Ranged || !plainBase(u, st) {
			continue
		}
		cands = append(cands, c)
	}
	return cands, worn
}

// PickImbue chooses the item Charsi should imbue: the plain base whose imbued (rare, unidentified: the leveler's flat
// rare expectation) score gains most over what is worn in its slot. Pure over the snapshot; never chooses a base that
// a runeword wants (a socketed-to-be white base is worth more than a random rare, docs/items.md 3.2) unless it is
// all there is. The imbue level of the result is not known (U), so only the base's own usability is checked.
func PickImbue(db *data.DB, p *d2.Player, inv *d2.Inventory, stats StatsFn) (Pick, bool) {
	cands, worn := candidates(db, p, inv, stats)
	var best Pick
	found, bestRW := false, false
	for _, c := range cands {
		if c.AvoidTwoHanded {
			continue
		}
		rare := c.Unit
		it := *rare.Item
		it.Quality, it.Flags = d2.QualityRare, it.Flags&^d2.FlagIdentified
		rare.Item = &it
		after := leveler.Candidate(db, p, rare, nil).Score
		gain := after - worn[c.Slots[0]]
		rw, _ := bestRuneword(db, c.Unit, nil)
		isRW := rw > 0
		// A base no runeword wants beats one a runeword wants; then the larger gain.
		if found && (isRW && !bestRW || isRW == bestRW && gain <= best.Score) {
			continue
		}
		best = Pick{Unit: c.Unit, Name: c.Name, Score: gain, Worn: c.Unit.Item.Where == d2.ContEquipment,
			Why: fmt.Sprintf("imbued rare scores %.0f over the worn %.0f in its slot", after, worn[c.Slots[0]])}
		found, bestRW = true, isRW
	}
	return best, found && best.Score > 0
}

// imbuedProof: the imbued item is magic or better (Charsi: rare, S) in the same pack cell; the id may change (U), so
// the cell is the key.
func imbuedProof(before d2.Unit, s *d2.Snapshot) bool {
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.X == before.X && u.Y == before.Y && u.Class == before.Class && u.Item.Quality >= d2.QualityMagic {
			return true
		}
	}
	return false
}

// Imbue talks to Charsi, picks the Imbue line and the item. Proof: the item in that cell is no longer normal.
func (t *Town) Imbue(ctx context.Context, w ImbueWant) (ServiceReport, error) {
	return t.itemService(ctx, ImbueFlow, func(s *d2.Snapshot) (Pick, error) {
		if w.Item != 0 {
			u, ok := findIn(s.Inventory, w.Item)
			if !ok {
				return Pick{}, fmt.Errorf("%w: item %d", ErrNoTarget, w.Item)
			}
			return Pick{Unit: u, Name: t.nameOf(u), Worn: u.Item.Where == d2.ContEquipment, Why: "asked for"}, nil
		}
		pk, ok := PickImbue(t.e.Data, s.Player, s.Inventory, t.statsOf)
		if !ok {
			return Pick{}, ErrNoTarget
		}
		return pk, nil
	}, imbuedProof)
}

func (t *Town) statsOf(u d2.Unit) d2.Stats {
	st, _ := t.unitStats(u.Addr)
	return st
}

// itemService is the shared flow: choose before talking (so a missing target never burns the one-time reward), take a
// worn target off, open the dialog, click the path, wait for the picker, click the item, confirm, wait for the proof.
func (t *Town) itemService(ctx context.Context, f ServiceFlow, choose func(*d2.Snapshot) (Pick, error),
	proof func(before d2.Unit, s *d2.Snapshot) bool) (ServiceReport, error) {
	var rep ServiceReport
	s, err := t.current()
	if err != nil {
		return rep, err
	}
	if a := ActOfLevel(s.Player.Area); a == nil || a.No != f.Act {
		return rep, fmt.Errorf("%w: %s needs act %d", ErrNotInTown, f.Name, f.Act)
	}
	pk, err := choose(s)
	if err != nil {
		return rep, err
	}
	rep.Name, rep.Why, rep.WasWorn = pk.Name, pk.Why, pk.Worn
	rep.Steps = append(rep.Steps, fmt.Sprintf("target %s: %s", pk.Name, pk.Why))
	if pk.Worn {
		if err := t.h.Unequip(ctx, pk.Unit.ID); err != nil {
			return rep, fmt.Errorf("take %s off: %w", pk.Name, err)
		}
		if s, err = t.current(); err != nil {
			return rep, err
		}
		u, ok := findIn(s.Inventory, pk.Unit.ID)
		if !ok || u.Item.Where != d2.ContInventory {
			return rep, fmt.Errorf("%w: %s is not in the pack after taking it off", ErrNoTarget, pk.Name)
		}
		pk.Unit = u
		rep.Steps = append(rep.Steps, "took it off")
	}
	// The dialog box is clicked with the NPC in the unit table; the picker needs the inventory, not a walk.
	npc, err := t.openNPC(ctx, []int{f.NPC}, t.npcPreset(f.NPC))
	if err != nil {
		return rep, err
	}
	rep.Steps = append(rep.Steps, "dialog open")
	for _, l := range f.Path {
		if err := t.PickLine(ctx, npc, l.Idx, l.N); err != nil {
			_ = t.h.CloseAll(ctx)
			return rep, err
		}
	}
	s, err = t.h.OpenInventory(ctx) // shows by itself or not (U): the hotkey is idempotent here
	if err != nil {
		_ = t.h.CloseAll(ctx)
		return rep, fmt.Errorf("%w: %v", ErrServiceUnshown, err)
	}
	lay, err := t.layout(s)
	if err != nil {
		return rep, err
	}
	cur, ok := findIn(s.Inventory, pk.Unit.ID)
	if !ok {
		_ = t.h.CloseAll(ctx)
		return rep, fmt.Errorf("%w: %s vanished", ErrNoTarget, pk.Name)
	}
	w, hh := t.h.Size(cur)
	c := lay.Inventory.ItemCenter(cur.X, cur.Y, w, hh)
	if err := t.e.In.Click(input.Left, c.X, c.Y); err != nil {
		return rep, err
	}
	t.sleep(t.tune().ClickSettleMs)
	if f.Confirm != (panel.Pt{}) {
		if err := t.e.In.Click(input.Left, f.Confirm.X, f.Confirm.Y); err != nil {
			return rep, err
		}
	}
	_, done, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return proof(cur, s) })
	if err != nil {
		return rep, err
	}
	if !done {
		_ = t.h.CloseAll(ctx)
		return rep, fmt.Errorf("%w: %s", ErrNotApplied, pk.Name)
	}
	rep.Item = cur
	rep.Steps = append(rep.Steps, "applied")
	return rep, t.h.CloseAll(ctx)
}
