// Package leveler spends stat and skill points per the plan data, evaluates and equips gear, and holds the
// point and gear rules the bot's loop calls. Everything that decides is pure and tested offline; everything that
// clicks goes through env.Env.In and proves its effect by re-reading memory (docs/leveler.md).
package leveler

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/plandata"
)

// Attribute indexes in the order the character panel lists them.
const (
	Str = iota
	Dex
	Vit
	Ene
	numAttr
)

var attrNames = [numAttr]string{"strength", "dexterity", "vitality", "energy"}

// Row is one level of a build table: points spent when reaching clvl.
type Row struct {
	Clvl   int
	Attr   [numAttr]int
	Skills []SkillPts
}

// SkillPts is "Points" points into the skill named as in skills.txt.
type SkillPts struct {
	Skill  string
	Points int
}

// Build is a per-level allocation table (build-*.json).
type Build struct {
	Name string
	Rows []Row
	// Respec is the data's respec trigger level (0 = none). Not implemented: docs/leveler.md "Respec".
	Respec int
	// RespecPool is the data's post-respec allocation (documentation for a future respec flow; the planner ignores it).
	RespecPool []SkillPts
	// Class is the class this table is for ("sorceress" when the file says nothing, which keeps the old files valid).
	Class string
	// Hotkeys is the build's hotkey plan for the combat agent; the leveler never presses it.
	Hotkeys []Hotkey
}

// Hotkey is one entry of a build's hotkey plan: bind Skill to Key on Hand ("left" or "right") from clvl FromClvl
// (until UntilClvl when set).
type Hotkey struct {
	Key       string `json:"key"`
	Skill     string `json:"skill"`
	Hand      string `json:"hand"`
	FromClvl  int    `json:"from_clvl,omitempty"`
	UntilClvl int    `json:"until_clvl,omitempty"`
}

type rawBuild struct {
	Build   string   `json:"build"`
	Class   string   `json:"class"`
	Hotkeys []Hotkey `json:"hotkeys"`
	Rows    []struct {
		Clvl      int `json:"clvl"`
		Strength  int `json:"strength"`
		Dexterity int `json:"dexterity"`
		Vitality  int `json:"vitality"`
		Energy    int `json:"energy"`
		Skills    []struct {
			Skill  string `json:"skill"`
			Points int    `json:"points"`
		} `json:"skills"`
	} `json:"rows"`
	Respec *struct {
		AtClvl int `json:"at_clvl"`
		Pool   []struct {
			Skill  string `json:"skill"`
			Points int    `json:"points"`
		} `json:"respend_pool"`
	} `json:"respec"`
}

// ParseBuild decodes a build-*.json file.
func ParseBuild(b []byte) (*Build, error) {
	var rb rawBuild
	if err := json.Unmarshal(b, &rb); err != nil {
		return nil, err
	}
	out := &Build{Name: rb.Build, Class: strings.ToLower(rb.Class), Hotkeys: rb.Hotkeys}
	if out.Class == "" {
		out.Class = "sorceress"
	}
	if rb.Respec != nil {
		out.Respec = rb.Respec.AtClvl
		for _, s := range rb.Respec.Pool {
			out.RespecPool = append(out.RespecPool, SkillPts{s.Skill, s.Points})
		}
	}
	last := 0
	for _, r := range rb.Rows {
		if r.Clvl <= last {
			return nil, fmt.Errorf("build %s: rows not ascending at clvl %d", rb.Build, r.Clvl)
		}
		last = r.Clvl
		row := Row{Clvl: r.Clvl, Attr: [numAttr]int{r.Strength, r.Dexterity, r.Vitality, r.Energy}}
		for _, s := range r.Skills {
			if s.Points > 0 {
				row.Skills = append(row.Skills, SkillPts{s.Skill, s.Points})
			}
		}
		out.Rows = append(out.Rows, row)
	}
	if len(out.Rows) == 0 {
		return nil, fmt.Errorf("build %s: no rows", rb.Build)
	}
	return out, nil
}

// Build files by selector name.
var buildFiles = map[string]string{
	"primary":  "build-primary-blizzard.json",
	"fallback": "build-fallback-lightning.json",
	// BUILDV2: tables for fresh characters (SelectBuild); the names above stay what existing characters run.
	FallbackV2: "build-fallback-lightning-v2.json",
	PaladinV2:  "build-paladin-normal30-v2.json",
	// Paladin, Normal to clvl 30 (docs/plan/data/build-paladin-normal30.json; sources and tags inside the file).
	"paladin-normal30": "build-paladin-normal30.json",
	// GEARWIRE: Necromancer and Amazon tables from internal/gear (same files, copied to plandata and docs/plan/data).
	"necromancer_leveling_normal30": "build-necromancer_leveling_normal30.json",
	"necromancer_summoner_normal30": "build-necromancer_summoner_normal30.json",
	"amazon_bow_normal30":           "build-amazon_bow_normal30.json",
	"amazon_javelin_normal30":       "build-amazon_javelin_normal30.json",
	"druid_fire_normal30":           "build-druid_fire_normal30.json",
	"assassin_fireblast_normal30":   "build-assassin_fireblast_normal30.json",
	"barbarian_melee_normal30":      "build-barbarian_melee_normal30.json",
	// ENDBUILD: clvl 2..90 tables (docs/builds-endgame.md), opt-in by name or by the "endgame" alias; never a default.
	"sorceress_blizzard_endgame90":     "build-sorceress_blizzard_endgame90.json",
	"paladin_hammerdin_endgame90":      "build-paladin_hammerdin_endgame90.json",
	"paladin_hammerdin_zeal_endgame90": "build-paladin_hammerdin_zeal_endgame90.json",
	"necromancer_summoner_endgame90":   "build-necromancer_summoner_endgame90.json",
	"amazon_multishot_endgame90":       "build-amazon_multishot_endgame90.json",
	"druid_tornado_endgame90":          "build-druid_tornado_endgame90.json",
	"assassin_sentry_endgame90":        "build-assassin_sentry_endgame90.json",
	"barbarian_warcry_endgame90":       "build-barbarian_warcry_endgame90.json",
}

// EndgameAlias is the build selector ("--build endgame", config levelerBuild) that means "the endgame table of the
// character's class".
const EndgameAlias = "endgame"

// endgameBuild is the clvl 2..90 table per class. The Sorceress table (Blizzard, Ice Bolt line from clvl 2) has no
// respec in its data (V: respec.enabled false in the file), so it is a safe choice for a new character; the Lightning
// "fallback" table stops at clvl 30 and cannot be continued, so it is not the endgame choice. A mid-game switch of an
// existing character is not supported (the lines differ, docs/gear.md "Endgame tables").
var endgameBuild = map[string]string{
	"sorceress":   "sorceress_blizzard_endgame90",
	"paladin":     "paladin_hammerdin_endgame90",
	"necromancer": "necromancer_summoner_endgame90",
	"amazon":      "amazon_multishot_endgame90",
	"druid":       "druid_tornado_endgame90",
	"assassin":    "assassin_sentry_endgame90",
	"barbarian":   "barbarian_warcry_endgame90",
}

// EndgameBuildName returns the endgame table of a class (case-insensitive), "" for an unknown class.
func EndgameBuildName(class string) string { return endgameBuild[strings.ToLower(class)] }

// ResolveBuild turns a configured selector into a table name: "" stays "" (the caller then uses DefaultBuildName, so
// the default selection is unchanged), "endgame" becomes the class's endgame table, anything else passes through.
func ResolveBuild(name, class string) string {
	if name == EndgameAlias {
		return EndgameBuildName(class)
	}
	return name
}

// BUILDV2 table names.
const (
	FallbackV2 = "fallback-lightning-v2"
	PaladinV2  = "paladin-normal30-v2"
)

// v2Rows is the last clvl whose rows the BUILDV2 tables changed.
const v2Rows = 15

// freshBuild is the BUILDV2 table of a class that has one; a fresh character gets it, an existing one keeps its table.
var freshBuild = map[string]string{"sorceress": FallbackV2, "paladin": PaladinV2}

// OwnedSkills maps the skills the player owns to their levels by skills.txt name (nil without tables).
func OwnedSkills(db *data.DB, p *d2.Player) map[string]int {
	m := map[string]int{}
	if db == nil || p == nil {
		return m
	}
	for _, s := range p.Skills {
		if sk := db.SkillByID(s.ID); sk != nil {
			m[sk.Key] = s.Level
		}
	}
	return m
}

// SelectBuild is the table of a class when the config names none: the BUILDV2 table for a fresh character (level 1, or
// every owned skill also in the new table), else the class default (DefaultBuildName). A character that already owns a
// skill the new table dropped (Soaky: Static Field, Paladin: Holy Fire or Resist Fire) keeps its old table, so a table
// switch never spends points differently from what the character has already bought. The check uses the embedded tables.
func SelectBuild(class string, level int, owned map[string]int) string {
	class = strings.ToLower(class)
	legacy := DefaultBuildName(class)
	v2 := freshBuild[class]
	if v2 == "" || legacy == "" {
		return legacy
	}
	if level <= 1 {
		return v2
	}
	lb, err1 := LoadBuild(legacy, "")
	nb, err2 := LoadBuild(v2, "")
	if err1 != nil || err2 != nil {
		return legacy
	}
	// Only the rows up to clvl v2Rows differ (rows after that are copies of the old table).
	inNew := map[string]bool{}
	for _, r := range nb.Rows {
		if r.Clvl > v2Rows {
			break
		}
		for _, s := range r.Skills {
			inNew[s.Skill] = true
		}
	}
	for _, r := range lb.Rows {
		if r.Clvl > v2Rows {
			break
		}
		for _, s := range r.Skills {
			if owned[s.Skill] > 0 && !inNew[s.Skill] {
				return legacy
			}
		}
	}
	return v2
}

// defaultBuild is the build a class gets when the config names none. Sorceress stays on "fallback" (unchanged behaviour).
var defaultBuild = map[string]string{
	"sorceress": "fallback",
	"paladin":   "paladin-normal30",
}

// DefaultBuildName returns the default build selector for a class name (case-insensitive); unknown classes get ""
// so the caller fails loudly instead of guessing.
func DefaultBuildName(class string) string {
	class = strings.ToLower(class)
	if n := defaultBuild[class]; n != "" {
		return n
	}
	return gear.DefaultBuild(class)
}

// LoadBuild loads "primary" (Blizzard) or "fallback" (Lightning) from dir, else from the embedded copies.
func LoadBuild(name, dir string) (*Build, error) {
	f, ok := buildFiles[name]
	if !ok {
		return nil, fmt.Errorf("leveler: unknown build %q (primary, fallback, paladin-normal30, or a name from gear.Builds)", name)
	}
	var b []byte
	var err error
	if dir != "" {
		if b, err = os.ReadFile(filepath.Join(dir, f)); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	if b == nil {
		if b, err = fs.ReadFile(plandata.FS, f); err != nil {
			return nil, err
		}
	}
	bd, err := ParseBuild(b)
	if err != nil {
		return nil, err
	}
	if name == "fallback" && bd.Class == "sorceress" {
		withEnergy(bd, sorcEnergyPerRow)
	}
	return bd, nil
}

// sorcEnergyPerRow: the Lightning table had 0 energy in every row, so a clvl 12 Sorceress had 60 mana and rested 30-40 s
// per pack (V, docs/leveler.md "Why Booti stayed at clvl 12"). Two of the five points of every vitality row go to
// energy (U whether 2 is the best split; 2 energy = +4 mana per level). The data file lives in plandata (not ours),
// so the shift is applied at load time.
const sorcEnergyPerRow = 2

// withEnergy moves n points per row from vitality to energy in rows that put more than n into vitality, so the
// strength rows and the total points per level stay as the table has them.
func withEnergy(b *Build, n int) {
	for i := range b.Rows {
		r := &b.Rows[i]
		if r.Attr[Vit] > n {
			r.Attr[Vit] -= n
			r.Attr[Ene] += n
		}
	}
}
