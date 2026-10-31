package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"d2lod-bot/internal/status"
	"d2lod-bot/internal/telemetry"
)

// simClock is wall time scaled by speed, so sim hours pass quickly.
type simClock struct {
	start time.Time
	speed float64
}

func (c *simClock) Now() time.Time {
	return c.start.Add(time.Duration(float64(time.Since(c.start)) * c.speed))
}

type simMon struct {
	id       uint32
	name     string
	txt      int
	x, y     float64
	hp       int
	mode     int
	corpse   int
	friendly bool
}

type simGround struct {
	it   status.Item
	x, y float64
}

// Sim is a plausible Sorceress leveling between town and Blood Moor. It
// implements status.CharacterSource, InventorySource, MapView and Namer, the same
// interfaces the real readers will satisfy.
type Sim struct {
	mu  sync.Mutex
	rng *rand.Rand
	tr  *telemetry.Tracker

	town, field, lvl *level
	px, py           float64
	tx, ty           float64
	hasTarget        bool
	blocked          int

	level, statPts, skillPts int
	exp                      uint32
	life, mana               int
	str, dex, vit, ene       int
	hpPots, mpPots           int
	gold, stashGold          int
	skills                   []status.Skill
	inv, stash               []status.Item
	equip                    map[string]*status.Item
	mons                     []*simMon
	ground                   []*simGround
	nextID                   uint32

	phase      string
	phaseTicks int
	runTicks   int
	deadTicks  int
	target     *simMon
}

var skillNames = []string{"Fire Bolt", "Charged Bolt", "Frozen Armor", "Ice Bolt", "Static Field", "Telekinesis", "Frost Nova", "Ice Blast"}

func newSim(tr *telemetry.Tracker, seed int64) *Sim {
	s := &Sim{rng: rand.New(rand.NewSource(seed)), tr: tr, town: townFixture(), field: fieldLevel(),
		level: 5, exp: 7900, str: 12, dex: 25, vit: 18, ene: 40, hpPots: 6, mpPots: 4, gold: 420, stashGold: 1500,
		equip: map[string]*status.Item{}, nextID: 100, phase: "town"}
	s.lvl = s.town
	s.px, s.py = float64(s.town.spawn[0]), float64(s.town.spawn[1])
	s.skills = []status.Skill{{ID: 36, Name: "Fire Bolt", Level: 4, Hotkey: "F1"}, {ID: 40, Name: "Frozen Armor", Level: 1, Hotkey: "F2"}, {ID: 38, Name: "Charged Bolt", Level: 1, Hotkey: "F3"}}
	s.life, s.mana = s.maxLife(), s.maxMana()
	s.equipNew("wnd", status.QNormal, 0)
	s.equipNew("lea", status.QMagic, 0)
	s.equipNew("rin", status.QMagic, 0)
	s.spawnNPCs()
	return s
}

func (s *Sim) maxLife() int { return 50 + s.vit*4 + s.level*2 }
func (s *Sim) maxMana() int { return 10 + s.ene*2 + s.level }

var npcs = []struct {
	name string
	x, y float64
}{{"Akara", 28, 34}, {"Charsi", 88, 32}, {"Gheed", 92, 52}, {"Kashya", 60, 20}, {"Warriv", 100, 44}, {"Cain", 66, 52}}

func (s *Sim) spawnNPCs() {
	for i, n := range npcs {
		s.mons = append(s.mons, &simMon{id: uint32(i + 1), name: n.name, txt: 100 + i, x: n.x, y: n.y, mode: 1, friendly: true, hp: 1})
	}
}

// --- item generation -------------------------------------------------------

type tmpl struct {
	code string
	w, h int
	slot string
}

var tmpls = []tmpl{{"cap", 2, 2, "head"}, {"amu", 1, 1, "neck"}, {"lea", 2, 3, "torso"}, {"buc", 2, 2, "lhand"}, {"lgl", 2, 2, "gloves"},
	{"bts", 2, 2, "feet"}, {"rin", 1, 1, "rring"}, {"wnd", 1, 2, "rhand"}, {"ssd", 1, 3, ""}, {"jew", 1, 1, ""}, {"rune", 1, 1, ""}, {"bel", 2, 1, "belt"}}

var qrank = map[string]int{"low": 0, "normal": 1, "superior": 2, "magic": 3, "rare": 4, "crafted": 5, "set": 6, "unique": 7, "runeword": 8}

func (s *Sim) newItem(code, q string, ilvl int) status.Item {
	var t tmpl
	for _, x := range tmpls {
		if x.code == code {
			t = x
		}
	}
	s.nextID++
	it := status.Item{ID: s.nextID, Code: code, Quality: q, Level: ilvl, W: t.w, H: t.h, Identified: true}
	n := map[string]int{"magic": 2, "rare": 4, "crafted": 4, "unique": 3, "set": 3, "runeword": 4}[q]
	for i := 0; i < n; i++ {
		it.Props = append(it.Props, propLine(s.rng))
	}
	if (code == "ssd" || code == "lea" || code == "cap") && s.rng.Intn(4) == 0 {
		it.Sockets = 1 + s.rng.Intn(3)
		if q == status.QNormal && s.rng.Intn(2) == 0 {
			q = "runeword"
		}
	}
	it.Ethereal = s.rng.Intn(15) == 0
	return it
}

func propLine(r *rand.Rand) string {
	switch r.Intn(6) {
	case 0:
		return fmt.Sprintf("+%d to Life", 5+r.Intn(30))
	case 1:
		return fmt.Sprintf("Fire Resist +%d%%", 5+r.Intn(30))
	case 2:
		return fmt.Sprintf("+%d to Mana", 5+r.Intn(25))
	case 3:
		return fmt.Sprintf("%d%% Faster Cast Rate", 5+5*r.Intn(4))
	case 4:
		return fmt.Sprintf("+%d to Dexterity", 1+r.Intn(10))
	}
	return fmt.Sprintf("Cold Resist +%d%%", 5+r.Intn(30))
}

func (s *Sim) randomItem() status.Item {
	t := tmpls[s.rng.Intn(len(tmpls))]
	q := status.QNormal
	switch r := s.rng.Intn(100); {
	case r < 4:
		q = status.QLow
	case r < 55:
		q = status.QNormal
	case r < 82:
		q = status.QMagic
	case r < 92:
		q = status.QRare
	case r < 96:
		q = status.QUnique
	case r < 98:
		q = status.QSet
	default:
		q = status.QCrafted
	}
	if t.code == "rune" {
		q = status.QNormal
	}
	return s.newItem(t.code, q, s.level+s.rng.Intn(4))
}

func (s *Sim) equipNew(code, q string, _ int) {
	it := s.newItem(code, q, s.level)
	for _, t := range tmpls {
		if t.code == code {
			s.equip[t.slot] = &it
		}
	}
}

func (s *Sim) tryEquip(it status.Item) bool {
	for _, t := range tmpls {
		if t.code != it.Code || t.slot == "" {
			continue
		}
		slot := t.slot
		if slot == "rring" && s.equip["rring"] != nil && s.equip["lring"] == nil {
			slot = "lring"
		}
		cur := s.equip[slot]
		if cur == nil || qrank[it.Quality] > qrank[cur.Quality] {
			if cur != nil {
				s.place(&s.inv, 10, 4, *cur)
			}
			c := it
			s.equip[slot] = &c
			return true
		}
	}
	return false
}

// place puts it in the first free spot of a cols x rows grid.
func (s *Sim) place(list *[]status.Item, cols, rows int, it status.Item) bool {
	occ := make([]bool, cols*rows)
	for _, o := range *list {
		for y := o.Y; y < o.Y+o.H; y++ {
			for x := o.X; x < o.X+o.W; x++ {
				if x < cols && y < rows {
					occ[y*cols+x] = true
				}
			}
		}
	}
	for y := 0; y+it.H <= rows; y++ {
		for x := 0; x+it.W <= cols; x++ {
			ok := true
			for dy := 0; dy < it.H && ok; dy++ {
				for dx := 0; dx < it.W; dx++ {
					if occ[(y+dy)*cols+x+dx] {
						ok = false
						break
					}
				}
			}
			if ok {
				it.X, it.Y = x, y
				*list = append(*list, it)
				return true
			}
		}
	}
	return false
}

// --- names (the Namer the DATA agent will replace) --------------------------

var baseNames = map[string]string{"cap": "Cap", "amu": "Amulet", "lea": "Leather Armor", "buc": "Buckler", "lgl": "Leather Gloves", "bts": "Boots",
	"rin": "Ring", "wnd": "Wand", "ssd": "Short Sword", "jew": "Jewel", "rune": "El Rune", "bel": "Sash"}
var pre = []string{"Jade", "Cobalt", "Sturdy", "Fine", "Amber", "Crimson", "Lizard's", "Sapphire"}
var suf = []string{"of the Fox", "of Vita", "of Frost", "of Quickness", "of the Whale", "of Readiness"}
var rareA = []string{"Death", "Storm", "Doom", "Raven", "Grim", "Beast", "Soul", "Dread"}
var rareB = []string{"Grip", "Veil", "Song", "Shell", "Coil", "Band", "Wrap", "Fist"}
var uniq = map[string]string{"cap": "Biggin's Bonnet", "amu": "Nokozan Relic", "lea": "Greyform", "buc": "Pelta Lunata", "lgl": "Chance Guards", "bts": "War Traveler", "rin": "Nagelring", "wnd": "Gravenspine", "ssd": "Rixot's Keen", "jew": "Rainbow Facet", "bel": "Gloom's Trap"}

type simNamer struct{}

func (simNamer) ItemName(it status.Item) string {
	b := baseNames[it.Code]
	if b == "" {
		b = it.Code
	}
	id := int(it.ID)
	switch it.Quality {
	case status.QMagic:
		return pre[id%len(pre)] + " " + b + " " + suf[(id/7)%len(suf)]
	case status.QRare, status.QCrafted:
		return rareA[id%len(rareA)] + " " + rareB[(id/5)%len(rareB)]
	case status.QUnique:
		if u := uniq[it.Code]; u != "" {
			return u
		}
	case status.QSet:
		return "Civerb's " + b
	case status.QRuneword:
		return "Spirit (" + b + ")"
	case status.QLow:
		return "Cracked " + b
	case status.QSuperior:
		return "Superior " + b
	}
	return b
}

func (s *Sim) ItemName(it status.Item) string { return simNamer{}.ItemName(it) }

// --- status interfaces -----------------------------------------------------

func (s *Sim) Character() status.CharacterData {
	s.mu.Lock()
	defer s.mu.Unlock()
	prev, _ := telemetry.DefaultExpForLevel(s.level)
	next, _ := telemetry.DefaultExpForLevel(s.level + 1)
	return status.CharacterData{Name: "Testy", Class: "Sorceress", Difficulty: "Normal", Level: s.level, Exp: s.exp, ExpPrev: prev, ExpNext: next,
		Life: s.life, MaxLife: s.maxLife(), Mana: s.mana, MaxMana: s.maxMana(), Stamina: 130, MaxStamina: 130,
		Str: s.str, Dex: s.dex, Vit: s.vit, Ene: s.ene, StatPoints: s.statPts, SkillPts: s.skillPts, Defense: 40 + s.dex/4,
		Resists: status.Resists{Fire: 12, Light: 0, Cold: 5, Poison: -0}, Skills: append([]status.Skill(nil), s.skills...), LeftSkill: 36, RightSkill: 40}
}

func (s *Sim) Inventory() status.InventoryData {
	s.mu.Lock()
	defer s.mu.Unlock()
	d := status.InventoryData{Equipment: map[string]*status.Item{}, Inventory: append([]status.Item{}, s.inv...), Stash: append([]status.Item{}, s.stash...),
		Gold: s.gold, StashGold: s.stashGold, BeltRows: 4, Belt: []status.Item{}}
	for k, v := range s.equip {
		c := *v
		d.Equipment[k] = &c
	}
	pot := func(col, n int, code, name string) {
		for r := 0; r < n && r < 4; r++ {
			d.Belt = append(d.Belt, status.Item{ID: uint32(9000 + col*10 + r), Code: code, Name: name, Quality: status.QNormal, X: col, Y: r, W: 1, H: 1, Identified: true})
		}
	}
	pot(0, s.hpPots, "hp1", "Minor Healing Potion")
	pot(1, s.hpPots-4, "hp1", "Minor Healing Potion")
	pot(2, s.mpPots, "mp1", "Minor Mana Potion")
	pot(3, s.mpPots-4, "mp1", "Minor Mana Potion")
	return d
}

func (s *Sim) AreaID() int { s.mu.Lock(); defer s.mu.Unlock(); return s.lvl.id }
func (s *Sim) Bounds() (x0, y0, x1, y1 int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return 0, 0, s.lvl.g.w, s.lvl.g.h
}
func (s *Sim) Walkable(x, y int) bool { s.mu.Lock(); defer s.mu.Unlock(); return s.lvl.g.at(x, y) }
func (s *Sim) Player() (int, int)     { s.mu.Lock(); defer s.mu.Unlock(); return int(s.px), int(s.py) }
func (s *Sim) Revision() int          { s.mu.Lock(); defer s.mu.Unlock(); return s.lvl.rev }
func (s *Sim) Exits() []status.MapExit {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]status.MapExit(nil), s.lvl.exits...)
}
func (s *Sim) Units() []status.MapUnit {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []status.MapUnit
	for _, m := range s.mons {
		k := "monster"
		if m.friendly {
			k = "npc"
		}
		out = append(out, status.MapUnit{ID: m.id, Kind: k, Name: m.name, X: int(m.x), Y: int(m.y), Dead: m.mode == 0 || m.mode == 12, Hostile: !m.friendly})
	}
	for _, g := range s.ground {
		out = append(out, status.MapUnit{ID: g.it.ID, Kind: "item", Name: simNamer{}.ItemName(g.it), X: int(g.x), Y: int(g.y)})
	}
	return out
}

// --- simulation ------------------------------------------------------------

func (s *Sim) randWalkable(g *grid) (float64, float64) {
	for i := 0; i < 500; i++ {
		x, y := 8+s.rng.Intn(g.w-16), 8+s.rng.Intn(g.h-16)
		if g.at(x, y) {
			return float64(x), float64(y)
		}
	}
	return s.px, s.py
}

func (s *Sim) walk(tx, ty, speed float64) bool {
	before := math.Hypot(tx-s.px, ty-s.py)
	if before <= speed {
		s.px, s.py = tx, ty
		return true
	}
	dx, dy := tx-s.px, ty-s.py
	nx, ny := s.px+dx/before*speed, s.py+dy/before*speed
	switch {
	case s.lvl.g.at(int(nx), int(ny)):
		s.px, s.py = nx, ny
	case s.lvl.g.at(int(nx), int(s.py)):
		s.px = nx
	case s.lvl.g.at(int(s.px), int(ny)):
		s.py = ny
	default:
		a := s.rng.Float64() * 6.28
		if jx, jy := s.px+math.Cos(a)*speed, s.py+math.Sin(a)*speed; s.lvl.g.at(int(jx), int(jy)) {
			s.px, s.py = jx, jy
		}
	}
	if math.Hypot(tx-s.px, ty-s.py) > before-speed*0.3 {
		s.blocked++
	} else {
		s.blocked = 0
	}
	if s.blocked > 12 { // stuck: sim cheat, the real bot path-finds
		s.px, s.py = tx, ty
		s.blocked = 0
		return true
	}
	return false
}

func (s *Sim) dist(x, y float64) float64 { return math.Hypot(x-s.px, y-s.py) }

func (s *Sim) enterLevel(l *level) {
	s.lvl = l
	s.px, s.py = float64(l.spawn[0]), float64(l.spawn[1])
	s.mons, s.ground, s.target, s.hasTarget = nil, nil, nil, false
	if l == s.town {
		s.spawnNPCs()
		return
	}
	names := []string{"Fallen", "Zombie", "Rogue Scout", "Quill Rat", "Carver"}
	for i := 0; i < 28; i++ {
		x, y := s.randWalkable(l.g)
		if math.Hypot(x-s.px, y-s.py) < 30 {
			continue
		}
		s.nextID++
		k := s.rng.Intn(len(names))
		s.mons = append(s.mons, &simMon{id: s.nextID, name: names[k], txt: 20 + k, x: x, y: y, mode: 1, hp: 20 + s.level*3 + s.rng.Intn(15)})
	}
}

func (s *Sim) alive() (out []*simMon) {
	for _, m := range s.mons {
		if !m.friendly && m.mode != 0 && m.mode != 12 {
			out = append(out, m)
		}
	}
	return
}

func (s *Sim) setPhase(p, activity, detail string) {
	s.phase, s.phaseTicks = p, 0
	s.hasTarget = false
	s.tr.SetActivity(activity, detail)
}

func (s *Sim) inTown() bool { return s.lvl == s.town }

// Step advances the simulation by one tick.
func (s *Sim) Step() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phaseTicks++
	dead := false
	loading := s.phase == "loading"
	// regen
	if s.phase != "dead" {
		if s.rng.Intn(3) == 0 && s.life < s.maxLife() {
			s.life++
		}
		if s.mana < s.maxMana() {
			s.mana++
		}
	}
	switch s.phase {
	case "town":
		if s.phaseTicks == 1 {
			s.tr.SetActivity("town", "restocking")
		}
		if !s.hasTarget {
			n := npcs[s.rng.Intn(3)]
			s.tx, s.ty, s.hasTarget = n.x+3, n.y+4, true
		}
		if s.walk(s.tx, s.ty, 5) {
			s.townChores()
			s.setPhase("togate", "town", "heading to the gate")
		}
	case "togate":
		if s.walk(117, 44, 5) {
			s.setPhase("loading", "loading", "Blood Moor")
		}
	case "loading":
		if s.phaseTicks >= 3 {
			if s.lvl == s.town {
				s.enterLevel(s.field)
				s.runTicks = 0
				s.tr.RunStart("Blood Moor")
				s.setPhase("explore", "exploring", "Blood Moor")
			} else {
				s.enterLevel(s.town)
				s.setPhase("town", "town", "restocking")
			}
		}
	case "explore", "fight", "loot", "return":
		s.runTicks++
		s.fieldTick()
	case "dead":
		dead = true
		if s.phaseTicks >= 4 {
			s.tr.RunEnd()
			s.gold = s.gold * 9 / 10
			s.enterLevel(s.town)
			s.life, s.mana = s.maxLife()/2, s.maxMana()/2
			s.setPhase("town", "town", "recovering after death")
		}
	}
	s.observe(dead, loading && s.phaseTicks < 3 && s.phaseTicks > 0)
}

func (s *Sim) townChores() {
	// restock, sell junk, stash valuables, spend points
	s.hpPots, s.mpPots = 8, 8
	s.life, s.mana = s.maxLife(), s.maxMana()
	keep := s.inv[:0:0]
	for _, it := range s.inv {
		switch it.Quality {
		case status.QMagic, status.QNormal, status.QLow, status.QSuperior:
			if it.Code != "rune" {
				s.gold += 30 + s.rng.Intn(200)
				continue
			}
			fallthrough
		default:
			it.X, it.Y = 0, 0
			if s.place(&s.stash, 6, 8, it) {
				continue
			}
			keep = append(keep, it)
		}
	}
	s.inv = nil
	for _, it := range keep {
		s.place(&s.inv, 10, 4, it)
	}
	s.gold -= 120
	if s.gold < 0 {
		s.gold = 0
	}
	for s.statPts > 0 {
		if s.statPts%2 == 0 {
			s.vit++
		} else {
			s.ene++
		}
		s.statPts--
	}
	for s.skillPts > 0 {
		i := s.rng.Intn(len(skillNames))
		found := false
		for k := range s.skills {
			if s.skills[k].Name == skillNames[i] {
				s.skills[k].Level++
				found = true
			}
		}
		if !found {
			hk := ""
			if len(s.skills) < 8 {
				hk = fmt.Sprintf("F%d", len(s.skills)+1)
			}
			s.skills = append(s.skills, status.Skill{ID: 30 + i, Name: skillNames[i], Level: 1, Hotkey: hk})
		}
		s.skillPts--
	}
}

func (s *Sim) fieldTick() {
	// emergency potions
	if s.life < s.maxLife()*35/100 && s.hpPots > 0 {
		s.hpPots--
		s.life += s.maxLife() * 4 / 10
		if s.life > s.maxLife() {
			s.life = s.maxLife()
		}
	}
	if s.mana < s.maxMana()/10 && s.mpPots > 0 {
		s.mpPots--
		s.mana += s.maxMana() / 2
	}
	// fighting damage to the player
	for _, m := range s.alive() {
		if s.dist(m.x, m.y) < 4 && s.rng.Intn(3) == 0 {
			s.life -= 3 + s.rng.Intn(8)
		}
	}
	if s.life <= 0 {
		s.life = 0
		s.setPhase("dead", "dead", "killed by monsters")
		return
	}
	// monsters shamble toward the player once close
	for _, m := range s.alive() {
		if d := s.dist(m.x, m.y); d < 22 && d > 3 {
			m.x += (s.px - m.x) / d * 2
			m.y += (s.py - m.y) / d * 2
		}
	}
	// corpses fade
	keep := s.mons[:0]
	for _, m := range s.mons {
		if m.mode == 0 {
			m.mode = 12
		} else if m.mode == 12 {
			m.corpse++
			if m.corpse > 6 {
				continue
			}
		}
		keep = append(keep, m)
	}
	s.mons = keep

	alive := s.alive()
	var near *simMon
	for _, m := range alive {
		if d := s.dist(m.x, m.y); d < 24 && (near == nil || d < s.dist(near.x, near.y)) {
			near = m
		}
	}
	switch {
	case near != nil && s.phase != "return":
		if s.phase != "fight" {
			s.setPhase("fight", "fighting", near.name)
		}
		d := s.dist(near.x, near.y)
		if d > 10 {
			s.walk(near.x, near.y, 5)
			return
		}
		dmg := 14 + s.level*2 + s.rng.Intn(12)
		if s.mana >= 3 {
			s.mana -= 3
		} else {
			dmg /= 3
		}
		near.hp -= dmg
		if near.hp <= 0 {
			near.mode = 0
			s.onKill(near)
		}
	case len(s.ground) > 0 && s.phase != "return":
		if s.phase != "loot" {
			s.setPhase("loot", "looting", "")
		}
		g := s.ground[0]
		for _, o := range s.ground {
			if s.dist(o.x, o.y) < s.dist(g.x, g.y) {
				g = o
			}
		}
		if s.walk(g.x, g.y, 5) {
			s.pickup(g)
		}
	case len(alive) <= 4 || s.runTicks > 900 || s.phase == "return" || s.invFull():
		if s.phase != "return" {
			s.setPhase("return", "exploring", "returning to town")
		}
		if s.walk(float64(s.field.exits[0].X+1), float64(s.field.exits[0].Y), 6) {
			s.tr.RunEnd()
			s.setPhase("loading", "loading", "Rogue Encampment")
		}
	default:
		if s.phase != "explore" {
			s.setPhase("explore", "exploring", "Blood Moor")
		}
		if !s.hasTarget || s.dist(s.tx, s.ty) < 3 {
			m := alive[s.rng.Intn(len(alive))]
			s.tx, s.ty, s.hasTarget = m.x, m.y, true
		}
		s.walk(s.tx, s.ty, 5)
	}
}

func (s *Sim) invFull() bool {
	probe := append([]status.Item(nil), s.inv...)
	return !s.place(&probe, 10, 4, status.Item{W: 2, H: 3})
}

func (s *Sim) onKill(m *simMon) {
	s.exp += uint32(float64(18+s.rng.Intn(10)) * (1 + float64(s.level)*0.3))
	for {
		next, ok := telemetry.DefaultExpForLevel(s.level + 1)
		if !ok || s.exp < next {
			break
		}
		s.level++
		s.statPts += 5
		s.skillPts++
		s.life, s.mana = s.maxLife(), s.maxMana()
	}
	s.gold += s.rng.Intn(40)
	for i := 0; i < 2; i++ {
		if s.rng.Intn(5) == 0 {
			s.ground = append(s.ground, &simGround{it: s.randomItem(), x: m.x + float64(i*2), y: m.y + 1})
		}
	}
}

func (s *Sim) pickup(g *simGround) {
	out := s.ground[:0]
	for _, o := range s.ground {
		if o != g {
			out = append(out, o)
		}
	}
	s.ground = out
	if !s.tryEquip(g.it) {
		s.place(&s.inv, 10, 4, g.it)
	}
}

func (s *Sim) observe(dead, loading bool) {
	f := telemetry.Frame{InGame: !loading, InTown: s.inTown(), Area: s.lvl.id, Level: s.level, Exp: s.exp, Gold: s.gold, StashGld: s.stashGold,
		Dead: dead, Belt: telemetry.Belt{Health: s.hpPots, Mana: s.mpPots}}
	for _, m := range s.mons {
		f.Monsters = append(f.Monsters, telemetry.Monster{ID: m.id, TxtNo: m.txt, Name: m.name, Mode: m.mode, X: int(m.x), Y: int(m.y), Friendly: m.friendly})
	}
	for _, g := range s.ground {
		f.Ground = append(f.Ground, telemetry.GroundItem{ID: g.it.ID, Name: simNamer{}.ItemName(g.it), Quality: g.it.Quality, X: int(g.x), Y: int(g.y)})
	}
	s.tr.Observe(f)
}
