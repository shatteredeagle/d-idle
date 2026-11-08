package watch

import (
	"sort"

	"d2lod-bot/internal/unlock"
)

// LadderUnlocker is the senses Unlocker on top of the unlock ladder (TASK-149, docs/senses-unlock.md): tier n is allowed
// when the ladder's senses entry of that tier is open. Precedence: the panel is off when no Senses is mounted
// (knob senses.tier 0); otherwise Cap, when above 0, is a maximum on top of the ladder (an explicit senses.tier
// override, e.g. to keep a stream calm); Cap 0 leaves the ladder alone. Without a ladder the knob's FixedTier applies.
type LadderUnlocker struct {
	Engine *unlock.Engine
	Cap    int
}

func (l LadderUnlocker) Allowed(tier int) bool {
	if l.Engine == nil || tier < 1 || (l.Cap > 0 && tier > l.Cap) {
		return false
	}
	return l.Engine.SenseTier(tier)
}

// Locked lists the senses tiers still closed, lowest first. A tier above Cap is not listed: it cannot open.
func (l LadderUnlocker) Locked() []SenseLock {
	if l.Engine == nil {
		return nil
	}
	var out []SenseLock
	for _, s := range l.Engine.States("senses") {
		if s.Open || (l.Cap > 0 && s.Tier > l.Cap) {
			continue
		}
		out = append(out, SenseLock{ID: s.ID, Tier: s.Tier, Title: s.Title, Teaser: s.Teaser, Price: s.Price})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tier < out[j].Tier })
	return out
}

// Met says whether a monster of that name is in the Codex: the per-class part of tier 3 (a kind not met shows a plain
// dot). Name is the Codex subject as the firsts table spells it.
func (l LadderUnlocker) Met(monster string) bool {
	return l.Engine != nil && l.Allowed(3) && l.Engine.Met("monster", monster)
}

// NewSensesLock picks the Unlocker for the knob senses.tier (0 means no panel: the caller mounts no Senses). Without a
// ladder it is FixedTier(knob). With one the ladder decides; a knob above 1 is a maximum on top of it, while 1 (the
// default, "map only" before the ladder existed) is not a cap, or the ladder could never open tier 2.
func NewSensesLock(eng *unlock.Engine, knob int) Unlocker {
	if eng == nil {
		return FixedTier(knob)
	}
	l := LadderUnlocker{Engine: eng}
	if knob > 1 {
		l.Cap = knob
	}
	return l
}
