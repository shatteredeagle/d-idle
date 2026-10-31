package bot

import (
	"context"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/resource"
)

// resInput fills the policy input from a snapshot outside a fight (docs/resource.md: TownTripTrigger every idle tick).
// Fight-only fields (danger, skills, merc) stay zero: the fighter feeds those itself.
func (r *Runner) resInput(s *d2.Snapshot) resource.Input {
	p := s.Player
	in := resource.Input{
		Life: p.Life, MaxLife: p.MaxLife, Mana: p.Mana, MaxMana: p.MaxMana, InTown: p.InTown(), Gold: p.Gold,
		Level: p.Level, Class: p.Class, Diff: p.Difficulty, Area: p.Area, LastTripGold: -1, NearTown: resource.NearTownArea(p.Area),
		Hostiles: r.hostileNear(s, r.e.Tune.Combat.FightRadius),
		Build:    leveler.ResolveBuild(r.cfg.LevelerBuild, p.Class.String()),
	}
	r.obMu.Lock()
	if r.ob.tripGoldSet {
		in.LastTripGold = r.ob.tripGold
	}
	in.IdleTrips = r.ob.idleTrips
	in.TripsRecent = r.tripsRecentLocked(r.e.Clock.Now())
	r.obMu.Unlock()
	in.Sellable = r.sellable(s)
	mc := d2.FindMerc(s)
	in.SetMerc(mc, r.merc.observe(mc, p.InTown(), r.e.Clock.Now()))
	if r.d.Catalog == nil || s.Inventory == nil {
		return in
	}
	for i := range s.Inventory.Items {
		u := &s.Inventory.Items[i]
		if u.Item == nil {
			continue
		}
		m, ok := r.d.Catalog.Item(u.Class)
		if !ok {
			continue
		}
		switch u.Item.Where {
		case d2.ContBelt:
			switch m.Potion {
			case PotionHealing:
				in.Belt.HP++
			case PotionMana:
				in.Belt.MP++
			case PotionRejuv:
				in.Belt.RV++
			}
		case d2.ContInventory:
			if strings.Contains(m.Name, "Town Portal") {
				in.HasTP = true
			}
			switch m.Potion {
			case PotionHealing:
				in.Belt.PackHP++
			case PotionRejuv:
				in.Belt.PackRV++
			}
		}
	}
	return in
}

// tripVerdict is the resource policy's answer to "is a town trip worth it now".
func (r *Runner) tripVerdict(s *d2.Snapshot) resource.Trip {
	return resource.TownTripTrigger(r.resInput(s), resource.FromEnv(*r.e.Tune))
}

// idleTripGold is the gold growth below which a finished chore counts as an idle trip.
const idleTripGold = 10

// noteTripGold remembers the gold a finished chore left, so a trip that could not fix a need is not repeated until
// gold has grown (TownTripTrigger.Futile).
func (r *Runner) noteTripGold(gold int) {
	r.obMu.Lock()
	// a chore that ended with (almost) no more gold than the previous one bought and sold nothing (U approximation)
	if r.ob.tripGoldSet && gold-r.ob.tripGold < idleTripGold {
		r.ob.idleTrips++
	} else {
		r.ob.idleTrips = 0
	}
	r.ob.tripGold, r.ob.tripGoldSet = gold, true
	now := r.e.Clock.Now()
	r.ob.tripTimes = append(r.tripTimesLocked(now), now)
	r.obMu.Unlock()
}

// logTrip writes the one line per finished trip: what it cost and how much of the budget is gone.
func (r *Runner) logTrip(t *trip, gold int) {
	r.obMu.Lock()
	n := len(r.ob.tripTimes)
	r.obMu.Unlock()
	rt := r.e.Tune.Resource
	r.e.Log.Info("trip budget: trip done", "reason", t.plan.Reason, "emergency", t.emergency, "goldBefore", t.startGold, "goldAfter", gold,
		"spent", t.startGold-gold, "tripsInWindow", n, "windowMin", rt.TripWindowMin, "budget", rt.TripBudget, "goldFloor", rt.TripGoldFloor)
}

// tripTimesLocked drops the trips older than the budget window and returns the rest (obMu held).
func (r *Runner) tripTimesLocked(now time.Time) []time.Time {
	win := time.Duration(r.e.Tune.Resource.TripWindowMin) * time.Minute
	keep := r.ob.tripTimes[:0]
	for _, t := range r.ob.tripTimes {
		if now.Sub(t) < win {
			keep = append(keep, t)
		}
	}
	r.ob.tripTimes = keep
	return keep
}

func (r *Runner) tripsRecentLocked(now time.Time) int { return len(r.tripTimesLocked(now)) }

// starveStep is the no-portal fallback of starveArea: the first non-town plan step of the character's own act (a
// character standing in act 2's town must not be sent to the Blood Moor), else the first step of the difficulty. ok is
// false when that step is cleared or banned.
func (r *Runner) starveStep(s *d2.Snapshot, diff string) (int, bool) {
	act := s.Player.Act + 1
	var first int
	for _, st := range r.d.Plan.Steps[diff] {
		if r.d.Plan.IsTown(st.Area) {
			continue
		}
		if r.d.Plan.Act(st.Area) == act {
			first = st.Area
			break
		}
		if first == 0 {
			first = st.Area
		}
	}
	if first == 0 || r.loop.cleared[first] || r.banned(first) {
		return 0, false
	}
	return first, true
}

// povertySetter is the part of a Fighter that takes the poverty verdict (combat.Fighter.SetPoverty).
type povertySetter interface {
	SetPoverty(poor bool, maxPackDanger int)
}

// pushPoverty hands the verdict to the fighter so its engage decision can skip packs a potion-less character cannot
// afford to fight (resource.PreEngage).
func (r *Runner) pushPoverty(poor bool, maxPackDanger int) {
	if ps, ok := r.d.Fighter.(povertySetter); ok {
		ps.SetPoverty(poor, maxPackDanger)
	}
}

// MercTown is the optional part of the Townsman that hires and revives mercenaries (adapter over town.Hirer; the
// policy for what to hire is town.HirePolicy). Without it the merc chore never runs.
type MercTown interface {
	// HireMerc hires per town.HirePolicy for the snapshot's difficulty, gold and merc; (false, nil) means the policy
	// declined or the hire dialog is unverified.
	HireMerc(ctx context.Context, s *d2.Snapshot) (bool, error)
	// ReviveMerc revives the dead merc; unverified is the Tuning.Bot.MercReviveUnverified knob.
	ReviveMerc(ctx context.Context, unverified bool) (bool, error)
}

// BeltFiller is the optional part of the Townsman that moves potions from the pack to the belt (town.FillBelt). The
// runner calls it in the field between fights and in town, so a pack full of potions never shows as a thin belt.
type BeltFiller interface {
	NeedsBeltFill(s *d2.Snapshot) bool
	FillBelt(ctx context.Context) (moved int, err error)
}

// beltFillBackoff spaces the attempts, so a belt fill that does nothing is not clicked again every tick.
const beltFillBackoff = 90 * time.Second

// beltFillStep fills the belt from the pack when no hostile is near and the pack holds potions the belt has room for.
// It reports whether it acted (the caller then takes a fresh snapshot).
func (r *Runner) beltFillStep(ctx context.Context, s *d2.Snapshot) bool {
	bf, ok := r.d.Town.(BeltFiller)
	if !ok || s.Inventory == nil || r.e.Clock.Now().Before(r.loop.beltFillNext) || !bf.NeedsBeltFill(s) ||
		r.hostileNear(s, r.e.Tune.Bot.ClearRadius) > 0 {
		return false
	}
	r.loop.beltFillNext = r.e.Clock.Now().Add(beltFillBackoff)
	n, err := bf.FillBelt(ctx)
	r.e.Log.Info("belt fill", "moved", n, "inTown", s.Player.InTown(), "err", err)
	return n > 0
}

// areaLutGholein is the Act 2 town (V levels.tsv id 40).
const areaLutGholein = 40

// mercGoneAfter is how long a known merc must stay out of the unit table in town before it counts as dead: a living
// merc follows through a portal a moment after the player.
const mercGoneAfter = 8 * time.Second

// mercState is the sticky merc memory. Guarded by mu: the guard goroutine reads resInput too.
type mercState struct {
	mu        sync.Mutex
	known     bool      // a merc was seen or hired this run (a dead one may vanish from the unit table, U)
	goneSince time.Time // known merc first missing in town, zero while present or in the field
	next      time.Time // earliest next chore attempt
}

// observe records what the snapshot shows and returns the sticky known flag.
func (m *mercState) observe(mc d2.Merc, inTown bool, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if mc.Present {
		m.known = true
	}
	if mc.Present || !inTown {
		m.goneSince = time.Time{}
	} else if m.goneSince.IsZero() {
		m.goneSince = now
	}
	return m.known
}

// mercWanted says what the merc chore would do now: "hire", "revive" or "" (nothing). Pure apart from the clock.
func (r *Runner) mercWanted(s *d2.Snapshot) string {
	if _, ok := r.d.Town.(MercTown); !ok || s.Player == nil || !s.Player.InTown() {
		return ""
	}
	t := r.e.Tune.Bot
	now := r.e.Clock.Now()
	m := &r.merc
	m.mu.Lock()
	defer m.mu.Unlock()
	if now.Before(m.next) {
		return ""
	}
	mc := d2.FindMerc(s)
	if mc.Present {
		m.known = true
	}
	gone := !mc.Present && !m.goneSince.IsZero() && now.Sub(m.goneSince) >= mercGoneAfter
	if !m.known {
		if gone && s.Player.Area == areaLutGholein && s.Player.Gold >= t.MercHireMinGold {
			return "hire"
		}
		return ""
	}
	if !t.MercReviveUnverified || !(gone || mc.Present && !mc.Alive) {
		return ""
	}
	in := resource.Input{Gold: s.Player.Gold}
	in.SetMerc(mc, true)
	if in.MercReviveWanted(t.GoldFloor) {
		return "revive"
	}
	return ""
}

// mercChore runs the wanted merc action once and backs off MercRetryMin minutes whatever the outcome.
func (r *Runner) mercChore(ctx context.Context, s *d2.Snapshot) {
	what := r.mercWanted(s)
	if what == "" {
		return
	}
	mt := r.d.Town.(MercTown)
	r.merc.mu.Lock()
	r.merc.next = r.e.Clock.Now().Add(time.Duration(r.e.Tune.Bot.MercRetryMin) * time.Minute)
	r.merc.mu.Unlock()
	var ok bool
	var err error
	if what == "hire" {
		ok, err = mt.HireMerc(ctx, s)
		if ok {
			r.merc.mu.Lock()
			r.merc.known = true
			r.merc.mu.Unlock()
		}
	} else {
		ok, err = mt.ReviveMerc(ctx, r.e.Tune.Bot.MercReviveUnverified)
	}
	r.e.Log.Info("merc chore", "what", what, "done", ok, "err", err)
}

// buildName is the leveler table of the character for logs: the configured one, else the class default.
func (r *Runner) buildName(p *d2.Player) string {
	if b := leveler.ResolveBuild(r.cfg.LevelerBuild, p.Class.String()); b != "" {
		return b
	}
	if b := leveler.SelectBuild(p.Class.String(), p.Level, leveler.OwnedSkills(r.e.Data, p)); b != "" {
		return b
	}
	return "-"
}
