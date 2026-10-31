package combat

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Druid and Assassin upkeep (docs/summon.md "Druid and Assassin"). Same method as the Necromancer: the census of
// friendly units (IsFriendly, docs/friendly.md) decides what is missing, and a cast that paid mana but raised nothing
// records the cap (necroState.capSeen, kinds are distinct from the Necromancer's). Monster class ids are V from the
// monstats extract in friendly.go (which AI rows exist); which skill makes which class is U, no druid or assassin
// summon was ever cast live.
const (
	classRaven               = 419
	classSpiritWolf          = 420 // U: 420 is the Spirit Wolf, 421 the Dire Wolf (Fenris); both AI DruidWolf (V)
	classDireWolf            = 421
	classGrizzly             = 428      // AI DruidBear (V)
	classSpiritLo, classSpHi = 422, 424 // AI Totem: Oak Sage, Heart of Wolverine, Spirit of Barbs (which is which: U)
	classVineLo, classVineHi = 425, 427 // AI Vines and CycleOfLife
	classShadowWarrior       = 417
	classShadowMaster        = 418
	classTrapLo, classTrapHi = 410, 415 // AI AssassinSentry 410-412 and 415, BladeCreeper 413, InvisoPet 414
)

// DruidPets is the census of a Druid's summons by kind.
type DruidPets struct {
	Ravens, SpiritWolves, DireWolves, Grizzlies, Spirits, Vines int
}

// Total is every counted pet.
func (d DruidPets) Total() int {
	return d.Ravens + d.SpiritWolves + d.DireWolves + d.Grizzlies + d.Spirits + d.Vines
}

// friendlyUnits lists living friendly monsters that are not townspeople or hirelings.
func friendlyUnits(snap *d2.Snapshot, cat Catalog) []d2.Unit {
	if snap == nil {
		return nil
	}
	fr := Friendlies(snap, cat)
	var out []d2.Unit
	for _, u := range snap.Monsters {
		if w, ok := fr[u.ID]; ok && u.Alive() && w != WhyNPC && w != WhyHirelingAI {
			out = append(out, u)
		}
	}
	return out
}

// CountDruidPets counts the Druid's summons.
func CountDruidPets(snap *d2.Snapshot, cat Catalog) DruidPets {
	var d DruidPets
	for _, u := range friendlyUnits(snap, cat) {
		switch c := u.Class; {
		case c == classRaven:
			d.Ravens++
		case c == classSpiritWolf:
			d.SpiritWolves++
		case c == classDireWolf:
			d.DireWolves++
		case c == classGrizzly:
			d.Grizzlies++
		case c >= classSpiritLo && c <= classSpHi:
			d.Spirits++
		case c >= classVineLo && c <= classVineHi:
			d.Vines++
		}
	}
	return d
}

// AssassinPets is the census of an Assassin's traps and shadows.
type AssassinPets struct {
	Traps   []d2.Unit // sentries and blade creepers, with positions
	Shadows int       // Shadow Warrior or Shadow Master
}

// CountAssassinPets counts traps (with positions) and shadows.
func CountAssassinPets(snap *d2.Snapshot, cat Catalog) AssassinPets {
	var a AssassinPets
	for _, u := range friendlyUnits(snap, cat) {
		switch c := u.Class; {
		case c >= classTrapLo && c <= classTrapHi:
			a.Traps = append(a.Traps, u)
		case c == classShadowWarrior || c == classShadowMaster:
			a.Shadows++
		}
	}
	return a
}

func isDruid(p *d2.Player) bool    { return p != nil && p.Class == d2.Druid }
func isAssassin(p *d2.Player) bool { return p != nil && p.Class == d2.Assassin }

// petPlan is one summon the Druid wants: skill, census kind, how many it can hold at this skill level and how many
// stand now. Caps: V docs/classes/druid.md (up to 5 spirit wolves, 3 dire wolves, 1 bear, 5 ravens); the count equals
// the skill level up to that cap is U (the real cap is learned by capSeen).
type petPlan struct {
	name, kind string
	max, have  int
}

func druidPlans(have DruidPets) []petPlan {
	return []petPlan{
		{"Oak Sage", "spirit", 1, 0}, {"Heart of Wolverine", "spirit", 2, 0}, {"Spirit of Barbs", "spirit", 3, 0},
		{"Summon Grizzly", "bear", 1, have.Grizzlies}, {"Summon Fenris", "dire", 3, have.DireWolves},
		{"Summon Spirit Wolf", "wolf", 5, have.SpiritWolves}, {"Raven", "raven", 5, have.Ravens},
	}
}

// planPet picks the next Druid summon: spirits one by one (the n-th spirit skill is wanted while fewer than n
// spirits stand), then bear, wolves, ravens. Only learned, bound, affordable summons count.
func (f *Fighter) planPet(snap *d2.Snapshot) (SkillSpec, bool) {
	p := snap.Player
	if !isDruid(p) || p.InTown() {
		return SkillSpec{}, false
	}
	f.necro.init()
	known := KnownSkills(p, f.cat, f.keys)
	have := CountDruidPets(snap, f.cat)
	for _, pl := range druidPlans(have) {
		s, ok := known[pl.name]
		if !ok || s.Kind != KindSummon || (s.Key == "" && p.RightSkill != s.ID) || s.Cost > p.Mana-p.MaxMana/10 {
			continue
		}
		n := pl.have
		if pl.kind == "spirit" {
			n = have.Spirits
		}
		want := min(s.Level, pl.max)
		if pl.kind == "spirit" {
			want = pl.max // the n-th spirit skill is wanted while fewer than n spirits stand
		}
		if n >= want {
			continue
		}
		if cap, capped := f.necro.capSeen[pl.kind]; capped {
			if n < cap {
				delete(f.necro.capSeen, pl.kind)
			} else {
				continue
			}
		}
		return s, true
	}
	return SkillSpec{}, false
}

// castPet casts one Druid summon at the caster and records a cap when mana was spent and the count did not grow.
func (f *Fighter) castPet(ctx context.Context, snap *d2.Snapshot, sp SkillSpec) (bool, error) {
	p := snap.Player
	before := CountDruidPets(snap, f.cat)
	if err := f.cast(snap, sp, input.Point{X: p.X, Y: p.Y}); err != nil {
		return false, err
	}
	if err := f.sleep(ctx, ms(f.tune().CastGapMs)+ms(500)); err != nil {
		return false, err
	}
	after, err := f.read(ctx)
	if err != nil {
		return true, err
	}
	now := CountDruidPets(after, f.cat)
	paid := after.Player.Mana < p.Mana-sp.Cost/2
	grew := now.Total() > before.Total()
	kind := petKind(sp.Name)
	if kind != "" && paid && !grew {
		f.necro.capSeen[kind] = map[string]int{"spirit": now.Spirits, "bear": now.Grizzlies, "dire": now.DireWolves,
			"wolf": now.SpiritWolves, "raven": now.Ravens}[kind]
	}
	f.info("pet", "skill", sp.Name, "pets", now.Total(), "grew", grew, "paid", paid)
	return true, nil
}

func petKind(name string) string {
	switch name {
	case "Oak Sage", "Heart of Wolverine", "Spirit of Barbs":
		return "spirit"
	case "Summon Grizzly":
		return "bear"
	case "Summon Fenris":
		return "dire"
	case "Summon Spirit Wolf":
		return "wolf"
	case "Raven":
		return "raven"
	}
	return ""
}

// KeepPets casts the missing Druid summons (no corpse, no walking), at most maxSummonsAt casts, never in town or
// while dead. Returns the number of casts.
func (f *Fighter) KeepPets(ctx context.Context) (int, error) {
	casts := 0
	for casts < maxSummonsAt {
		snap, err := f.read(ctx)
		if err != nil {
			return casts, err
		}
		if !isDruid(snap.Player) || snap.Player.InTown() || isDead(snap.Player) {
			return casts, nil
		}
		sp, ok := f.planPet(snap)
		if !ok {
			return casts, nil
		}
		if done, err := f.castPet(ctx, snap, sp); err != nil || !done {
			return casts, err
		}
		casts++
	}
	return casts, nil
}

// druidTick is the fight-loop hook: one missing summon per tick while life is above 40%. true = acted.
func (f *Fighter) druidTick(ctx context.Context, snap *d2.Snapshot) (bool, error) {
	p := snap.Player
	if !isDruid(p) || p.InTown() || p.MaxLife > 0 && p.Life*100 < p.MaxLife*40 {
		return false, nil
	}
	if sp, ok := f.planPet(snap); ok {
		return f.castPet(ctx, snap, sp)
	}
	return false, nil
}

// classTick runs the class hook of the fight loop (necroTick, druidTick, assassinTick).
func (f *Fighter) classTick(ctx context.Context, snap *d2.Snapshot) (bool, error) {
	switch p := snap.Player; {
	case isNecro(p):
		return f.necroTick(ctx, snap)
	case isDruid(p):
		return f.druidTick(ctx, snap)
	case isAssassin(p):
		return f.assassinTick(ctx, snap)
	}
	return false, nil
}

// classUpkeep runs after a Clear: raise, summon or re-cast what is missing.
func (f *Fighter) classUpkeep(ctx context.Context) {
	snap, err := f.read(ctx)
	if err != nil {
		return
	}
	switch p := snap.Player; {
	case isNecro(p):
		_, _ = f.KeepSummons(ctx)
	case isDruid(p):
		_, _ = f.KeepPets(ctx)
	case isAssassin(p):
		_, _ = f.KeepShadow(ctx)
	}
}
