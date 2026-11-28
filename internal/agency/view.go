package agency

import (
	"fmt"
	"strings"
)

// View is the plan as the API and the Points page show it: names, one reason each, the warnings and the viability line.
// Nothing in it is a number a player has to read; the numbers are for the log (ShadowLine).
type View struct {
	Schema      string      `json:"schema"` // didle.plan/1
	Enabled     bool        `json:"enabled"`
	Mode        string      `json:"mode"` // off | shadow
	Hero        string      `json:"hero,omitempty"`
	Class       string      `json:"class,omitempty"`
	Level       int         `json:"level,omitempty"`
	Rotation    []ViewSkill `json:"rotation"`
	Aura        *ViewSkill  `json:"aura,omitempty"`
	Archetype   string      `json:"archetype,omitempty"`
	Profile     string      `json:"profile,omitempty"` // what the combat profile would attack with today
	Agrees      *bool       `json:"agrees,omitempty"`  // the plan's attack and aura equal the profile's
	Unplayed    []string    `json:"unplayed,omitempty"`
	Warnings    []ViewWarn  `json:"warnings"`
	Viability   *Viability  `json:"viability,omitempty"`
	Line        string      `json:"line"` // "your build: ...; viability: ..." for the Points page
	Note        string      `json:"note,omitempty"`
	ShadowOnly  bool        `json:"shadowOnly"` // always true in P1: the hero does not play by this plan yet
	Unsupported []string    `json:"unsupported,omitempty"`
}

// ViewSkill is one slot of the rotation.
type ViewSkill struct {
	Slot  string `json:"slot"` // main, second element, aura, buff, summon, movement
	Skill string `json:"skill"`
	Level int    `json:"level"`
	Tier  string `json:"tier"`
	Why   string `json:"why,omitempty"`
}

// ViewWarn is one warning.
type ViewWarn struct {
	Code string `json:"code"`
	Text string `json:"text"`
}

// ViewOf turns a plan into the view. profile is the combat profile's pick ("" when it has none); v may be nil.
func ViewOf(pl Plan, profile string, v *Viability) View {
	out := View{Schema: "didle.plan/1", Enabled: true, Mode: "shadow", ShadowOnly: true, Archetype: pl.Archetype, Profile: profile, Viability: v, Warnings: []ViewWarn{}, Rotation: []ViewSkill{}}
	add := func(slot string, c *Choice) {
		if c != nil {
			out.Rotation = append(out.Rotation, ViewSkill{Slot: slot, Skill: c.Skill, Level: c.Level, Tier: c.Tier, Why: c.Why})
		}
	}
	add("main", pl.Main)
	if pl.Attack != nil && pl.Main != nil && pl.Attack.Skill != pl.Main.Skill {
		add("attack", pl.Attack)
	}
	add("second element", pl.Secondary)
	for i := range pl.Buffs {
		add("buff", &pl.Buffs[i])
	}
	for i := range pl.Summons {
		if pl.Main == nil || pl.Summons[i].Skill != pl.Main.Skill {
			add("summon", &pl.Summons[i])
		}
	}
	for i := range pl.Movement {
		add("movement", &pl.Movement[i])
	}
	if pl.Aura != nil {
		a := ViewSkill{Slot: "aura", Skill: pl.Aura.Skill, Level: pl.Aura.Level, Tier: pl.Aura.Tier, Why: pl.Aura.Why}
		out.Aura = &a
		out.Rotation = append(out.Rotation, a)
	}
	for _, w := range pl.Warnings {
		out.Warnings = append(out.Warnings, ViewWarn{w.Code, w.Text})
	}
	for _, n := range sortedKeys(pl.Unplayed.Skills) {
		out.Unplayed = append(out.Unplayed, fmt.Sprintf("%s %d", n, pl.Unplayed.Skills[n]))
	}
	out.Unsupported = out.Unplayed
	if profile != "" && pl.Attack != nil {
		b := profile == pl.Attack.Skill
		if pl.Aura != nil && pl.PriorAura != "" { // a differing aura pick is a disagreement too (auraflap)
			b = b && pl.Aura.Skill == pl.PriorAura
		}
		out.Agrees = &b
	}
	out.Line = lineOf(out)
	return out
}

// lineOf is "your build: Zeal 4 (supported), Might aura; viability: Normal act 1: safe (...)".
func lineOf(v View) string {
	var parts []string
	for _, r := range v.Rotation {
		switch r.Slot {
		case "main", "second element", "attack":
			parts = append(parts, fmt.Sprintf("%s %d", r.Skill, r.Level))
		case "aura":
			parts = append(parts, r.Skill+" aura")
		}
	}
	rot := "no attack skill"
	if len(parts) > 0 {
		rot = strings.Join(parts, ", ")
	}
	s := "your build: " + rot
	if v.Archetype != "" {
		s += " (" + strings.ReplaceAll(v.Archetype, "_", " ") + ")"
	}
	if len(v.Unplayed) > 0 {
		s += "; points the hero cannot use yet: " + strings.Join(v.Unplayed, ", ")
	}
	if v.Viability != nil {
		s += "; viability: " + v.Viability.Line
	} else {
		s += "; viability: not judged (no zone table or no archetype)"
	}
	return s
}
