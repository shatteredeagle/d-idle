package bot

import (
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/resource"
)

// errAreaGated is a refusal to walk into an area the character must not enter now (naked, destitute or banned after
// repeated deaths). The task layer maps it to a danger fault, so the quest waits and the planner levels instead.
var errAreaGated = errors.New("area not allowed now")

// gearRebuyGold is the gold at which a naked character goes to town to rebuy a weapon and body armor (U: judgment; the
// cheapest vendor weapon and armor of Normal cost well under this, not measured here).
const gearRebuyGold = 150

// gearVerdict is resource.GearGate for the snapshot, with no particular area in mind. An unreadable inventory is
// never gated.
func (r *Runner) gearVerdict(s *d2.Snapshot) resource.GearVerdict {
	v, ok := r.gearVerdictFor(s, 0)
	if ok {
		r.noteGate(v, s.Player.Gold)
	}
	return v
}

// gearVerdictFor adds the area (monster level and deaths there) to the verdict. It never logs.
func (r *Runner) gearVerdictFor(s *d2.Snapshot, area int) (resource.GearVerdict, bool) {
	if s == nil || s.Player == nil || s.Inventory == nil || r.d.Catalog == nil {
		return resource.GearVerdict{}, false
	}
	p, inv := s.Player, s.Inventory
	healing, _, _ := r.beltAndInv(s)
	now := r.e.Clock.Now()
	t := r.e.Tune.Bot
	r.obMu.Lock()
	hour := r.dlog.since(now.Add(-time.Hour), 0)
	here := 0
	if area != 0 {
		here = r.dlog.since(now.Add(-time.Duration(t.SpiralMinutes)*time.Minute), area)
	}
	r.obMu.Unlock()
	in := resource.GearInput{
		Level: p.Level, Gold: p.Gold, Potions: healing,
		Weapon:     inv.Equipped(d2.BodyRHand) != nil || inv.Equipped(d2.BodyAltR) != nil || r.gearBlocked("weapon", p),
		Body:       inv.Equipped(d2.BodyTorso) != nil || r.gearBlocked("body armor", p),
		Caster:     p.Class == d2.Sorceress || p.Class == d2.Necromancer || p.Class == d2.Druid,
		NeedWeapon: resource.NeedsWeapon(p.Class),
		Difficulty: int(p.Difficulty), Act: p.Act + 1, MaxLife: p.MaxLife,
		MinResist: min(p.Resist.Fire, p.Resist.Cold, p.Resist.Lightning),
		Deaths:    hour, DeathsHere: here,
	}
	if area != 0 {
		in.AreaMLvl = r.areaMLvl(area, p.Difficulty)
	}
	return resource.GearGate(in, gearRebuyGold, resource.FromEnv(*r.e.Tune).Gear()), true
}

// noteGate logs the gate when its state or reason changes, so a log shows how long a character stayed gated and why a
// rebuy did or did not follow (G1 attempt 6 left this invisible). In graded mode it also logs every change of the
// risk level: "gear risk: low|medium|high (reasons)".
func (r *Runner) noteGate(v resource.GearVerdict, gold int) {
	key := ""
	if v.Gated {
		key = fmt.Sprintf("naked=%v rebuy=%v", v.Naked, v.CanRebuy)
	}
	r.obMu.Lock()
	changed := key != r.ob.gateKey
	r.ob.gateKey = key
	riskChanged := v.Risk != r.ob.riskKey || !r.ob.riskSeen
	r.ob.riskKey, r.ob.riskSeen = v.Risk, true
	r.obMu.Unlock()
	if riskChanged && r.e.Tune.Resource.GearGateMode != resource.GateHard {
		r.e.Log.Info(fmt.Sprintf("gear risk: %s (%s)", v.Risk, v.Reason), "gold", gold)
	}
	switch {
	case changed && v.Gated:
		r.e.Log.Info("gear gate: on", "reason", v.Reason, "canRebuy", v.CanRebuy, "gold", gold)
	case changed:
		r.e.Log.Info("gear gate: off", "gold", gold)
	}
}

// safeFarmArea is the first field area of the character's act, where a gated character farms gold (the walk home works
// from there without a portal). 0 when the plan has none.
func (r *Runner) safeFarmArea(s *d2.Snapshot) int {
	act := s.Player.Act + 1
	first := 0
	for _, st := range r.d.Plan.Steps[difficultyName(s.Player.Difficulty)] {
		if r.d.Plan.IsTown(st.Area) {
			continue
		}
		if r.d.Plan.Act(st.Area) == act {
			return st.Area
		}
		if first == 0 {
			first = st.Area
		}
	}
	return first
}

// areaAllowed says why the character may not be in area now ("" = fine): towns always pass, a banned area never does,
// and a gated character only gets the safe farm area.
func (r *Runner) areaAllowed(s *d2.Snapshot, area int) string {
	if area == 0 || r.d.Plan.IsTown(area) {
		return ""
	}
	if r.banned(area) {
		return "banned after repeated deaths"
	}
	if why := r.areaGuardWhy(s, area); why != "" {
		return "route guard (" + why + ")"
	}
	if v := r.gearVerdict(s); v.Gated && area != r.safeFarmArea(s) {
		return v.Reason + ": only the first field is open"
	}
	if v, ok := r.gearVerdictFor(s, area); ok && v.Gated && r.e.Tune.Resource.GearGateMode != resource.GateHard && area != r.safeFarmArea(s) {
		return "gear risk high here (" + v.Reason + "): only the first field is open"
	}
	return ""
}

// routeAllowed checks every area on a walk to target, not only the target itself: the Soaky deaths were on the way to
// the Barracks. A walk to town is always fine (it is the way out), and a walk to the safe farm area only has to avoid
// banned areas.
func (r *Runner) routeAllowed(s *d2.Snapshot, cur, target int, hops []int) error {
	if s == nil || s.Player == nil || r.d.Plan.IsTown(target) {
		return nil
	}
	safe := r.gearVerdict(s).Gated && target == r.safeFarmArea(s)
	for _, h := range hops {
		if h == cur || r.d.Plan.IsTown(h) {
			continue
		}
		why := ""
		if err := r.routeGuardHop(s, h); err != nil {
			return err
		}
		if r.banned(h) {
			why = "banned after repeated deaths"
		} else if !safe {
			why = r.areaAllowed(s, h)
		}
		if why != "" {
			return fmt.Errorf("%w: %s (%d): %s", errAreaGated, r.areaName(h), h, why)
		}
	}
	return nil
}

// areaMLvl is the monster level of area in the difficulty (levels.txt, V: data.Level.MonLvl), 0 when unknown.
func (r *Runner) areaMLvl(area int, d d2.Difficulty) int {
	if r.e.Data == nil {
		return 0
	}
	if l := r.e.Data.AreaByID(area); l != nil {
		return l.MonLvl[min(max(int(d), 0), 2)]
	}
	return 0
}

// noteBlocked remembers the slots a rebuy could not fill because the vendor's stock needs more strength, dexterity or
// level than the character has (V: G1 attempt 7, a Sorceress with Str 10 cannot wear the cheapest body armor, Quilted
// needs 12; the gate then refused every quest step for good). Gold cannot fix that, so while the stats stay as they
// are the gate treats the slot as settled; a level or stat point re-opens it.
func (r *Runner) noteBlocked(p *d2.Player, slots []string) {
	if p == nil {
		return
	}
	r.obMu.Lock()
	defer r.obMu.Unlock()
	r.ob.blocked = map[string][3]int{}
	for _, sl := range slots {
		r.ob.blocked[sl] = [3]int{p.Strength, p.Dexterity, p.Level}
		r.e.Log.Warn("gear gate: slot cannot be filled yet, not gating on it", "slot", sl, "str", p.Strength, "dex", p.Dexterity, "clvl", p.Level)
	}
}

// gearBlocked says whether slot is settled by noteBlocked for the character's current stats.
func (r *Runner) gearBlocked(slot string, p *d2.Player) bool {
	r.obMu.Lock()
	defer r.obMu.Unlock()
	at, ok := r.ob.blocked[slot]
	return ok && at == [3]int{p.Strength, p.Dexterity, p.Level}
}
