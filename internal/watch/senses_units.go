package watch

import "d2lod-bot/internal/senses"

// PublicUnit is a unit as the watcher may see it (docs/senses-units.md). Which fields are filled depends on the open
// tiers: tier 2 gives id, kind and position only; tier 3 adds name and life for the monster being attacked, and only
// when its class is in the Codex; tier 4 adds the elite kind, own name and mods of elites whose kind the Codex holds.
type PublicUnit struct {
	ID      uint32   `json:"id"`
	Kind    string   `json:"kind"`
	X       int      `json:"x"`
	Y       int      `json:"y"`
	Name    string   `json:"name,omitempty"`
	LifePct int      `json:"lifePct,omitempty"`
	Target  bool     `json:"target,omitempty"`
	Elite   string   `json:"elite,omitempty"`
	Mods    []string `json:"mods,omitempty"`
}

// UnitGate is the Codex side of the per-kind rules; LadderUnlocker implements it. An Unlocker without it (FixedTier)
// shows every class and every elite once the tier is open.
type UnitGate interface {
	Met(monster string) bool
	EliteSeen(kind, name string) bool
}

// publicUnits cuts the bot's units down to what the open tiers allow. Locked data never reaches the result: the
// copy is built field by field.
func publicUnits(us []senses.Unit, lock Unlocker, tier3, tier4 bool) []PublicUnit {
	if len(us) == 0 {
		return nil
	}
	gate, _ := lock.(UnitGate)
	out := make([]PublicUnit, 0, len(us))
	for _, u := range us {
		p := PublicUnit{ID: u.ID, Kind: u.Kind, X: u.X, Y: u.Y}
		if u.Kind == senses.KindMonster {
			if tier3 && u.Target && u.Name != "" && (gate == nil || gate.Met(u.Name)) {
				p.Target, p.Name, p.LifePct = true, u.Name, u.LifePct
			}
			if tier4 {
				applyElite(&p, u, gate)
			}
		}
		out = append(out, p)
	}
	return out
}
