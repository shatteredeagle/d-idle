// Package town does the chores between fights: heal, restock the belt and scrolls, repair, identify,
// sell, stash, and the trips to and from town. Contract: docs/archive/wave2.md "Townsman". Details and the list of
// what is verified live: docs/town.md.
package town

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

// Mover is the slice of the navigator the town code uses.
type Mover interface {
	GoTo(ctx context.Context, p input.Point) error
	GoToUnit(ctx context.Context, unitID uint32) error
	ExitTo(ctx context.Context, levelNo int) error
	TakeWaypoint(ctx context.Context, levelNo int) error
}

// Judge rates items. *loot.Looter implements it.
type Judge interface {
	EvaluateUnit(u d2.Unit) loot.Verdict
	EvaluateWith(u d2.Unit, c loot.Ctx) loot.Verdict
}

var (
	ErrNotInTown = errors.New("town: not in a town")
	ErrNoNPC     = errors.New("town: npc not found or not reachable")
	ErrNoPortal  = errors.New("town: no town portal available")
	ErrNoMoney   = errors.New("town: not enough gold")
)

// Town is the Townsman implementation.
type Town struct {
	e     *env.Env
	mv    Mover
	judge Judge
	h     *hand.Hand

	lastVisit Visit // measurement of the last NPC open (skip.go)

	// Seam for tests (the snapshot seam is t.h.Snap).
	owner            func(it d2.Unit) (uint32, error)         // unit address of the container owner of an item
	itemsFn          func() ([]d2.Unit, error)                // all item units (default: Reader.Units)
	potionCap        int                                      // when > 0, buyPotions spends at most this much in total (poverty mode)
	arrived          time.Time                                // when ToTown last arrived
	pages            []int                                    // stock pages of the open vendor, sorted (tab layout)
	tab              int                                      // vendor tab currently shown (Page+1; 0 = unknown); reset when a shop opens
	sellByCursor     bool                                     // right-click selling did not work live: sell by dropping the item on the vendor grid
	menuAdj          map[int]int                              // pixel adjustment of NPC dialog clicks that worked, by NPC class
	statsFn          func(addr uint32) (d2.Stats, error)      // unit stat list (default: Reader.UnitStats)
	req              RequirementSource                        // effective item requirements (require.go; default: item tables + affix names)
	beltFullAt       *beltSignature                           // belt contents at the last failed move (beltfill.go)
	skipLogged       map[string]bool                          // equip skips already logged, by item/level/reason
	planFn           func(class d2.Class, clvl int) belt.Band // belt plan override (SetPlan)
	beltBuyGold      int                                      // gold at the last belt purchase attempt that bought nothing; 0 = none
	repairFail       repairLatch                              // BASICSFIX: last failed repair
	futile           map[string]futileMark                    // trip reason -> when and with how much gold a visit left it unfixed
	presetsFn        func() []d2map.Preset                    // test seam: preset units of the current level
	ammoPath         string                                   // "bow" or "javelin" for an Amazon (SetAmmoPath)
	reviveUnverified bool                                     // Revive may click the unverified resurrect dialog
	mercKnown        bool                                     // a merc was hired this run
	build            string                                   // leveler build name, picks the mana-heavy belt columns (SetBuild)
	stashFull        bool                                     // stash gold is at the cap (one log line per fill)
	stashCapSeen     int                                      // stash gold at the last refused deposit (0 = none seen)
	stashLeft        []d2.Unit                                // pack items the last stashItems could not fit
	stashRefused     map[uint32]bool                          // pack items a full stash refused and relieveStash could not place
	cubeBlocked      string                                   // state signature after a cube chore found no pack room
	shopClass        int                                      // class of the NPC whose trade window is open (0 = none): chores of one NPC share it
	noWalk           bool                                     // openNPC clicks a loaded NPC without walking next to it first
	menuEntry        func(Entry) (input.Point, bool)          // backend-reported dialog line point (replaces the pixel model of entryPoint)
	menuIndex        func(idx int) (input.Point, bool)        // same, by line index below the header
	fixedTabs        bool                                     // the vendor panel always draws four tabs; a page's tab slot is its page number
}

// New builds the Townsman. It creates its own loot.Looter as the Judge; use SetJudge to share one.
func New(e *env.Env, mv Mover) *Town {
	t := &Town{e: e, mv: mv, h: hand.New(e), sellByCursor: true}
	t.judge = loot.New(e, mv)
	t.owner = t.readOwner
	return t
}

// SetPlan replaces the belt plan (which columns hold which potion at which level). The default is
// belt.PlanForClass; a build that needs no mana potions passes a plan without mana columns.
func (t *Town) SetPlan(f func(class d2.Class, clvl int) belt.Band) { t.planFn = f }

// SetBuild tells the shop which build the character plays, so Nightmare and Hell shopping uses the mana-heavy belt
// columns of resource.PlanForBuild. Normal shopping ignores it.
func (t *Town) SetBuild(name string) {
	t.build = name
	if l, ok := t.judge.(*loot.Looter); ok {
		l.SetBuild(name)
	}
}

func (t *Town) plan(p *d2.Player) belt.Band {
	if t.planFn != nil {
		return t.planFn(p.Class, p.Level)
	}
	return belt.PlanForBuild(p.Class, p.Level, p.Difficulty, t.build)
}

// SetJudge replaces the item judge.
func (t *Town) SetJudge(j Judge) {
	t.judge = j
	if t.build != "" {
		if l, ok := j.(*loot.Looter); ok {
			l.SetBuild(t.build)
		}
	}
}

func (t *Town) tune() *env.TownTuning { return &t.e.Tune.Town }
func (t *Town) uiTimeout() time.Duration {
	return time.Duration(t.tune().UITimeoutMs) * time.Millisecond
}
func (t *Town) sleep(ms int) { t.e.Clock.Sleep(time.Duration(ms) * time.Millisecond) }

// ChorePlan says which chores to attempt. Each is skipped when its need check says nothing is to do, unless
// Force is set.
type ChorePlan struct {
	Heal, Identify, Sell, Potions, Scrolls, Repair, Stash, Gold bool
	Antidotes                                                   int  // > 0: top up antidote potions (belt and pack) to this many at the shop (the Andariel kit, task.BossGate)
	Ammo                                                        bool // ammo and ranged gear (ammo.go); no-op for classes without a launcher
	Insure                                                      bool // also stash valuables (see loot.Valuables) before a dangerous trip
	Force                                                       bool
	PotionFloor                                                 int            // > 0: buy healing potions only up to this many in the belt, no mana or rejuvenation (recovery keeps the gold for the kit)
	Cube                                                        bool           // gem and rune cube upgrades (cubejobs.go); the Tuning.Bot.CubeJobs knob decides in the adapter
	CubeKeep                                                    map[string]int // item codes the cube chore must not consume
}

// FullPlan is every chore.
func FullPlan() ChorePlan {
	return ChorePlan{Heal: true, Identify: true, Sell: true, Potions: true, Scrolls: true, Repair: true, Stash: true, Gold: true, Ammo: true}
}

// Step is the outcome of one chore.
type Step struct {
	Name   string
	Ran    bool // false: skipped because nothing was needed
	Err    string
	Detail string
}

// ChoreReport is what Chores did.
type ChoreReport struct {
	Steps      []Step
	GoldBefore int
	GoldAfter  int
	Bought     map[string]int
	Spent      int
	Earned     int
	Sold       int
	Stashed    int
	Identified int
	Healed     bool
	Repaired   bool
	PackFull   bool // the pack is still at or above the trip threshold after the chores
	PackPct    int  // backpack cells in use after the chores, percent
	StashPct   int  // stash cells in use (as far as the snapshot lists the stash), percent
	Errors     int
	Cubed      int  // cube transmutes done
	Broke      bool // a purchase was skipped because gold ran out: a clean "cannot afford", not an error
}

func (r *ChoreReport) add(name string, ran bool, detail string, err error) {
	s := Step{Name: name, Ran: ran, Detail: detail}
	if err != nil {
		s.Err = err.Error()
		r.Errors++
	}
	r.Steps = append(r.Steps, s)
}

// readOwner returns the address of the unit that owns the container an item sits in:
// ItemData(+0x14 of the unit) -> pOwnerInventory (+0x5C) -> Inventory.pOwner (+0x08). The ItemData part
// is UNVERIFIED (D2BS layout); docs/town.md records whether live reads confirmed it.
func (t *Town) readOwner(it d2.Unit) (uint32, error) {
	src := t.e.Mem
	data, err := mem.U32(src, it.Addr+0x14)
	if err != nil || data == 0 {
		return 0, fmt.Errorf("item data: %v", err)
	}
	inv, err := mem.U32(src, data+0x5C)
	if err != nil || inv == 0 {
		return 0, fmt.Errorf("owner inventory: %v", err)
	}
	return mem.U32(src, inv+0x08)
}

// current is a fresh in-game snapshot.
func (t *Town) current() (*d2.Snapshot, error) { return t.h.Fresh() }

func dist(ax, ay, bx, by int) float64 { return math.Hypot(float64(ax-bx), float64(ay-by)) }

func (t *Town) layout(s *d2.Snapshot) (*panel.Layout, error) { return t.h.Layout(s) }

// Source replaces where the Town reads the game from. Zero fields keep the defaults (the Env's Reader).
// It exists for replay and tests; the simulator in internal/town/sim fills all four.
type Source struct {
	Snapshot func() (*d2.Snapshot, error)
	Items    func() ([]d2.Unit, error)
	Owner    func(it d2.Unit) (uint32, error)
	Stats    func(addr uint32) (d2.Stats, error)
	Presets  func() []d2map.Preset // preset units of the current level (default: the Env's World tracker)

	// Backend seams for a game that reports its own layout (the d2d backend, internal/d2dtown). All optional.
	MenuEntry func(Entry) (input.Point, bool)   // click point of a dialog entry of the open NPC menu
	MenuIndex func(idx int) (input.Point, bool) // click point of line idx (0 = first line below the header)
	FixedTabs bool                              // vendor tab slot = stock page, whatever pages the NPC stocks
}

// UseSource installs a Source (and the same snapshot function for the shared hand and the Judge when it is a *loot.Looter).
func (t *Town) UseSource(s Source) {
	if s.Snapshot != nil {
		t.h.Snap = s.Snapshot
	}
	if s.Items != nil {
		t.itemsFn = s.Items
	}
	if s.Owner != nil {
		t.owner = s.Owner
	}
	if s.Stats != nil {
		t.statsFn = s.Stats
	}
	if s.Presets != nil {
		t.presetsFn = s.Presets
	}
	if s.MenuEntry != nil {
		t.menuEntry = s.MenuEntry
	}
	if s.MenuIndex != nil {
		t.menuIndex = s.MenuIndex
	}
	t.fixedTabs = t.fixedTabs || s.FixedTabs
	if l, ok := t.judge.(*loot.Looter); ok {
		l.UseSource(s.Snapshot, s.Stats)
	}
}

// Live-check hooks (used by internal/town/cmd/townlive): open an NPC dialog by class, the point of a dialog
// entry, and the stash. They are the same code paths Chores uses.
func (t *Town) OpenForTest(ctx context.Context, class int) (d2.Unit, error) {
	if _, err := t.current(); err != nil {
		return d2.Unit{}, err
	}
	return t.openNPC(ctx, []int{class}, t.npcPreset(class))
}

// OpenDirectForTest is OpenForTest without the walk next to the NPC: the game itself walks to the clicked NPC (merc.go).
func (t *Town) OpenDirectForTest(ctx context.Context, class int) (d2.Unit, error) {
	t.noWalk = true
	defer func() { t.noWalk = false }()
	return t.OpenForTest(ctx, class)
}

// PickEntryForTest clicks dialog line entry (an Entry) of the open dialog and waits for flag (-1: no wait).
func (t *Town) PickEntryForTest(ctx context.Context, npc d2.Unit, entry Entry, flag int) error {
	return t.pickEntry(ctx, npc, entry, flag)
}

// EquipForTest equips a pack item of the base code into its slot by pick-up and click (the slot-swap path).
func (t *Town) EquipForTest(ctx context.Context, code string) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if b := t.base(u); b != nil && b.Code == code {
			for _, g := range gearSlots {
				if ty := t.e.Data.ItemType(b.Type); ty != nil && (ty.BodyLocs[0] == g.loc || ty.BodyLocs[1] == g.loc) {
					return t.equip(ctx, g, u.Class)
				}
			}
		}
	}
	return fmt.Errorf("no %s in the pack", code)
}

func (t *Town) OpenStashForTest(ctx context.Context) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return ErrNotInTown
	}
	return t.openStash(ctx, a)
}

// SellForTest sells one pack item to the open vendor; it returns the gold gained.
func (t *Town) SellForTest(ctx context.Context, u d2.Unit) (int, error) {
	var rep ChoreReport
	refused, err := t.sellTo(ctx, []d2.Unit{u}, &rep)
	if err == nil && len(refused) > 0 {
		err = fmt.Errorf("vendor did not buy %d item(s)", len(refused))
	}
	return rep.Earned, err
}

// BuyForTest buys one item with the base code from the open vendor npc; it returns the price paid.
func (t *Town) BuyForTest(ctx context.Context, npc d2.Unit, code string) (int, error) {
	s, err := t.current()
	if err != nil {
		return 0, err
	}
	lay, err := t.layout(s)
	if err != nil {
		return 0, err
	}
	stock, err := t.Stock(npc)
	if err != nil {
		return 0, err
	}
	for _, u := range stock {
		if b := t.base(u); b != nil && b.Code == code {
			return t.buyUnit(ctx, lay, u)
		}
	}
	return 0, fmt.Errorf("no %s in stock", code)
}

// StashRoundTripForTest moves the first pack item into the stash and back (stash must be open) and reports
// each step; every move is verified in memory by hand.Transfer.
func (t *Town) StashRoundTripForTest(ctx context.Context) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	pack := s.Inventory.Container(d2.ContInventory)
	if len(pack) == 0 {
		return fmt.Errorf("pack is empty")
	}
	u := pack[0]
	if err := t.h.Transfer(ctx, u, lay.Inventory, lay.Stash, d2.ContStash); err != nil {
		return fmt.Errorf("into stash: %w", err)
	}
	s, _ = t.current()
	fmt.Println("in stash:", len(s.Inventory.Container(d2.ContStash)), "stash items")
	moved, _ := hand.Find(s.Inventory, u.ID)
	if err := t.h.Transfer(ctx, moved, lay.Stash, lay.Inventory, d2.ContInventory); err != nil {
		return fmt.Errorf("back to pack: %w", err)
	}
	return nil
}

// StashGoldForTest runs the gold deposit on the open stash (the threshold knob decides).
func (t *Town) StashGoldForTest(ctx context.Context) error {
	var rep ChoreReport
	return t.stashGold(ctx, &rep)
}

// StripForTest sells every equipped item to the open vendor (it must buy that type): the equipment slot is
// clicked (item to cursor) and the item is dropped on the vendor grid. It returns the gold gained.
func (t *Town) StripForTest(ctx context.Context) (int, error) {
	s, err := t.current()
	if err != nil {
		return 0, err
	}
	lay, err := t.layout(s)
	if err != nil {
		return 0, err
	}
	g0 := s.Player.Gold
	for _, g := range gearSlots {
		cur, err := t.current()
		if err != nil {
			return 0, err
		}
		u := cur.Inventory.Equipped(g.body)
		if u == nil {
			continue
		}
		c := lay.Slots[g.slot].Center()
		if err := t.e.In.Click(input.Left, c.X, c.Y); err != nil {
			return 0, err
		}
		if _, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return s.Inventory.Cursor != nil }); err != nil || !ok {
			return 0, fmt.Errorf("strip %s: item did not reach the cursor (%v)", g.name, err)
		}
		v := lay.Vendor.Cell(lay.Vendor.Cols/2, lay.Vendor.Rows-2)
		if err := t.e.In.Click(input.Left, v.X, v.Y); err != nil {
			return 0, err
		}
		if _, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool { return s.Inventory.Cursor == nil && s.Inventory.Equipped(g.body) == nil }); err != nil || !ok {
			_ = t.h.ClearCursor(ctx)
			return 0, fmt.Errorf("strip %s: vendor did not take the item (%v)", g.name, err)
		}
	}
	s, err = t.current()
	if err != nil {
		return 0, err
	}
	return s.Player.Gold - g0, nil
}

// UseWeapon swaps the quest weapon with the base code (or display name) into the weapon slot, runs fn, and swaps the
// old weapon back (or takes the quest weapon off when the slot was empty). The swap back also runs when fn fails; its
// error is joined. Panels are left to the caller's CloseAll.
func (t *Town) UseWeapon(ctx context.Context, code string, fn func(context.Context) error) error {
	old, err := t.h.WornWeapon(ctx)
	if err != nil {
		return err
	}
	if err := t.h.Equip(ctx, code); err != nil {
		return fmt.Errorf("town: equip %s: %w", code, err)
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return err
	}
	ferr := fn(ctx)
	rctx := ctx
	if ctx.Err() != nil { // a cancelled run still restores the weapon
		var cancel context.CancelFunc
		rctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
	}
	var rerr error
	if old != "" {
		rerr = t.h.Equip(rctx, old)
	} else if s, err := t.h.Fresh(); err == nil {
		if u := s.Inventory.Equipped(d2.BodyRHand); u != nil {
			rerr = t.h.Unequip(rctx, u.ID)
		}
	}
	if rerr != nil {
		rerr = fmt.Errorf("town: restore weapon: %w", rerr)
	}
	return errors.Join(ferr, rerr)
}
