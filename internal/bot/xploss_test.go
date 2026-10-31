package bot

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// observeAt feeds one world snapshot to the observer the way the guard does.
func (g *rig) observeAt(now time.Time) {
	s, _ := g.w.snapshot()
	g.r.obMu.Lock()
	g.r.observe(s, now)
	g.r.obMu.Unlock()
}

func TestXPLostAfterReloadAndAtRisk(t *testing.T) {
	var saved uint32
	var haveSaved bool
	g := newRig(t, func(_ *Config, d *Deps, _ *env.Tuning) {
		d.SavedExp = func() (uint32, bool) { return saved, haveSaved }
	})
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	now := time.Unix(1000, 0)
	g.w.set(func(w *world) { w.exp = 1000 })
	g.observeAt(now)
	g.w.set(func(w *world) { w.exp = 1800 })
	g.observeAt(now.Add(time.Second))

	g.r.obMu.Lock()
	if got := g.r.ob.xpAtRiskLocked(nil); got != 800 {
		t.Fatalf("at risk from entry exp = %d, want 800", got)
	}
	g.r.obMu.Unlock()

	g.r.logXPAtRisk("test")
	if !strings.Contains(buf.String(), "xp at risk") || !strings.Contains(buf.String(), "atRisk=800") {
		t.Fatalf("no at-risk line: %s", buf.String())
	}
	saved, haveSaved = 1500, true // the .d2s is newer than the entry
	g.r.obMu.Lock()
	if got := g.r.ob.xpAtRiskLocked(g.r.d.SavedExp); got != 300 {
		t.Fatalf("at risk from the save = %d, want 300", got)
	}
	g.r.obMu.Unlock()

	// The game dies and comes back from the save: menu, then in game with the old exp.
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.observeAt(now.Add(2 * time.Second))
	g.w.set(func(w *world) { w.state, w.exp = d2.StateInGame, 1000 })
	g.observeAt(now.Add(3 * time.Second))
	g.observeAt(now.Add(4 * time.Second))

	if !strings.Contains(buf.String(), "msg=\"xp lost\"") || !strings.Contains(buf.String(), "lost=800") {
		t.Fatalf("no xp lost line: %s", buf.String())
	}
	g.r.obMu.Lock()
	defer g.r.obMu.Unlock()
	if g.r.ob.xpLost != 800 || g.r.ob.savedExp != 1000 || g.r.ob.pendingXP != 800 {
		t.Fatalf("lost %d saved %d pending %d, want 800/1000/800 (pending is the 1000->1800 gain, the reload adds nothing)", g.r.ob.xpLost, g.r.ob.savedExp, g.r.ob.pendingXP)
	}
}
