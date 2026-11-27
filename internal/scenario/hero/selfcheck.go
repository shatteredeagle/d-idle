package hero

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Observed is what the prep read from the game after the verbs; nil or absent means not read.
type Observed struct {
	Level    *int
	StatPts  *int
	SkillPts *int
	// SkillLevels is the dump's skills.base_levels (hard points in class order, the order `cmd skillpt` indexes).
	SkillLevels []int
	// LeftSkill and RightSkill are the skill ids on the two hands (the state verb's lskill and rskill).
	LeftSkill, RightSkill *int
	// Weapon is the code of the right-hand weapon, "" for none; nil when not read.
	Weapon *string
}

// Check is one row of the self-check.
type Check struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Want string `json:"want"`
	Got  string `json:"got"`
}

// SelfCheck compares the hero with the game (the minimal half of design section 6.2: level, points, attack, weapon;
// life, potions, purse, goal and order belong to card pgselfcheck). A fact that was not read fails its row: a check
// that cannot see is no pass.
func (h Hero) SelfCheck(o Observed) []Check {
	var out []Check
	eq := func(name string, want int, got *int) {
		c := Check{Name: name, Want: fmt.Sprint(want), Got: "not read"}
		if got != nil {
			c.Got, c.OK = fmt.Sprint(*got), *got == want
		}
		out = append(out, c)
	}
	eq("level", h.Level, o.Level)
	eq("statpts", h.StatPts, o.StatPts)
	eq("skillpts", h.SkillPts, o.SkillPts)
	if h.Profile == Played {
		c := Check{Name: "attack", Want: fmt.Sprintf("%s (id %d) learned and on a hand", h.Attack, h.AttackID)}
		learned := h.AttackIndex < 0 || (h.AttackIndex < len(o.SkillLevels) && o.SkillLevels[h.AttackIndex] > 0)
		bound := (o.LeftSkill != nil && *o.LeftSkill == h.AttackID) || (o.RightSkill != nil && *o.RightSkill == h.AttackID)
		c.OK = learned && bound
		c.Got = fmt.Sprintf("learned %v, hands %s %s", learned, show(o.LeftSkill), show(o.RightSkill))
		out = append(out, c)
	}
	if h.Weapon != "" {
		c := Check{Name: "weapon", Want: h.Weapon + " in the right hand", Got: "not read"}
		if o.Weapon != nil {
			c.Got, c.OK = *o.Weapon, *o.Weapon == h.Weapon
		}
		out = append(out, c)
	}
	return out
}

// Failed names the failed rows ("" when all pass), the form of `selfcheck: <name>`.
func Failed(cs []Check) string {
	var bad []string
	for _, c := range cs {
		if !c.OK {
			bad = append(bad, c.Name)
		}
	}
	return strings.Join(bad, ",")
}

func show(p *int) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprint(*p)
}

// ReadPointDump fills the points of o from one line of `dump player,stats,skills` (V cmd/d2scenario: player.stat_pts,
// player.skill_pts, skills.base_levels).
func ReadPointDump(o *Observed, line string) error {
	var d struct {
		Player struct {
			Level    *int `json:"level"`
			StatPts  *int `json:"stat_pts"`
			SkillPts *int `json:"skill_pts"`
		} `json:"player"`
		Skills struct {
			Levels []int `json:"base_levels"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(line), &d); err != nil {
		return fmt.Errorf("hero: point dump: %w", err)
	}
	if d.Player.Level != nil {
		o.Level = d.Player.Level
	}
	o.StatPts, o.SkillPts, o.SkillLevels = d.Player.StatPts, d.Player.SkillPts, d.Skills.Levels
	return nil
}
