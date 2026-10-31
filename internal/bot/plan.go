package bot

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/plandata"
)

// Difficulty names used by the plan files.
var difficulties = []string{"normal", "nightmare", "hell"}

// Step is one progression entry: farm Area while the character level is in [EnterLvl, LeaveLvl].
type Step struct {
	Area     int
	Name     string
	EnterLvl int
	LeaveLvl int
	Purpose  string
	// KitLvl (progression "kit_clvl") is the level below which the step is refused unless the character carries a full
	// kit (PickOptions.FullKit). Found in the Soaky soak 2026-10-29 (V): an empty-belt clvl 2-3 sorceress was sent to Cold
	// Plains whenever Blood Moor and the Den were held back, and chickened out every 2 minutes. 0 = no extra gate.
	KitLvl int
	// MeleeEnter (progression "melee_enter_clvl") is a lower entry level for the strong melee classes (Barbarian,
	// Paladin: PickOptions.Melee); 0 = same as EnterLvl. BUILDV2, SME: Stony Field at 7 for them, 8 for casters (U).
	MeleeEnter int
}

// enter is the entry level of the step for the character kind.
func (s Step) enter(melee bool) int {
	if melee && s.MeleeEnter > 0 && s.MeleeEnter < s.EnterLvl {
		return s.MeleeEnter
	}
	return s.EnterLvl
}

// MeleeTank says whether a class enters the harder early areas earlier (BUILDV2).
func MeleeTank(c d2.Class) bool { return c == d2.Paladin || c == d2.Barbarian }

// AreaInfo is a row of areas.json.
type AreaInfo struct {
	ID   int
	Name string
	Act  int // 1-based
	Town bool
}

// Run is a named boss/area run from runs.json.
type Run struct {
	Key       string
	Name      string
	Act       int
	WPArea    int
	Stops     []int
	FinalArea int
	Kill      []string
	MinLvl    map[string]int // by difficulty name
}

// Plan is the loaded progression data. Build it with LoadPlan; it is read-only afterwards and safe to share.
type Plan struct {
	Areas     map[int]AreaInfo
	Steps     map[string][]Step // by difficulty name, in progression order
	Runs      map[string]Run    // by normalised key
	Waypoints map[int]bool      // area ids with a waypoint
	Towns     map[int]int       // act (1-based) -> town area id
	Warnings  []string          // entries that were skipped or odd
	byName    map[string]int
}

// LoadPlan reads areas.json, runs.json and progression-<difficulty>.json from dir, falling back per file to the
// embedded copies (dir "" uses only the embedded ones). The loader is tolerant: unknown fields are ignored, field
// aliases are accepted and malformed entries are skipped with a warning rather than failing the load.
func LoadPlan(dir string) (*Plan, error) {
	p := &Plan{
		Areas: map[int]AreaInfo{}, Steps: map[string][]Step{}, Runs: map[string]Run{},
		Waypoints: map[int]bool{}, Towns: map[int]int{}, byName: map[string]int{},
	}
	read := func(name string) ([]byte, error) {
		if dir != "" {
			b, err := os.ReadFile(dir + string(os.PathSeparator) + name)
			if err == nil {
				return b, nil
			}
			if !os.IsNotExist(err) {
				return nil, err
			}
		}
		return fs.ReadFile(plandata.FS, name)
	}
	b, err := read("areas.json")
	if err != nil {
		return nil, err
	}
	if err := p.parseAreas(b); err != nil {
		return nil, fmt.Errorf("areas.json: %w", err)
	}
	for _, d := range difficulties {
		b, err := read("progression-" + d + ".json")
		if err != nil {
			p.warn("progression-%s.json missing: %v", d, err)
			continue
		}
		if err := p.parseSteps(d, b); err != nil {
			return nil, fmt.Errorf("progression-%s.json: %w", d, err)
		}
	}
	if b, err := read("runs.json"); err == nil {
		if err := p.parseRuns(b); err != nil {
			return nil, fmt.Errorf("runs.json: %w", err)
		}
	}
	return p, nil
}

func (p *Plan) warn(format string, a ...any) {
	p.Warnings = append(p.Warnings, fmt.Sprintf(format, a...))
}

type rawObj map[string]json.RawMessage

// num returns the first key present as a number or numeric string.
func (o rawObj) num(keys ...string) (int, bool) {
	for _, k := range keys {
		raw, ok := o[k]
		if !ok {
			continue
		}
		var f float64
		if json.Unmarshal(raw, &f) == nil {
			return int(f), true
		}
		var s string
		if json.Unmarshal(raw, &s) == nil {
			if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

func (o rawObj) str(keys ...string) string {
	for _, k := range keys {
		var s string
		if raw, ok := o[k]; ok && json.Unmarshal(raw, &s) == nil {
			return s
		}
	}
	return ""
}

func (o rawObj) boolean(keys ...string) bool {
	for _, k := range keys {
		var v bool
		if raw, ok := o[k]; ok && json.Unmarshal(raw, &v) == nil {
			return v
		}
	}
	return false
}

func (o rawObj) ints(keys ...string) []int {
	for _, k := range keys {
		var v []int
		if raw, ok := o[k]; ok && json.Unmarshal(raw, &v) == nil {
			return v
		}
	}
	return nil
}

func (o rawObj) list(keys ...string) []rawObj {
	for _, k := range keys {
		var v []rawObj
		if raw, ok := o[k]; ok && json.Unmarshal(raw, &v) == nil {
			return v
		}
	}
	return nil
}

func (p *Plan) parseAreas(b []byte) error {
	var top rawObj
	if err := json.Unmarshal(b, &top); err != nil {
		return err
	}
	for _, a := range top.list("areas") {
		id, ok := a.num("id", "area", "area_id")
		if !ok {
			p.warn("areas.json: entry without id")
			continue
		}
		info := AreaInfo{ID: id, Name: a.str("name"), Town: a.boolean("town")}
		info.Act, _ = a.num("act")
		p.Areas[id] = info
		if info.Name != "" {
			p.byName[normName(info.Name)] = id
		}
	}
	var tb map[string]int
	if raw, ok := top["town_by_act"]; ok && json.Unmarshal(raw, &tb) == nil {
		for k, v := range tb {
			if act, err := strconv.Atoi(k); err == nil {
				p.Towns[act] = v
			}
		}
	}
	var wp map[string][]int
	if raw, ok := top["waypoints_by_act"]; ok && json.Unmarshal(raw, &wp) == nil {
		for _, ids := range wp {
			for _, id := range ids {
				p.Waypoints[id] = true
			}
		}
	}
	for id, a := range p.Areas {
		if a.Town {
			if _, ok := p.Towns[a.Act]; !ok {
				p.Towns[a.Act] = id
			}
		}
	}
	return nil
}

func (p *Plan) parseSteps(diff string, b []byte) error {
	var top rawObj
	if err := json.Unmarshal(b, &top); err != nil {
		return err
	}
	alias := map[string][]string{"normal": {"normal", "n"}, "nightmare": {"nightmare", "nm"}, "hell": {"hell", "h"}}
	for _, e := range top.list(alias[diff]...) {
		id, ok := e.num("area", "area_id", "id")
		if !ok {
			p.warn("progression-%s: step without area", diff)
			continue
		}
		s := Step{Area: id, Name: e.str("name"), Purpose: e.str("purpose", "note")}
		s.EnterLvl, _ = e.num("enter_clvl", "min_clvl", "enter")
		s.KitLvl, _ = e.num("kit_clvl")
		s.MeleeEnter, _ = e.num("melee_enter_clvl")
		s.LeaveLvl, ok = e.num("leave_clvl", "target_clvl", "leave")
		if !ok || s.LeaveLvl < s.EnterLvl {
			p.warn("progression-%s: step %d has no usable leave level", diff, id)
			continue
		}
		if s.Name == "" {
			s.Name = p.Areas[id].Name
		}
		p.Steps[diff] = append(p.Steps[diff], s)
	}
	return nil
}

func (p *Plan) parseRuns(b []byte) error {
	var top rawObj
	if err := json.Unmarshal(b, &top); err != nil {
		return err
	}
	for _, r := range top.list("runs") {
		name := r.str("name")
		if name == "" {
			continue
		}
		run := Run{Name: name, Key: normName(name), Stops: r.ints("stops"), Kill: nil, MinLvl: map[string]int{}}
		run.Act, _ = r.num("act")
		run.WPArea, _ = r.num("wp_area")
		run.FinalArea, _ = r.num("final_area")
		if run.FinalArea == 0 && len(run.Stops) > 0 {
			run.FinalArea = run.Stops[len(run.Stops)-1]
		}
		var k []string
		if raw, ok := r["kill"]; ok && json.Unmarshal(raw, &k) == nil {
			run.Kill = k
		}
		if raw, ok := r["min_clvl"]; ok {
			var m map[string]int
			if json.Unmarshal(raw, &m) == nil {
				run.MinLvl["normal"], run.MinLvl["nightmare"], run.MinLvl["hell"] = m["n"], m["nm"], m["h"]
			}
		}
		if run.FinalArea == 0 {
			p.warn("runs.json: %s has no final area", name)
			continue
		}
		p.Runs[run.Key] = run
	}
	return nil
}

func normName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Act of an area (1-based), 0 if unknown.
func (p *Plan) Act(area int) int { return p.Areas[area].Act }

// IsTown reports whether area is a town.
func (p *Plan) IsTown(area int) bool {
	if a, ok := p.Areas[area]; ok {
		return a.Town
	}
	for _, t := range p.Towns {
		if t == area {
			return true
		}
	}
	return false
}

// Resolve turns a farm_area target into an area id: an explicit id, a "run:<key>" (the run's final area), or a name
// from the game tables (cat, may be nil) or the plan's areas.json. A name that is only known by a nonzero goals
// catalog id can be passed as fallbackID.
func (p *Plan) Resolve(name string, id int, cat Catalog, fallback func(string) int) (int, error) {
	if id > 0 {
		return id, nil
	}
	name = strings.TrimSpace(name)
	if key, ok := strings.CutPrefix(strings.ToLower(name), "run:"); ok {
		if r, ok := p.Runs[normName(key)]; ok {
			return r.FinalArea, nil
		}
		return 0, fmt.Errorf("unknown run %q", name)
	}
	if cat != nil {
		if a, ok := cat.AreaByName(name); ok {
			return a, nil
		}
	}
	if a, ok := p.byName[normName(name)]; ok {
		return a, nil
	}
	if fallback != nil {
		if a := fallback(name); a > 0 {
			return a, nil
		}
	}
	return 0, fmt.Errorf("unknown area %q", name)
}

// PickOptions tunes the planner.
type PickOptions struct {
	Margin  int          // Tuning.Bot.LevelMargin
	Slack   int          // Tuning.Bot.LeaveSlack
	Acts    map[int]bool // acts the bot can reach (1-based); nil = all
	Exclude map[int]bool // areas to skip (cleared in this game, too dangerous)
	Cleared map[int]bool // with Floor: only these count as "excluded window" (nil: all of Exclude)
	// Floor > 0 refuses the "most advanced eligible step" fallback when that step's leave level is more than Floor
	// levels below the character while an excluded step still covers the level: farming monsters that far below the
	// character gives next to no experience (LIVE: Den of Evil at clvl 12 yielded 8 XP in 12 minutes), so Pick reports
	// nothing eligible and the caller starts a fresh game instead.
	Floor int
	// FullKit says the character has a full belt and a town portal: steps with a KitLvl above the level are allowed.
	FullKit bool
	// Melee: the character is a Barbarian or Paladin (steps with a MeleeEnter level open earlier).
	Melee bool
}

// Pick chooses the area to level in for difficulty diff at character level. Rule: among steps (in progression order)
// in reachable acts, not towns, not excluded, with enter+margin <= level, take the last one whose leave+slack >= level
// (the most advanced area that is still worth it); if the character is past every window, the last eligible step; if
// below every entry level, the first eligible step. ok is false when nothing is eligible.
func (p *Plan) Pick(diff string, level int, o PickOptions) (Step, bool) {
	var eligible []Step
	for _, s := range p.Steps[diff] {
		if p.IsTown(s.Area) || o.Exclude[s.Area] {
			continue
		}
		if s.KitLvl > level && !o.FullKit {
			continue
		}
		if o.Acts != nil && !o.Acts[p.Act(s.Area)] {
			continue
		}
		eligible = append(eligible, s)
	}
	if len(eligible) == 0 {
		return Step{}, false
	}
	var best *Step
	var last *Step
	for i := range eligible {
		s := &eligible[i]
		if s.enter(o.Melee)+o.Margin > level {
			continue
		}
		last = s
		if level <= s.LeaveLvl+o.Slack {
			best = s
		}
	}
	switch {
	case best != nil:
		return *best, true
	case last != nil:
		if o.Floor > 0 && level-last.LeaveLvl > o.Floor && p.windowExcluded(diff, level, o) {
			return Step{}, false
		}
		return *last, true
	}
	// Below every entry level: the first step is still better than nothing, but not when it is far above the character
	// and an excluded (cleared) step covers the level: a clvl 3 Sorceress with no potions was sent to Burial Grounds
	// (enter 6) after Blood Moor was cleared (live 2026-10-29, Soaky). The caller starts a fresh game instead.
	if o.Floor > 0 && eligible[0].enter(o.Melee)-level >= earlyGap && p.windowExcluded(diff, level, o) {
		return Step{}, false
	}
	return eligible[0], true
}

// earlyGap is how many levels below its entry level a step is refused when a cleared step still covers the character.
const earlyGap = 2

// windowExcluded reports whether an excluded, reachable-act step still has the level inside its window.
func (p *Plan) windowExcluded(diff string, level int, o PickOptions) bool {
	excl := o.Exclude
	if o.Cleared != nil {
		excl = o.Cleared // an area that is only avoided for a while or held back (Den rule) is not a cleared one
	}
	for _, s := range p.Steps[diff] {
		if !excl[s.Area] || p.IsTown(s.Area) || o.Acts != nil && !o.Acts[p.Act(s.Area)] {
			continue
		}
		if s.enter(o.Melee)+o.Margin <= level && level <= s.LeaveLvl+o.Slack {
			return true
		}
	}
	return false
}

// DifficultyFor suggests the difficulty for a character level: the highest difficulty whose first step the character
// qualifies for (with margin) and whose resist gate the raw resists res pass (nil: not checked). Switching difficulty needs a new game, so callers treat it as a wish.
func (p *Plan) DifficultyFor(level, margin int, res *d2.Resists) string {
	best := "normal"
	for _, d := range difficulties {
		if st := p.Steps[d]; len(st) > 0 && level >= st[0].EnterLvl+margin && p.EntryAllowed(d, level, st[0].Area, res).OK {
			best = d
		}
	}
	return best
}

// StepAreas lists the ids of all steps of a difficulty (for tests and docs).
func (p *Plan) StepAreas(diff string) []int {
	var out []int
	for _, s := range p.Steps[diff] {
		out = append(out, s.Area)
	}
	sort.Ints(out)
	return out
}

// EntryAllowed is combat.EntryGate for a plan area (docs/difficulty.md 2.2): Nightmare and Hell are refused below the
// recommended level of the area's act. The area monster level is not known here (0), so only the act floor and the
// Hell act 1 dungeon floor apply. Unknown areas and Normal pass. res are the raw resists (leveler.RawResists) for the
// resist gate of an ENTRY into diff (leveler.ResistGate); nil skips it. The planner passes nil for the difficulty the
// character already stands in: refusing every area there would only pause it, it keeps farming instead.
func (p *Plan) EntryAllowed(diff string, level, area int, res *d2.Resists) combat.EntryDecision {
	a, ok := p.Areas[area]
	if !ok || a.Act < 1 {
		return combat.EntryDecision{OK: true}
	}
	// Act 1 ids 2-7 and 17 (Burial Grounds) are outdoors; the rest are caves, crypts and towers (S: levels.tsv ids).
	dungeon := a.Act == 1 && area > 7 && area != 17
	d := combat.EntryGate(d2.Difficulty(rank(diff)), a.Act, level, 0, dungeon)
	if d.OK && res != nil {
		if g := leveler.ResistGate(d2.Difficulty(rank(diff)), *res); !g.OK {
			return combat.EntryDecision{Why: g.Why}
		}
	}
	return d
}
