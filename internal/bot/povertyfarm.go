package bot

import (
	"encoding/json"
	"fmt"
	"sync"

	"d2lod-bot/internal/admit"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/task"
)

// The poverty farm (user request 2026-11-26): a hero with no healing potion and not enough gold for the kit died six times on the
// Questy bed in fields above what it could survive without potions. The data (plandata/povertyfarm.json) lists the farm areas with
// their measured gold per minute; the hero farms the best open one whose monster level is at most clvl minus noPotionGap.

type povertyFarmData struct {
	NoPotionGap int `json:"noPotionGap"`
	FromLevel   int `json:"noPotionFromLevel"`
	Farm        []struct {
		Area       int     `json:"area"`
		Name       string  `json:"name"`
		GoldPerMin float64 `json:"goldPerMin"`
	} `json:"farm"`
	EarlyShop struct {
		SurplusPct int `json:"surplusPct"`
	} `json:"earlyShop"`
	RecoveryFromLevel int               `json:"-"`
	KitFloor          admit.KitFloorRow `json:"kitFloor"`
	Lean              struct {
		PotionsBelow int `json:"potionsBelow"`
		EntryLifePct int `json:"entryLifePct"`
	} `json:"noPortalLean"`
}

var (
	povOnce sync.Once
	povData povertyFarmData
)

func loadPovertyFarm() povertyFarmData {
	povOnce.Do(func() {
		if b, err := plandata.FS.ReadFile("povertyfarm.json"); err == nil {
			_ = json.Unmarshal(b, &povData)
		}
		if rows, err := admit.Load(); err == nil { // the floors live in admission.json (card admit1)
			povData.NoPotionGap, povData.FromLevel = rows.Poverty.NoPotionGap, rows.Poverty.FromLevel
			povData.RecoveryFromLevel = rows.Poverty.RecoveryFromLevel
			povData.KitFloor, povData.Lean = rows.KitFloor, rows.Lean
		}
	})
	return povData
}

// noPotions: the hero carries no healing potion at all (belt and pack, rejuvenation counted).
func (r *Runner) noPotions(s *d2.Snapshot) bool {
	return s != nil && s.Player != nil && healingOf(r.resInput(s)) == 0
}

// povertyAreaWhy says why a hero with no potions may not enter area a ("" = fine): its monsters are above clvl minus the gap.
func (r *Runner) povertyAreaWhy(s *d2.Snapshot, a int) string {
	d := loadPovertyFarm()
	if d.NoPotionGap <= 0 || r.e.Tune.Route.EarlyRoute == 0 || !r.noPotions(s) || s.Player.Level < d.FromLevel || s.Player.Difficulty != d2.Normal || task.TryActive() ||
		r.corpseRun.Load() || a == 0 || r.d.Plan.IsTown(a) {
		return ""
	}
	if ml := r.areaMLvl(a, s.Player.Difficulty); ml > s.Player.Level-d.NoPotionGap {
		return fmt.Sprintf("no healing potions: monster level %d is above clvl %d minus %d", ml, s.Player.Level, d.NoPotionGap)
	}
	return ""
}

// povertyFarmArea is the best open farm area for a hero with no potions and no gold for them: the highest measured gold per
// minute among the data's areas that its level allows and that are neither cleared, banned nor guarded.
func (r *Runner) povertyFarmArea(s *d2.Snapshot) (int, bool) {
	d := loadPovertyFarm()
	if d.NoPotionGap <= 0 || s == nil || s.Player == nil || healingOf(r.resInput(s)) >= max(d.KitFloor.PotionsBelow, 1) || s.Player.Difficulty != d2.Normal || s.Player.Level < d.FromLevel {
		return 0, false // the farm is for a hero short of the kit (no potion at all, or below the kit floor)
	}
	ex := map[int]bool{}
	for a := range r.loop.cleared {
		ex[a] = true
	}
	for a := range r.avoidedNow() {
		ex[a] = true
	}
	r.guardedAreas(s, ex)
	best, bestG := 0, -1.0
	for _, f := range d.Farm {
		if ex[f.Area] || r.povertyAreaWhy(s, f.Area) != "" || r.kitFloorWhy(s, f.Area) != "" || f.GoldPerMin <= bestG {
			continue
		}
		best, bestG = f.Area, f.GoldPerMin
	}
	return best, best != 0
}

// leanHero: no town portal and fewer healing potions than noPortalLean.potionsBelow in Normal: the walk home is its only way out
// of a field, so it stays within the first field.
func (r *Runner) leanHero(s *d2.Snapshot) bool {
	d := loadPovertyFarm()
	if d.Lean.PotionsBelow <= 0 || s == nil || s.Player == nil || s.Player.Difficulty != d2.Normal || r.e.Tune.Route.EarlyRoute == 0 {
		return false
	}
	in := r.resInput(s)
	return !in.HasTP && healingOf(in) < d.Lean.PotionsBelow
}

// leanEntryWhy is the entry hold of a lean hero: it does not move on into another field below noPortalLean.entryLifePct of its life
// (Questy entered Stony Field at 90/176 with no portal). "" = fine. A hero in town is held by the heal chore, not here.
func (r *Runner) leanEntryWhy(s *d2.Snapshot, a int) string {
	d := loadPovertyFarm()
	if !r.leanHero(s) || a == 0 || a == s.Player.Area || s.Player.InTown() || r.d.Plan.IsTown(a) || r.corpseRun.Load() || s.Player.MaxLife <= 0 {
		return ""
	}
	if pct := s.Player.LifePct(); pct < d.Lean.EntryLifePct {
		return fmt.Sprintf("no portal and %d%% life: not entering another field below %d%%", pct, d.Lean.EntryLifePct)
	}
	return ""
}

// fieldsOfAct lists the non-town areas of the hero's own act in plan order (Blood Moor, Cold Plains, ... in act 1).
func (r *Runner) fieldsOfAct(s *d2.Snapshot) []int {
	diff := difficultyName(s.Player.Difficulty)
	act := s.Player.Act + 1
	var out, other []int
	seen := map[int]bool{}
	for _, st := range r.d.Plan.Steps[diff] {
		if r.d.Plan.IsTown(st.Area) || seen[st.Area] {
			continue
		}
		seen[st.Area] = true
		if r.d.Plan.Act(st.Area) == act {
			out = append(out, st.Area)
		} else {
			other = append(other, st.Area)
		}
	}
	if len(out) == 0 {
		return other
	}
	return out
}

// firstFieldArea is the first non-town plan step of the hero's own act: the area next to the camp (Blood Moor in act 1).
func (r *Runner) firstFieldArea(s *d2.Snapshot) int {
	if f := r.fieldsOfAct(s); len(f) > 0 {
		return f[0]
	}
	return 0
}

// kitFloorWhy says why a hero below the kit floor may not pick area a ("" = fine): from kitFloor.fromLevel in Normal with fewer than
// kitFloor.potionsBelow healing potions it enters the first field only, scroll or not; the tiers of the row open more (two potions:
// the first two fields and the areas whose monsters are far below the hero). The area the hero stands in is not asked: the kit trip
// rules decide when it leaves.
func (r *Runner) kitFloorWhy(s *d2.Snapshot, a int) string {
	d := loadPovertyFarm()
	if d.KitFloor.PotionsBelow <= 0 || s == nil || s.Player == nil || r.e.Tune.Route.EarlyRoute == 0 || s.Player.Difficulty != d2.Normal ||
		s.Player.Level < d.KitFloor.FromLevel || a == 0 || a == s.Player.Area || r.d.Plan.IsTown(a) || task.TryActive() || r.corpseRun.Load() {
		return ""
	}
	n := healingOf(r.resInput(s))
	if n >= d.KitFloor.PotionsBelow {
		return ""
	}
	fields := r.fieldsOfAct(s)
	if d.KitFloor.Open(n, s.Player.Level, a, fields, func(x int) int { return r.areaMLvl(x, s.Player.Difficulty) }) {
		return ""
	}
	first := 0
	if len(fields) > 0 {
		first = fields[0]
	}
	return fmt.Sprintf("%d healing potions < %d: the first field only (area %d) unless a tier opens more, not area %d", n, d.KitFloor.PotionsBelow, first, a)
}

// earlyShopBudget is the gold the gear shop may spend below earlyGearMax: 0 unless the kit is met (potions at the kitFloor,
// a portal scroll) and the hero has gold above the gear shop minimum; then earlyShop.surplusPct of the gold above bot.gearReserve.
func (r *Runner) earlyShopBudget(s *d2.Snapshot) int {
	d := loadPovertyFarm()
	t := r.e.Tune.Bot
	p := s.Player
	if d.EarlyShop.SurplusPct <= 0 || d.KitFloor.PotionsBelow <= 0 || p.Gold < t.GearMinGold {
		return 0
	}
	in := r.resInput(s)
	if !in.HasTP || healingOf(in) < d.KitFloor.PotionsBelow {
		return 0
	}
	return max(p.Gold-t.GearReserve, 0) * d.EarlyShop.SurplusPct / 100
}

// recoveryPoorExclude adds to ex the plan areas a recovering hero with no healing potion must not farm: monster level above clvl minus
// poverty.noPotionGap (Doofer died at clvl 4 in Cold Plains, monster level 2, with an empty belt). It reports whether the rule applied.
func (r *Runner) recoveryPoorExclude(s *d2.Snapshot, diff string, ex map[int]bool) bool {
	d := loadPovertyFarm()
	if d.NoPotionGap <= 0 || d.RecoveryFromLevel <= 0 || diff != "normal" || s.Player.Level < d.RecoveryFromLevel || !r.noPotions(s) {
		return false
	}
	for _, st := range r.d.Plan.Steps[diff] {
		if ml := r.recoveryMLvl(st.Area, s.Player.Difficulty); ml > s.Player.Level-d.NoPotionGap {
			ex[st.Area] = true
		}
	}
	return true
}
