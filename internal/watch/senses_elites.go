package watch

import "d2lod-bot/internal/senses"

// applyElite adds the tier 4 marker to p: the elite kind, the unit's own name (a superunique) and its mod names. It is
// the spoiler rule of docs/senses-units.md: with a UnitGate the marker appears only when the Codex already holds the
// first of that kind (a champion pack, or the superunique's own entry); a kind not met yet stays the plain dot of tier 2.
// Without a gate (FixedTier) every elite is marked. The mod names come ready from the feed (decide.ModName), so an id
// the table lacks reads "unknown mod #N" and never blanks the marker.
func applyElite(p *PublicUnit, u senses.Unit, gate UnitGate) {
	if u.Elite == "" {
		return
	}
	name := u.Title
	if name == "" {
		name = u.Name
	}
	if gate != nil && !gate.EliteSeen(u.Elite, name) {
		return
	}
	p.Elite = u.Elite
	p.Mods = append([]string(nil), u.Mods...)
	if p.Name == "" && u.Title != "" {
		p.Name = u.Title
	}
}
