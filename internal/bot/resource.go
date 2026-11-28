package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/supply"
)

// resInput fills the policy input from a snapshot outside a fight (docs/resource.md: TownTripTrigger every idle tick).
// Fight-only fields (danger, skills, merc) stay zero: the fighter feeds those itself.
func (r *Runner) resInput(s *d2.Snapshot) resource.Input { return r.resInputOpt(s, true) }

// resInputOpt is resInput; judge false takes the sell count from the memo instead of judging the pack (callers off
// the bot loop, sellableCached).
func (r *Runner) resInputOpt(s *d2.Snapshot, judge bool) resource.Input {
	p := s.Player
	in := resource.Input{
		Life: p.Life, MaxLife: p.MaxLife, Mana: p.Mana, MaxMana: p.MaxMana, InTown: p.InTown(), Gold: p.Gold,
		Level: p.Level, Class: p.Class, Diff: p.Difficulty, Area: p.Area, LastTripGold: -1, NearTown: resource.NearTownArea(p.Area), ChainHome: resource.ChainHomeArea(p.Area),
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
	if judge {
		in.Sellable = r.sellableOnce(s)
	} else {
		in.Sellable = r.sellableCached()
	}
	in.HasTP = r.hasPortalSource(s)
	mc := d2.FindMerc(s)
	in.SetMerc(mc, r.merc.observe(mc, p.InTown(), r.e.Clock.Now()))
	c, known := r.supplyCount(s)
	if !known {
		return in
	}
	in.Belt.HP, in.Belt.MP, in.Belt.RV = c.Belt[supply.HP], c.Belt[supply.MP], c.Belt[supply.RV]
	in.Belt.HPTier = c.BeltTier[supply.HP]
	in.Belt.PackHP, in.Belt.PackRV = c.Pack[supply.HP], c.Pack[supply.RV]
	// BELTFIX5: with no healing potion anywhere the town visit withdraws the stash gold for the kit (Town.withdraw), so
	// every purse test of the policy (poor, futile, trip floor, recovery) sees purse plus bank. Gate 1 20261103-0338:
	// 6 gold in the pack, 1418 in the stash, 7 virtual minutes without a potion.
	if healingOf(in) == 0 {
		in.Gold += p.StashGold
	}
	return in
}

// tripVerdict is the resource policy's answer to "is a town trip worth it now".
func (r *Runner) tripVerdict(s *d2.Snapshot) resource.Trip {
	in, rt := r.resInput(s), resource.FromEnv(*r.e.Tune)
	trip := resource.TownTripTrigger(in, rt)
	return r.askTrip(s, in, rt, trip)
}

// askTrip is point t1.town.trigger (docs/t0-slice.md): the policy's verdict is the answer until the ladder (when on)
// learns a rule for the context. The ladder can only stop a trip the policy wants (a learned "keep going"): a trip
// needs the policy's method (portal or walk), which an answer cannot supply, so only Go verdicts are asked.
func (r *Runner) askTrip(s *d2.Snapshot, in resource.Input, rt resource.Tuning, trip resource.Trip) resource.Trip {
	l := ladder.Active()
	if l == nil || !trip.Go { // nothing to veto: no question for a trip the policy does not want
		return trip
	}
	_, pct, _ := r.beltAndInv(s)
	fv := resource.FieldGate(in, rt) // HaveHP and WantHP are filled whether or not the belt gate is on
	v, err := l.TripWanted(ladder.HeroFrom(s.Player).With("have", fv.HaveHP, "want", fv.WantHP), ladder.BeltBand(fv.HaveHP, fv.WantHP), pct, trip.Go)
	if err != nil {
		l.Fault(err)
		return trip
	}
	if trip.Go && !v {
		return resource.Trip{Reason: "ladder: keep going (" + trip.Reason + ")"}
	}
	return trip
}

// idleTripGold is the gold growth below which a finished chore counts as an idle trip.
const idleTripGold = 10

// noteTripEnd records the gold a finished trip left on the basis of the policy's Gold (codefix F2): the stash counts
// while no healing potion is carried, so a visit that could not turn it into a potion is futile the next time too.
func (r *Runner) noteTripEnd(s *d2.Snapshot) { r.noteTripGold(r.resInput(s).Gold) }

// noteTownFutile sets the in-town latch (codefix67 F7) after a finished trip that left the character in town with no
// healing potion and no more gold on the policy basis than the trip began with: another round would repeat the same
// shop, stash and withdraw walks. Gold growth, a potion, or leaving town lifts it (townLatched).
func (r *Runner) noteTownFutile(t *trip, s *d2.Snapshot) {
	in := r.resInput(s)
	r.obMu.Lock()
	defer r.obMu.Unlock()
	r.ob.townLatch = s.Player.InTown() && in.Gold < t.startBasis+idleTripGold && healingOf(in) == 0
	r.ob.townLatchGold = in.Gold
}

// healingOf is the one healing count of the bot side: belt and pack, healing and rejuvenation.
func healingOf(in resource.Input) int {
	return in.Belt.HP + in.Belt.RV + in.Belt.PackHP + in.Belt.PackRV
}

// townLatched reports whether the latch holds now; it clears itself outside town, with a healing potion, or after gold growth.
func (r *Runner) townLatched(s *d2.Snapshot) bool {
	in := r.resInput(s)
	r.obMu.Lock()
	defer r.obMu.Unlock()
	if !r.ob.townLatch {
		return false
	}
	if !s.Player.InTown() || healingOf(in) > 0 || in.Gold >= r.ob.townLatchGold+idleTripGold {
		r.ob.townLatch = false
		return false
	}
	return true
}

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
	kitN, level := 0, r.ob.level
	_, kitReason, isKit := strings.Cut(t.plan.Reason, "kit: ")
	if strings.Contains(t.plan.Reason, resource.RecoveryKitReason) {
		r.ob.recKitTrips++
	}
	if isKit {
		r.ob.kitTrips++
		kitN = r.ob.kitTrips
	}
	r.obMu.Unlock()
	if isKit {
		missing, _, _ := strings.Cut(kitReason, "]")
		r.e.Log.Info("kitgate-trip", "level", level, "missing", missing, "gold", gold, "trip", kitN)
	}
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
	return r.starveStepEx(s, diff, nil)
}

// starveStepEx is starveStep with areas the own-level pick must not take (the recovery no-potion rule). The floor is the
// first field of the hero's act; when ban holds that field (a hero in act 2 or later with no potion: every area of the
// act is above the limit, review 11 N4) it is the first field of the difficulty, which stays the floor whatever ban
// says. Only a hero standing in town gets the lower floor: the way there is the town waypoint (travelTo, wpActTown), and
// a hero already in the field of the later act has no route west. U: that trip is not run live.
func (r *Runner) starveStepEx(s *d2.Snapshot, diff string, ban map[int]bool) (int, bool) {
	act := s.Player.Act + 1
	var first, lowest int
	for _, st := range r.d.Plan.Steps[diff] {
		if r.d.Plan.IsTown(st.Area) {
			continue
		}
		if lowest == 0 {
			lowest = st.Area
		}
		if r.d.Plan.Act(st.Area) == act {
			first = st.Area
			break
		}
	}
	if first == 0 || ban[first] && r.d.Plan.IsTown(s.Player.Area) {
		first = lowest
	}
	if first == 0 || r.loop.cleared[first] || r.banned(first) {
		return 0, false
	}
	if a, ok := r.homeBandArea(s, diff, first, ban); ok {
		return a, true
	}
	return first, true
}

// homeReach is the set of areas a character without a town portal can leave for town: the first field (a short walk)
// and every area whose waypoint is active (the bot takes it home, "no town portal: taking the waypoint to town").
func (r *Runner) homeReach(s *d2.Snapshot, first int) map[int]bool {
	reach := map[int]bool{first: true}
	if r.leanHero(s) {
		return reach // no portal, few potions: the waypoint walk home crosses crowds (povertyfarm.json noPortalLean)
	}
	if wps, _ := r.activeWaypoints(s); wps != nil {
		for a := range wps {
			reach[a] = true
		}
	}
	return reach
}

// homeBandArea picks the plan area of the character's own level among the areas it can leave without a portal. A clvl 14
// hero was sent to Blood Moor (0.6 xp and 0.26 gold per kill) because the first field was the only option (sellskip,
// docs/sellskip.md). ok is false when the band holds nothing reachable (the caller keeps the first field).
func (r *Runner) homeBandArea(s *d2.Snapshot, diff string, first int, ban map[int]bool) (int, bool) {
	if r.kitGateActive(s) {
		return 0, false // the early kit gate keeps the walk home short: first field only
	}
	reach := r.homeReach(s, first)
	ex := map[int]bool{}
	for _, st := range r.d.Plan.Steps[diff] {
		if !reach[st.Area] || ban[st.Area] || r.loop.cleared[st.Area] || r.banned(st.Area) {
			ex[st.Area] = true
		}
	}
	r.guardedAreas(s, ex)
	t := r.e.Tune.Bot
	st, ok := r.d.Plan.Pick(diff, s.Player.Level, PickOptions{Margin: t.LevelMargin, Slack: t.LeaveSlack, Acts: r.allowedActs(s), Exclude: ex, FullKit: r.fullKit(s), Melee: MeleeTank(s.Player.Class), Class: planClass(s.Player.Class), Merc: mercUp()})
	if !ok || st.Area == first {
		return 0, false
	}
	return st.Area, true
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

// beltFillUrgentRadius is the calm radius (subtiles) of a belt fill when the belt holds no healing potion at all.
const beltFillUrgentRadius = 12

// beltFillStep fills the belt from the pack when no hostile is near and the pack holds potions the belt has room for.
// It reports whether it acted (the caller then takes a fresh snapshot).
func (r *Runner) beltFillStep(ctx context.Context, s *d2.Snapshot) bool {
	bf, ok := r.d.Town.(BeltFiller)
	if !ok || s.Inventory == nil || r.e.Clock.Now().Before(r.loop.beltFillNext) || !bf.NeedsBeltFill(s) {
		return false
	}
	// The calm radius is the sweep radius (30 subtiles), which a crowded level such as the Barracks never clears: a belt
	// with no healing potion and a pack full of them then stayed empty until the hero chickened (V live, scenario
	// act1.tools.malus: belt hp=0, six potions in the pack). With the belt out of healing the gate shrinks to
	// beltFillUrgentRadius: nothing within easy reach of the hero.
	radius := r.e.Tune.Bot.ClearRadius
	if in := r.resInput(s); in.Belt.HP+in.Belt.RV == 0 && in.Belt.PackHP+in.Belt.PackRV > 0 {
		radius = min(radius, beltFillUrgentRadius)
	}
	if r.hostileNear(s, radius) > 0 {
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
	deadSince time.Time // known merc first seen dead or gone; zero while he lives
	offNoted  time.Time // last `revive off` line
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

// mercWanted says what the merc chore would do now: "hire", "revive" or "" (nothing). Pure apart from the clock; with the
// ladder installed the legacy verdict is put to point t2.mercenary (askMerc) after the merc lock is released.
func (r *Runner) mercWanted(s *d2.Snapshot) string {
	what := r.mercLegacy(s)
	if what == "" || what == "revive" && r.zoneLeaveMercDead() {
		return ""
	}
	merc := "none"
	if what == "revive" {
		merc = "dead"
	}
	if !r.askMerc(s, merc) {
		return ""
	}
	return what
}

// mercLegacy is the legacy merc verdict, under the merc lock.
func (r *Runner) mercLegacy(s *d2.Snapshot) string {
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
	dead := gone || mc.Present && !mc.Alive
	if !dead {
		m.deadSince = time.Time{}
		return ""
	}
	if m.deadSince.IsZero() {
		m.deadSince = now
	}
	if !t.MercReviveUnverified {
		r.noteMercReviveOff(s, now.Sub(m.deadSince), now)
		return ""
	}
	in := resource.Input{Gold: s.Player.Gold}
	in.SetMerc(mc, true)
	if in.MercReviveWanted(t.GoldFloor) {
		return "revive"
	}
	return ""
}

// mercOffEvery spaces the `revive off` line. Caller holds r.merc.mu.
const mercOffEvery = 5 * time.Minute

// noteMercReviveOff makes the refusal visible: a dead merc stays dead while the unverified revive knob is off, and
// without a line nobody can tell that from a bug (2026-11-27: Doofer's merc stayed dead 22 minutes). Caller holds
// r.merc.mu; when the gold would pay it also records a standing refusal for /api/explain.
func (r *Runner) noteMercReviveOff(s *d2.Snapshot, dead time.Duration, now time.Time) {
	m := &r.merc
	if !m.offNoted.IsZero() && now.Sub(m.offNoted) < mercOffEvery {
		return
	}
	m.offNoted = now
	mins := int(dead / time.Minute)
	r.e.Log.Info(fmt.Sprintf("merc dead since %dm, revive off (bot.mercReviveUnverified)", mins), "gold", s.Player.Gold)
	r.noteStanding("merc revive", fmt.Sprintf("merc dead since %dm; bot.mercReviveUnverified=0 (revive dialog unverified live, docs/probes/merc-revive-live-checklist.md)", mins))
}

// askMerc is point t2.mercenary (docs/t0-slice.md), asked only where the legacy gate already says hire or revive: the
// gate is the answer until the ladder, when installed, learns a rule; a learned wait postpones the action (the chore
// asks again after its retry pause), a learned hire never creates one the gold or the town does not allow. merc is the
// key's state: none (never seen) or dead.
func (r *Runner) askMerc(s *d2.Snapshot, merc string) bool {
	l := ladder.Active()
	if l == nil {
		return true
	}
	in := r.resInput(s)
	kit := ladder.KitFor(in.HasTP, in.Belt.HP+in.Belt.RV+in.Belt.PackHP+in.Belt.PackRV)
	hire, err := l.MercHire(ladder.HeroFrom(s.Player), merc, kit, true)
	if err != nil {
		l.Fault(err)
		return true
	}
	return hire
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

// portalClasses adapts the bot's catalog for combat.PortalSource: a Town Portal item is a portal, the tome is the one
// named "Tome". combat's own picker and this kit check then agree on what is a usable portal.
type portalClasses struct{ Catalog }

func (c portalClasses) Item(class int) combat.ItemClass {
	m, ok := c.Catalog.Item(class)
	if !ok || !strings.Contains(m.Name, "Town Portal") {
		return combat.ItemClass{}
	}
	return combat.ItemClass{Portal: true, Tome: strings.Contains(m.Name, "Tome")}
}

// hasPortalSource is true when the pack holds a scroll or a tome with charges. An empty tome is not a portal source
// (F5: it used to count, so the kit gate and trip logic believed a portal existed).
func (r *Runner) hasPortalSource(s *d2.Snapshot) bool {
	if r.d.Catalog == nil {
		return false
	}
	var st combat.Stats
	if r.e.Read != nil {
		st = r.e.Read.UnitStats
	}
	cat := portalClasses{r.d.Catalog}
	src := combat.PortalSource(s, cat, st)
	r.dumpTP(s, src != nil, cat, st, src)
	v, ev := r.loop.tpSeen.look(src != nil, r.e.Clock.Now())
	if ev.kind != "" && r.e.Log != nil {
		r.e.Log.Info("tp-debounce-"+ev.kind, "verdict", v, "raw", src != nil, "looks", ev.looks, "held_s", int(ev.held.Seconds()))
	}
	return v
}

// mercReserve is the gold a visit keeps for reviving a dead merc: the revive price (merc.ReviveCost of the merc's level,
// else of the level a hire offers at clvl-5, which matched the quoted 600 at clvl 14). The shop honours it for optional
// buys only (town.mercReserve). 0 when the merc is alive, unlisted or the knob is off.
func (r *Runner) mercReserve(s *d2.Snapshot) int {
	if r.e.Tune.Town.MercReserve == 0 || s == nil || s.Player == nil {
		return 0
	}
	mc := d2.FindMerc(s)
	if !mc.Present || mc.Alive {
		return 0
	}
	lvl := mc.Level
	if lvl <= 0 {
		lvl = merc.OfferedLevel(s.Player.Level, 0)
	}
	return merc.ReviveCost(lvl)
}
