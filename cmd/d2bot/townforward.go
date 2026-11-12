package main

import (
	"context"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// Optional Townsman parts the runner finds by type assertion (docs/wiring.md). townAdapter holds the Town in a named
// field and mercTown embeds the Townsman interface, so neither promotes these: without the forwards below the
// runner's assertion fails and the feature is silently off. wiring_test.go fails when one goes missing again.

// ForgetBelt clears the "belt is full" memory at a new game (goal.go).
func (a townAdapter) ForgetBelt() { a.t.ForgetBelt() }

// PlannedSpend is the gold the next town visit would spend (the trip floor, goal.go spendPlanner).
func (a townAdapter) PlannedSpend(s *d2.Snapshot) int { return a.t.PlannedSpend(s) }

// UseWeapon swaps to the weapon with item code for the duration of fn (quest steps that need one).
func (a townAdapter) UseWeapon(ctx context.Context, item string, fn func(context.Context) error) error {
	return a.t.UseWeapon(ctx, item, fn)
}

// ReconcileMarks and MarkOf carry the player's sell/keep marks (leveling.go pack-room count).
func (a townAdapter) ReconcileMarks(s *d2.Snapshot) { a.t.ReconcileMarks(s) }
func (a townAdapter) MarkOf(u d2.Unit) string       { return a.t.MarkOf(u) }

// The merc wrapper forwards the same parts to the Townsman it wraps (absent on the inner one: the zero answer that
// makes the runner skip the feature, as without the wrapper).

func (t *mercTown) ForgetBelt() {
	if f, ok := t.Townsman.(interface{ ForgetBelt() }); ok {
		f.ForgetBelt()
	}
}

func (t *mercTown) PlannedSpend(s *d2.Snapshot) int {
	if p, ok := t.Townsman.(interface{ PlannedSpend(*d2.Snapshot) int }); ok {
		return p.PlannedSpend(s)
	}
	return 0
}

func (t *mercTown) UseWeapon(ctx context.Context, item string, fn func(context.Context) error) error {
	if u, ok := t.Townsman.(interface {
		UseWeapon(context.Context, string, func(context.Context) error) error
	}); ok {
		return u.UseWeapon(ctx, item, fn)
	}
	return task.Errorf(task.KindPrereq, "the town module cannot swap weapons")
}

func (t *mercTown) ReconcileMarks(s *d2.Snapshot) {
	if r, ok := t.Townsman.(interface{ ReconcileMarks(*d2.Snapshot) }); ok {
		r.ReconcileMarks(s)
	}
}

func (t *mercTown) MarkOf(u d2.Unit) string {
	if m, ok := t.Townsman.(interface{ MarkOf(d2.Unit) string }); ok {
		return m.MarkOf(u)
	}
	return ""
}

func (t *mercTown) NeedsBeltFill(s *d2.Snapshot) bool {
	if b, ok := t.Townsman.(bot.BeltFiller); ok {
		return b.NeedsBeltFill(s)
	}
	return false
}

func (t *mercTown) FillBelt(ctx context.Context) (int, error) {
	if b, ok := t.Townsman.(bot.BeltFiller); ok {
		return b.FillBelt(ctx)
	}
	return 0, nil
}
