package combat

import (
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
)

// Press rows (card dlv-press; docs/design/fable-review-2026-11-21/combat-engine-b-grammar.md, use.press). A repeat or
// channel skill (Zeal, Strafe, Fend, Arctic Blast) is held for one slice per fight tick; the fight loop (potions,
// evasion, the surround rule) runs between slices, so survival between slices is the tick itself. A profile's press row
// names a skill and gives its slice (SliceMs) and, for a skill that cannot be interrupted once started (Commit, a
// spin), the life percent under which the press is not started (CommitLifePct): the fight picks the next slot instead.
// Rows are data; Go names no skill.
//
// Release and timeout: the cast verb carries the slice as its press time, so the release is explicit and no hold can
// outlast pressSliceMax; the cast expectation (expect.go) settles on its own deadline.
//
// Knob combat.deliveryUpkeep (the delivery driver knob): 0 the profile-wide Melee.HoldMs slice and no commit gate as
// before, 1 shadow (the old slice is used, a differing row slice or a refused start is counted and logged), 2 the row
// slice is used and the commit gate refuses a start. No rate claim before fidelity A1 (attack-sequence rate).

// PressRule is one press row of a profile.
type PressRule struct {
	Name          string `json:"name"`
	SliceMs       int    `json:"sliceMs"`
	Commit        bool   `json:"commit,omitempty"`
	CommitLifePct int    `json:"commitLifePct,omitempty"`
	Source        string `json:"source"` // V, S or U first: where SliceMs and CommitLifePct come from
}

const (
	pressSliceMin = 100 * time.Millisecond
	pressSliceMax = 3 * time.Second
)

// validatePresses checks the press rows at load time against the skill table: an error names the profile and the row.
func validatePresses(ps []Profile, skills map[string]skillMeta) error {
	for i := range ps {
		p := &ps[i]
		seen := map[string]bool{}
		for _, r := range p.Presses {
			m, ok := skills[r.Name]
			d := time.Duration(r.SliceMs) * time.Millisecond
			switch {
			case !ok:
				return fmt.Errorf("profile %q: press row %q names no skill of the table", p.ProfileID(), r.Name)
			case !m.Hold:
				return fmt.Errorf("profile %q: press row %q on a skill that is not held", p.ProfileID(), r.Name)
			case seen[r.Name]:
				return fmt.Errorf("profile %q: press row %q twice", p.ProfileID(), r.Name)
			case d < pressSliceMin || d > pressSliceMax:
				return fmt.Errorf("profile %q: press row %q: sliceMs %d outside %d..%d", p.ProfileID(), r.Name, r.SliceMs, pressSliceMin.Milliseconds(), pressSliceMax.Milliseconds())
			case r.Commit != (r.CommitLifePct > 0) || r.CommitLifePct > 100:
				return fmt.Errorf("profile %q: press row %q: commitLifePct 1..100 goes with commit and only with it", p.ProfileID(), r.Name)
			case !validSource(r.Source):
				return fmt.Errorf("profile %q: press row %q needs a source tag (V, S or U)", p.ProfileID(), r.Name)
			}
			seen[r.Name] = true
		}
	}
	return nil
}

func (p *Profile) pressRow(name string) *PressRule {
	for i := range p.Presses {
		if p.Presses[i].Name == name {
			return &p.Presses[i]
		}
	}
	return nil
}

// pressSlice is how long the held skill sp keeps the button down this tick.
func (f *Fighter) pressSlice(p *d2.Player, sp SkillSpec) time.Duration {
	old := f.holdFor(p)
	r := f.profileFor(p).pressRow(sp.Name)
	if r == nil || f.upkeepMode() == 0 {
		return old
	}
	d := time.Duration(r.SliceMs) * time.Millisecond
	if f.upkeepMode() == 1 {
		if d != old && f.press.noteDiff(sp.Name, "slice") {
			f.info("press shadow", "skill", sp.Name, "slice", d, "old", old)
		}
		return old
	}
	return d
}

// pressMayStart is the commit gate: false when the skill is a commit press and life is under its row's percent.
func (f *Fighter) pressMayStart(p *d2.Player, sp SkillSpec) bool {
	if p == nil || f.upkeepMode() == 0 {
		return true
	}
	r := f.profileFor(p).pressRow(sp.Name)
	if r == nil || !r.Commit || p.MaxLife <= 0 || p.Life*100 >= p.MaxLife*r.CommitLifePct {
		return true
	}
	if f.upkeepMode() == 1 {
		if f.press.noteDiff(sp.Name, "commit") {
			f.info("press shadow", "skill", sp.Name, "refused", "commit gate", "life", p.Life, "of", p.MaxLife)
		}
		return true
	}
	f.press.refused++
	return false
}

// pressState counts what the shadow and the gate did.
type pressState struct {
	diffs, refused int
	seen           map[string]bool
}

// noteDiff counts a shadow difference and reports whether it is the first of its (skill, kind).
func (s *pressState) noteDiff(skill, kind string) bool {
	s.diffs++
	k := skill + "|" + kind
	if s.seen[k] {
		return false
	}
	if s.seen == nil {
		s.seen = map[string]bool{}
	}
	s.seen[k] = true
	return true
}

// PressShadowDiffs is the number of shadow differences (combat.deliveryUpkeep 1); PressRefusals the commit starts refused.
func (f *Fighter) PressShadowDiffs() int { return f.press.diffs }
func (f *Fighter) PressRefusals() int    { return f.press.refused }
