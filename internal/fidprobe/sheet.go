// Package fidprobe compares two d2.Snapshot-derived sheets of the SAME save, one read from game.exe and
// one from d2d, field by field (docs/research/fidelity-probe-design.md). It is pure: no process, socket
// or file access, so the engine is tested on fakes. Tolerance rules, known deviations and the scenario
// catalogue are data (rules.json, known.json, scenarios.json), not code.
package fidprobe

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
)

// Kind of a field value.
const (
	KindInt   = "int"
	KindFloat = "float"
	KindSet   = "set"
)

// Field is one comparable quantity. Why != "" means the side cannot read it (the value is then void).
type Field struct {
	Kind string   `json:"kind,omitempty"`
	I    int64    `json:"i,omitempty"`
	F    float64  `json:"f,omitempty"`
	Set  []string `json:"set,omitempty"` // sorted, unique
	Why  string   `json:"why,omitempty"`
}

// Readable reports whether the field carries a value.
func (f Field) Readable() bool { return f.Why == "" && f.Kind != "" }

// String is the value for the report.
func (f Field) String() string {
	switch {
	case !f.Readable():
		return "?"
	case f.Kind == KindInt:
		return fmt.Sprint(f.I)
	case f.Kind == KindFloat:
		return fmt.Sprintf("%g", f.F)
	}
	return fmt.Sprintf("{%d members}", len(f.Set))
}

// Sheet is the normalised quantity set of one side. Keys are "group.name"; the group is the part before
// the first dot and is what a scenario lists as comparable.
type Sheet struct {
	Source string           `json:"source"`
	Fields map[string]Field `json:"fields"`
}

// NewSheet returns an empty sheet.
func NewSheet(source string) *Sheet { return &Sheet{Source: source, Fields: map[string]Field{}} }

// Int, Float, Set and Unreadable add a field.
func (s *Sheet) Int(key string, v int64)     { s.Fields[key] = Field{Kind: KindInt, I: v} }
func (s *Sheet) Float(key string, v float64) { s.Fields[key] = Field{Kind: KindFloat, F: v} }
func (s *Sheet) Unreadable(key, why string)  { s.Fields[key] = Field{Why: why} }
func (s *Sheet) Set(key string, members []string) {
	m := append([]string(nil), members...)
	sort.Strings(m)
	out := m[:0]
	for i, x := range m {
		if i == 0 || x != m[i-1] {
			out = append(out, x)
		}
	}
	s.Fields[key] = Field{Kind: KindSet, Set: out}
}

// Group returns the group of a key.
func Group(key string) string {
	if i := strings.IndexByte(key, '.'); i >= 0 {
		return key[:i]
	}
	return key
}

// Keys returns the sorted field keys.
func (s *Sheet) Keys() []string {
	k := make([]string, 0, len(s.Fields))
	for x := range s.Fields {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

// Input is everything one side produced for a sheet. Progress is nil when quest words and waypoints
// could not be read. Unreadable maps a producer-side field name (d2dsrc.Report.Unknown) to its reason.
type Input struct {
	Source      string
	Snap        *d2.Snapshot
	Progress    *d2.PlayerProgress
	ProgressWhy string
	Unreadable  map[string]string
}

// FromInput flattens a snapshot to a Sheet. Only deterministic quantities enter (design section 3):
// unit ids, addresses, cursor, mouse and screen are never compared; names and strings are not compared.
func FromInput(in Input) (*Sheet, error) {
	s := NewSheet(in.Source)
	if in.Snap == nil || in.Snap.Player == nil || in.Snap.Inventory == nil {
		return nil, fmt.Errorf("fidprobe: %s: snapshot has no player (state %v): not in game", in.Source, stateOf(in.Snap))
	}
	p := in.Snap.Player
	i := func(k string, v int) { s.Int(k, int64(v)) }
	i("base.class", int(p.Class))
	i("base.difficulty", int(p.Difficulty))
	i("base.act", p.Act)
	i("base.area", p.Area)
	i("base.level", p.Level)
	i("base.exp", int(p.Exp))
	i("base.gold", p.Gold)
	i("base.stash_gold", p.StashGold)
	i("base.str", p.Strength)
	i("base.dex", p.Dexterity)
	i("base.vit", p.Vitality)
	i("base.ene", p.Energy)
	i("base.stat_points", p.StatPoints)
	i("base.skill_points", p.SkillPoints)
	i("derived.life", p.Life)
	i("derived.max_life", p.MaxLife)
	i("derived.mana", p.Mana)
	i("derived.max_mana", p.MaxMana)
	i("derived.stamina", p.Stamina)
	i("derived.max_stamina", p.MaxStamina)
	i("derived.res_fire", p.Resist.Fire)
	i("derived.res_cold", p.Resist.Cold)
	i("derived.res_light", p.Resist.Lightning)
	i("derived.res_poison", p.Resist.Poison)
	i("derived.res_magic", p.Resist.Magic)
	i("position.x", p.X)
	i("position.y", p.Y)
	i("skills.left", p.LeftSkill)
	i("skills.right", p.RightSkill)

	rl, err := DefaultRules()
	if err != nil {
		return nil, err
	}
	var sk, bonus []string
	for _, x := range p.Skills {
		if rl.InClassSkills(int(p.Class), x.ID) && x.Level != 0 {
			sk = append(sk, fmt.Sprintf("%d=%d", x.ID, x.Level))
		}
		if x.Bonus != 0 {
			bonus = append(bonus, fmt.Sprintf("%d=%d", x.ID, x.Bonus))
		}
	}
	s.Set("skills.base", sk)
	s.Set("skills.bonus", bonus)

	var st, stx []string
	for k, v := range p.Stats {
		if v == 0 {
			continue // absent and zero are the same quantity (game.exe omits zeros, d2d lists them)
		}
		e := fmt.Sprintf("%d/%d=%d", k>>16, k&0xffff, v)
		if rl.scopeOnly(int(k & 0xffff)) {
			stx = append(stx, e)
		} else {
			st = append(st, e)
		}
	}
	s.Set("stats.vector", st)
	s.Set("statsx.vector", stx)
	// load.*: the current and maximum life, mana and stamina with their 8 fractional bits, from the stat
	// vector (stats 6 to 11). Kept apart from derived.* (integers) for the load-time comparison.
	for i, n := range []string{"life", "max_life", "mana", "max_mana", "stamina", "max_stamina"} {
		if v, ok := p.Stats[uint32(6+i)]; ok {
			s.Float("load."+n, float64(v)/256)
		} else {
			s.Unreadable("load."+n, "stat "+fmt.Sprint(6+i)+" not in the stat vector")
		}
	}

	items := map[d2.Container][]string{}
	for _, u := range in.Snap.Inventory.Items {
		it := u.Item
		if it == nil || it.Where == d2.ContCursor {
			continue
		}
		pos := fmt.Sprintf("%d,%d", it.GX, it.GY)
		if it.Where == d2.ContEquipment {
			pos = fmt.Sprintf("body%d", it.Body)
		}
		items[it.Where] = append(items[it.Where], fmt.Sprintf("%s:class%d/q%d/ilvl%d/%s", pos, u.Class, it.Quality, it.ILvl, flagLetters(it)))
	}
	for c := d2.ContInventory; c <= d2.ContEquipment; c++ {
		s.Set("items."+c.String(), items[c])
	}

	var npcs, objs []string
	for _, u := range in.Snap.Monsters {
		npcs = append(npcs, fmt.Sprintf("class%d@%d,%d", u.Class, u.X, u.Y))
	}
	for _, u := range in.Snap.Objects {
		objs = append(objs, fmt.Sprintf("class%d@%d,%d", u.Class, u.X, u.Y))
	}
	s.Set("npcs.units", npcs)
	s.Set("objects.units", objs)

	for d := 0; d < 3; d++ {
		wk, qk := fmt.Sprintf("waypoints.d%d", d), fmt.Sprintf("quests.d%d", d)
		if in.Progress == nil {
			why := in.ProgressWhy
			if why == "" {
				why = "no progress read"
			}
			s.Unreadable(wk, why)
			s.Unreadable(qk, why)
			continue
		}
		var wp, qw []string
		for b := 0; b < 64; b++ {
			if in.Progress.Waypoints[d]>>uint(b)&1 != 0 {
				wp = append(wp, fmt.Sprint(b))
			}
		}
		for n, w := range in.Progress.Quest[d] {
			if w != 0 {
				qw = append(qw, fmt.Sprintf("w%d=%#x", n, w))
			}
		}
		s.Set(wk, wp)
		s.Set(qk, qw)
	}
	applyUnreadable(s, in.Unreadable)
	return s, nil
}

func stateOf(s *d2.Snapshot) any {
	if s == nil {
		return "nil"
	}
	return s.State
}

func flagLetters(it *d2.Item) string {
	var b []byte
	for _, f := range []struct {
		on bool
		c  byte
	}{{it.Identified(), 'i'}, {it.Ethereal(), 'e'}, {it.Socketed(), 's'}, {it.Flags&d2.FlagRuneword != 0, 'r'}} {
		if f.on {
			b = append(b, f.c)
		}
	}
	return string(b)
}
