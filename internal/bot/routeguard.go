package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
	"d2lod-bot/internal/task"
)

// fillRoute copies the route guard and imbue knobs into the World, computes the verdict and logs it once per change
// (`route guard: blocked monastery (no merc, clvl 12 < 20)`), so a repeat shows in one grep. The guard is a Normal
// rule; the knob 0 turns it off.
func (r *Runner) fillRoute(w *task.World, class d2.Class) {
	t := r.e.Tune
	w.RouteMinLevel, w.RouteMinLevelMerc = t.Route.MonasteryMinLevel, t.Route.MonasteryMinLevelMerc
	w.ImbueMinLevel, w.ImbueNeedsMerc = t.Campaign.ImbueMinLevel, t.Campaign.ImbueNeedsMerc != 0
	w.ImbueUseful = class == d2.Barbarian || class == d2.Paladin || class == d2.Assassin
	w.TowerMinLevel, w.TowerMinLevelMerc, w.TowerNeedsMerc = t.Route.TowerMinLevel, t.Route.TowerMinLevelMerc, t.Route.TowerNeedsMerc != 0
	w.RouteBlocked = task.RouteWhy(*w)
	w.TowerBlocked = task.TowerWhy(*w)
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
	r.noteFloors(*w)
}

// noteFloors logs the effective floor of each guard once per change (`route guard: monastery floor=16 (merc=none)`), so
// a grep shows which number a decision used and whether the merc state moved it. Normal only; a guard that is off is silent.
func (r *Runner) noteFloors(w task.World) {
	if w.Difficulty != "" && w.Difficulty != "normal" {
		return
	}
	w.AndarielMinLevel, w.AndarielMinLevelMerc = r.e.Tune.Boss.AndarielMinLevel, r.e.Tune.Boss.AndarielMinLevelMerc
	var lines []string
	for _, g := range []struct {
		name  string
		floor int
	}{{"monastery", task.MonasteryFloor(w)}, {"forgotten tower", task.TowerFloor(w)}, {"andariel", task.AndarielFloor(w)}} {
		if g.floor > 0 {
			lines = append(lines, task.FloorLine(g.name, g.floor, w))
		}
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
	if task.MercAlive != nil {
		w.MercAlive, w.MercKnown = task.MercAlive()
	}
	in := r.resInput(s)
	w.Potions, _, _ = r.beltAndInv(s)
	w.Potions += in.Belt.PackHP + in.Belt.PackRV
	w.HasTP, w.ManaPotions = in.HasTP, in.Belt.MP
	w.Caster = s.Player.Class == d2.Sorceress || s.Player.Class == d2.Necromancer || s.Player.Class == d2.Druid
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
		return task.TowerWhy(*w)
	}
	return ""
}

// areaGuardWhy says why guarded area a is closed now ("" = not guarded or open): the Monastery road and the tower.
func (r *Runner) areaGuardWhy(s *d2.Snapshot, a int) string {
	switch {
	case task.RouteGuarded(task.Area(a)):
		return r.routeGuardWhy(s)
	case task.TowerGuarded(task.Area(a)):
		return r.towerGuardWhy(s)
	}
	return ""
}

// guardedAreas adds the Monastery road and the tower to ex while their guards are closed, so the level picker never offers it.
func (r *Runner) guardedAreas(s *d2.Snapshot, ex map[int]bool) {
	if r.denGated(s) {
		ex[areaDenOfEvil] = true // the Den waits for its class floor (campaign.denMinLevel*): level in Blood Moor
	}
	route, tower := r.routeGuardWhy(s) != "", r.towerGuardWhy(s) != ""
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
	why := r.areaGuardWhy(snap, p.Area)
	if why == "" {
		return false
	}
	crowd := r.hostileNear(snap, guardCrowdRadius) >= guardCrowdN
	r.e.Log.Info(fmt.Sprintf("guard: leaving guarded area %d first (%s)", p.Area, why), "name", r.areaName(p.Area), "crowd", crowd)
	return r.do(ctx, c, StateResupplying, "Leaving a guarded area ("+why+")", sec(r.cfg.TownTimeoutSec), "guard exit", func(ctx context.Context) error {
		if crowd {
			if err := r.travelTo(ctx, int(task.RogueEncampment)); err == nil || ctx.Err() != nil {
				return err
			}
		}
		return r.d.Town.ToTown(ctx)
	})
}

// routeGuardHop is the error of a walk through a guarded area while the guard is closed (nil = fine).
func (r *Runner) routeGuardHop(s *d2.Snapshot, h int) error {
	if !task.AreaGuarded(task.Area(h)) {
		return nil
	}
	if why := r.areaGuardWhy(s, h); why != "" {
		return fmt.Errorf("%w: %s (%d): route guard (%s)", errAreaGated, r.areaName(h), h, why)
	}
	return nil
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
	switch p.Class {
	case d2.Sorceress, d2.Necromancer, d2.Druid:
		return c.DenMinLevelCaster
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
