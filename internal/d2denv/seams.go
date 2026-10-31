package d2denv

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
)

// Seams are the hook points where the behaviour packages plug in. A nil field gets the stub below, which fails every
// action with ErrNotYet and logs "not yet" (once per action kind per minute), so the bot boots, reads, ticks the goal
// manager and stops on d2d before combat and town exist for it.
//
//	Fighter: internal/d2dcombat builds a bot.Fighter over e (reads, semantic input) and nv (movement).
//	Town:    internal/d2dtown builds a bot.Townsman.
//	Loot:    optional bot.Looter (nil keeps the bot's "no pickup" path).
type Seams struct {
	Fighter func(e *env.Env, nv *nav.Navigator) bot.Fighter
	Town    func(e *env.Env, nv *nav.Navigator) bot.Townsman
	Loot    func(e *env.Env, nv *nav.Navigator) bot.Looter
}

// ErrNotYet is what the stubs answer.
var ErrNotYet = errNotYet{}

type errNotYet struct{}

func (errNotYet) Error() string { return "d2d backend: this behaviour is not yet implemented" }

type nag struct {
	log  *slog.Logger
	mu   sync.Mutex
	last map[string]time.Time
}

func (n *nag) say(what string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.last == nil {
		n.last = map[string]time.Time{}
	}
	if t, ok := n.last[what]; ok && time.Since(t) < time.Minute {
		return
	}
	n.last[what] = time.Now()
	n.log.Info("not yet", "what", what)
}

// NotYetFighter is the combat stub: Safety never asks for anything.
func NotYetFighter(log *slog.Logger) bot.Fighter { return notYetFighter{&nag{log: log}} }

type notYetFighter struct{ n *nag }

func (f notYetFighter) Clear(context.Context, bot.ClearOpts) (bot.ClearResult, error) {
	f.n.say("combat.Clear")
	return bot.ClearResult{}, ErrNotYet
}
func (f notYetFighter) Kill(context.Context, uint32) error { f.n.say("combat.Kill"); return ErrNotYet }
func (f notYetFighter) Buff(context.Context) error         { f.n.say("combat.Buff"); return ErrNotYet }
func (notYetFighter) Safety(*d2.Snapshot) bot.SafetyAction { return bot.SafetyNone }
func (f notYetFighter) Act(context.Context, bot.SafetyAction) error {
	f.n.say("combat.Act")
	return ErrNotYet
}

// NotYetTown is the town stub: no chores, no trips.
func NotYetTown(log *slog.Logger) bot.Townsman { return notYetTown{&nag{log: log}} }

type notYetTown struct{ n *nag }

func (t notYetTown) ToTown(context.Context) error { t.n.say("town.ToTown"); return ErrNotYet }
func (t notYetTown) Chores(context.Context, bot.ChorePlan) (bot.ChoreReport, error) {
	t.n.say("town.Chores")
	return bot.ChoreReport{}, ErrNotYet
}
func (t notYetTown) BackToField(context.Context, int) error {
	t.n.say("town.BackToField")
	return ErrNotYet
}
func (t notYetTown) Rebuy(context.Context, int) (bot.RebuyReport, error) {
	t.n.say("town.Rebuy")
	return bot.RebuyReport{}, ErrNotYet
}
func (notYetTown) NeedsTrip(*d2.Snapshot) (bool, string) { return false, "" }

// Behaviors resolves the seams for one backend.
func (b *Backend) Behaviors(s Seams, nv *nav.Navigator) (bot.Fighter, bot.Townsman, bot.Looter) {
	var f bot.Fighter
	var t bot.Townsman
	var l bot.Looter
	if s.Fighter != nil {
		f = s.Fighter(b.Env, nv)
	} else {
		f = NotYetFighter(b.Env.Log)
	}
	if s.Town != nil {
		t = s.Town(b.Env, nv)
	} else {
		t = NotYetTown(b.Env.Log)
	}
	if s.Loot != nil {
		l = s.Loot(b.Env, nv)
	}
	return f, t, l
}
