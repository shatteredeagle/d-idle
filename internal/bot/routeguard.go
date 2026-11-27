package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/task"
)

// fillRoute copies the route guard and imbue knobs into the World, computes the verdict and logs it once per change
// (`route guard: blocked monastery (no merc, clvl 12 < 20)`), so a repeat shows in one grep. The guard is a Normal
// rule; the knob 0 turns it off.
func (r *Runner) fillRoute(w *task.World, class d2.Class, p *d2.Player) {
	t := r.e.Tune
	w.RouteMinLevel, w.RouteMinLevelMerc = t.Route.MonasteryMinLevel, t.Route.MonasteryMinLevelMerc
	w.ImbueMinLevel, w.ImbueNeedsMerc = t.Campaign.ImbueMinLevel, t.Campaign.ImbueNeedsMerc != 0
	w.ImbueUseful = class == d2.Barbarian || class == d2.Paladin || class == d2.Assassin
	w.TowerMinLevel, w.TowerMinLevelMerc, w.TowerNeedsMerc = t.Route.TowerMinLevel, t.Route.TowerMinLevelMerc, t.Route.TowerNeedsMerc != 0
	r.fillTower(w, t.Route, class, p)
	w.RouteBlocked = task.RouteWhy(*w)
	w.TowerBlocked = r.towerWhy(*w, p)
	line := task.RouteLine(*w)
	if tl := task.TowerLine(*w); tl != "" {
		line += "; " + tl
	}
	guard := line
	if why := task.ImbueSkipWhy(*w); why != "" {
		line += "; imbue quest skipped: optional (" + why + ")"
	}
	r.obMu.Lock()
	changed := r.loop.routeNote != line
	r.loop.routeNote = line
	r.ob.routeLine = guard
	r.obMu.Unlock()
	if changed {
		r.e.Log.Info(line)
	}
	r.fillEarly(w)
	r.fillAndariel(w, class)
	r.noteFloors(*w)
}

// fillTower copies the life and kit knobs of the Forgotten Tower guard. The mana part of the kit is asked only of a class
// that uses mana every attack (resource.ManaNeed), so a Barbarian is unchanged. A max life that cannot be read stays 0
// and the guard falls back to the old floor (route.towerMinLevelNoLife) and says so in its log line.
func (r *Runner) fillTower(w *task.World, rt env.RouteTuning, class d2.Class, p *d2.Player) {
	w.TowerMinLife, w.TowerMinLevelNoLife = rt.TowerMinLife, rt.TowerMinLevelNoLife
	w.BarracksMinLevel, w.BarracksMinLife, w.BarracksLifeWaiver = rt.BarracksMinLevel, rt.BarracksMinLife, rt.BarracksLifeWaiver
	if class.Caster() {
		w.BarracksMinLevel, w.BarracksMinLife = rt.BarracksMinLevelCaster, rt.BarracksMinLifeCaster
	}
	if p != nil {
		w.MaxLife = p.MaxLife
	}
	if resource.ManaNeed(class) >= 2 {
		w.TowerKitMana = rt.TowerKitMana
	}
}

// noteFloors logs the effective floor of each guard once per change (`route guard: monastery floor=16 (merc=none)`), so
// a grep shows which number a decision used and whether the merc state moved it. Normal only; a guard that is off is silent.
func (r *Runner) noteFloors(w task.World) {
	if w.Difficulty != "" && w.Difficulty != "normal" {
		return
	}
	var lines []string
	for _, g := range []struct {
		name  string
		floor int
	}{{"monastery", task.MonasteryFloor(w)}, {"forgotten tower", task.TowerFloor(w)}, {"andariel", task.AndarielFloor(w)}} {
		if g.floor > 0 {
			lines = append(lines, task.FloorLine(g.name, g.floor, w))
			if g.name == "forgotten tower" {
				lines[len(lines)-1] += task.TowerLifeNote(w)
			}
		}
	}
	if task.BarracksLifeWaived(w) {
		lines = append(lines, fmt.Sprintf("route guard: barracks life floor waived (clvl %d, max life %d < %d)", w.Level, w.MaxLife, w.BarracksMinLife))
	}
	key := strings.Join(lines, "\n")
	r.obMu.Lock()
	changed := r.loop.floorNote != key
	r.loop.floorNote = key
	r.obMu.Unlock()
	if changed {
		for _, l := range lines {
			r.e.Log.Info(l)
		}
	}
}

// guardWorld is the slice of the World the guards read, built from a snapshot (nil when there is no player).
func (r *Runner) guardWorld(s *d2.Snapshot) *task.World {
	if s == nil || s.Player == nil {
		return nil
	}
	w := task.World{Difficulty: difficultyName(s.Player.Difficulty), Level: s.Player.Level, HasMerc: d2.FindMerc(s).Alive}
	w.RouteMinLevel, w.RouteMinLevelMerc = r.e.Tune.Route.MonasteryMinLevel, r.e.Tune.Route.MonasteryMinLevelMerc
	w.TowerMinLevel, w.TowerMinLevelMerc, w.TowerNeedsMerc = r.e.Tune.Route.TowerMinLevel, r.e.Tune.Route.TowerMinLevelMerc, r.e.Tune.Route.TowerNeedsMerc != 0
	r.fillTower(&w, r.e.Tune.Route, s.Player.Class, s.Player)
	if task.MercAlive != nil {
		w.MercAlive, w.MercKnown = task.MercAlive()
	}
	in := r.resInputOpt(s, false) // the watch API calls this too: no sell judgement (sellsnap)
	w.Potions, _, _ = r.beltAndInv(s)
	w.Potions += in.Belt.PackHP + in.Belt.PackRV
	w.HasTP, w.ManaPotions = in.HasTP, in.Belt.MP
	r.fillKit(&w, s)
	w.Caster = s.Player.Class.Caster()
	return &w
}

// routeGuardWhy says why the Monastery road is closed now ("" = open) from the latest snapshot.
func (r *Runner) routeGuardWhy(s *d2.Snapshot) string {
	if w := r.guardWorld(s); w != nil {
		return task.RouteWhy(*w)
	}
	return ""
}

// towerGuardWhy says why the Forgotten Tower (areas 20-25) is closed now ("" = open).
func (r *Runner) towerGuardWhy(s *d2.Snapshot) string {
	if w := r.guardWorld(s); w != nil {
		return r.towerWhy(*w, s.Player)
	}
	return ""
}

// towerWhy is task.TowerWhy with point t0.area.toohard (docs/t0-slice.md): the level guard's verdict is the answer
// (go when open, else lower with no kit, else skip) until the ladder, when installed, learns a rule for the context.
// A learned go lifts the level block (the kit part of the guard stays); a learned skip or lower still blocks. The point
// is asked only while the level guard blocks (never when the hero is at or above the floor). Asked only while the guard is on (Normal), so the other difficulties are untouched. p may be nil.
func (r *Runner) towerWhy(w task.World, p *d2.Player) string {
	why := task.TowerWhy(w)
	l := ladder.Active()
	if l == nil || p == nil || task.TowerFloor(w) == 0 {
		return why
	}
	if task.TowerLevelWhy(w) == "" {
		// The level passes (clvl at the floor counts): there is nothing "too hard" to ask about. The point used to be asked
		// anyway, which put "the bot wants hero level 15 there and the hero is level 15" on the phone while only the kit
		// was thin (live 2026-11-17). A thin kit stays a chores matter (why).
		return why
	}
	kit := ladder.KitFor(w.HasTP, w.Potions)
	legacy := "go"
	switch {
	case kit == ladder.KitNone:
		legacy = "lower"
	default:
		legacy = "skip"
	}
	a, err := l.TooHard(ladder.HeroFrom(p).With("dest", "the Forgotten Tower", "floor", task.TowerFloor(w)), w.Level, task.MercState(w), kit, legacy)
	if err != nil {
		l.Fault(err)
		return why
	}
	switch {
	case a == legacy || a != "go" && legacy != "go":
		return why
	case a == "go":
		return task.TowerKitWhy(w)
	}
	return "ladder: " + a
}

// areaGuardWhy says why guarded area a is closed now ("" = not guarded or open): the Monastery road and the tower.
func (r *Runner) areaGuardWhy(s *d2.Snapshot, a int) string {
	switch {
	case task.RouteGuarded(task.Area(a)):
		if why := r.routeGuardWhy(s); why != "" {
			return why
		}
		return r.barracksGuardWhy(s, a)
	case task.TowerGuarded(task.Area(a)):
		return r.towerGuardWhy(s)
	}
	return r.admitEnter(s, a)
}

// areaLeaveWhy is areaGuardWhy for a hero that already stands in area a: the road guard uses its mid-route verdict.
func (r *Runner) areaLeaveWhy(s *d2.Snapshot, a int) string {
	if task.RouteGuarded(task.Area(a)) {
		if w := r.guardWorld(s); w != nil {
			if why := task.RouteLeaveWhy(*w); why != "" {
				return why
			}
		}
		return r.barracksGuardWhy(s, a)
	}
	if _, early := earlyRouteAt(a); early {
		return "" // a hero already in the Underground Passage or the Dark Wood walks on or home: the guard is about entering
	}
	return r.areaGuardWhy(s, a)
}

// barracksGuardWhy is the Barracks floor for the areas that need it (28 to 37): a hero below the class level or max life
// neither walks through the Barracks arrival nor farms past it (V live 2026-11-23: the farm walk to the Catacombs crossed
// it at clvl 20 and chickened or portalled out seven times in ten minutes).
func (r *Runner) barracksGuardWhy(s *d2.Snapshot, a int) string {
	if !task.BarracksBeyond(task.Area(a)) || r.e.Tune.Route.MonasteryMinLevel <= 0 { // the road guard knob 0 turns the Barracks floor off too
		return ""
	}
	if w := r.guardWorld(s); w != nil {
		return task.BarracksWhy(*w)
	}
	return ""
}

// MercGoldFarm is set by the merc wiring (cmd/d2bot/mercwire.go): true while the merc is missing, gold cannot pay the
// next merc action and the character is below merc.goldFarmMaxLevel. nil = never.
var MercGoldFarm func(s *d2.Snapshot) bool

// farmExclude adds every area outside the plan steps flagged farm to ex while the gold farm is on (Normal act 1) and
// returns the areas it added, so the picker can drop them again when nothing farmable is left (farm = preference).
// A missing plan table is logged and the farm is skipped, never replaced by a built-in set.
func (r *Runner) farmExclude(s *d2.Snapshot, ex map[int]bool) []int {
	f := MercGoldFarm
	if f == nil || s == nil || s.Player == nil || s.Player.Difficulty != d2.Normal || s.Player.Act != 0 || !f(s) {
		return nil
	}
	farm, err := r.d.Plan.FarmAreas("normal")
	if err != nil {
		r.e.Log.Warn("farm: no farm areas", "err", err)
		return nil
	}
	var added []int
	for a := 2; a <= 132; a++ {
		if !farm[a] && !ex[a] {
			ex[a] = true
			added = append(added, a)
		}
	}
	return added
}

// guardedAreas adds the Monastery road and the tower to ex while their guards are closed, so the level picker never offers it.
// While the gold farm is on (Normal act 1) every area outside the plan steps flagged farm is added too.
func (r *Runner) guardedAreas(s *d2.Snapshot, ex map[int]bool) {
	r.farmExclude(s, ex)
	r.guardedAreasNoFarm(s, ex)
}

func (r *Runner) guardedAreasNoFarm(s *d2.Snapshot, ex map[int]bool) {
	if a := r.stalePackArea(s); a != 0 {
		ex[a] = true // the pack we fled from is still there: level elsewhere until the kit is rebuilt
	}
	for _, a := range []int{10, 5} { // the early road (earlyroutes.json): the level picker never offers it while it is closed
		if r.earlyRouteWhy(s, a) != "" {
			ex[a] = true
		}
	}
	if r.denGated(s) {
		ex[areaDenOfEvil] = true // the Den waits for its class floor (campaign.denMinLevel*): level in Blood Moor
	}
	route, tower := r.routeGuardWhy(s) != "", r.towerGuardWhy(s) != ""
	if r.barracksGuardWhy(s, int(task.Barracks)) != "" {
		for a := int(task.Barracks); a <= 37; a++ {
			ex[a] = true // the picker never offers the Barracks or what lies past it below the floor
		}
	}
	for a := 7; a <= 37; a++ {
		if (route && task.RouteGuarded(task.Area(a))) || (tower && task.TowerGuarded(task.Area(a))) {
			ex[a] = true
		}
	}
}

// guardCrowdN is the hostile count within guardCrowdRadius at which a town portal is not worth its cast time (the crowd
// rule, docs/combat.md): the character walks out instead. U: judgment, same order as combat.crowdN.
const (
	guardCrowdN      = 3
	guardCrowdRadius = 6
)

// guardedExit is the first action of a character that stands in a guarded area it may not be in (no merc and below the
// level floor, or a thin kit; a portal landing or a mistaken entry): leave toward town before any planning. With a crowd
// around, walk to the entry first (travelTo allows a walk to town through the guarded areas), else portal or walk home
// (Town.ToTown). Reports whether it acted.
func (r *Runner) guardedExit(ctx context.Context, snap *d2.Snapshot, c control) bool {
	p := snap.Player
	if p.InTown() {
		return false
	}
	why := r.areaLeaveWhy(snap, p.Area)
	if why == "" {
		return false
	}
	crowd := r.hostileNear(snap, guardCrowdRadius) >= guardCrowdN
	r.e.Log.Info(fmt.Sprintf("guard: leaving guarded area %d first (%s)", p.Area, why), "name", r.areaName(p.Area), "crowd", crowd)
	r.setWhy(WhyGuardExit, "why", why)
	return r.do(ctx, c, StateResupplying, "Leaving a guarded area ("+why+")", sec(r.cfg.TownTimeoutSec), "guard exit", func(ctx context.Context) error {
		if crowd {
			if err := r.travelTo(ctx, int(task.RogueEncampment)); err == nil || ctx.Err() != nil {
				return err
			}
		}
		err := r.d.Town.ToTown(ctx)
		if err == nil || ctx.Err() != nil {
			return err
		}
		// No portal scroll: walk home through the level chain. The area gate lets a walk to town through every guarded level (it is
		// the way out); ToTown's own walk knows only the short list of fields. V live 2026-11-25 02:40Z to 03:00Z: cellar 2 of the Forgotten Tower
		// with no scroll, 204 "no town portal available" and 20 minutes without a kill.
		r.e.Log.Info("guard: no way home by portal, walking", "err", err, "area", p.Area)
		if werr := r.travelTo(ctx, int(task.RogueEncampment)); werr != nil {
			return homeWalkErr(err, werr)
		}
		return nil
	})
}

// routeGuardHop is the error of a walk through a guarded area while the guard is closed (nil = fine).
func (r *Runner) routeGuardHop(s *d2.Snapshot, h int) error {
	if !task.AreaGuarded(task.Area(h)) {
		return nil
	}
	if why := r.areaGuardWhy(s, h); why != "" {
		return r.refuse(fmt.Errorf("%w: %s (%d): route guard (%s)", errAreaGated, r.areaName(h), h, why))
	}
	return nil
}

// homeWalkErr is the error of a guard exit whose portal failed (err) and whose walk home failed too (werr). A typed
// refusal of the walk stays what it is, so do handles it as a refusal and not as a fault; any other fault is joined
// with the portal error as before.
func homeWalkErr(err, werr error) error {
	var ref *Refused
	if errors.As(werr, &ref) {
		return werr
	}
	return fmt.Errorf("%w (walk home: %v)", err, werr)
}

// fillRouteFacts adds the ROUTEGUARD autopsy fields. Called with obMu held from buildAutopsy.
func (r *Runner) fillRouteFacts(au *Autopsy, a aliveInfo, now time.Time) {
	au.Nearest = a.near
	au.RouteGuard = r.ob.routeLine
	switch {
	case a.merc == nil:
		au.MercState = "none"
	case *a.merc:
		au.MercState = "alive"
	default:
		au.MercState = "dead"
	}
	switch au.State {
	case "traveling":
		au.Mode = "moving"
	case "clearing":
		au.Mode = "fighting"
	default:
		au.Mode = au.State
	}
	// life samples at least 100 ms apart, newest last, at most 6
	var rev []LifeSample
	last := time.Time{}
	for i := len(r.ob.curve) - 1; i >= 0 && len(rev) < 6; i-- {
		c := r.ob.curve[i]
		if !last.IsZero() && last.Sub(c.at) < 100*time.Millisecond {
			continue
		}
		last = c.at
		rev = append(rev, LifeSample{MsBefore: int(now.Sub(c.at) / time.Millisecond), Pct: c.pct})
	}
	for i := len(rev) - 1; i >= 0; i-- {
		au.LastLife = append(au.LastLife, rev[i])
	}
	if d, ok := r.d.Fighter.(fighterDiag); ok {
		if pc := d.LastPortalCast(); pc.Started && now.Sub(pc.Start) < 30*time.Second {
			au.PortalCast = &PortalNote{SecBefore: int(now.Sub(pc.Start) / time.Second), DurMs: int(pc.Dur / time.Millisecond), OK: pc.OK}
		}
		n, by := d.RecentCasts(5 * time.Second)
		au.Casts5s = n
		if n > 0 {
			au.CastSkills = map[string]int{}
			for id, c := range by {
				au.CastSkills[fmt.Sprint(id)] = c
			}
		}
	}
}

// denFloor is the clvl at which this character may enter the Den of Evil: the caster knob for casters and an Amazon
// without a launcher, the melee knob for the melee classes and a bowed Amazon.
func (r *Runner) denFloor(s *d2.Snapshot) int {
	p, c := s.Player, r.e.Tune.Campaign
	if p.Class.Caster() {
		return c.DenMinLevelCaster
	}
	switch p.Class {
	case d2.Amazon:
		if r.hasLauncher(s) {
			return c.DenMinLevelMelee
		}
		return c.DenMinLevelCaster
	}
	return c.DenMinLevelMelee
}

// hasLauncher reports a bow or crossbow in the weapon hand (U: base-code lookup through the item table, like town.gearOf).
func (r *Runner) hasLauncher(s *d2.Snapshot) bool {
	if s.Inventory == nil || r.e.Data == nil {
		return false
	}
	u := s.Inventory.Equipped(d2.BodyRHand)
	if u == nil || u.Class < 0 || u.Class >= len(r.e.Data.Items) {
		return false
	}
	return gear.IsA(gear.TypeOf(r.e.Data.Items[u.Class].Code), "miss")
}

// denGated says the character is below its Den floor in Normal.
func (r *Runner) denGated(s *d2.Snapshot) bool {
	return s != nil && s.Player != nil && s.Player.Difficulty == d2.Normal && s.Player.Level < r.denFloor(s)
}

// supplyPlanner is the town's planner, reached through an optional interface so a backend without one keeps the old
// constants.
type supplyPlanner interface {
	SupplyPlan(s *d2.Snapshot, profile string, reserve int) (supply.Plan, error)
}

// kitProfile is the supply profile the route and tower guards ask for (internal/supply/profiles.json).
const kitProfile = "monastery_road"

// kitPlanOn: the guards and the shop read the supply plan at stage 2 of town.plan, and always with the quest board's
// state machine (quests.boardSM). The guard's constants ask for more healing than a belt with a mana column holds, so a
// chores trip ended one potion short and the chunk was parked (Proving Ground boardsmrun, hold.restock); the plan is
// clamped to what the hero can carry and names the line it cannot close.
func (r *Runner) kitPlanOn() bool { return r.e.Tune.Town.Plan >= 2 || !r.boardLegacy() }

// kitPlanOf is the town planner's view of the guard kit at stage 2 of the knob town.plan: nil at stage 0 or 1 (the
// guards keep their constants) or when the planner cannot answer, which is logged once.
func (r *Runner) kitPlanOf(s *d2.Snapshot) *task.KitPlan {
	if !r.kitPlanOn() || s == nil || s.Player == nil {
		return nil
	}
	sp, ok := r.d.Town.(supplyPlanner)
	if !ok {
		return nil
	}
	profile := kitProfile
	if r.andarielKitOn(s) {
		profile = "andariel" // the boss kit: more healing than the road asks for, and the boss purse (supply/profiles.json)
	}
	pl, err := sp.SupplyPlan(s, profile, 0)
	if err != nil {
		r.planWarn.Do(func() { r.e.Log.Warn("town plan: " + err.Error()) })
		return nil
	}
	hand := pl.Handover(supply.Gold{Sellable: r.sellableCached() > 0, VaultPrompt: r.e.Tune.Loot.Vault != 0 && r.e.Tune.Loot.VaultSellPrompt != 0})
	return &task.KitPlan{Healing: pl.MinOf(supply.Healing), Mana: pl.MinOf(supply.Mana), Portals: pl.MinOf(supply.Portal), GoldCarry: pl.MinOf(supply.GoldCarry), Short: pl.Why(), Hand: hand.Reason}
}

// fillKit counts the kit by the one counter and, at stage 2, attaches the plan.
func (r *Runner) fillKit(w *task.World, s *d2.Snapshot) {
	// Mana counts belt plus pack at every stage, like the healing count and like the shop that decides what to buy (rule
	// conflict I8): with the belt only a caster with its potions in the pack was refused "0 mana potions" by the quest
	// gate while the shop saw nothing to buy (run 20261122-190434, 29 minutes stuck).
	if c, ok := r.supplyCount(s); ok {
		w.ManaPotions = c.Mana()
	}
	if !r.kitPlanOn() {
		return
	}
	w.KitPlan = r.kitPlanOf(s)
}

// andarielKitOn: the hero is close enough to Andariel's floor in Normal and has not killed her, so the town visits buy the boss
// kit (the antidote rule of antidotesWanted).
func (r *Runner) andarielKitOn(s *d2.Snapshot) bool {
	// While the campaign holds Andariel back (boss.andarielHold, no owner try) the hero levels: the boss floor of 8 closed the Forgotten Tower
	// Cellars and the road and shuttled it between town and the field (V live 2026-11-25).
	held := r.e.Tune.Boss.AndarielHold != 0 && !task.TryActive()
	return s != nil && s.Player != nil && !held && s.Player.Difficulty == d2.Normal && s.Player.Level >= r.andarielLow(s.Player.Class)-2 && !r.killedBoss(task.BossAndy)
}

// bossPurse is the gold a town visit leaves with when the stash has it: the boss profile's gold_carry, 0 outside the boss kit.
func (r *Runner) bossPurse() int {
	s := r.latest()
	if s == nil || !r.andarielKitOn(s) {
		return 0
	}
	if k := r.kitPlanOf(s); k != nil {
		return k.GoldCarry
	}
	return 0
}

// fillEarly hands the early road guard to the task layer, so a quest step into the Dark Wood waits as a hold (a chores trip for a
// missing kit) instead of failing at the hop and counting as "too weak".
func (r *Runner) fillEarly(w *task.World) {
	s := r.latest()
	rs, _ := loadEarlyRoutes()
	w.EarlyBlocked, w.WalkBlocked = nil, task.EarlyBlock{}
	if s != nil && s.Player != nil {
		w.FirstField = task.Area(r.firstFieldArea(s))
	}
	if why, kit := r.admitLeave(s); why != "" {
		w.WalkBlocked = task.EarlyBlock{Why: why, KitOnly: kit}
	}
	for _, rt := range rs {
		for _, a := range rt.Areas {
			if why, kit := r.earlyRouteCheck(s, a); why != "" {
				if w.EarlyBlocked == nil {
					w.EarlyBlocked = map[task.Area]task.EarlyBlock{}
				}
				w.EarlyBlocked[task.Area(a)] = task.EarlyBlock{Why: why, KitOnly: kit}
			}
		}
	}
}
