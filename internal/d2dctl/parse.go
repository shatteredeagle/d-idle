package d2dctl

import (
	"fmt"
	"strconv"
	"strings"
)

// Parsers are tolerant on purpose: the channel's verbs change upstream. Every parsed record keeps its Raw
// line, extra trailing fields land in Extra, and a line that cannot be parsed is reported through
// Reply.Bad instead of failing the call.

// fields splits a payload line: on tabs when it has any (the documented format), else on whitespace
// (what our spike transcript looked like).
func fields(line string) []string {
	if strings.Contains(line, "\t") {
		return strings.Split(line, "\t")
	}
	return strings.Fields(line)
}

func atoi(s string) (int, error)     { return strconv.Atoi(strings.TrimSpace(s)) }
func atof(s string) (float64, error) { return strconv.ParseFloat(strings.TrimSpace(s), 64) }
func unitID(s string) (int, error)   { return atoi(strings.TrimPrefix(strings.TrimSpace(s), "#")) }
func isUnitTok(s string) bool        { return len(s) > 1 && s[0] == '#' && s[1] >= '0' && s[1] <= '9' }

// firstErr returns the first non-nil error.
func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// Pair is a "cur/max" value (life, mana, hp).
type Pair struct{ Cur, Max int }

// ParsePair parses "45/45".
func ParsePair(s string) (Pair, error) {
	a, b, ok := strings.Cut(s, "/")
	if !ok {
		return Pair{}, fmt.Errorf("not cur/max: %q", s)
	}
	c, e1 := atoi(a)
	m, e2 := atoi(b)
	return Pair{c, m}, firstErr(e1, e2)
}

// KV is a parsed "key=value key=value" line; tokens without '=' are kept in Loose.
type KV struct {
	Raw   string
	M     map[string]string
	Order []string
	Loose []string
}

// ParseKV splits on whitespace and then on the first '='. A value never contains spaces in the
// channel's key=value lines we have seen (V: state, info, debug attack, quest show).
func ParseKV(line string) KV {
	kv := KV{Raw: line, M: map[string]string{}}
	for _, t := range strings.Fields(line) {
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			kv.Loose = append(kv.Loose, t)
			continue
		}
		if _, dup := kv.M[k]; !dup {
			kv.Order = append(kv.Order, k)
		}
		kv.M[k] = v
	}
	return kv
}

// Get returns a value and whether the key was present.
func (kv KV) Get(k string) (string, bool) { v, ok := kv.M[k]; return v, ok }

// Int returns a key as int; ok is false when missing or not a number.
func (kv KV) Int(k string) (int, bool) {
	v, ok := kv.M[k]
	if !ok {
		return 0, false
	}
	n, err := atoi(v)
	return n, err == nil
}

// Pair returns a "cur/max" key.
func (kv KV) Pair(k string) (Pair, bool) {
	v, ok := kv.M[k]
	if !ok {
		return Pair{}, false
	}
	p, err := ParsePair(v)
	return p, err == nil
}

// Info is the reply of "info".
type Info struct {
	KV
	W, H, Frame int
}

func parseInfo(line string) Info {
	kv := ParseKV(line)
	in := Info{KV: kv}
	in.W, _ = kv.Int("w")
	in.H, _ = kv.Int("h")
	in.Frame, _ = kv.Int("frame")
	return in
}

// State is the one-line "state" reply (about 50 key=value fields, V). Typed fields are the ones we rely
// on; anything else is in KV.
type State struct {
	KV
	Screen, Class, Name string
	Difficulty          int // -1 when "?" or missing
	Level, Gold, Exp    int
	Life, Mana          Pair
	// GameMenu is the raw "gamemenu=" value (V: "0:0"); check it before assuming Escape closed the menu.
	GameMenu string
}

func parseState(line string) State {
	kv := ParseKV(line)
	s := State{KV: kv, Difficulty: -1}
	s.Screen, _ = kv.Get("screen")
	s.Class, _ = kv.Get("class")
	s.Name, _ = kv.Get("name")
	s.GameMenu, _ = kv.Get("gamemenu")
	if n, ok := kv.Int("difficulty"); ok {
		s.Difficulty = n
	}
	s.Level, _ = kv.Int("level")
	s.Gold, _ = kv.Int("gold")
	s.Exp, _ = kv.Int("exp")
	s.Life, _ = kv.Pair("life")
	s.Mana, _ = kv.Pair("mana")
	return s
}

// GameMenuOpen reports whether the in-game (Escape) menu looks open. The gamemenu value is "a:b"; V only
// saw "0:0" for closed, so any non-zero half counts as open (U for the meaning of each half).
func (s State) GameMenuOpen() bool {
	a, b, ok := strings.Cut(s.GameMenu, ":")
	return ok && (a != "0" || b != "0")
}

// NPC is one line of "npcs".
type NPC struct {
	Raw          string
	Name         string
	SX, SY       int // feet on the 800x600 screen
	HasMenu      bool
	Index        int // for "cmd interact"
	CellX, CellY float64
}

// ParseNPC parses <name> <sx> <sy> <menu> <index> <cellx> <celly>. Names may contain spaces: the last six
// fields are fixed.
func ParseNPC(line string) (NPC, error) {
	n := NPC{Raw: line}
	f := fields(line)
	if len(f) < 7 {
		return n, fmt.Errorf("npc: want 7 fields, got %d", len(f))
	}
	k := len(f) - 6
	n.Name = strings.Join(f[:k], " ")
	var e [6]error
	n.SX, e[0] = atoi(f[k])
	n.SY, e[1] = atoi(f[k+1])
	var m int
	m, e[2] = atoi(f[k+2])
	n.HasMenu = m != 0
	n.Index, e[3] = atoi(f[k+3])
	n.CellX, e[4] = atof(f[k+4])
	n.CellY, e[5] = atof(f[k+5])
	return n, firstErr(e[:]...)
}

// Monster is one line of "monsters" (current level).
type Monster struct {
	Raw          string
	ClassID      string // the first column (V: a short id like "fallen1")
	CellX, CellY float64
	SX, SY       int
	HP           Pair
	Mode         string // animation mode, e.g. NU, A1, GH, DT, DD
	Boss         string // champion / unique / superunique / minion, or "-"
	Mods         []int  // MonUMod ids
	Level        int
	Name         string
	Unit         int    // what "cmd skill" takes
	Row          string // skill sequence row "<mode>:<frame>" or "-"
	Extra        []string
}

// IsBoss reports whether the boss column is set.
func (m Monster) IsBoss() bool { return m.Boss != "" && m.Boss != "-" }

// Dead reports a death mode (DD) or hp at or below 0.
func (m Monster) Dead() bool { return m.Mode == "DD" || m.HP.Cur <= 0 }

// ParseMonster parses a monsters line. The name may hold spaces: it is whatever sits between "lvl<n>" and
// the "#<unit>" token. Layout V from the spike transcript.
func ParseMonster(line string) (Monster, error) {
	m := Monster{Raw: line}
	f := strings.Fields(line)
	if len(f) < 12 {
		return m, fmt.Errorf("monster: want >=12 fields, got %d", len(f))
	}
	u := -1
	for i := 10; i < len(f); i++ {
		if isUnitTok(f[i]) {
			u = i
			break
		}
	}
	if u < 0 || !strings.HasPrefix(f[9], "lvl") {
		return m, fmt.Errorf("monster: layout changed (no lvl/#unit tokens)")
	}
	var e [8]error
	m.ClassID = f[0]
	m.CellX, e[0] = atof(f[1])
	m.CellY, e[1] = atof(f[2])
	m.SX, e[2] = atoi(f[3])
	m.SY, e[3] = atoi(f[4])
	m.HP, e[4] = ParsePair(f[5])
	m.Mode, m.Boss = f[6], f[7]
	if f[8] != "-" {
		for _, t := range strings.Split(f[8], ",") {
			n, err := atoi(t)
			if err != nil {
				e[5] = err
				continue
			}
			m.Mods = append(m.Mods, n)
		}
	}
	m.Level, e[6] = atoi(strings.TrimPrefix(f[9], "lvl"))
	m.Name = strings.Join(f[10:u], " ")
	m.Unit, e[7] = unitID(f[u])
	if u+1 < len(f) {
		m.Row = f[u+1]
		m.Extra = f[u+2:]
	}
	return m, firstErr(e[:]...)
}

// GroundItem is one line of "ground".
type GroundItem struct {
	Raw    string
	Code   string // "gld" for gold
	Label  string
	SX, SY int
	Unit   int // what "cmd pickup" takes
	// Extra holds trailing fields (V: our spike saw the cell x y after the unit id; not in the upstream notes).
	Extra []string
}

// IsGold reports a gold pile.
func (g GroundItem) IsGold() bool { return g.Code == "gld" }

// Cell returns the trailing cell position when the Extra fields carry one.
func (g GroundItem) Cell() (x, y float64, ok bool) {
	if len(g.Extra) < 2 {
		return 0, 0, false
	}
	x, e1 := atof(g.Extra[0])
	y, e2 := atof(g.Extra[1])
	return x, y, e1 == nil && e2 == nil
}

// ParseGround parses <code> <label...> <sx> <sy> #<unit> [more]. The label (may contain spaces and numbers,
// V: "8800 Gold") is whatever lies between the code and the two screen coordinates before the #unit token.
func ParseGround(line string) (GroundItem, error) {
	g := GroundItem{Raw: line}
	f := fields(line)
	u := -1
	for i := range f {
		if isUnitTok(f[i]) {
			u = i
			break
		}
	}
	if u < 3 {
		return g, fmt.Errorf("ground: no #unit token or too few fields")
	}
	var e [3]error
	g.Code = f[0]
	g.Label = strings.Join(f[1:u-2], " ")
	g.SX, e[0] = atoi(f[u-2])
	g.SY, e[1] = atoi(f[u-1])
	g.Unit, e[2] = unitID(f[u])
	g.Extra = f[u+1:]
	return g, firstErr(e[:]...)
}

// MenuLine is one line of the open NPC menu.
type MenuLine struct {
	Raw  string
	Text string
	X, Y int // a point inside the line, game pixels
}

// ParseMenuLine parses <text> <x> <y>; text may contain spaces and digits ("Aliza - Lvl: 5 ...").
func ParseMenuLine(line string) (MenuLine, error) {
	m := MenuLine{Raw: line}
	f := fields(line)
	if len(f) < 3 {
		return m, fmt.Errorf("menu: want text x y")
	}
	var e [2]error
	m.X, e[0] = atoi(f[len(f)-2])
	m.Y, e[1] = atoi(f[len(f)-1])
	m.Text = strings.Join(f[:len(f)-2], " ")
	return m, firstErr(e[:]...)
}

// Player is one line of "debug players".
type Player struct {
	Raw         string
	ID          int
	Name        string
	X, Y        float64
	Seat, PMode int // seat -1: the first player; pmode -1: none
	Ghost       bool
}

// ParsePlayer parses <id> <name> <x> <y> <seat> <pmode> <ghost>; the last five fields are fixed.
func ParsePlayer(line string) (Player, error) {
	p := Player{Raw: line}
	f := fields(line)
	if len(f) < 7 {
		return p, fmt.Errorf("player: want 7 fields, got %d", len(f))
	}
	k := len(f) - 5
	var e [6]error
	p.ID, e[0] = atoi(f[0])
	p.Name = strings.Join(f[1:k], " ")
	p.X, e[1] = atof(f[k])
	p.Y, e[2] = atof(f[k+1])
	p.Seat, e[3] = atoi(f[k+2])
	p.PMode, e[4] = atoi(f[k+3])
	var g int
	g, e[5] = atoi(f[k+4])
	p.Ghost = g != 0
	return p, firstErr(e[:]...)
}

// Corpse is one line of "debug corpses".
type Corpse struct {
	Raw   string
	Level int
	X, Y  float64
	Items int
}

// ParseCorpse parses <level> <x> <y> <items>.
func ParseCorpse(line string) (Corpse, error) {
	c := Corpse{Raw: line}
	f := fields(line)
	if len(f) < 4 {
		return c, fmt.Errorf("corpse: want 4 fields")
	}
	var e [4]error
	c.Level, e[0] = atoi(f[0])
	c.X, e[1] = atof(f[1])
	c.Y, e[2] = atof(f[2])
	c.Items, e[3] = atoi(f[3])
	return c, firstErr(e[:]...)
}

// Warp is one line of "debug warps": a cave mouth or stairs.
type Warp struct {
	Raw   string
	Index int // for "debug enter"
	X, Y  float64
	To    int // Levels.txt id
}

// ParseWarp parses <i> <x> <y> <to>.
func ParseWarp(line string) (Warp, error) {
	w := Warp{Raw: line}
	f := fields(line)
	if len(f) < 4 {
		return w, fmt.Errorf("warp: want 4 fields")
	}
	var e [4]error
	w.Index, e[0] = atoi(f[0])
	w.X, e[1] = atof(f[1])
	w.Y, e[2] = atof(f[2])
	w.To, e[3] = atoi(f[3])
	return w, firstErr(e[:]...)
}

// Preset is one line of "debug presets": every object or NPC the level placed, hidden ones included.
type Preset struct {
	Raw      string
	Index    int // for "cmd interact"
	ObjectID int
	X, Y     float64
	Hidden   bool
	Name     string
}

// ParsePreset parses <i> <object id> <x> <y> <hidden> <name...>.
func ParsePreset(line string) (Preset, error) {
	p := Preset{Raw: line}
	f := fields(line)
	if len(f) < 6 {
		return p, fmt.Errorf("preset: want 6 fields")
	}
	var e [5]error
	p.Index, e[0] = atoi(f[0])
	p.ObjectID, e[1] = atoi(f[1])
	p.X, e[2] = atof(f[2])
	p.Y, e[3] = atof(f[3])
	var h int
	h, e[4] = atoi(f[4])
	p.Hidden = h != 0
	p.Name = strings.Join(f[5:], " ")
	return p, firstErr(e[:]...)
}

// Object is one line of "debug objects": a shrine or chest.
type Object struct {
	Raw    string
	Index  int
	X, Y   float64
	Kind   string // shrine | chest
	Row    string // Shrines.txt row, or the chest trap type 0..8 / "trap"
	Locked bool
	Mode   string
}

// ParseObject parses <i> <x> <y> <kind> <row|trap> <locked|-> <mode>.
func ParseObject(line string) (Object, error) {
	o := Object{Raw: line}
	f := fields(line)
	if len(f) < 7 {
		return o, fmt.Errorf("object: want 7 fields")
	}
	var e [3]error
	o.Index, e[0] = atoi(f[0])
	o.X, e[1] = atof(f[1])
	o.Y, e[2] = atof(f[2])
	o.Kind, o.Row, o.Locked, o.Mode = f[3], f[4], f[5] == "locked", f[6]
	return o, firstErr(e[:]...)
}

// Portal is one line of "debug portals".
type Portal struct {
	Raw   string
	Which int // 0 where it was cast, 1 its twin in camp
	Level int
	X, Y  float64
}

// ParsePortal parses <which> <level> <x> <y>.
func ParsePortal(line string) (Portal, error) {
	p := Portal{Raw: line}
	f := fields(line)
	if len(f) < 4 {
		return p, fmt.Errorf("portal: want 4 fields")
	}
	var e [4]error
	p.Which, e[0] = atoi(f[0])
	p.Level, e[1] = atoi(f[1])
	p.X, e[2] = atof(f[2])
	p.Y, e[3] = atof(f[3])
	return p, firstErr(e[:]...)
}

// Item is one entry of "items": a header "[code loc=.. slot=.. q=.. panel=.. id=.. size=WxH at=c,r]" and
// the indented tooltip lines that follow it.
type Item struct {
	Raw     string
	Code    string
	Attrs   map[string]string // loc, slot, q, panel, id, size, at
	Tooltip []string
}

// Loc returns the integer "loc=" attribute (V: 1 body slot, 2 belt; other values U).
func (it Item) Loc() int { n, _ := atoi(it.Attrs["loc"]); return n }

// ParseItems groups the "items" payload: a line starting with '[' opens an item, other lines belong to it.
func ParseItems(lines []string) (items []Item, bad []string) {
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(l, "[") && strings.HasSuffix(t, "]") {
			kv := ParseKV(strings.Trim(t, "[]"))
			it := Item{Raw: l, Attrs: kv.M}
			if len(kv.Loose) > 0 {
				it.Code = kv.Loose[0]
			}
			items = append(items, it)
			continue
		}
		if len(items) == 0 {
			bad = append(bad, l)
			continue
		}
		items[len(items)-1].Tooltip = append(items[len(items)-1].Tooltip, t)
	}
	return
}

// LevelInfo is the rest of "debug level": id x y w h wx wy (V).
type LevelInfo struct {
	Raw              string
	ID               int
	X, Y             float64 // player cell
	W, H             int     // level size in cells
	OriginX, OriginY int     // act-tile origin
}

func parseLevel(rest string) (LevelInfo, error) {
	l := LevelInfo{Raw: rest}
	f := strings.Fields(rest)
	if len(f) < 7 {
		return l, fmt.Errorf("level: want 7 fields, got %d", len(f))
	}
	var e [7]error
	l.ID, e[0] = atoi(f[0])
	l.X, e[1] = atof(f[1])
	l.Y, e[2] = atof(f[2])
	l.W, e[3] = atoi(f[3])
	l.H, e[4] = atoi(f[4])
	l.OriginX, e[5] = atoi(f[5])
	l.OriginY, e[6] = atoi(f[6])
	return l, firstErr(e[:]...)
}

// QuestInfo is the rest of "debug quest <q> show": bits=0x.. den=N skillpts=N states=a,b,c,... (V for Act 1).
type QuestInfo struct {
	KV
	Bits   uint32
	States []int // Act 1: den, burial, tools, cain, tower, andy
}

func parseQuest(rest string) (QuestInfo, error) {
	kv := ParseKV(rest)
	q := QuestInfo{KV: kv}
	v, ok := kv.Get("bits")
	if !ok {
		return q, fmt.Errorf("quest: no bits= field")
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(v, "0x"), 16, 32)
	if err != nil {
		return q, fmt.Errorf("quest bits %q: %w", v, err)
	}
	q.Bits = uint32(n)
	if v, ok := kv.Get("states"); ok {
		for _, t := range strings.Split(v, ",") {
			n, err := atoi(t)
			if err != nil {
				return q, err
			}
			q.States = append(q.States, n)
		}
	}
	return q, nil
}
