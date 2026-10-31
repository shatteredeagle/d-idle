package main

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/town"
)

// HireMerc and ReviveMerc make townAdapter a bot.MercTown. Hire stops with ErrMercRowUnknown until a RowPicker is
// set (the aura of a hire-window row is only readable on screen, U); the bot logs it and retries after its backoff.
func (a townAdapter) HireMerc(ctx context.Context, s *d2.Snapshot) (bool, error) {
	w, ok := town.HirePolicy(data.Difficulty(s.Player.Difficulty), d2.FindMerc(s), s.Player.Gold)
	if !ok {
		return false, nil
	}
	rep, err := a.t.Hire(ctx, w)
	return rep.Hired, err
}

func (a townAdapter) ReviveMerc(ctx context.Context, unverified bool) (bool, error) {
	a.t.SetReviveUnverified(unverified)
	a.t.SetMercKnown(true)
	rep, err := a.t.Revive(ctx)
	return rep.Revived, err
}

var _ interface {
	HireMerc(context.Context, *d2.Snapshot) (bool, error)
	ReviveMerc(context.Context, bool) (bool, error)
} = townAdapter{}
