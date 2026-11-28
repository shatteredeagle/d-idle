package combat

import (
	"time"

	"d2lod-bot/internal/data"
)

// evidenceGap is the least time between two evidence lines of one rule and outcome.
const evidenceGap = 30 * time.Second

// evidence writes the line `combat evidence rule=<knob> outcome=<what>` when a rule behind a default-0 knob met its
// case, also when it then did nothing: a log window without the line says the rule was never exercised, which a
// missing action line alone cannot (docs/probes/graduations-2026-11-28.md).
func (f *Fighter) evidence(rule, outcome string, args ...any) {
	now, k := f.now(), rule+"/"+outcome
	f.diagMu.Lock()
	if at, ok := f.evidAt[k]; ok && now.Sub(at) < evidenceGap {
		f.diagMu.Unlock()
		return
	}
	if f.evidAt == nil {
		f.evidAt = map[string]time.Time{}
	}
	f.evidAt[k] = now
	f.diagMu.Unlock()
	f.info("evidence", append([]any{"rule", rule, "outcome", outcome}, args...)...)
}

// noteCastGuard logs what the portal guard did with a town portal the survival rule asked for in a fight: the
// refusal (both modes), or the mode 1 pass with its terms.
func (f *Fighter) noteCastGuard(in SafetyInput, a SafetyAction) {
	if !in.InFight || decideBase(in) != TownPortal {
		return
	}
	b, why := in.castBlocked()
	switch g := in.CastGuard; {
	case b && a != TownPortal:
		f.noteCastRefused(why, in)
	case !b && g.Mode == 1:
		f.evidence("portalGuard", "allowed", "adjacent", in.Adjacent, "incoming", int(g.Incoming), "dmgKnown", g.DmgKnown, "life", pct(in.Life, in.Max), "mode", g.Mode)
	}
}

// noteThreat logs what the threat-model dodge made of the ranked hostile missiles; nothing when none was near.
func (f *Fighter) noteThreat(inc Incoming, dodged bool) {
	if len(inc.Dodge)+len(inc.Shooters) == 0 {
		return
	}
	outcome := "dodge"
	switch {
	case dodged:
	case len(inc.Dodge) == 0:
		outcome = "kill-shooter"
	default:
		outcome = "no-room"
	}
	f.evidence("threatDodge", outcome, "missiles", len(inc.Dodge), "shooters", len(inc.Shooters))
}

// standKept reports whether the StandOK knob keeps a target the reach rule would drop as never in range.
func standKept(standOK int, m *data.Monster) bool { return standOK != 0 && Stands(m) }
