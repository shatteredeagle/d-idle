// Package d2dmerc is the merc.Reader of the d2d backend: it answers from d2dsrc snapshots (the dump "merc" section)
// and reads nothing else.
//
// What the d2d dump carries about the merc (V: internal/d2dsrc/player.go merc(), testdata/dump_head_late.json):
// present, code, position, life current/max, dead. It carries NO level, name, aura, stats or equipped items
// (d2dsrc marks merc.level, merc.name, merc.aura unknown; docs/d2d-probes.md section 4: no verb lists merc gear).
// So State is Known with owned/alive/class(RG only)/life, and Level 0, Aura "", StatsKnown false, Gear nil: the policy
// then fails closed on everything that needs those (revive without a level estimate, gear, requirements).
package d2dmerc

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/merc"
)

// ErrNoOffers: d2d has no hire window reader; the rows are only visible on screen (docs/d2d-probes.md section 4).
var ErrNoOffers = errors.New("d2dmerc: hire window rows are not readable on d2d")

// Reader implements merc.Reader over a d2dsrc read.
type Reader struct {
	// Read returns one snapshot with its report: (*d2dsrc.Source).Read.
	Read func(ctx context.Context) (*d2dsrc.Result, error)
	// LevelEstimate, when set, supplies a merc level for the price formula while the dump has none (U: the caller
	// decides what to guess; an overestimate only makes the revive stricter). Zero or nil keeps Level 0.
	LevelEstimate func(s *d2.Snapshot) int
	// Timeout per read, 0 = 10 s.
	Timeout time.Duration
}

var _ merc.Reader = (*Reader)(nil)

// State reads the merc. An error from the source is returned as is; an unreadable merc section is Known false.
func (r *Reader) State() (merc.MercState, error) {
	d := r.Timeout
	if d == 0 {
		d = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	res, err := r.Read(ctx)
	if err != nil {
		return merc.MercState{}, err
	}
	st := FromResult(res)
	if st.Owned && st.Level == 0 && r.LevelEstimate != nil {
		st.Level = r.LevelEstimate(res.Snap)
	}
	return st, nil
}

// Offers always fails: see ErrNoOffers.
func (r *Reader) Offers() ([]merc.HireOffer, error) { return nil, ErrNoOffers }

// FromResult converts one d2dsrc snapshot. Known needs the dump's merc.present flag: on stock d2d (no dump) the
// merc section is unknown and so is the answer, which keeps "no merc" apart from "cannot tell".
func FromResult(res *d2dsrc.Result) merc.MercState {
	out := merc.MercState{FreeCells: -1}
	if res == nil || res.Snap == nil || res.Snap.Player == nil || !res.Report.Has("merc.present") {
		return out
	}
	out.Known = true
	x := res.Extras.Merc
	if x == nil {
		return out // present=false: no merc
	}
	out.Owned = true
	out.Alive = !x.Dead
	if res.Report.Has("merc.class") {
		out.Class = d2.FindMerc(res.Snap).Class
	}
	if out.Alive && res.Report.Has("merc.life") {
		out.LifePct = max(d2.FindMerc(res.Snap).LifePct, 1)
	}
	return out
}
