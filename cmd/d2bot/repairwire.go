package main

import (
	"os"
	"path/filepath"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goldfarm"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/repair"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/watch"
)

// Repair is the player's call (docs/repair.md): the controller holds the mode, the Repair now request and the gear
// readings; the town asks it at every visit (town.SetRepairGate), the watch screen shows and drives it. A named run
// (D2BOT_RUN: gate, soak, drill) keeps the legacy durability rule, like the policy sheet's class starts.

// newRepair makes the controller and the durability reader of the inventory page. The town and the policy sheet bind
// later (wireRepair, bindRepair), so until then the mode is the default and nothing is unlocked.
func newRepair(stateDir string, d *display, ref *runnerRef) error {
	cfg := repair.Config{
		Gear:     func() []repair.Cond { return ref.gear() },
		Estimate: ref.estimate,
		Gold:     ref.gold,
		InTown: func() bool {
			s := ref.snap()
			return s != nil && s.Player != nil && s.Player.InTown()
		},
		Now: time.Now,
	}
	if stateDir != "" {
		cfg.Path = filepath.Join(stateDir, repair.StateFile)
	}
	c, err := repair.New(cfg)
	if err != nil {
		return err
	}
	ref.rep = c
	gcfg := ""
	if stateDir != "" {
		gcfg = filepath.Join(stateDir, goldfarm.StateFile)
	}
	if ref.gf, err = goldfarm.New(gcfg, time.Now); err != nil {
		return err
	}
	if d != nil {
		d.dur = func(u d2.Unit) (int, int, bool) {
			if tw := ref.tw.Load(); tw != nil {
				return tw.Durability(u)
			}
			return 0, 0, false
		}
	}
	return nil
}

func (r *runnerRef) snap() *d2.Snapshot {
	if r == nil {
		return nil
	}
	if run := r.p.Load(); run != nil {
		return run.LastSnapshot()
	}
	return nil
}

// gear reads the worn items' durability from the newest snapshot (memory reads only). Slot keys and names are the
// inventory page's, so the two screens agree.
func (r *runnerRef) gear() []repair.Cond {
	tw, s := r.tw.Load(), r.snap()
	if tw == nil || s == nil || s.Inventory == nil || r.rep == nil {
		return nil
	}
	warn := r.rep.Table().WarnPct
	slots := map[int]string{}
	for k, b := range invdesk.SlotBody {
		slots[b] = k
	}
	var out []repair.Cond
	for _, u := range s.Inventory.Container(d2.ContEquipment) {
		dur, mx, ok := tw.Durability(u)
		if !ok || u.Item == nil {
			continue
		}
		slot := slots[u.Item.Body]
		if c, ok := repair.Classify(slot, tw.ItemName(u), dur, mx, warn); ok {
			out = append(out, c)
		}
	}
	sortConds(out)
	return out
}

// sortConds orders the slots the way the inventory page lists them.
func sortConds(c []repair.Cond) {
	order := map[string]int{"head": 0, "neck": 1, "torso": 2, "rhand": 3, "lhand": 4, "rring": 5, "lring": 6, "belt": 7, "feet": 8, "gloves": 9}
	for i := 1; i < len(c); i++ {
		for j := i; j > 0 && order[c[j].Slot] < order[c[j-1].Slot]; j-- {
			c[j], c[j-1] = c[j-1], c[j]
		}
	}
}

// gold is the live pack gold of the newest snapshot (the stash is not counted); -1 when there is none.
func (r *runnerRef) gold() int {
	if s := r.snap(); s != nil && s.Player != nil {
		return s.Player.Gold
	}
	return -1
}

// estimate is the gold Repair All would cost now (a lower bound), 0 when unknown.
func (r *runnerRef) estimate() int {
	tw, s := r.tw.Load(), r.snap()
	if tw == nil || s == nil || s.Inventory == nil || s.Player == nil {
		return 0
	}
	return tw.RepairEstimate(s)
}

// wireRepair gives the town its gate and the controller its readings once the town exists.
func wireRepair(ref *runnerRef, tw *town.Town) {
	if ref == nil || ref.rep == nil || tw == nil {
		return
	}
	ref.tw.Store(tw)
	if os.Getenv("D2BOT_RUN") == "" {
		tw.SetRepairGate(ref.rep)
	}
}

// bindRepair connects the controller to the policy sheet (the mode is the row's current option), the unlock ladder
// (a mode is open when its row is), the alerts (gear alerts) and the routes. No-op without a controller or a Fun.
func bindRepair(f *watch.Fun, c *repair.Controller, gf *goldfarm.Controller) {
	if f == nil || c == nil {
		return
	}
	var allowed func(string) bool
	var lock func(string) (bool, int64, string)
	if f.Unlocks != nil && f.Unlocks.Engine != nil {
		eng := f.Unlocks.Engine
		allowed = eng.Allowed
		lock = func(id string) (bool, int64, string) {
			if eng.Allowed(id) {
				return true, 0, ""
			}
			for _, st := range eng.States("ui") {
				if st.ID == id {
					return false, st.Price, st.Teaser
				}
			}
			return false, 0, "Locked."
		}
	}
	var mode func() string
	if f.Policy != nil {
		mode = func() string { return f.Policy.CurrentOption("repair") }
		f.Policy.SetUnlocker(lock)
	}
	c.Bind(mode, allowed)
	if f.Alerts != nil {
		f.Alerts.SetGear(c.Gear)
	}
	api := watch.NewRepairAPI()
	api.Bind(repairAdapter{c, gf})
	f.Repair = api
	if gf != nil {
		fa := watch.NewGoldFarmAPI()
		fa.Bind(farmAdapter{gf, c})
		f.GoldFarm = fa
	}
}

// wireGoldFarm gives the runner the Farm gold order. A named run (D2BOT_RUN) never farms on an order: measured runs
// stay comparable, like the repair gate.
func wireGoldFarm(ref *runnerRef, r *bot.Runner) {
	if ref == nil || ref.gf == nil || r == nil || os.Getenv("D2BOT_RUN") != "" {
		return
	}
	r.SetGoldFarm(ref.gf, ref.estimate)
	r.SetRepairTrip(ref.rep)
}

// repairAdapter lets the watch routes talk to the controller without importing it. The reply carries the Farm gold
// status as one additive field, so the gear panel needs no second poll.
type repairAdapter struct {
	c  *repair.Controller
	gf *goldfarm.Controller
}

type repairWithFarm struct {
	repair.View
	Farm *goldfarm.View `json:"farm,omitempty"`
}

func (a repairAdapter) View() any {
	v := repairWithFarm{View: a.c.View()}
	if a.gf != nil {
		f := a.gf.View()
		v.Farm = &f
	}
	return v
}

// Request is the Repair now button; a request that has to farm first also switches the Farm gold order on for itself.
func (a repairAdapter) Request() (any, error) {
	q, err := a.c.Request()
	if err == nil && q.Kind == repair.KindFarm && a.gf != nil {
		err = a.gf.StartForRepair()
	}
	return q, err
}

// Cancel withdraws the request and the farm order it owns.
func (a repairAdapter) Cancel() error {
	err := a.c.Cancel()
	if a.gf != nil {
		_ = a.gf.StopForRepair()
	}
	return err
}

// farmAdapter is the Farm gold desk of the watch routes. Stopping an order a repair request owns withdraws the request
// too (one Stop farming means stop).
type farmAdapter struct {
	c   *goldfarm.Controller
	rep *repair.Controller
}

func (a farmAdapter) View() any { return a.c.View() }
func (a farmAdapter) Set(on bool) error {
	if on {
		return a.c.Start()
	}
	if a.rep != nil && a.c.ForRepair() {
		_ = a.rep.Cancel()
	}
	return a.c.Stop()
}

var (
	_ watch.RepairDesk   = repairAdapter{}
	_ watch.GoldFarmDesk = farmAdapter{}
)
