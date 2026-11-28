package bot

import (
	"fmt"
	"sync"

	"d2lod-bot/internal/admit"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// The early route guard (user request 2026-11-25): a quest walk through the Underground Passage to the Dark Wood killed a clvl 4
// Paladin on the Questy bed (no potions, Vile Hunters at monster level 6 to 7). The legs and their floors are data
// (internal/plandata/admission.json, roads): below the level of its class family or short of the kit a hero does not enter them, and the
// level plan farms elsewhere meanwhile. The owner's try anyway overrides, like the other route guards.

type earlyRoute struct {
	Name     string         `json:"name"`
	Areas    []int          `json:"areas"`
	MinLevel map[string]int `json:"minLevel"`
	MinLife  map[string]int `json:"minLife"` // max life by class family; 0 = no rule
	Kit      struct {
		Healing int `json:"healing"`
		Portal  int `json:"portal"`
	} `json:"kit"`
}

type questWalkRule struct {
	FromLevel  int `json:"fromLevel"`
	BelowLevel int `json:"belowLevel"`
	Healing    int `json:"healing"`
	Portal     int `json:"portal"`
}

var (
	earlyWalk   questWalkRule
	earlyOnce   sync.Once
	earlyRoutes []earlyRoute
	earlyErr    error
)

func loadEarlyRoutes() ([]earlyRoute, error) {
	earlyOnce.Do(func() {
		rows, err := admit.Load()
		if err != nil {
			earlyErr = err
			return
		}
		for _, rd := range rows.Roads {
			er := earlyRoute{Name: rd.ID, Areas: rd.Areas, MinLevel: rd.MinLevel, MinLife: rd.MinMaxLife}
			er.Kit.Healing, er.Kit.Portal = rd.Kit.Healing, rd.Kit.Portal
			earlyRoutes = append(earlyRoutes, er)
		}
		w := rows.QuestWalk
		earlyWalk = questWalkRule{FromLevel: w.FromLevel, BelowLevel: w.BelowLevel, Healing: w.Healing, Portal: w.Portal}
	})
	return earlyRoutes, earlyErr
}

// earlyRoutesAt lists every route that holds area a, in file order. An area may be on two roads (Duriel's Lair is on
// the tombs road and has its own row with the caster life floor): all of them are asked, as admit.MayEnter does.
func earlyRoutesAt(a int) []earlyRoute {
	rs, _ := loadEarlyRoutes()
	var out []earlyRoute
	for _, r := range rs {
		for _, x := range r.Areas {
			if x == a {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// earlyRouteWhy says why area a is closed to the hero now ("" = open). Normal difficulty only.
func (r *Runner) earlyRouteWhy(s *d2.Snapshot, a int) string {
	why, _ := r.earlyRouteCheck(s, a)
	return why
}

// earlyRouteCheck is earlyRouteWhy that also says whether only the kit is missing (a chores trip fixes it).
func (r *Runner) earlyRouteCheck(s *d2.Snapshot, a int) (why string, kitOnly bool) {
	if r.corpseRun.Load() {
		return "", false // a corpse run is exempt: the corpse is worth more than the kit rule (Questy 2026-11-26: skipped for 1 potion, then lost)
	}
	rts := earlyRoutesAt(a)
	if len(rts) == 0 || s == nil || s.Player == nil || r.e.Tune.Route.EarlyRoute == 0 || s.Player.Difficulty != d2.Normal || task.TryActive() {
		return "", false
	}
	family := "melee"
	if s.Player.Class.Caster() {
		family = "caster"
	}
	in := r.resInput(s)
	for _, rt := range rts {
		if min := rt.MinLevel[family]; min > 0 && s.Player.Level < min {
			return fmt.Sprintf("level %d needed for the %s road (clvl %d)", min, rt.Name, s.Player.Level), false
		}
		if min := rt.MinLife[family]; min > 0 && s.Player.MaxLife > 0 && s.Player.MaxLife < min {
			return fmt.Sprintf("life %d needed for the %s road (max life %d)", min, rt.Name, s.Player.MaxLife), false
		}
		if onRoad(rt, s.Player.Area) {
			// The kit is an entry rule: a hero already inside passed it and goes on with a kit that thinned, like the route
			// guard. The fix (a chores trip) needs the portal that is missing, so the refusal parked the chunk and left the
			// hero idle in the lair (V lab night 2026-11-28, drill.act2.shaft.stepexec: "no town portal for the
			// act2_maggot_lair road" in Maggot Lair Level 1, parked after 3 tries, then "Waiting for a fresh game").
			continue
		}
		if n := healingOf(in); n < rt.Kit.Healing {
			return fmt.Sprintf("%d healing potions < %d for the %s road", n, rt.Kit.Healing, rt.Name), true
		}
		if rt.Kit.Portal > 0 && !in.HasTP {
			return "no town portal for the " + rt.Name + " road", true
		}
	}
	return "", false
}

func onRoad(rt earlyRoute, area int) bool {
	for _, a := range rt.Areas {
		if a == area {
			return true
		}
	}
	return false
}

// questWalkCheck is the kit rule of every quest walk into a field (earlyroutes.json questWalk): from fromLevel to below belowLevel in
// Normal the hero needs the healing potions and a portal source before it leaves town for a quest. kitOnly says the purse can pay
// the missing potions (a chores trip fixes it); otherwise the hero farms gold first (a hold, nothing counted).
func (r *Runner) questWalkCheck(s *d2.Snapshot) (why string, kitOnly bool) {
	if s == nil || s.Player == nil || r.e.Tune.Route.EarlyRoute == 0 || s.Player.Difficulty != d2.Normal || task.TryActive() || r.corpseRun.Load() {
		return "", false
	}
	_, _ = loadEarlyRoutes()
	w := earlyWalk
	if w.BelowLevel == 0 || s.Player.Level < w.FromLevel || s.Player.Level >= w.BelowLevel {
		return "", false
	}
	in := r.resInput(s)
	n := healingOf(in)
	switch {
	case n < w.Healing:
		need := (w.Healing - n) * r.e.Tune.Resource.KitPotionGold
		if s.Player.Gold < need {
			return fmt.Sprintf("%d healing potions < %d for a quest walk, %d gold needed: farm gold first", n, w.Healing, need), false
		}
		return fmt.Sprintf("%d healing potions < %d for a quest walk", n, w.Healing), true
	case w.Portal > 0 && !in.HasTP:
		if s.Player.Gold < r.e.Tune.Resource.KitTPGold {
			return "no town portal for a quest walk, " + fmt.Sprint(r.e.Tune.Resource.KitTPGold) + " gold needed: farm gold first", false
		}
		return "no town portal for a quest walk", true
	}
	return "", false
}
