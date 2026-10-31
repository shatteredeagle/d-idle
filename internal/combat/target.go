package combat

import (
	"math"
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Monster modes that mean the unit is gone (d2.MonModeDeath = 0, d2.MonModeDead = 12).

// Affix is a monster modifier (monumod.txt row ids as stored in MonsterData.Enchants). The ids were read from the
// real monumod.txt (row order = id: 5 strong, 6 fast, 7 curse, 8 resist, 9 fire, 17 lightning, 18 cold, 25 manahit,
// 26 teleport, 27 spectralhit, 28 stoneskin, 29 multishot, 30 aura). That the client's Enchants byte array holds
// these ids is from D2BS and UNVERIFIED live (no unique monster has been seen yet).
type Affix uint32

const (
	AffixExtraStrong Affix = 1 << iota
	AffixExtraFast
	AffixCursed
	AffixMagicResistant
	AffixFireEnchanted
	AffixLightningEnchanted
	AffixColdEnchanted
	AffixManaBurn
	AffixTeleportation
	AffixSpectralHit
	AffixStoneSkin
	AffixMultiShot
	AffixAura // aura enchanted; which aura is not stored in the enchant list
)

var affixByMod = map[uint8]Affix{
	5: AffixExtraStrong, 6: AffixExtraFast, 7: AffixCursed, 8: AffixMagicResistant, 9: AffixFireEnchanted,
	17: AffixLightningEnchanted, 18: AffixColdEnchanted, 25: AffixManaBurn, 26: AffixTeleportation,
	27: AffixSpectralHit, 28: AffixStoneSkin, 29: AffixMultiShot, 30: AffixAura,
}

func affixesOf(m *d2.MonsterInfo) Affix {
	var a Affix
	if m == nil {
		return a
	}
	ids, _ := m.EnchantIDs() // unknown (unverified offsets) reads as no affixes
	for _, id := range ids {
		a |= affixByMod[id]
	}
	return a
}

func (a Affix) String() string {
	names := []string{"extra-strong", "extra-fast", "cursed", "magic-resistant", "fire-enchanted", "lightning-enchanted",
		"cold-enchanted", "mana-burn", "teleportation", "spectral-hit", "stone-skin", "multiple-shots", "aura-enchanted"}
	var out []string
	for i, n := range names {
		if a&(1<<uint(i)) != 0 {
			out = append(out, n)
		}
	}
	return strings.Join(out, ",")
}

// killFirstAffixes are fought before the rest of the pack (docs/plan/data/danger-table.json, "kill_first").
const killFirstAffixes = AffixCursed | AffixManaBurn | AffixMultiShot | AffixAura

// avoidHell are not fought in Hell below avoidHellLevel (danger table: avoid / skip).
const (
	avoidHell      = AffixCursed | AffixManaBurn | AffixLightningEnchanted | AffixSpectralHit
	avoidHellLevel = 75
)

// killFirstNames are monster id/name fragments that raise or buff others and die first (04-combat.md section 2).
var killFirstNames = []string{"shaman", "oblivion", "unraveler", "bloodraven"}

// explodeNames are monsters that explode or drain when near: keep range (danger table "Dolls", "Souls").
var explodeNames = []string{"doll", "soul"}

// Target is a hostile monster worth a decision.
type Target struct {
	Unit     d2.Unit
	Mon      *data.Monster // nil for an unknown class
	Name     string
	Dist     float64 // subtiles from the player
	Affixes  Affix
	Elite    bool // unique, champion or minion of a unique
	Boss     bool // unique with a name, or a monstats boss; never a champion (docs/champions.md)
	Champion bool // flag byte 0x0c champion (no minions): Elite, not Boss; selects the tightened early potion lines
	LifePct  int  // 0-100, 100 when unknown
	KillFast bool // support monster or kill-first affix
	Explodes bool
	Danger   float64 // death-history weight
	Avoid    bool    // the danger table says do not fight it at this level / difficulty
	// Res are the effective resistances per damage type (stat list when readable, else monstats); HasRes tells if set.
	Res    [6]int
	HasRes bool
}

// Immune reports whether the target takes no damage of this type.
func (t *Target) Immune(d data.DamageType) bool { return t.HasRes && t.Res[d] >= 100 }

// ImmuneAll reports whether the target is immune to every given type.
func (t *Target) ImmuneAll(ds ...data.DamageType) bool {
	if len(ds) == 0 {
		return false
	}
	for _, d := range ds {
		if !t.Immune(d) {
			return false
		}
	}
	return true
}

// HostileCtx carries what the hostile filter needs besides the snapshot.
type HostileCtx struct {
	Cat      Catalog
	Diff     d2.Difficulty
	Resists  func(u d2.Unit, m *data.Monster) ([6]int, bool) // optional: stat-list resistances
	ClassLvl int
	Weights  DeathWeights // lessons from past deaths
}

func dist(ax, ay, bx, by int) float64 { return math.Hypot(float64(ax-bx), float64(ay-by)) }

// Champions spawn in groups of 3-5 and have about 3x life and +50% damage: a group of this many in view is too much for
// a character under championGroupLevel (SME rule of thumb, EXP). Only stricter than before: the targets are ignored like
// any Avoid target and the Kill target path is unaffected.
const (
	championGroupMin   = 3
	championGroupLevel = 8
)

func avoidChampionGroup(ts []Target, clvl int) {
	if clvl <= 0 || clvl >= championGroupLevel {
		return
	}
	n := 0
	for _, t := range ts {
		if t.Champion && !t.Boss {
			n++
		}
	}
	if n < championGroupMin {
		return
	}
	for i := range ts {
		if ts[i].Champion && !ts[i].Boss {
			ts[i].Avoid = true
		}
	}
}

// Hostiles picks the monsters in the snapshot that should be fought: alive, not friendly (IsFriendly: town NPCs,
// mercenaries, minions, pets, summons), not non-killable. Nothing is hostile in town.
// Sorted nearest first; use Rank for the fight order.
func Hostiles(snap *d2.Snapshot, hc HostileCtx) []Target {
	if snap == nil || snap.Player == nil || snap.State != d2.StateInGame || snap.Player.InTown() {
		return nil
	}
	p := snap.Player
	friendly := Friendlies(snap, hc.Cat)
	var out []Target
	for _, u := range snap.Monsters {
		if _, ok := friendly[u.ID]; ok || !u.Alive() || u.ID == p.ID {
			continue
		}
		mon := hc.Cat.Monster(u.Class)
		if mon != nil && (mon.NPC || !mon.Killable) {
			continue
		}
		t := Target{Unit: u, Mon: mon, Dist: dist(p.X, p.Y, u.X, u.Y), LifePct: 100}
		t.Name = hc.Cat.MonsterName(mon)
		if u.Monster != nil {
			t.Affixes = affixesOf(u.Monster)
			t.Elite = u.Monster.IsUnique() || u.Monster.IsChampion() || u.Monster.IsMinion()
			t.Champion = u.Monster.IsChampion() && !u.Monster.IsUnique()
			if u.Monster.IsUnique() && u.Monster.Name != "" {
				t.Boss = true
				t.Name = u.Monster.Name
			}
			if u.Monster.LifePct > 0 {
				t.LifePct = min(u.Monster.LifePct, 100)
			}
		}
		if mon != nil && mon.Boss {
			t.Boss, t.Elite = true, true
		}
		key := ""
		if mon != nil {
			key = strings.ToLower(mon.Key + " " + t.Name)
		}
		for _, n := range killFirstNames {
			if strings.Contains(key, n) {
				t.KillFast = true
			}
		}
		for _, n := range explodeNames {
			if strings.Contains(key, n) {
				t.Explodes = true
			}
		}
		if t.Affixes&killFirstAffixes != 0 {
			t.KillFast = true
		}
		if hc.Diff == d2.Hell && t.Affixes&avoidHell != 0 && hc.ClassLvl < avoidHellLevel && !t.Boss {
			t.Avoid = true
		}
		if mon != nil {
			t.Danger = hc.Weights.of(mon.Key, t.Affixes)
		} else {
			t.Danger = hc.Weights.of("", t.Affixes)
		}
		if t.Danger >= avoidWeight && !t.Boss {
			t.Avoid = true
		}
		t.Res, t.HasRes = resistsOf(u, mon, hc)
		out = append(out, t)
	}
	avoidChampionGroup(out, hc.ClassLvl)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Dist != out[j].Dist {
			return out[i].Dist < out[j].Dist
		}
		return out[i].Unit.ID < out[j].Unit.ID
	})
	return out
}

// resistsOf prefers the live stat list (it includes curses and difficulty penalties) and falls back to the monstats row.
func resistsOf(u d2.Unit, mon *data.Monster, hc HostileCtx) ([6]int, bool) {
	if hc.Resists != nil {
		if r, ok := hc.Resists(u, mon); ok {
			return r, true
		}
	}
	if mon == nil {
		return [6]int{}, false
	}
	d := data.Difficulty(hc.Diff)
	var r [6]int
	for i := range r {
		r[i] = mon.Res[d][i]
	}
	return AffixResists(r, affixesOf(u.Monster)), true
}

// resistStat are the stat ids for physical, magic, fire, lightning, cold, poison in data.DamageType order
// (damageresist 36, magicresist 37, fireresist 39, lightresist 41, coldresist 43, poisonresist 45).
var resistStat = [6]int{36, 37, 39, 41, 43, 45}

// ResistsFromStats merges a monster's stat list with its monstats row: a stat that is present wins.
func ResistsFromStats(st d2.Stats, mon *data.Monster, diff d2.Difficulty) [6]int {
	var r [6]int
	if mon != nil {
		for i := range r {
			r[i] = mon.Res[data.Difficulty(diff)][i]
		}
	}
	for i, id := range resistStat {
		if st.Has(id) {
			r[i] = int(st.Get(id))
		}
	}
	return r
}

// Rank orders targets for the fight: kill-first monsters, then elites that are close, then the nearest.
// Targets the danger table says to avoid are dropped (counted by the caller via len difference).
// quality (0-100) is the handicap knob: with probability (100-quality)% the best target is replaced by the nearest one;
// roll is a uniform [0,1) draw supplied by the caller so this stays deterministic.
func Rank(ts []Target, quality int, roll float64) []Target {
	out := make([]Target, 0, len(ts))
	for _, t := range ts {
		if !t.Avoid {
			out = append(out, t)
		}
	}
	score := func(t *Target) float64 {
		s := -t.Dist
		if t.KillFast {
			s += 3000
		}
		if t.Elite && t.Dist < 25 {
			s += 600
		}
		s += t.Danger * 400 // killed us before: deal with it first
		if t.Dist <= 6 {    // something already on us comes first
			s += 1500
		}
		return s
	}
	sort.SliceStable(out, func(i, j int) bool {
		si, sj := score(&out[i]), score(&out[j])
		if si != sj {
			return si > sj
		}
		return out[i].Unit.ID < out[j].Unit.ID
	})
	if quality < 100 && len(out) > 1 && roll*100 >= float64(quality) {
		// careless pick: the nearest monster, ignoring priorities
		best := 0
		for i := range out {
			if out[i].Dist < out[best].Dist {
				best = i
			}
		}
		out[0], out[best] = out[best], out[0]
	}
	return out
}
