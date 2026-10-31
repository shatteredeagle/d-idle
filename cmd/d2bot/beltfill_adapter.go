package main

import (
	"context"

	"d2lod-bot/internal/d2"
)

// NeedsBeltFill and FillBelt make townAdapter a bot.BeltFiller.
func (a townAdapter) NeedsBeltFill(s *d2.Snapshot) bool { return a.t.NeedsBeltFill(s) }

func (a townAdapter) FillBelt(ctx context.Context) (int, error) {
	rep, err := a.t.FillBelt(ctx)
	return rep.Moved, err
}
