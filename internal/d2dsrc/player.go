package d2dsrc

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
)

// builder assembles one snapshot.
type builder struct {
	s    *Source
	rp   *rep
	res  *Result
	snap *d2.Snapshot
	st   d2dctl.State
	pl   *d2dctl.Player // "debug players" row of the local player, nil if unavailable
	lvl  *d2dctl.LevelInfo
	// origin in subtiles added to positions when Options.AddOrigin
	ox, oy int
	held   string // dump ui held: base code of the item on the cursor
}

func (b *builder) pos(cx, cy float64) (int, int) {
	return d2dctl.SubTile(cx) + b.ox, d2dctl.SubTile(cy) + b.oy
}

// head fills Player, UI, merc and the clock from the dump sections player,stats,skills,ui,merc,time.
func (b *builder) head(ctx context.Context, d obj) {
	b.clock(d.sub("time"))
	p := d.sub("player")
	if p == nil {
		b.rp.unknown("player", "dump has no player section")
		return
	}
	b.levelOrigin(ctx, false)
	pl := &d2.Player{}
	b.snap.Player = pl
	b.playerCore(pl, p)
	b.playerStats(pl, d.sub("stats"), p)
	b.playerSkills(pl, d.sub("skills"), p)
	b.ui(d.sub("ui"))
	b.merc(d.sub("merc"))
}

func (b *builder) levelOrigin(ctx context.Context, force bool) {
	if !b.s.o.AddOrigin && !force {
		return
	}
	l, err := b.s.c.Level(ctx)
	if err != nil {
		b.rp.unknown("coords.origin", "debug level failed: "+err.Error())
		return
	}
	b.lvl = &l
	if b.s.o.AddOrigin {
		b.ox, b.oy = 5*l.OriginX, 5*l.OriginY
		b.res.Extras.Coords = "absolute(origin*5, U)"
		b.rp.derived("coords.origin", "act-tile origin of debug level x5; unit of the origin is U")
	}
}

func (b *builder) clock(t obj) {
	if t == nil {
		b.rp.unknown("time", "dump has no time section (stock d2d: only info frame=)")
		return
	}
	c := Clock{Valid: true}
	c.Frame, _ = t.int("frame")
	c.GameMS, _ = t.int("game_ms")
	c.Tick, _ = t.int("tick")
	c.Lockstep, _ = t.bool("lockstep")
	c.Scale, _ = t.num("scale")
	b.res.Extras.Clock = c
	b.rp.have("time")
}

func (b *builder) playerCore(pl *d2.Player, p obj) {
	rp := b.rp
	if n, ok := p.str("name"); ok {
		pl.Name = n
		rp.have("player.name")
	} else {
		rp.unknown("player.name", "missing in dump player")
	}
	if cn, ok := p.str("class"); ok {
		if c, ok := classOf(cn); ok {
			pl.Class = c
			rp.have("player.class")
		} else {
			rp.unknown("player.class", fmt.Sprintf("class name %q not recognised", cn))
		}
	} else {
		rp.unknown("player.class", "missing in dump player")
	}
	if b.pl != nil {
		pl.ID = uint32(b.pl.ID)
		rp.have("player.id")
	} else {
		rp.unknown("player.id", "debug players gave no row")
	}
	if x, ok := p.num("x"); ok {
		if y, ok := p.num("y"); ok {
			pl.X, pl.Y = b.pos(x, y)
			rp.have("player.pos")
		}
	}
	if !rp.Has("player.pos") && b.pl != nil {
		pl.X, pl.Y = b.pos(b.pl.X, b.pl.Y)
		rp.have("player.pos")
	}
	if !rp.Has("player.pos") {
		rp.unknown("player.pos", "no x/y in dump or debug players")
	}
	if area, ok := p.int("area"); ok {
		pl.Area, pl.Act = area, ActOf(area)
		rp.have("player.area")
		rp.derived("player.act", "from the area id (act boundaries by town ids)")
	} else {
		rp.unknown("player.area", "missing in dump player")
		rp.unknown("player.act", "needs the area")
	}
	pl.Mode = -1
	if m, ok := p.str("mode"); ok {
		if n, ok := plrModes[m]; ok {
			pl.Mode = n
			rp.derived("player.mode", "mode code "+m+" via PlrMode.txt order (d2d's numbering U)")
		} else {
			rp.unknown("player.mode", fmt.Sprintf("mode code %q not in the player mode table", m))
		}
	} else {
		rp.unknown("player.mode", "missing in dump player")
	}
	pl.Level, _ = p.int("level")
	if _, ok := p.int("level"); ok {
		rp.have("player.level")
	} else {
		rp.unknown("player.level", "missing")
	}
	if v, ok := p.num("exp"); ok {
		pl.Exp = uint32(v)
		rp.have("player.exp")
	} else {
		rp.unknown("player.exp", "missing")
	}
	if v, ok := p.int("gold"); ok {
		pl.Gold = v
		rp.have("player.gold")
	} else {
		rp.unknown("player.gold", "missing")
	}
	if v, ok := p.int("stash_gold"); ok {
		pl.StashGold = v
		rp.have("player.stash_gold")
	} else {
		rp.unknown("player.stash_gold", "not in state; dump only")
	}
	pl.Life, pl.MaxLife, _ = p.pair("life")
	pl.Mana, pl.MaxMana, _ = p.pair("mana")
	pl.Stamina, pl.MaxStamina, _ = p.pair("stamina")
	if pl.MaxStamina >= 10000 { // whole stamina never gets near this; "state" prints the raw 8.8 value (V)
		pl.Stamina >>= 8
		pl.MaxStamina >>= 8
		rp.derived("player.stamina_units", "dump pair looked like raw 8.8 fixed point (>= 10000), shifted down")
	}
	for _, k := range []string{"life", "mana", "stamina"} {
		if _, _, ok := p.pair(k); ok {
			rp.have("player." + k)
		} else {
			rp.unknown("player."+k, "missing")
		}
	}
	if v, ok := p.int("stat_pts"); ok {
		pl.StatPoints = v
		rp.have("player.stat_points")
	} else {
		rp.unknown("player.stat_points", "missing")
	}
	if v, ok := p.int("skill_pts"); ok {
		pl.SkillPoints = v
		rp.have("player.skill_points")
	} else {
		rp.unknown("player.skill_points", "missing")
	}
	if v, ok := p.int("lskill"); ok {
		pl.LeftSkill = v
		rp.have("player.left_skill")
	} else {
		rp.unknown("player.left_skill", "missing")
	}
	if v, ok := p.int("rskill"); ok {
		pl.RightSkill = v
		rp.have("player.right_skill")
	} else {
		rp.unknown("player.right_skill", "missing")
	}
	if v, ok := p.int("difficulty"); ok && v >= 0 && v <= 2 {
		pl.Difficulty = d2.Difficulty(v)
		rp.have("player.difficulty")
	} else {
		rp.unknown("player.difficulty", "missing or out of range")
	}
	rp.unknown("player.states", "no state list in dump (poison, curses, buffs); poisoned() is blind")
	rp.unknown("player.hotkeys", "no hotkey binding readback; semantic input selects skills by id")
}

// playerStats fills attributes, resists and the Stats vector from the dump stats section. d2d only knows
// base attributes, the panel bonuses and the resist panel, so Stats holds only what is known: Stats.Has(id)
// is false for everything else (FCR, FRW, ... gear and aura sums).
func (b *builder) playerStats(pl *d2.Player, st, p obj) {
	rp := b.rp
	pl.Stats = d2.Stats{}
	put := func(id int, v int) { pl.Stats[uint32(id)] = int32(v) }
	fixed := func(id int, v int) { pl.Stats[uint32(id)] = int32(v) << 8 }
	if rp.Has("player.life") {
		fixed(d2.StatLife, pl.Life)
		fixed(d2.StatMaxLife, pl.MaxLife)
	}
	if rp.Has("player.mana") {
		fixed(d2.StatMana, pl.Mana)
		fixed(d2.StatMaxMana, pl.MaxMana)
	}
	if rp.Has("player.stamina") {
		fixed(d2.StatStamina, pl.Stamina)
		fixed(d2.StatMaxStamina, pl.MaxStamina)
	}
	if rp.Has("player.level") {
		put(d2.StatLevel, pl.Level)
	}
	if rp.Has("player.exp") {
		put(d2.StatExp, int(pl.Exp))
	}
	if rp.Has("player.gold") {
		put(d2.StatGold, pl.Gold)
	}
	if rp.Has("player.stash_gold") {
		put(d2.StatStashGold, pl.StashGold)
	}
	if rp.Has("player.stat_points") {
		put(d2.StatPoints, pl.StatPoints)
	}
	if rp.Has("player.skill_points") {
		put(d2.StatSkillPts, pl.SkillPoints)
	}
	if st == nil {
		for _, f := range []string{"player.str", "player.dex", "player.vit", "player.ene", "player.resist"} {
			rp.unknown(f, "dump has no stats section")
		}
		rp.unknown("player.stats.effective", "no stats section")
		return
	}
	base := st.sub("base")
	bonus, bok := st.ints("bonus")
	attrs := []struct {
		key  string
		id   int
		dst  *int
		name string
	}{
		{"str", d2.StatStrength, &pl.Strength, "player.str"},
		{"dex", d2.StatDexterity, &pl.Dexterity, "player.dex"},
		{"vit", d2.StatVitality, &pl.Vitality, "player.vit"},
		{"ene", d2.StatEnergy, &pl.Energy, "player.ene"},
	}
	for i, a := range attrs {
		v, ok := base.int(a.key)
		if !ok {
			rp.unknown(a.name, "missing in stats.base")
			continue
		}
		*a.dst = v
		how := "base only (no gear bonus)"
		if bok && len(bonus) == 4 {
			*a.dst = v + bonus[i]
			how = "base + stats.bonus[i]; bonus order str,dex,vit,ene is U"
		}
		put(a.id, *a.dst)
		rp.derived(a.name, how)
	}
	if res := st.sub("res"); res != nil {
		pl.Resist.Fire, _ = res.int("fire")
		pl.Resist.Cold, _ = res.int("cold")
		pl.Resist.Lightning, _ = res.int("light")
		pl.Resist.Poison, _ = res.int("poison")
		put(d2.StatFireRes, pl.Resist.Fire)
		put(d2.StatColdRes, pl.Resist.Cold)
		put(d2.StatLightRes, pl.Resist.Lightning)
		put(d2.StatPoisonRes, pl.Resist.Poison)
		rp.derived("player.resist", "stats.res panel values (after difficulty penalty and caps? U)")
	} else {
		rp.unknown("player.resist", "missing in stats")
	}
	rp.unknown("player.resist.magic", "panel shows 4 resists; no magic resist")
	rp.unknown("player.stats.effective", "no effective stat vector (FCR, FRW, IAS, FHR, MF ...): Stats holds only the keys d2d gives")
}

// playerSkills maps skills.base_levels (class order) to ids. Bonus levels are unknown.
func (b *builder) playerSkills(pl *d2.Player, sk, p obj) {
	rp := b.rp
	if sk == nil {
		rp.unknown("player.skills", "dump has no skills section")
		return
	}
	lv, ok := sk.ints("base_levels")
	if !ok || int(pl.Class) >= len(classSkillBase) || !rp.Has("player.class") {
		rp.unknown("player.skills", "no base_levels or class unknown")
		return
	}
	base := classSkillBase[pl.Class]
	for i, l := range lv {
		id := base + i
		if l > 0 || id == pl.LeftSkill || id == pl.RightSkill {
			pl.Skills = append(pl.Skills, d2.Skill{ID: id, Level: l})
		}
	}
	rp.derived("player.skills", "skill id = class base id + index in base_levels (ascending Skills.txt order U)")
	rp.unknown("player.skills.bonus", "item/aura skill bonus not in dump (Bonus stays 0)")
}

// ui maps the dump ui flags onto our panel flag array. Panels d2d does not report stay 0 and are listed.
func (b *builder) ui(u obj) {
	rp := b.rp
	if u == nil {
		rp.unknown("ui", "dump has no ui section")
		return
	}
	set := func(key string, idx int) {
		v, ok := u.bool(key)
		if !ok {
			rp.unknown("ui."+key, "missing in dump ui")
			return
		}
		if v {
			b.snap.UI[idx] = 1
		}
		rp.have("ui." + key)
	}
	b.held, _ = u.str("held")
	set("stash", d2.UIStash)
	set("cube", d2.UICube)
	set("npc_menu", d2.UINPCMenu)
	set("vendor", d2.UIShop)
	set("skill_tree", d2.UISkillTree)
	set("waypoint", d2.UIWaypoint)
	set("game_menu", d2.UIEscMenu)
	set("mini_panel", d2.UIMiniPanel)
	set("automap", d2.UIAutomap)
	if r := u.sub("rects"); r != nil { // lab3+: only open panels appear, inv_grid is the backpack (V live in D2DVERBS)
		_, open := r["inv_grid"]
		if open {
			b.snap.UI[d2.UIInventory] = 1
		}
		rp.have("ui.inventory")
	} else if b.s.o.NoInvHint {
		rp.unknown("ui.inventory", "dump ui has no rects (lab2)")
	} else {
		if b.s.c.InventoryHint() {
			b.snap.UI[d2.UIInventory] = 1
		}
		rp.derived("ui.inventory", "no ui.rects (lab2): tracked from the i/Escape keys sent, U")
	}
	if b.s.c.CharacterHint() {
		b.snap.UI[d2.UICharacter] = 1
	}
	rp.derived("ui.character", "tracked from the c/Escape keys sent, U")
	for _, f := range []string{"quest", "party", "trade", "merc", "chat"} {
		rp.unknown("ui."+f, "dump ui has no such flag: CloseAll would not see this panel")
	}
}

func (b *builder) merc(m obj) {
	rp := b.rp
	if m == nil {
		rp.unknown("merc", "dump has no merc section")
		return
	}
	present, ok := m.bool("present")
	if !ok {
		rp.unknown("merc", "no present flag")
		return
	}
	rp.have("merc.present")
	if !present {
		return
	}
	code, _ := m.str("code")
	dead, _ := m.bool("dead")
	u := d2.Unit{Type: d2.TypeMonster, ID: MercID, Class: -1, Mode: d2.MonModeStand, Monster: &d2.MonsterInfo{}}
	if cls, ok := mercClass[code]; ok {
		u.Class = cls
		rp.have("merc.class")
	} else {
		rp.unknown("merc.class", fmt.Sprintf("merc code %q has no known monstats class (only RG is V)", code))
	}
	if dead {
		u.Mode = d2.MonModeDead
	}
	if x, ok := m.num("x"); ok {
		if y, ok := m.num("y"); ok {
			u.X, u.Y = b.pos(x, y)
			rp.have("merc.pos")
		}
	}
	if c, mx, ok := m.pair("life"); ok {
		u.Monster.Life, u.Monster.MaxLife = c, mx
		if mx > 0 {
			u.Monster.LifePct = c * 100 / mx
		}
		rp.have("merc.life")
	}
	rp.unknown("merc.level", "not in dump")
	rp.unknown("merc.name", "not in dump")
	rp.unknown("merc.aura", "not in dump")
	b.res.Extras.Merc = &MercExtra{Code: code, Dead: dead}
	b.snap.Monsters = append(b.snap.Monsters, u)
}

// degradedHead is the stock-d2d path: state + level + attack verbs only. It fills what they carry and
// leaves the rest unknown.
func (b *builder) degradedHead(ctx context.Context) error {
	rp := b.rp
	st := b.st
	pl := &d2.Player{Name: st.Name, Level: st.Level, Gold: st.Gold, Exp: uint32(st.Exp), Mode: -1}
	b.snap.Player = pl
	if c, ok := classOf(st.Class); ok {
		pl.Class = c
		rp.have("player.class")
	} else {
		rp.unknown("player.class", fmt.Sprintf("state class %q not recognised", st.Class))
	}
	if st.Name != "" {
		rp.have("player.name")
	}
	rp.have("player.level", "player.exp", "player.gold")
	pl.Life, pl.MaxLife = st.Life.Cur, st.Life.Max
	pl.Mana, pl.MaxMana = st.Mana.Cur, st.Mana.Max
	rp.have("player.life", "player.mana")
	if v, ok := st.Int("lskill"); ok {
		pl.LeftSkill = v
		rp.have("player.left_skill")
	}
	if v, ok := st.Int("rskill"); ok {
		pl.RightSkill = v
		rp.have("player.right_skill")
	}
	if st.Difficulty >= 0 && st.Difficulty <= 2 {
		pl.Difficulty = d2.Difficulty(st.Difficulty)
		rp.have("player.difficulty")
	}
	if b.pl != nil {
		pl.ID = uint32(b.pl.ID)
		pl.X, pl.Y = b.pos(b.pl.X, b.pl.Y)
		rp.have("player.id", "player.pos")
	}
	if l, err := b.s.c.Level(ctx); err == nil {
		b.lvl = &l
		pl.Area, pl.Act = l.ID, ActOf(l.ID)
		rp.have("player.area")
		rp.derived("player.act", "from the area id")
		if b.s.o.AddOrigin {
			b.ox, b.oy = 5*l.OriginX, 5*l.OriginY
			b.res.Extras.Coords = "absolute(origin*5, U)"
			// positions read before the origin was known: redo
			if b.pl != nil {
				pl.X, pl.Y = b.pos(b.pl.X, b.pl.Y)
			}
		}
	} else if !isCmdErr(err) {
		return err
	}
	for _, f := range []string{"player.stash_gold", "player.stamina", "player.stat_points", "player.skill_points", "player.str", "player.dex",
		"player.vit", "player.ene", "player.resist", "player.skills", "player.states", "player.mode", "ui", "merc", "time", "player.stats.effective"} {
		rp.unknown(f, "stock d2d has no such verb (needs the dump fork)")
	}
	return nil
}
