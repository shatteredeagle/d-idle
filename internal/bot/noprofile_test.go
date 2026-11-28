package bot

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// profFighter is the fake fighter with the profile check of combat.Fighter.
type profFighter struct {
	*fakeFighter
	err atomic.Pointer[error]
}

func (f *profFighter) CheckProfile(*d2.Player) error {
	if e := f.err.Load(); e != nil {
		return *e
	}
	return nil
}

// Review C B1: a hero without a combat profile is parked in town with the reason instead of walking a farm route it
// cannot fight on, the row is hard, and the park ends on the tick the profile resolves. Two classes. Must fail while
// the only reader of the profile error is the invariant tick.
func TestNoCombatProfileParksInTownUntilItResolves(t *testing.T) {
	for _, cl := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(cl.String(), func(t *testing.T) {
			pf := &profFighter{}
			g := newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) {
				pf.fakeFighter = d.Fighter.(*fakeFighter)
				d.Fighter = pf
			})
			bad := errors.New(`table "x_build" has no combat key`)
			pf.err.Store(&bad)
			g.w.set(func(w *world) { w.class = cl })
			g.run()
			defer g.finish()
			g.r.Start(farm("Blood Moor"))
			want := "no combat profile for " + cl.String() + "/"
			g.waitFor("park status", func() bool { return strings.Contains(g.r.Status().Activity, want) })
			time.Sleep(150 * time.Millisecond)
			g.w.mu.Lock()
			area := g.w.area
			g.w.mu.Unlock()
			if area != 1 || fieldWork(g.w.log()) {
				t.Fatalf("a hero without a combat profile left town (area %d): %s", area, g.w.log())
			}
			if !g.r.noProfile.Load() {
				t.Fatal("the park is not flagged")
			}
			g.r.inv.mu.Lock()
			n := g.r.inv.total["no-combat-profile"]
			g.r.inv.mu.Unlock()
			if n == 0 {
				t.Error("the invariant row did not count")
			}
			pf.err.Store(nil)
			g.waitFor("field work after the profile resolved", func() bool { return fieldWork(g.w.log()) })
			if g.r.noProfile.Load() {
				t.Fatal("the park stayed after the profile resolved")
			}
		})
	}
}
