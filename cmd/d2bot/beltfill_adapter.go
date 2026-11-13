package main

import (
	"context"

	"d2lod-bot/internal/d2"
)

// NeedsBeltFill and FillBelt make townAdapter a bot.BeltFiller.
func (a townAdapter) NeedsBeltFill(s *d2.Snapshot) bool { return a.t.NeedsBeltFill(s) }

func (a townAdapter) FillBelt(ctx context.Context) (int, error) {
	rep, err := a.t.FillBelt(ctx)
	return rep.Moved + rep.Repaired, err // a repair alone changed the belt too: the runner takes a fresh snapshot
}

// TPCount and TakeTPBought make townAdapter (and so d2dTown) a bot.TPDiag: the adapter holds the Town in a named field,
// so without these the runner's type assertion for the minute line fields tp and tpBuy failed and tp read 0 on every
// line (g2g: 2500 lines, while inv-tp-dump counted a tome).
func (a townAdapter) TPCount(s *d2.Snapshot) int { return a.t.TPCount(s) }
func (a townAdapter) TakeTPBought() int          { return a.t.TakeTPBought() }
