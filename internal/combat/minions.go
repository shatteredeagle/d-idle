package combat

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Minion driver, census-observed (card dlv-upkeep-minions; combat plan, upkeep half). It keeps the profile's minion
// groups standing from the census alone, never from a class id or a skill name in Go:
//   - a group is a row of the profile's minions block (a Necromancer profile without one derives it from its necro
//     block: the golems as one exclusive group, each summon rule a corpse group). Exclusive groups hold one unit of
//     any of their skills (golem, spirit, shadow); the others hold min(skill level, Max), the real cap is learned by
//     evidence (a paid cast that raised nothing), shared with the old loop's memory (necroState.capSeen).
//   - a unit belongs to a group by the Skills.txt summon column of the group's skills (data.Skill.Summon, the monstats
//     key it makes; S stock tables, not checked live per class), or with combat.minionRead by the engine row's skill
//     id. A raised unit that matches no key (Revive raises the corpse's own monster: empty summon column) counts for
//     the first corpse group whose skill has no summon monster.
//   - corpse groups aim at the nearest raisable corpse in reach, or the strongest when the row says so.
//
// Knob combat.deliveryUpkeep (shared with the upkeep driver): 0 the old class loops, 1 shadow (the old loops act, the
// driver's picks that differ are logged and counted), 2 the driver raises and summons; curses, corpse explosion and
// traps stay in the class hooks until their own cards (dlv-mark, dlv-aim-corpse, dlv-aim-point).

// MinionParams is a profile's minions block.
type MinionParams struct {
	// LifePct: in a fight, summon only while life stays above this percent (0: always).
	LifePct int           `json:"lifePct,omitempty"`
	Groups  []MinionGroup `json:"groups"`
}

// MinionGroup is one census group: the skills that fill it, in preference order.
type MinionGroup struct {
	Name      string   `json:"name"`
	Skills    []string `json:"skills"`
	Max       int      `json:"max,omitempty"`
	Exclusive bool     `json:"exclusive,omitempty"`
	Corpse    bool     `json:"corpse,omitempty"`    // the cast targets a corpse in reach
	Strongest bool     `json:"strongest,omitempty"` // corpse aim: the strongest corpse in reach instead of the nearest
	// AfterFight: only the pass after a clear summons this group, never the fight tick.
	AfterFight bool `json:"afterFight,omitempty"`
	// Source says where Max comes from: V, S or U first. Required unless Exclusive.
	Source string `json:"source,omitempty"`
}

// minionParams is the profile's minions block, derived from the necro block when the profile has none.
func (p *Profile) minionParams() *MinionParams {
	if p == nil {
		return nil
	}
	if p.Minions != nil {
		return p.Minions
	}
	n := p.Necro
	if n == nil {
		return nil
	}
	mp := &MinionParams{}
	if len(n.Golems) > 0 {
		mp.Groups = append(mp.Groups, MinionGroup{Name: "golem", Skills: n.Golems, Exclusive: true})
	}
	for _, s := range n.Summons {
		mp.Groups = append(mp.Groups, MinionGroup{Name: s.Kind, Skills: []string{s.Name}, Max: s.Max, Corpse: true,
			Strongest: s.Kind == "revive", Source: s.Source})
	}
	return mp
}

// validateMinions checks the minions blocks at load time: an error names the profile and the group.
func validateMinions(ps []Profile) error {
	for _, p := range ps {
		if p.Minions == nil {
			continue
		}
		if p.Minions.LifePct < 0 || p.Minions.LifePct >= 100 {
			return fmt.Errorf("profile %q: minions lifePct %d out of 0..99", p.ProfileID(), p.Minions.LifePct)
		}
		seen := map[string]bool{}
		for _, g := range p.Minions.Groups {
			switch {
			case g.Name == "" || seen[g.Name]:
				return fmt.Errorf("profile %q: minion group name %q empty or repeated", p.ProfileID(), g.Name)
			case len(g.Skills) == 0:
				return fmt.Errorf("profile %q: minion group %q has no skills", p.ProfileID(), g.Name)
			case !g.Exclusive && (g.Max < 1 || !validSource(g.Source)):
				return fmt.Errorf("profile %q: minion group %q needs max >= 1 with a source tag (V, S or U)", p.ProfileID(), g.Name)
			}
			seen[g.Name] = true
		}
	}
	return nil
}

// minionCensus counts the standing minions per group name.
func (f *Fighter) minionCensus(snap *d2.Snapshot, groups []MinionGroup) map[string]int {
	out := map[string]int{}
	if snap == nil || f.cat == nil {
		return out
	}
	bySkill, byMon, raised := map[int]string{}, map[string]string{}, ""
	for _, g := range groups {
		for _, name := range g.Skills {
			s := f.cat.SkillByName(name)
			if s == nil {
				continue
			}
			bySkill[s.ID] = g.Name
			if s.Summon != "" {
				byMon[s.Summon] = g.Name
			} else if g.Corpse && raised == "" {
				raised = g.Name
			}
		}
	}
	if f.e.Tune.Combat.MinionRead != 0 && f.MinionRead != nil {
		if rows, err := f.MinionRead(); err == nil {
			for _, r := range rows {
				if g, ok := bySkill[r.Skill]; ok {
					out[g]++
				} else if raised != "" {
					out[raised]++
				}
			}
			return out
		}
	}
	for _, u := range friendlyUnits(snap, f.cat) {
		key := ""
		if m := f.cat.Monster(u.Class); m != nil {
			key = m.Key
		}
		if g, ok := byMon[key]; ok {
			out[g]++
		} else if raised != "" && u.Monster.HasState(d2.StateMinion) {
			out[raised]++
		}
	}
	return out
}

// minionPlan is the driver's next summon.
type minionPlan struct {
	sp     SkillSpec
	group  string
	corpse *d2.Unit
	walk   bool
}

// corpseReach is the cast reach and the walk search of the profile (necro rules, else the package defaults).
func corpseReach(pr *Profile) (reach, search float64) {
	if pr != nil && pr.Necro != nil {
		return pr.Necro.summonReach(), pr.Necro.corpseSearch()
	}
	return summonReach, corpseSearch
}

// planMinion picks the next summon: groups in profile order, the first usable skill of a group that stands below
// its want and below a cap seen. inFight skips AfterFight groups and applies LifePct. Pure given the Fighter's memory.
// No skill-kind check: the group names its skills, and Shadow Warrior is a buff row of skills.json.
func (f *Fighter) planMinion(snap *d2.Snapshot, pr *Profile, inFight, allowWalk bool) (minionPlan, bool) {
	p := snap.Player
	mp := pr.minionParams()
	if p == nil || mp == nil || len(mp.Groups) == 0 || p.InTown() {
		return minionPlan{}, false
	}
	if inFight && mp.LifePct > 0 && p.MaxLife > 0 && p.Life*100 < p.MaxLife*mp.LifePct {
		return minionPlan{}, false
	}
	f.necro.init()
	known := KnownSkills(p, f.cat, f.keys)
	have := f.minionCensus(snap, mp.Groups)
	me := input.Point{X: p.X, Y: p.Y}
	castReach, search := corpseReach(pr)
	reach := castReach
	if allowWalk {
		reach = search
	}
	var corpses []d2.Unit
	corpsesRead := false
	for _, g := range mp.Groups {
		if inFight && g.AfterFight {
			continue
		}
		var sp SkillSpec
		found := false
		for _, name := range g.Skills {
			s, ok := known[name]
			if ok && (s.Key != "" || p.RightSkill == s.ID) && s.Cost <= p.Mana-p.MaxMana/10 {
				sp, found = s, true
				break
			}
		}
		if !found {
			continue
		}
		want := 1
		if !g.Exclusive {
			want = min(sp.Level, g.Max)
		}
		n := have[g.Name]
		if n >= want {
			continue
		}
		if cap, capped := f.necro.capSeen[g.Name]; capped {
			if n >= cap {
				continue
			}
			delete(f.necro.capSeen, g.Name)
		}
		if !g.Corpse {
			return minionPlan{sp: sp, group: g.Name}, true
		}
		if !corpsesRead {
			corpses, corpsesRead = Corpses(snap, f.cat, me, reach, f.necro.tried), true
		}
		if len(corpses) == 0 {
			continue
		}
		u := corpses[0]
		if g.Strongest {
			u = bestRevivable(corpses, f.cat)
		}
		return minionPlan{sp: sp, group: g.Name, corpse: &u, walk: dist(me.X, me.Y, u.X, u.Y) > castReach}, true
	}
	return minionPlan{}, false
}

// castMinion casts one planned summon and records the evidence: the corpse is used, a paid cast that raised nothing
// is the group's cap.
func (f *Fighter) castMinion(ctx context.Context, snap *d2.Snapshot, pr *Profile, pl minionPlan) (bool, error) {
	p := snap.Player
	groups := pr.minionParams().Groups
	at := input.Point{X: p.X, Y: p.Y}
	if pl.corpse != nil {
		at = input.Point{X: pl.corpse.X, Y: pl.corpse.Y}
	}
	before := f.minionCensus(snap, groups)
	if err := f.cast(snap, pl.sp, at); err != nil {
		return false, err
	}
	if pl.corpse != nil {
		f.necro.tried[pl.corpse.ID] = true
	}
	if err := f.sleep(ctx, ms(f.tune().CastGapMs)+ms(500)); err != nil {
		return false, err
	}
	after, err := f.read(ctx)
	if err != nil {
		return true, err
	}
	now := f.minionCensus(after, groups)
	paid := after.Player.Mana < p.Mana-pl.sp.Cost/2
	grew := censusTotal(now) > censusTotal(before)
	if paid && !grew {
		f.necro.capSeen[pl.group] = now[pl.group]
	}
	f.info("minion", "skill", pl.sp.Name, "group", pl.group, "have", now[pl.group], "grew", grew, "paid", paid)
	return true, nil
}

func censusTotal(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// minionTick is the fight-loop hook of mode 2: one summon whose corpse (if any) is in reach. true = acted.
func (f *Fighter) minionTick(ctx context.Context, snap *d2.Snapshot) (bool, error) {
	pr := f.profileFor(snap.Player)
	if pl, ok := f.planMinion(snap, pr, true, false); ok && !pl.walk {
		return f.castMinion(ctx, snap, pr, pl)
	}
	return false, nil
}

// keepMinions is the pass after a clear in mode 2: walk to a corpse when calm and a mover exists, cast, re-read.
// It never acts in town or dead, and returns the number of casts (at most maxSummonsAt).
func (f *Fighter) keepMinions(ctx context.Context) (int, error) {
	casts := 0
	for casts < maxSummonsAt {
		snap, err := f.read(ctx)
		if err != nil {
			return casts, err
		}
		p := snap.Player
		if p == nil || p.InTown() || isDead(p) {
			return casts, nil
		}
		pr := f.profileFor(p)
		pl, ok := f.planMinion(snap, pr, false, f.calm(snap) && f.mv != nil)
		if !ok {
			return casts, nil
		}
		if pl.walk {
			if err := f.mv.GoTo(ctx, input.Point{X: pl.corpse.X, Y: pl.corpse.Y}); err != nil {
				f.necro.tried[pl.corpse.ID] = true
				continue
			}
			if snap, err = f.read(ctx); err != nil {
				return casts, err
			}
		}
		done, err := f.castMinion(ctx, snap, pr, pl)
		if err != nil || !done {
			return casts, err
		}
		casts++
	}
	return casts, nil
}

// calm says no hostile stands within 10 subtiles (walking to a corpse is allowed).
func (f *Fighter) calm(snap *d2.Snapshot) bool {
	for _, h := range Hostiles(snap, f.hostileCtx(snap.Player)) {
		if h.Dist < 10 {
			return false
		}
	}
	return true
}

// minionShadow is mode 1: compare the driver's pick with the old loop's on the same snapshot and log a difference
// once while it lasts. what is "tick" or "after".
func (f *Fighter) minionShadow(snap *d2.Snapshot, what string, allowWalk bool) {
	if snap == nil || snap.Player == nil {
		return
	}
	inFight := what == "tick"
	old := f.oldMinionPick(snap, inFight, allowWalk)
	drv := ""
	if pl, ok := f.planMinion(snap, f.profileFor(snap.Player), inFight, allowWalk); ok && !(inFight && pl.walk) {
		drv = pl.sp.Name
	}
	key := what + "|" + old + "|" + drv
	if old == drv {
		if f.upk.minionLast[what] != "" {
			f.upk.minionLast[what] = ""
		}
		return
	}
	if f.upk.minionLast == nil {
		f.upk.minionLast = map[string]string{}
	}
	if f.upk.minionLast[what] == key {
		return
	}
	f.upk.minionLast[what] = key
	f.upk.shadowDiffs++
	f.info("minion shadow", "what", what, "old", old, "driver", drv)
}

// oldMinionPick is the skill the old class loops would summon now ("" none). Shadow only; it goes with the old loops.
// Each old planner gates its own class, so no class switch is added here.
func (f *Fighter) oldMinionPick(snap *d2.Snapshot, inFight, allowWalk bool) string {
	p := snap.Player
	if nd, err := f.necroParams(p); err == nil {
		if sp, _, walk, ok := f.planSummon(snap, nd, allowWalk && !inFight); ok && !(inFight && walk) {
			return sp.Name
		}
		return ""
	}
	if sp, ok := f.planPet(snap); ok {
		if inFight && p.MaxLife > 0 && p.Life*100 < p.MaxLife*40 {
			return ""
		}
		return sp.Name
	}
	if inFight || CountAssassinPets(snap, f.cat).Shadows > 0 {
		return ""
	}
	known := KnownSkills(p, f.cat, f.keys)
	for _, g := range f.profileFor(p).minionParams().groupsAfterFight() {
		for _, name := range g.Skills {
			if sp, ok := known[name]; ok && (sp.Key != "" || p.RightSkill == sp.ID) && sp.Cost <= p.Mana-p.MaxMana/10 {
				return sp.Name
			}
		}
	}
	return ""
}

func (mp *MinionParams) groupsAfterFight() []MinionGroup {
	if mp == nil {
		return nil
	}
	var out []MinionGroup
	for _, g := range mp.Groups {
		if g.AfterFight {
			out = append(out, g)
		}
	}
	return out
}
