package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town"
)

// Thin adapters between the behavior packages and internal/bot/contracts.go.

// ---- nav ----

// moverAdapter makes "nothing left to explore" a normal result: the area is exhausted, the runner marks it cleared.
type moverAdapter struct{ *nav.Navigator }

func (m moverAdapter) Explore(ctx context.Context, done func() bool) error {
	if err := m.Navigator.Explore(ctx, done); err != nil && !errors.Is(err, nav.ErrExplored) {
		return err
	}
	return nil
}

// CanTeleport feeds task.World.CanTeleport: the navigator has a Teleport key.
func (m moverAdapter) CanTeleport() bool { return m.Navigator.TeleportKey != "" }

// TeleportToArea hops toward the exit into levelNo (the navigator teleports along the way); without the Teleport key
// there is nothing to hop with.
func (m moverAdapter) TeleportToArea(ctx context.Context, levelNo int) error {
	if m.Navigator.TeleportKey == "" {
		return task.ErrNoTeleport
	}
	return m.Navigator.ToArea(ctx, levelNo)
}

// ---- combat ----

type fighterAdapter struct{ f *combat.Fighter }

func (a fighterAdapter) IsIgnored(id uint32, dist float64) bool { return a.f.IsIgnored(id, dist) }

func (a fighterAdapter) Clear(ctx context.Context, o bot.ClearOpts) (bot.ClearResult, error) {
	res, err := a.f.Clear(ctx, combat.ClearOpts{Radius: o.Radius})
	return bot.ClearResult{Killed: res.Kills, Remaining: res.Skipped + res.Avoided}, mapCombatErr(err)
}
func (a fighterAdapter) Kill(ctx context.Context, id uint32) error {
	return mapCombatErr(a.f.Kill(ctx, id))
}
func (a fighterAdapter) Buff(ctx context.Context) error { return a.f.Buff(ctx) }

var toBot = map[combat.SafetyAction]bot.SafetyAction{
	combat.DrinkLife: bot.SafetyDrinkLife, combat.DrinkMana: bot.SafetyDrinkMana, combat.DrinkRejuv: bot.SafetyDrinkRejuv,
	combat.TownPortal: bot.SafetyTownPortal, combat.Chicken: bot.SafetyChicken,
	combat.CrowdRetreat: bot.SafetyCrowdRetreat, combat.CrowdAbandon: bot.SafetyCrowdAbandon,
}
var toCombat = map[bot.SafetyAction]combat.SafetyAction{
	bot.SafetyDrinkLife: combat.DrinkLife, bot.SafetyDrinkMana: combat.DrinkMana, bot.SafetyDrinkRejuv: combat.DrinkRejuv,
	bot.SafetyTownPortal: combat.TownPortal, bot.SafetyChicken: combat.Chicken,
	bot.SafetyCrowdRetreat: combat.CrowdRetreat, bot.SafetyCrowdAbandon: combat.CrowdAbandon,
}

// LastPortalCast and RecentCasts feed the death autopsy (bot.fighterDiag).
func (a fighterAdapter) LastPortalCast() combat.PortalCast              { return a.f.LastPortalCast() }
func (a fighterAdapter) RecentCasts(w time.Duration) (int, map[int]int) { return a.f.RecentCasts(w) }

// SetWalking tells the fighter the bot is on a travel or quest walk (the crowd-abandon rule applies to walks).
func (a fighterAdapter) SetWalking(on bool) { a.f.SetWalking(on) }

func (a fighterAdapter) Safety(s *d2.Snapshot) bot.SafetyAction { return toBot[a.f.Safety(s)] }
func (a fighterAdapter) Act(ctx context.Context, act bot.SafetyAction) error {
	return mapCombatErr(a.f.Act(ctx, toCombat[act]))
}

// mapCombatErr turns combat's "I opened a portal / I left the game" results into the runner's sentinels.
func mapCombatErr(err error) error {
	switch {
	case errors.Is(err, combat.ErrTownPortal):
		return bot.ErrPortalOpened
	case errors.Is(err, combat.ErrChicken):
		return bot.ErrLeftGame
	}
	return err
}

// ---- town ----

type townAdapter struct {
	t  *town.Town
	e  *env.Env
	mv interface {
		GoToUnitOf(ctx context.Context, typ int, id uint32) error
	}
}

// ToTown reuses a portal the fighter already opened next to the player, else lets the Townsman open one.
func (a townAdapter) ToTown(ctx context.Context) error {
	s, err := a.e.Snapshot()
	if err == nil && s.Player != nil && !s.Player.InTown() {
		for _, o := range s.Objects {
			if o.Class != town.ObjPortal || max(abs(o.X-s.Player.X), abs(o.Y-s.Player.Y)) > 25 {
				continue
			}
			if err := a.e.In.Interact(input.Target{Type: town.TypeObject, ID: o.ID, Pos: input.Point{X: o.X, Y: o.Y}}); err != nil {
				break
			}
			if a.waitTown(ctx, 15*time.Second) == nil {
				return nil
			}
			break
		}
	}
	return a.t.ToTown(ctx)
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (a townAdapter) waitTown(ctx context.Context, d time.Duration) error {
	deadline := a.e.Clock.Now().Add(d)
	for a.e.Clock.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s, err := a.e.Snapshot(); err == nil && s.State == d2.StateInGame && s.Player != nil && s.Player.InTown() {
			return nil
		}
		a.e.Clock.Sleep(150 * time.Millisecond)
	}
	return errors.New("portal did not lead to town")
}

func (a townAdapter) BackToField(ctx context.Context, level int) error {
	err := a.t.BackToField(ctx, level)
	if errors.Is(err, town.ErrWrongLevel) {
		return fmt.Errorf("%w: %v", bot.ErrPortalWrongLevel, err)
	}
	return err
}

func (a townAdapter) Chores(ctx context.Context, p bot.ChorePlan) (bot.ChoreReport, error) {
	// Every visit may restock: the shop chore checks its own need, so a trip that came for a quest, a heal or a full
	// pack still buys the town portal and the potion kit when gold allows (G1 attempt 6).
	tp := town.ChorePlan{Heal: p.Heal, Identify: p.Identify, Sell: p.Sell, Potions: true, Scrolls: true, Repair: p.Repair,
		Stash: p.Stash, Gold: true, Insure: p.Insure, Ammo: true, Force: p.Poverty, PotionFloor: p.PotionFloor, Antidotes: p.Antidotes, Cube: a.e.Tune.Bot.CubeJobs && p.Stash}
	rep, err := a.t.Chores(ctx, tp)
	out := bot.ChoreReport{}
	for _, s := range rep.Steps {
		switch {
		case s.Err != "":
			a.e.Log.Warn("chore failed", "step", s.Name, "err", s.Err)
			out.Skipped = append(out.Skipped, s.Name+": "+s.Err)
		case s.Ran:
			out.Done = append(out.Done, s.Name)
		default:
			out.Skipped = append(out.Skipped, s.Name)
		}
	}
	if err == nil && p.Poverty {
		budget := max(rep.GoldAfter-p.GoldFloor, 0)
		rb, rerr := a.t.Rebuy(ctx, budget)
		a.e.Log.Info("poverty rebuy", "budget", budget, "equipped", rb.Equipped, "missing", rb.Missing, "spent", rb.Spent, "errors", rb.Errors)
		out.Blocked = rb.Blocked
		out.NeedStr, out.NeedDex = rb.NeedStr, rb.NeedDex
		if rerr != nil {
			err = fmt.Errorf("rebuy: %w", rerr)
		}
	}
	return out, err
}

func (a townAdapter) Rebuy(ctx context.Context, budget int) (bot.RebuyReport, error) {
	r, err := a.t.Rebuy(ctx, budget)
	return bot.RebuyReport{Equipped: r.Equipped, Spent: r.Spent, Missing: r.Missing, Blocked: r.Blocked}, err
}
func (a townAdapter) NeedsTrip(s *d2.Snapshot) (bool, string) { return a.t.NeedsTrip(s) }

// ---- loot ----

type looterAdapter struct{ l *loot.Looter }

func (a looterAdapter) PickUp(ctx context.Context, o bot.PickOpts) (bot.PickReport, error) {
	r, err := a.l.PickUp(ctx, loot.PickOpts{Radius: o.Radius, Names: o.Names})
	return bot.PickReport{Picked: len(r.Picked), Skipped: len(r.Skipped)}, err
}

// LeftBehind and ClearLeftBehind feed the runner's "loot: pack full" trip reason.
func (a looterAdapter) LeftBehind() int                 { return a.l.LeftBehind() }
func (a looterAdapter) ClearLeftBehind()                { a.l.ClearLeftBehind() }
func (a looterAdapter) Evaluate(it d2.Item) bot.Verdict { return verdictOf(a.l.Evaluate(it)) }

func (a looterAdapter) EvaluateUnit(u d2.Unit, clvl, gold int) bot.Verdict {
	return verdictOf(a.l.EvaluateWith(u, loot.Ctx{CLvl: clvl, Gold: gold}))
}

func verdictOf(v loot.Verdict) bot.Verdict {
	switch v.Action {
	case loot.Stash:
		return bot.VerdictStash
	case loot.Sell:
		return bot.VerdictSell
	case loot.Drop:
		return bot.VerdictDrop
	}
	return bot.VerdictKeep
}

// ---- out-of-game flow ----

type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

// cycler drives launch.Flow for the bot (fresh games, re-entry after a chicken exit). UNVERIFIED end to end.
type cycler struct {
	flow  *launch.Flow
	name  func() string
	world *d2map.Tracker
	// onCycle is told about every leave/re-enter the bot itself makes (the recycler counts them, docs/soak.md).
	onCycle func()
}

func (c cycler) spec(diff string) launch.CharSpec {
	if diff == "" {
		diff = "normal"
	}
	return launch.CharSpec{Name: c.name(), Difficulty: diff}
}
func (c cycler) EnterGame(ctx context.Context, diff string) error {
	err := c.flow.Start(ctx, c.spec(diff))
	c.forgetMap(err)
	c.noteCycle(err)
	return err
}
func (c cycler) NewGame(ctx context.Context, diff string) error {
	if err := c.flow.LeaveGame(ctx, 30*time.Second); err != nil {
		return err
	}
	err := c.flow.Start(ctx, c.spec(diff))
	c.forgetMap(err)
	c.noteCycle(err)
	return err
}

func (c cycler) noteCycle(err error) {
	if err == nil && c.onCycle != nil {
		c.onCycle()
	}
}

// forgetMap drops the map tracker's caches after a (re-)entry: they are keyed by heap addresses of the old game.
func (c cycler) forgetMap(err error) {
	if err == nil && c.world != nil {
		c.world.Reset()
	}
}

// ---- leveler ----

// gearAdapter exposes the leveler's equip and shop passes as bot.Gear.
type gearAdapter struct {
	l *leveler.Leveler
	v leveler.Vendor
	e *env.Env
}

func (a gearAdapter) PendingSwaps(s *d2.Snapshot) int { return len(a.l.SwapsFor(s)) }
func (a gearAdapter) Equip(ctx context.Context) (bot.GearReport, error) {
	r := a.l.Equip(ctx)
	err := r.Err
	if err == nil && len(r.Failed) > 0 && len(r.Swaps) == 0 {
		err = errors.New(r.Failed[0])
	}
	return bot.GearReport{Swaps: r.Swaps}, err
}
func (a gearAdapter) Shop(ctx context.Context, budget int) (bot.GearReport, error) {
	r := a.l.Shop(ctx, a.v, budget)
	return bot.GearReport{Bought: r.Bought, Spent: r.Spent}, r.Err
}

// questAdapter claims quest rewards by talking to the quest giver. UNVERIFIED end to end: only the Den of Evil (Akara) is
// known, and success is judged by a rising unspent skill point count.
type questAdapter struct {
	t *town.Town
	e *env.Env
}

const npcAkara = 148

func (a questAdapter) Claim(ctx context.Context, quest string) error {
	if quest != "den_of_evil" {
		return fmt.Errorf("no claim procedure for quest %q", quest)
	}
	before, err := a.e.Snapshot()
	if err != nil || before.Player == nil {
		return fmt.Errorf("claim: no snapshot: %v", err)
	}
	npc, err := a.t.OpenForTest(ctx, npcAkara)
	if err != nil {
		return fmt.Errorf("open Akara: %w", err)
	}
	err = a.t.PickEntryForTest(ctx, npc, town.EntryTalk, -1)
	a.e.Clock.Sleep(2500 * time.Millisecond) // her speech
	_ = a.e.In.CloseAll()
	if err != nil {
		return fmt.Errorf("talk to Akara: %w", err)
	}
	after, err := a.e.Snapshot()
	if err == nil && after.Player != nil && after.Player.SkillPoints > before.Player.SkillPoints {
		return nil
	}
	return errors.New("no skill point after talking to Akara (quest not complete, or the dialog missed)")
}
