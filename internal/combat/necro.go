package combat

import (
	"context"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// Necromancer upkeep: summons, curses, armor by state (docs/summon.md). Everything here asks the snapshot, never a timer:
// minions are counted (IsFriendly census), corpses are monster units in mode 12, curses and Bone Armor are read from
// state numbers. Where a state number is not yet proven readable the code falls back to a cast memory.
const (
	// summonReach: Raise Skeleton only produced a minion when the caster stood within 4 subtiles of the corpse
	// (V docs/classes/smoke-necromancer.md finding 2: 30 subtiles = no mana spent, 4 = cast). The exact limit is U.
	summonReach  = 4
	corpseSearch = 14 // KeepSummons walks to a corpse at most this far
	curseRange   = 20 // V: a curse landed at 11 subtiles and not at 28 (finding 4); 20 is a U middle
	packRadius   = 8  // hostiles within this of a target make it a pack
	curseLifeSec = 30 // cast memory when the curse state is not readable (Amplify Damage lasts 24 s at level 1, U for the rest)
	maxSummonsAt = 6  // casts per KeepSummons call
	curseCover   = 5  // a unit this close to a recent cast of the same curse counts as cursed (U radius; V live 2026-10-29: one cast per unit burned a cast every 1.2 s on a pack)
)

// States by name. stateBoneArmor and stateAmplify are V (docs/classes/smoke-necromancer.md findings 4-5); Weaken 19 is S
// (states.txt row order from memory), Lower Resist and the rest are not listed (cast memory is used for them).
const (
	stateAmplify   = 9
	stateBoneArmor = 14
	stateWeaken    = 19
)

var (
	curseStates = map[string]int{"Amplify Damage": stateAmplify, "Weaken": stateWeaken}
	buffStates  = map[string]int{"Bone Armor": stateBoneArmor}
	curseNames  = map[string]bool{"Amplify Damage": true, "Weaken": true, "Lower Resist": true, "Dim Vision": true,
		"Terror": true, "Confuse": true, "Life Tap": true, "Attract": true, "Decrepify": true}
	// golems in preference order. Iron Golem needs an item in the world and Blood Golem costs life: not used (U).
	golemLadder = []string{"Fire Golem", "Clay Golem"}
)

// Minions is the census of the player's own minions by kind.
type Minions struct {
	Skeletons, Mages, Golems, Other int
}

// Total is every counted minion.
func (m Minions) Total() int { return m.Skeletons + m.Mages + m.Golems + m.Other }

// Monster classes of the Necromancer's summons: V in friendly.go (monstats rows of AI NecroPet, live: 363 and 289).
func minionKind(class int) string {
	switch {
	case class == 363:
		return "skeleton"
	case class == 364:
		return "mage"
	case class >= 289 && class <= 292:
		return "golem"
	}
	return "other"
}

// CountMinions counts living friendly monsters that are summons (not mercenaries, townspeople or walls).
func CountMinions(snap *d2.Snapshot, cat Catalog) Minions {
	var m Minions
	if snap == nil {
		return m
	}
	fr := Friendlies(snap, cat)
	for _, u := range snap.Monsters {
		w, ok := fr[u.ID]
		if !ok || !u.Alive() || w == WhyNPC || w == WhyHirelingAI {
			continue
		}
		switch minionKind(u.Class) {
		case "skeleton":
			m.Skeletons++
		case "mage":
			m.Mages++
		case "golem":
			m.Golems++
		default:
			if u.Monster.HasState(d2.StateMinion) { // revives and other raised monsters (U: state 105 meaning)
				m.Other++
			}
		}
	}
	return m
}

// Corpses lists raisable corpses within maxDist of me, nearest first: hostile-class monsters in mode 12 that are not
// friendly, not townspeople, not killable-false and not a monstats boss (U: bosses and some classes cannot be raised),
// minus the ids in skip.
func Corpses(snap *d2.Snapshot, cat Catalog, me input.Point, maxDist float64, skip map[uint32]bool) []d2.Unit {
	if snap == nil {
		return nil
	}
	fr := Friendlies(snap, cat)
	var out []d2.Unit
	seen := map[uint32]bool{}
	for _, u := range snap.Monsters {
		if u.Mode != d2.MonModeDead || skip[u.ID] || seen[u.ID] || (u.X == 0 && u.Y == 0) { // 0,0 is an unread position (U)
			continue
		}
		seen[u.ID] = true
		if _, f := fr[u.ID]; f {
			continue
		}
		if cat != nil {
			if m := cat.Monster(u.Class); m != nil && (m.NPC || !m.Killable || m.Boss) {
				continue
			}
		}
		if dist(me.X, me.Y, u.X, u.Y) <= maxDist {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return dist(me.X, me.Y, out[i].X, out[i].Y) < dist(me.X, me.Y, out[j].X, out[j].Y)
	})
	return out
}

// necroState is the Fighter's memory for the upkeep (zero value is usable after init).
type necroState struct {
	capSeen map[string]int  // kind -> minion count at which a paid cast raised nothing (the cap); cleared when the count drops
	tried   map[uint32]bool // corpses a cast was aimed at (a used corpse vanishes; V a second cast spent no mana)
	learned map[int]bool    // state numbers seen present after our own cast: proven readable
	curseAt map[curseKey]curseRec
}

type curseKey struct {
	unit  uint32
	skill int
}

type curseRec struct {
	at    time.Time
	tries int
	x, y  int // where the unit stood at the cast: curses are area effects, neighbours count as covered
}

func (n *necroState) init() {
	if n.capSeen == nil {
		n.capSeen, n.tried, n.learned, n.curseAt = map[string]int{}, map[uint32]bool{}, map[int]bool{}, map[curseKey]curseRec{}
	}
}

func isNecro(p *d2.Player) bool { return p != nil && p.Class == d2.Necromancer }

// armyMin is the minion count from which a Necromancer no longer rests for a mana budget before a pack: the army
// fights while mana regenerates (V live 2026-10-29: after golem + 4 skeletons mana sat at 14-17 and the 40 s rest
// timed out twice, 36 s per pack idle).
const armyMin = 3

func (f *Fighter) armyStands(snap *d2.Snapshot) bool {
	return isNecro(snap.Player) && CountMinions(snap, f.cat).Total() >= armyMin
}

// wantMinions is how many of a kind we try to hold. Raise Skeleton gives about one skeleton per skill level (U:
// skills.txt param not read); the real cap is found by evidence (capSeen), so a high wish only costs one probe cast.
func wantMinions(kind string, level int) int {
	switch kind {
	case "mage":
		return min(level, 4)
	case "revive":
		return min(level, 8) // U: Revive count per level was not read from skills.txt; capSeen finds the real cap
	}
	return min(level, 12)
}

// planSummon picks the next summon: a golem when none stands, then skeletons, then mages. corpse is nil for a golem.
// Walk reports whether the corpse is farther than summonReach. Pure given the Fighter's memory.
func (f *Fighter) planSummon(snap *d2.Snapshot, allowWalk bool) (sp SkillSpec, corpse *d2.Unit, walk bool, ok bool) {
	p := snap.Player
	if !isNecro(p) || p.InTown() {
		return
	}
	f.necro.init()
	known := KnownSkills(p, f.cat, f.keys)
	usable := func(name string) (SkillSpec, bool) {
		s, ok := known[name]
		if !ok || s.Kind != KindSummon || (s.Key == "" && p.RightSkill != s.ID) || s.Cost > p.Mana-p.MaxMana/10 {
			return SkillSpec{}, false
		}
		return s, true
	}
	have := CountMinions(snap, f.cat)
	if have.Golems == 0 {
		for _, g := range golemLadder {
			if s, ok := usable(g); ok {
				return s, nil, false, true
			}
		}
	}
	me := input.Point{X: p.X, Y: p.Y}
	reach := float64(summonReach)
	if allowWalk {
		reach = corpseSearch
	}
	cs := Corpses(snap, f.cat, me, reach, f.necro.tried)
	if len(cs) == 0 {
		return
	}
	for _, c := range []struct {
		name, kind string
		n          int
	}{{"Raise Skeleton", "skeleton", have.Skeletons}, {"Raise Skeletal Mage", "mage", have.Mages}} {
		s, ok := usable(c.name)
		if !ok || c.n >= wantMinions(c.kind, s.Level) {
			continue
		}
		if cap, capped := f.necro.capSeen[c.kind]; capped {
			if c.n < cap {
				delete(f.necro.capSeen, c.kind)
			} else {
				continue
			}
		}
		u := cs[0]
		return s, &u, dist(me.X, me.Y, u.X, u.Y) > summonReach, true
	}
	// Revive last: the skeletons are cheap and plentiful, the revive takes the strongest corpse in reach.
	if s, ok := usable("Revive"); ok && have.Other < wantMinions("revive", s.Level) {
		if cap, capped := f.necro.capSeen["revive"]; !capped || have.Other < cap {
			delete(f.necro.capSeen, "revive")
			u := bestRevivable(cs, f.cat)
			return s, &u, dist(me.X, me.Y, u.X, u.Y) > summonReach, true
		}
	}
	return
}

// KeepSummons raises the missing minions: walk to a corpse (when allowed and a mover exists), cast, check the census.
// It never acts in town or with no corpse, and returns the number of casts. Call it after Clear (Clear does) and
// before leaving an area. Walking is skipped while a hostile stands within 10 subtiles.
func (f *Fighter) KeepSummons(ctx context.Context) (int, error) {
	casts := 0
	for casts < maxSummonsAt {
		snap, err := f.read(ctx)
		if err != nil {
			return casts, err
		}
		p := snap.Player
		if !isNecro(p) || p.InTown() || isDead(p) {
			return casts, nil
		}
		calm := true
		for _, h := range Hostiles(snap, f.hostileCtx(p)) {
			if h.Dist < 10 {
				calm = false
			}
		}
		sp, corpse, walk, ok := f.planSummon(snap, calm && f.mv != nil)
		if !ok {
			return casts, nil
		}
		if walk {
			if err := f.mv.GoTo(ctx, input.Point{X: corpse.X, Y: corpse.Y}); err != nil {
				f.necro.tried[corpse.ID] = true
				continue
			}
			if snap, err = f.read(ctx); err != nil {
				return casts, err
			}
			p = snap.Player
		}
		if done, err := f.castSummon(ctx, snap, sp, corpse); err != nil {
			return casts, err
		} else if done {
			casts++
		} else {
			return casts, nil
		}
	}
	return casts, nil
}

// castSummon casts one summon and records the result: corpse used (blacklist), cap reached (paid, nothing raised).
func (f *Fighter) castSummon(ctx context.Context, snap *d2.Snapshot, sp SkillSpec, corpse *d2.Unit) (bool, error) {
	p := snap.Player
	at := input.Point{X: p.X, Y: p.Y}
	if corpse != nil {
		at = input.Point{X: corpse.X, Y: corpse.Y}
	}
	before := CountMinions(snap, f.cat)
	if err := f.cast(snap, sp, at); err != nil {
		return false, err
	}
	if corpse != nil {
		f.necro.tried[corpse.ID] = true
	}
	if err := f.sleep(ctx, ms(f.tune().CastGapMs)+ms(500)); err != nil {
		return false, err
	}
	after, err := f.read(ctx)
	if err != nil {
		return true, err
	}
	now := CountMinions(after, f.cat)
	paid := after.Player.Mana < p.Mana-sp.Cost/2
	kind := map[string]string{"Raise Skeleton": "skeleton", "Raise Skeletal Mage": "mage", "Revive": "revive"}[sp.Name]
	grew := now.Total() > before.Total()
	if kind != "" && paid && !grew {
		f.necro.capSeen[kind] = map[string]int{"skeleton": now.Skeletons, "mage": now.Mages, "revive": now.Other}[kind]
	}
	f.info("summon", "skill", sp.Name, "minions", now.Total(), "grew", grew, "paid", paid)
	return true, nil
}

// ---- curses ---------------------------------------------------------------------------------------------

// pickCurseTarget chooses where to curse: a boss or elite first, else the hostile with the most others within
// packRadius (at least one other), nearest on ties. Only targets within curseRange and above 25% life qualify.
func pickCurseTarget(hs []Target) (Target, bool) {
	var best Target
	bestScore, found := -1, false
	for _, t := range hs {
		if t.Dist > curseRange || t.LifePct < 25 || t.Unit.Mode == d2.MonModeDead || t.Unit.Mode == d2.MonModeDeath {
			continue
		}
		n := 0
		for _, o := range hs {
			if o.Unit.ID != t.Unit.ID && dist(t.Unit.X, t.Unit.Y, o.Unit.X, o.Unit.Y) <= packRadius {
				n++
			}
		}
		score := n * 10
		if t.Boss || t.Elite {
			score += 100
		} else if n == 0 {
			continue
		}
		score -= int(t.Dist / 4)
		if score > bestScore {
			best, bestScore, found = t, score, true
		}
	}
	return best, found
}

// curseFor picks the curse for a target: Amplify Damage by default (the minions hit physically), Lower Resist when
// the target resists magic (Teeth and Bone Spear are magic) strongly and physical not, Weaken on an elite or boss when
// Amplify Damage is not learned. Only learned, bound curses are considered.
func curseFor(known map[string]SkillSpec, p *d2.Player, tg Target) (SkillSpec, bool) {
	ok := func(n string) (SkillSpec, bool) {
		s, has := known[n]
		return s, has && (s.Key != "" || p.RightSkill == s.ID) && s.Cost*2 <= p.Mana
	}
	if tg.HasRes && tg.Res[data.Physical] < 50 && tg.Res[data.Magic] >= 50 {
		if s, has := ok("Lower Resist"); has {
			return s, true
		}
	}
	for _, n := range []string{"Amplify Damage", "Weaken"} {
		if n == "Weaken" && !tg.Boss && !tg.Elite {
			continue
		}
		if s, has := ok(n); has {
			return s, true
		}
	}
	return SkillSpec{}, false
}

// curseUp says whether the target already carries this curse: state present when the state is proven readable, else
// the cast memory. Any other known curse state also counts (a second curse would replace the first, no gain).
func (f *Fighter) curseUp(u d2.Unit, sp SkillSpec) bool {
	f.necro.init()
	for name, st := range curseStates {
		if u.Monster.HasState(st) && (name == sp.Name || f.necro.learned[st]) {
			if name == sp.Name {
				f.necro.learned[st] = true // seen present on a unit we cursed: the state is readable
			}
			return true
		}
	}
	rec, seen := f.necro.curseAt[curseKey{u.ID, sp.ID}]
	if !seen {
		for k, r := range f.necro.curseAt {
			if k.skill == sp.ID && r.tries < 3 && f.now().Sub(r.at) < curseLifeSec*time.Second && dist(u.X, u.Y, r.x, r.y) <= curseCover {
				return true
			}
		}
		return false
	}
	if rec.tries >= 3 {
		return true // gave up on this unit: the cast does not register (too close, no line, immune), do not burn mana
	}
	if st, ok := curseStates[sp.Name]; ok && f.necro.learned[st] {
		return f.now().Sub(rec.at) < 2*time.Second // cast just now: give the state time to show
	}
	return f.now().Sub(rec.at) < curseLifeSec*time.Second
}

// Curse casts the best curse at the best target of the pack, once, and re-reads the state to learn whether the state
// number is readable. It returns true when a cast was made.
func (f *Fighter) Curse(ctx context.Context, snap *d2.Snapshot, hs []Target) (bool, error) {
	p := snap.Player
	if !isNecro(p) || p.InTown() {
		return false, nil
	}
	f.necro.init()
	tg, ok := pickCurseTarget(hs)
	if !ok {
		return false, nil
	}
	known := KnownSkills(p, f.cat, f.keys)
	sp, ok := curseFor(known, p, tg)
	if !ok || f.curseUp(tg.Unit, sp) {
		return false, nil
	}
	key := curseKey{tg.Unit.ID, sp.ID}
	if err := f.cast(snap, sp, input.Point{X: tg.Unit.X, Y: tg.Unit.Y}); err != nil {
		return false, err
	}
	rec := f.necro.curseAt[key]
	rec.at, rec.tries, rec.x, rec.y = f.now(), rec.tries+1, tg.Unit.X, tg.Unit.Y
	f.necro.curseAt[key] = rec
	if err := f.sleep(ctx, ms(f.tune().CastGapMs)+ms(300)); err != nil {
		return true, err
	}
	if after, err := f.read(ctx); err == nil {
		for _, u := range after.Monsters {
			if u.ID == tg.Unit.ID {
				if st, ok := curseStates[sp.Name]; ok && u.Monster.HasState(st) {
					f.necro.learned[st] = true
					rec.tries = 0
					f.necro.curseAt[key] = rec
				}
				break
			}
		}
	}
	st := curseStates[sp.Name]
	f.info("curse", "skill", sp.Name, "target", tg.Name, "dist", int(tg.Dist), "stateReadable", st != 0 && f.necro.learned[st])
	return true, nil
}

// necroTick is the fight-loop hook: curse the pack, raise a minion from a corpse within reach. true = acted this tick.
func (f *Fighter) necroTick(ctx context.Context, snap *d2.Snapshot) (bool, error) {
	p := snap.Player
	if !isNecro(p) || p.InTown() {
		return false, nil
	}
	hs := Hostiles(snap, f.hostileCtx(p))
	if done, err := f.Curse(ctx, snap, hs); done || err != nil {
		return done, err
	}
	if sp, corpse, walk, ok := f.planSummon(snap, false); ok && !walk {
		return f.castSummon(ctx, snap, sp, corpse)
	}
	return f.corpseExplode(ctx, snap, hs)
}

// buffDue says whether an armor or shield needs a cast. Bone Armor: by state when the state is proven readable
// (seen present once), with a 4 s grace after a cast; otherwise (and for every other buff) the BuffRecastSec timer.
func (f *Fighter) buffDue(p *d2.Player, sp SkillSpec) bool {
	f.necro.init()
	at, seen := f.buffAt[sp.ID]
	if st, ok := buffStates[sp.Name]; ok {
		if p.States.Has(st) {
			f.necro.learned[st] = true
			return false
		}
		if f.necro.learned[st] {
			return !seen || f.now().Sub(at) >= 4*time.Second
		}
	}
	return !seen || f.now().Sub(at) >= time.Duration(f.tune().BuffRecastSec)*time.Second
}
