package hero

import (
	"encoding/json"
	"fmt"
	"strings"

	"d2lod-bot/internal/leveler"
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
	// Gold is the purse; Potions the healing, mana and rejuvenation potions in belt and pack.
	Gold, Potions *int
}

// Floors are minimums a run declares beside the hero (0: not checked): the purse and the potions its setup gave.
type Floors struct {
	Purse   int `json:"purse,omitempty"`
	Potions int `json:"potions,omitempty"`
}

// Check is one row of the self-check.
type Check struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
	Want string `json:"want"`
	Got  string `json:"got"`
}

// SelfCheck compares the hero with the game (design section 6.2: level, points, attack, weapon; purse and potions in
// SelfCheckFloors; goal and order are static, in d2scenario validate; life is not checked yet). A fact that was not read fails its row: a check
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

// SelfCheckFloors is SelfCheck plus a row for each declared floor: a setup verb that answered ok and gave nothing
// (an empty purse, no potions) is caught before the run, not hours into it.
func (h Hero) SelfCheckFloors(o Observed, f Floors) []Check {
	out := h.SelfCheck(o)
	atLeast := func(name string, want int, got *int) {
		if want <= 0 {
			return
		}
		c := Check{Name: name, Want: fmt.Sprintf("at least %d", want), Got: "not read"}
		if got != nil {
			c.Got, c.OK = fmt.Sprint(*got), *got >= want
		}
		out = append(out, c)
	}
	atLeast("purse", f.Purse, o.Gold)
	atLeast("potions", f.Potions, o.Potions)
	return out
}

// Static is the half of the self-check that needs no game (design 6.2, T0): the declaration resolves (a known class
// and build of that class, a level the tables cover) and, for the played profile, the build's main attack gets its
// point at that level with the declared quest points. t may be nil for a banked hero.
func Static(d Decl, t Tables) []Check {
	bd := d
	bd.Profile, bd.Weapon = Banked, "none"
	c := Check{Name: "build", Want: fmt.Sprintf("%s build %q resolves at clvl %d", d.Class, d.Build, d.Level), Got: "ok"}
	if _, err := Resolve(bd, nil); err != nil {
		c.Got = err.Error()
		return []Check{c}
	}
	c.OK = true
	out := []Check{c}
	if d.Profile == Banked {
		return out
	}
	a := Check{Name: "attack", Want: fmt.Sprintf("main attack learned at clvl %d with %d quest skill points", d.Level, d.QuestSkillPts)}
	if h, err := Resolve(d, t); err != nil {
		a.Got = err.Error()
	} else {
		a.Got, a.OK = h.Attack, true
	}
	return append(out, a)
}

// NameTables are Tables without the game files, for the static check: every skill name resolves (ids by first use),
// no learn rule (the plan table's own order stands in for required levels, so a pass here can still fail live; U),
// no starting weapon.
func NameTables() Tables { return &nameTables{ids: map[string]int{"Attack": 0}} }

type nameTables struct{ ids map[string]int }

func (n *nameTables) SkillID(name string) (int, bool) {
	id, ok := n.ids[name]
	if !ok {
		id = len(n.ids)
		n.ids[name] = id
	}
	return id, true
}
func (n *nameTables) SkillIndex(string, int) (int, bool) { return 0, true }
func (n *nameTables) Rule(string) leveler.Rule           { return nil }
func (n *nameTables) StarterWeapon(string) string        { return "" }

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
