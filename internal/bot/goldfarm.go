package bot

import (
	"context"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goldfarm"
	"d2lod-bot/internal/resource"
)

// The player's Farm gold order (docs/goldfarm.md). While it is on, the leveling area choice is one rung below the plan
// area for the character's level (a zone it clearly out-levels), until the player stops it. Town trips, selling,
// banking and the repair gate run as always; poverty (updateStarvation) keeps its own hysteresis and lean mode.

// SetGoldFarm connects the order. repairNeed reads the gold Repair All would cost now (0 or nil = unknown).
func (r *Runner) SetGoldFarm(c *goldfarm.Controller, repairNeed func() int) {
	r.gf, r.gfRepair = c, repairNeed
}

func (r *Runner) goldFarmOn() bool { return r.gf != nil && r.gf.On() }

// RungState is the verdict of Plan.FarmRung.
type RungState int

const (
	RungOK     RungState = iota // a zone was found
	RungNone                    // every zone of the rung is excluded right now (cleared, banned, guarded): wait
	RungTooLow                  // the first field of the plan is above the character's level: no safe zone at all
)

// FarmRung is the farm zone one rung below the zone Pick would choose for level among the zones that may be farmed at
// all (static: the safe list, never changes during a game): the plan's own order says what the level is ready for, the
// zone before it is one the character out-levels. A level that Pick puts on the first zone farms that zone. Zones in
// o.Exclude (cleared, avoided, guarded: they change minute by minute) and thin-kit zones are skipped and the search
// walks further down. Data only: the zone list, entry levels and the melee entry are the plan's, no class is named
// (the caller tells Pick Melee from the class traits).
func (p *Plan) FarmRung(diff string, level int, o PickOptions, static map[int]bool) (Step, RungState) {
	var list []Step
	for _, s := range p.Steps[diff] {
		if p.IsTown(s.Area) || static[s.Area] || o.Acts != nil && !o.Acts[p.Act(s.Area)] {
			continue
		}
		list = append(list, s)
	}
	if len(list) == 0 {
		return Step{}, RungTooLow
	}
	entry := func(s Step) int {
		if o.Melee && s.MeleeEnter > 0 {
			return s.MeleeEnter
		}
		return s.EnterLvl
	}
	if entry(list[0]) > level {
		return Step{}, RungTooLow
	}
	free := o
	free.Exclude, free.Cleared, free.Floor, free.XP = static, nil, 0, nil
	idx := -1
	if cur, ok := p.Pick(diff, level, free); ok {
		for i, s := range list {
			if s.Area == cur.Area {
				idx = i
			}
		}
	}
	if idx < 0 { // the pick is outside the allowed acts: the last zone the level has reached
		for i, s := range list {
			if entry(s) <= level {
				idx = i
			}
		}
	}
	for i := max(idx-1, 0); i >= 0; i-- {
		s := list[i]
		if o.Exclude[s.Area] || s.KitLvl > level && !o.FullKit {
			continue
		}
		return s, RungOK
	}
	return Step{}, RungNone
}

// goldFarmTarget is the order's area choice. ok: farm there. stop: no target now (nothing open, or the order ended
// because no safe zone exists for this level); the caller returns no target.
func (r *Runner) goldFarmTarget(s *d2.Snapshot) (area int, ok, stop bool) {
	p := s.Player
	r.updateStarvation(s) // poverty keeps its state, trips and lean mode while the order runs
	have, want, repair := r.goldFarmFigures(s)
	rep := goldfarm.Status{Have: have, Want: want, Repair: repair}
	diff := difficultyName(p.Difficulty)
	var st Step
	state := RungOK
	if g := r.gearVerdict(s); g.Gated {
		// a gated hero is only allowed the first field (areaRulesWhy): that is the safe zone
		st.Area = r.safeFarmArea(s)
		if st.Area == 0 || r.banned(st.Area) || r.loop.cleared[st.Area] {
			state = RungNone
		}
	} else {
		ex, static := map[int]bool{}, map[int]bool{}
		for a := range r.loop.cleared {
			ex[a] = true
		}
		for a := range r.avoidedNow() { // the border-loop and death breakers: a skipped zone drops one rung further
			ex[a] = true
		}
		r.guardedAreas(s, ex)
		if diff == "normal" && p.Act == 0 { // the same safe list a broke character farms (safeTier)
			for _, ps := range r.d.Plan.Steps[diff] {
				static[ps.Area] = !safeTier[ps.Area]
			}
		}
		if healing, _, known := r.beltAndInv(s); known && healing < 2 {
			ex[areaDenOfEvil] = true
		}
		t := r.e.Tune.Bot
		st, state = r.d.Plan.FarmRung(diff, p.Level, PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex, FullKit: r.fullKit(s), Melee: MeleeTank(p.Class)}, static)
		if !r.resInput(s).HasTP && state == RungOK {
			// no portal: the walk home only works from the first field of the act (as starveArea)
			if first, ok := r.starveStep(s, diff); ok && !ex[first] {
				st = Step{Area: first}
			} else {
				state = RungNone
			}
		}
	}
	switch state {
	case RungTooLow:
		why := "level " + itoa(p.Level) + " is below the first safe zone"
		r.e.Log.Warn("goldfarm: stopped, no safe zone", "level", p.Level)
		r.gf.Halt(why)
		return 0, false, false // the normal plan carries on
	case RungNone:
		rep.Note = "waiting, every safe zone is cleared or avoided for now"
		r.gf.Report(rep)
		return 0, false, true
	}
	name := r.areaName(st.Area)
	rep.Area = name
	if r.gfArea != st.Area {
		r.gfArea = st.Area
		r.e.Log.Info("goldfarm: farming gold on user request", "area", st.Area, "name", name, "gold", have, "want", want)
	}
	r.gf.Report(rep)
	return st.Area, true, false
}

// goldFarmFigures is gold on hand (pack plus stash: banking is part of the order, not a loss), the target and its
// repair part.
func (r *Runner) goldFarmFigures(s *d2.Snapshot) (have, want, repair int) {
	if r.gfRepair != nil {
		repair = r.gfRepair()
	}
	return s.Player.Gold + s.Player.StashGold, goldfarm.Want(repair, resource.FromEnv(*r.e.Tune)), repair
}

// goldFarmNumbers feeds the whyline: have and want while the order is on and a zone is being farmed.
func (r *Runner) goldFarmNumbers(s *d2.Snapshot) (have, want int, on bool) {
	if !r.goldFarmOn() || s == nil || s.Player == nil {
		return 0, 0, false
	}
	have, want, _ = r.goldFarmFigures(s)
	return have, want, true
}

// goldFarmStep replaces the campaign's quest step while the order runs: one leveling slice in the farm zone.
func (r *Runner) goldFarmStep(ctx context.Context, snap *d2.Snapshot, c control) bool {
	have, want, _ := r.goldFarmFigures(snap)
	r.setWhy(WhyFarmUser, "area", "the farm zone", "have", itoa(have), "want", itoa(want))
	return r.levelSlice(ctx, snap, c, "Farming gold on your order", "goldfarm")
}
