package bot

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
)

// profileChecker is the optional Fighter part that says whether the hero has a combat profile (combat.Fighter
// .CheckProfile selects it now, so the answer is known before the first fight and clears when the profile resolves).
type profileChecker interface {
	CheckProfile(p *d2.Player) error
}

// profileErr is the reason the hero has no combat profile, nil when it has one or the fighter cannot say.
func (r *Runner) profileErr(p *d2.Player) error {
	switch f := r.d.Fighter.(type) {
	case profileChecker:
		return f.CheckProfile(p)
	case interface{ ProfileErr() error }:
		return f.ProfileErr()
	}
	return nil
}

// noProfilePark holds a hero without a combat profile in town (review C B1): the fighter refuses to fight, so any goal
// that leaves town would walk it through packs without attacking. Home first, then the town chores; one status and one
// ops log line name the class and the build. The park ends on the tick the profile resolves.
func (r *Runner) noProfilePark(ctx context.Context, snap *d2.Snapshot, c control) (acted, parked bool) {
	err := r.profileErr(snap.Player)
	if err == nil {
		if r.noProfile.Swap(false) {
			r.noProfileWhat = ""
			r.e.Log.Info("combat profile resolved: the hero leaves the park")
		}
		return false, false
	}
	what := fmt.Sprintf("no combat profile for %s/%s", snap.Player.Class, r.buildName(snap.Player))
	r.noProfile.Store(true)
	if what != r.noProfileWhat {
		r.noProfileWhat = what
		r.e.Log.Error(what+": parked in town, not fighting (fix the table's class and combat keys)", "err", err)
	}
	acted = r.rest(ctx, snap, c)
	if snap.Player.InTown() {
		r.setState(StateInTown, "Parked in town: "+what)
	}
	return acted, true
}
