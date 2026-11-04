package bot

import (
	"fmt"

	"d2lod-bot/internal/d2"
)

// Life log fields (docs/lifelog.md): logging only, nothing here feeds a decision.

// mercWord is alive, dead or none for the zone entry line.
func mercWord(s *d2.Snapshot) string {
	mc := d2.FindMerc(s)
	switch {
	case !mc.Present:
		return "none"
	case mc.Alive:
		return "alive"
	}
	return "dead"
}

// vitalsKV is the life, max life and potion counts added to the retreat and death lines.
func (r *Runner) vitalsKV(s *d2.Snapshot, p *d2.Player) []any {
	k := r.kitOf(s)
	return []any{"life", p.Life, "maxLife", p.MaxLife, "heal", k.BeltHP + k.PackHP, "beltMP", k.BeltMP, "rejuv", k.BeltRV, "tome", k.TownPortals}
}

// zoneEntry writes the one line per area change: `zone entry area=<id> clvl=<n> life=<n>/<n> heal=<n> mana=<n>/<n>
// tome=<n> merc=<alive|dead|none>`. heal is belt plus pack healing potions, tome the town portals in the kit.
func (r *Runner) zoneEntry(s *d2.Snapshot, p *d2.Player) {
	k := r.kitOf(s)
	r.e.Log.Info("zone entry", "area", p.Area, "clvl", p.Level, "life", fmt.Sprintf("%d/%d", p.Life, p.MaxLife),
		"heal", k.BeltHP+k.PackHP, "mana", fmt.Sprintf("%d/%d", p.Mana, p.MaxMana), "tome", k.TownPortals, "merc", mercWord(s))
}
