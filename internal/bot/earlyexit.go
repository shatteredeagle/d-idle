package bot

import (
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// Early-death rules (TASK-39, docs/research/g2f-diagnosis.md 4.1): seed 13 drank 11 potions in about 17 s in a 12-hostile
// pack, left, bought five Minor Healing and walked back into the same pack 2 s before it died.

// stalePack remembers where and how big the pack was at the last exit forced by a fight.
type stalePack struct {
	area, hostiles int
	at             time.Time
}

// earlyExit is called for every safety action about to be carried out. It records the pack at a chicken, portal or
// crowd exit, and turns the drink that would continue a potion burst into a town portal. snap may be nil.
func (r *Runner) earlyExit(snap *d2.Snapshot, a SafetyAction) SafetyAction {
	if snap == nil || snap.Player == nil || snap.Player.InTown() {
		return a
	}
	t, now := r.e.Tune.Bot, r.e.Clock.Now()
	host := r.hostileNear(snap, r.e.Tune.Combat.FightRadius)
	switch a {
	case SafetyDrinkLife:
		if t.PotionBurstN <= 0 {
			return a
		}
		r.obMu.Lock()
		win := time.Duration(t.PotionBurstSec) * time.Second
		keep := r.ob.drinkTimes[:0]
		for _, d := range r.ob.drinkTimes {
			if now.Sub(d) < win {
				keep = append(keep, d)
			}
		}
		keep = append(keep, now)
		r.ob.drinkTimes = keep
		burst := len(keep) > t.PotionBurstN
		if burst && host >= t.PotionBurstHostiles && r.hasPortalSource(snap) {
			r.ob.drinkTimes = nil
			r.obMu.Unlock()
			r.e.Log.Info(fmt.Sprintf("guard: potion burst (%d drinks in %ds, %d hostiles): town portal instead of another potion", len(keep)-1, t.PotionBurstSec, host))
			r.notePack(snap, host, now)
			return SafetyTownPortal
		}
		r.obMu.Unlock()
	case SafetyTownPortal, SafetyChicken, SafetyCrowdAbandon:
		r.notePack(snap, host, now)
	}
	return a
}

func (r *Runner) notePack(snap *d2.Snapshot, host int, now time.Time) {
	if r.e.Tune.Bot.StalePackHostiles <= 0 || host < r.e.Tune.Bot.StalePackHostiles {
		return
	}
	r.obMu.Lock()
	r.ob.stale = stalePack{area: snap.Player.Area, hostiles: host, at: now}
	r.obMu.Unlock()
	r.e.Log.Info("stale pack noted", "area", snap.Player.Area, "hostiles", host)
}

// stalePackArea is the area the hero fled from a big pack in, while the hold lasts and the kit is not rebuilt (0 = none).
func (r *Runner) stalePackArea(s *d2.Snapshot) int {
	t := r.e.Tune.Bot
	r.obMu.Lock()
	st := r.ob.stale
	r.obMu.Unlock()
	if st.area == 0 || t.StalePackHostiles <= 0 || r.e.Clock.Now().Sub(st.at) >= time.Duration(t.StalePackMinutes)*time.Minute {
		return 0
	}
	if s != nil && s.Player != nil {
		in := r.resInput(s)
		if healingOf(in) >= t.StalePackMinHP && in.Belt.HPTier >= t.StalePackMinTier {
			return 0
		}
	}
	return st.area
}

// redirectStalePack sends a town trip that would return into the stale pack to the first field instead.
func (r *Runner) redirectStalePack(s *d2.Snapshot, t *trip) {
	a := r.stalePackArea(s)
	if a == 0 || t.returnTo != a {
		return
	}
	if first, ok := r.starveStep(s, difficultyName(s.Player.Difficulty)); ok && first != a {
		r.e.Log.Info("field gate: stale pack, first field instead", "area", a, "first", first)
		t.returnTo = first
	}
}
