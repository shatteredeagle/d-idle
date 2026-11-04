package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dcombat"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dloot"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2dtown"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/drill"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/belt"
)

// lab is the live instance: a hidden d2d with a throwaway character, wired like cmd/d2bot's d2d stack (d2dwire.go
// buildD2DStack: town.Town over d2dtown with the hover-then-click input, the stack's Looter attached to d2d's ground
// reads, combat's Fighter over d2dcombat). Drills call the same exported functions the bot calls.
type lab struct {
	be  *d2denv.Backend
	db  *data.DB
	tw  *town.Town
	lt  *loot.Looter
	rig *d2dcombat.Rig
	log *slog.Logger
	rec *recorder
	nv  interface {
		GoToUnitOf(ctx context.Context, typ int, id uint32) error
	}
	led  *cheat.Ledger
	char string
	inst string
	tune *env.Tuning
	navr *nav.Navigator
	lv   *leveler.Leveler

	mu  sync.Mutex
	seq int

	wpNav  *nav.Navigator // DW: the bot's navigator with the spy in front (wp_lab.go)
	wpTabs string         // wp_tabs rects seen at the last tab click (DW)
	wpSpy  *spy

	runPol *d2dinput.RunPolicy // D-run: the controller's policy while it is switched off
	left   bool                // the chicken hook ran Save and Exit (RT.leave)
	runDir int                 // D-run: the direction of the measured leg (-1 not found yet; set by Leg)
}

func newLab(be *d2denv.Backend, db *data.DB, led *cheat.Ledger, char, inst string, log *slog.Logger) *lab {
	// d2dctl.Pause waits wall time, which does nothing under lockstep (the game only moves when stepped): EnterPortal,
	// HoverClick and the NPC walks would act on a frozen world. The lab pauses in virtual ticks instead.
	d2dctl.Pause = func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		_, err := be.Client.Step(ctx, max(int(d/(40*time.Millisecond)), 1))
		return err
	}
	e := be.Env
	nv := be.Navigator()
	rig := d2dcombat.Attach(context.Background(), e, nv, be.Ctl)
	te := *e
	te.In = d2dtown.Input(be.Ctl, be.Client)
	tw := town.New(&te, nv)
	ad := d2dtown.New(be.Client, be.Source, d2dsrc.NewDBTables(db))
	tw.UseSource(ad.Source())
	nv.Escape = tw.ToTown
	nv.ClearCursor = tw.ClearCursor
	lt := loot.New(rig.Env, nv)
	tw.SetJudge(lt)
	d2dloot.Attach(lt, be.Source, db)
	l := &lab{be: be, db: db, tw: tw, lt: lt, rig: rig, log: log, led: led, char: char, inst: inst, nv: nv, navr: nv, runDir: -1}
	rig.Fighter.Leave = leaveHook(be.Cycler, func() { l.left = true })
	return l
}

// ---- drill.World ----

// UI reads "dump ui": panels, dialogs and the cursor item. The mini panel and the automap are not panels.
func (l *lab) UI(ctx context.Context) (drill.UI, error) {
	f, err := l.be.Client.UI(ctx)
	if err != nil {
		return drill.UI{}, err
	}
	u := drill.UI{Cursor: f.Held}
	for _, p := range []struct {
		on   bool
		name string
	}{{f.Inventory, "inventory"}, {f.Stash, "stash"}, {f.Cube, "cube"}, {f.NPCMenu, "npc-dialog"}, {f.Vendor, "vendor"},
		{f.SkillTree, "skill-tree"}, {f.Waypoint, "waypoint"}, {f.GameMenu, "game-menu"}} {
		if p.on {
			u.Panels = append(u.Panels, p.name)
		}
	}
	return u, nil
}

// Recover closes what the gate found: a click on the ground drops a cursor item, Escape closes panels and dialogs
// (never sent on a clean screen: a second Escape would open the game menu).
func (l *lab) Recover(ctx context.Context) error {
	c := l.be.Client
	u, err := l.UI(ctx)
	if err != nil {
		return err
	}
	if u.Cursor != "" {
		if err := c.Click(ctx, 400, 300); err != nil {
			return err
		}
		_, _ = c.Step(ctx, 10)
	}
	if len(u.Panels) > 0 {
		if err := c.Key(ctx, "Escape"); err != nil {
			return err
		}
		_, _ = c.Step(ctx, 10)
	}
	return nil
}

// Cheat runs one d2d "debug" verb as a setup cheat and writes it to the cheat ledger first (the ledger must be
// writable before anything is applied) and completes the entry with the reply.
func (l *lab) Cheat(ctx context.Context, actor, verb, reason string) (string, error) {
	if !strings.HasPrefix(verb, "debug ") {
		return "", fmt.Errorf("cheat %q is not a d2d debug verb", verb)
	}
	l.mu.Lock()
	l.seq++
	id := fmt.Sprintf("%s-%d", time.Now().UTC().Format("20060102T150405"), l.seq)
	l.mu.Unlock()
	e := cheat.Entry{Time: time.Now().UTC(), Actor: actor, Instance: l.inst, Character: l.char, Kind: "d2d-debug",
		Params: map[string]any{"id": id, "verb": verb}, Reason: reason}
	if err := l.led.Append(e); err != nil { // refuses an empty reason too
		return "", err
	}
	r, err := l.be.Client.Do(ctx, verb)
	out := ""
	if r != nil {
		out = strings.TrimSpace(strings.Join(append(append([]string{}, r.Lines...), r.Rest), "\n"))
	}
	e.Time, e.Success, e.After = time.Now().UTC(), err == nil, map[string]any{"reply": out}
	if err != nil {
		e.Error = err.Error()
	}
	_ = l.led.Append(e)
	return out, err
}

// ---- drill.BeltLab ----

func (l *lab) Tick(ctx context.Context, n int) error {
	_, err := l.be.Client.Step(ctx, n)
	l.syncPacer() // F17: keep the navigator's clock on game time
	return err
}

// syncPacer steps one tick through the Pacer (the navigator's clock under lockstep) and returns how far its virtual time
// lagged the game: ticks stepped with Client.Step directly are invisible to it until its next step, which then jumps
// by all of them at once. A walk takes its deadline and stall timers before that first step, so a lag near the walk
// timeout expires it after a few seconds (F17: DW.m1 run 1). 0 on a wall clock.
func (l *lab) syncPacer() time.Duration {
	if l.be.Clock == nil || !l.be.Clock.Lockstep() {
		return 0
	}
	_, _, v0, _ := l.be.Clock.Pace()
	l.be.Clock.Sleep(d2dinput.TickDur)
	_, _, v1, _ := l.be.Clock.Pace()
	return v1 - v0 - d2dinput.TickDur
}

// PlanCols is the plan town.Town.plan returns for the character (belt.PlanForBuild; the build name is unset here).
func (l *lab) PlanCols(ctx context.Context) ([4]drill.Kind, error) {
	var out [4]drill.Kind
	s, err := l.be.Env.Snapshot()
	if err != nil || s.Player == nil {
		return out, fmt.Errorf("no player: %v", err)
	}
	for i, k := range belt.PlanForBuild(s.Player.Class, s.Player.Level, s.Player.Difficulty, "").Cols {
		switch k {
		case belt.HP:
			out[i] = "hp"
		case belt.MP:
			out[i] = "mp"
		}
	}
	return out, nil
}

func (l *lab) code(class int) string {
	if class >= 0 && class < len(l.db.Items) {
		return l.db.Items[class].Code
	}
	return fmt.Sprintf("?%d", class)
}

func (l *lab) View(ctx context.Context) (drill.View, error) {
	s, err := l.be.Env.Snapshot()
	if err != nil {
		return drill.View{}, err
	}
	if s.Player == nil || s.Inventory == nil {
		return drill.View{}, errors.New("no player or inventory in the snapshot")
	}
	v := drill.View{Life: s.Player.Life, MaxLife: s.Player.MaxLife, Mana: s.Player.Mana, MaxMana: s.Player.MaxMana, Level: s.Player.Level,
		BeltSlots: belt.Slots(l.db, s.Inventory), Belt: map[int]string{}, Pack: map[string]int{}}
	for _, u := range s.Inventory.Container(d2.ContBelt) {
		v.Belt[u.Item.GY*belt.Columns+u.Item.GX] = l.code(u.Class)
		v.BeltRaw = append(v.BeltRaw, fmt.Sprintf("%d,%d %s", u.Item.GX, u.Item.GY, l.code(u.Class)))
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		v.Pack[l.code(u.Class)]++
	}
	for _, u := range s.Ground {
		v.Ground = append(v.Ground, l.code(u.Class))
	}
	return v, nil
}

func (l *lab) GiveGround(ctx context.Context, r *drill.Run, code string) error {
	_, err := r.Cheat(ctx, "debug give "+code, "put a "+code+" on the ground at the character's feet: the drill's potion supply (no other way to create an item)")
	return err
}

func (l *lab) ClearPack(ctx context.Context, r *drill.Run) error {
	if err := l.ensureAlive(ctx, r); err != nil {
		return err
	}
	if _, err := r.Cheat(ctx, "debug clearinv", "empty the pack grid so the drill's potion counts are exact"); err != nil {
		return err
	}
	return l.Tick(ctx, 5) // the grid reaches the reader one tick later (V: D4.b run 2 saw the old, full pack)
}

func (l *lab) SetLife(ctx context.Context, r *drill.Run, points int) error {
	_, err := r.Cheat(ctx, fmt.Sprintf("debug stat 6 %d", points*256), fmt.Sprintf("set current life to %d (24.8 fixed point) so the drink has something to heal", points))
	if err != nil {
		return err
	}
	return l.Tick(ctx, 5) // the stat reaches the reader one tick later (V)
}

func (l *lab) SetMana(ctx context.Context, r *drill.Run, points int) error {
	_, err := r.Cheat(ctx, fmt.Sprintf("debug stat 8 %d", points*256), fmt.Sprintf("set current mana to %d (24.8 fixed point) so the drink has something to refill", points))
	if err != nil {
		return err
	}
	return l.Tick(ctx, 5)
}

func (l *lab) EquipBelt(ctx context.Context, r *drill.Run, code string) error {
	_, err := r.Cheat(ctx, "debug equip "+code+" 8", "wear a "+code+" belt in body slot 8: the belt type under test")
	if err != nil {
		return err
	}
	return l.Tick(ctx, 10)
}

// PickupRaw is the engine's own pickup (verb "cmd pickup" on every ground item), not the bot's pickit: setup only.
func (l *lab) PickupRaw(ctx context.Context) (int, error) {
	c := l.be.Client
	n := 0
	for round := 0; round < 4; round++ {
		g, err := c.Ground(ctx)
		if err != nil {
			return n, err
		}
		before := n
		for _, it := range g.Items {
			if it.IsGold() {
				continue
			}
			if err := c.CmdPickup(ctx, it.Unit); err != nil {
				return n, err
			}
			n++
			if _, err := c.Step(ctx, 60); err != nil {
				return n, err
			}
		}
		if n == before {
			break
		}
	}
	return n, nil
}

// DrainBelt drinks every belt potion with the belt verb until the belt is empty (a game action, not a cheat).
func (l *lab) DrainBelt(ctx context.Context) error {
	for i := 0; i < 24; i++ {
		v, err := l.View(ctx)
		if err != nil {
			return err
		}
		if len(v.Belt) == 0 {
			return nil
		}
		for col := 0; col < belt.Columns; col++ {
			if err := l.be.Client.Drink(ctx, col); err != nil {
				return err
			}
			if _, err := l.be.Client.Step(ctx, 10); err != nil {
				return err
			}
		}
	}
	return errors.New("belt did not empty after 24 rounds of drinking")
}

func (l *lab) FillBelt(ctx context.Context) (drill.FillInfo, error) {
	mark := l.rec.Mark()
	rep, err := l.tw.FillBelt(ctx)
	return drill.FillInfo{Moved: rep.Moved, Failed: rep.Failed, Err: err, Log: l.rec.Since(mark)}, err
}

func (l *lab) Pickit(ctx context.Context) (drill.PickInfo, error) {
	mark := l.rec.Mark()
	rep, err := l.lt.PickUp(ctx, loot.PickOpts{Radius: 60})
	info := drill.PickInfo{Seen: rep.Seen, Log: l.rec.Since(mark)}
	for _, p := range rep.Picked {
		info.Picked = append(info.Picked, fmt.Sprintf("%s (%s)", p.Name, p.Verdict))
	}
	for _, s := range rep.Skipped {
		info.Skipped = append(info.Skipped, fmt.Sprintf("%s: %s", s.Name, s.Why))
	}
	for _, s := range rep.Failed {
		info.Failed = append(info.Failed, fmt.Sprintf("%s: %s", s.Name, s.Why))
	}
	return info, err
}

func (l *lab) Drink(ctx context.Context, k drill.Kind) error {
	a := combat.DrinkLife
	if k == "mp" {
		a = combat.DrinkMana
	}
	return l.rig.Fighter.Act(ctx, a)
}

var _ drill.BeltLab = (*lab)(nil)
var _ drill.World = (*lab)(nil)

// ShiftClickPack sends the lab binary's click modifier verb ("click x y shift", fork branch with the click modifier) on
// the first pack item with this code: open the pack, "debug itemat" for the pixel, shift-click, close the pack. The bot
// does not call this on d2d (its belt fill uses "cmd belt put"), so the drill drives the verb directly.
func (l *lab) ShiftClickPack(ctx context.Context, code string) error {
	s, err := l.be.Env.Snapshot()
	if err != nil || s.Inventory == nil {
		return fmt.Errorf("no inventory: %v", err)
	}
	id := -1
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if l.code(u.Class) == code {
			id = int(u.ID)
			break
		}
	}
	if id < 0 {
		return fmt.Errorf("no %s in the pack", code)
	}
	c := l.be.Client
	if err := c.Key(ctx, "i"); err != nil {
		return err
	}
	defer func() { _ = c.Key(ctx, "i"); _, _ = c.Step(ctx, 5) }()
	if _, err := c.Step(ctx, 18); err != nil {
		return err
	}
	x, y, err := c.ItemAt(ctx, id)
	if err != nil {
		return err
	}
	if err := c.Hover(ctx, x, y); err != nil {
		return err
	}
	if _, err := c.Step(ctx, 4); err != nil {
		return err
	}
	if _, err := c.Do(ctx, fmt.Sprintf("click %d %d shift", x, y)); err != nil {
		return fmt.Errorf("click with shift: %w", err)
	}
	_, err = c.Step(ctx, 6)
	return err
}

// GiveBelt is the cheat `debug give <code> belt <box>`: a potion straight into a belt box (the drill's way to fill columns).
func (l *lab) GiveBelt(ctx context.Context, r *drill.Run, code string, box int) error {
	_, err := r.Cheat(ctx, fmt.Sprintf("debug give %s belt %d", code, box), "put a "+code+" in belt box "+fmt.Sprint(box)+": the drill needs healing in chosen columns and no other way fills one box")
	if err != nil {
		return err
	}
	return l.Tick(ctx, 5)
}

// ensureAlive is the per-case setup guard of the drills7 harness fix (DRILLS6 F9): a drill that leaves the character dead
// (a `debug hit` with low life) or hurt must not break the next one. A dead character takes the game's own Esc respawn
// (the Fighter's, not a cheat: `debug revive` does not exist in the lab build); life is then topped up by the logged
// cheat. Nothing happens to a healthy character.
func (l *lab) ensureAlive(ctx context.Context, r *drill.Run) error {
	s, err := l.be.Env.Snapshot()
	if err != nil || s == nil || s.Player == nil {
		return nil
	}
	if combat.IsDead(s.Player) {
		if err := l.rig.Fighter.Respawn(ctx, 40*time.Second); err != nil {
			return fmt.Errorf("harness: respawn after a lethal hit: %w", err)
		}
		_, _ = l.be.Client.Step(ctx, 40)
		if s, err = l.be.Env.Snapshot(); err != nil || s == nil || s.Player == nil {
			return nil
		}
	}
	if mx := s.Player.MaxLife; mx > 0 && s.Player.Life < mx {
		if _, err := r.Cheat(ctx, fmt.Sprintf("debug stat 6 %d", mx*256), "top up life before a drill's setup: an earlier drill's hit or death must not decide this one (drills7 harness)"); err != nil {
			return err
		}
		_, _ = l.be.Client.Step(ctx, 5)
	}
	return nil
}
