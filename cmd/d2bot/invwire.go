package main

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/unlock"
	"d2lod-bot/internal/watch"
)

// The inventory page on the real bot (docs/inventory-ui.md): the desk holds the player's sell and keep marks, equip
// requests and slot pins. The town honours the marks, the leveler and the town leave pinned slots alone, and the runner
// performs queued equips at the gear step in town. Backend independent: both runLive and runD2D call these.

// newInvDesk makes the desk over the display's item snapshot and the runner (late-bound through ref). The state file
// lives in the bot's state directory, so marks and pins survive a restart.
func newInvDesk(stateDir string, d *display, ref *runnerRef) error {
	cfg := invdesk.Config{
		Inventory: invSource{d}.Inventory,
		Advice: func(it status.Item) string {
			if r := ref.p.Load(); r != nil {
				return r.GearAdviceFor(it.ID)
			}
			return ""
		},
		Prompt: func() string {
			if r := ref.p.Load(); r != nil {
				return r.GearPrompt()
			}
			return ""
		},
		Snap: func() *d2.Snapshot {
			if r := ref.p.Load(); r != nil {
				return r.LastSnapshot()
			}
			return nil
		},
	}
	if stateDir != "" {
		cfg.Path = filepath.Join(stateDir, invdesk.StateFile)
	}
	desk, err := invdesk.New(cfg)
	if err != nil {
		return err
	}
	ref.inv = desk
	return newRepair(stateDir, d, ref)
}

// pinned is leveler.Options.Pinned; safe before the desk exists.
func (r *runnerRef) pinned(body int) bool {
	if r == nil {
		return false
	}
	return r.inv.Pinned(body)
}

// wireInv connects the desk to the town, the looter's verdicts and the leveler once they exist.
func wireInv(ref *runnerRef, tw *town.Town, lt *loot.Looter, lv *leveler.Leveler, db *data.DB) {
	if ref == nil || ref.inv == nil {
		return
	}
	d := ref.inv
	wireRepair(ref, tw)
	if tw != nil {
		tw.SetItemMarks(d)
		tw.SetPinned(d.Pinned)
	}
	if tw != nil {
		d.SetServices(tw)
		d.SetTidyFor(tw.TidyWhyFor)
		d.SetCainOpen(cainOpen(ref))
	}
	if lv != nil {
		d.SetEquipper(lv)
		d.SetTrasher(lv)
		d.SetUnequipper(lv)
		d.SetHands(lv)
	}
	if lt != nil {
		// items the player trashed and items the Tidy chore dropped stay on the ground (no drop-pickup loop)
		lt.SetIgnore(func(u d2.Unit) bool { return d.Ignored(u) || tw.TidyDropped(u) })
	}
	if lt != nil {
		d.SetWhyFor(func(s *d2.Snapshot) func(u d2.Unit) string {
			ev := lt.EvaluatorFor(s) // one context per page view; EvaluateUnit would read the game once per item
			return func(u d2.Unit) string { return ev(u).String() }
		})
	}
	if db != nil {
		never := func(u d2.Unit) string {
			var b *data.ItemBase
			if u.Class >= 0 && u.Class < len(db.Items) {
				b = db.Items[u.Class]
			}
			return town.NeverSellReason(b, u)
		}
		d.SetSellBlock(never)
		d.SetTrashBlock(never) // quest items, keys, tomes and the start kit are never trashed either
	}
}

var _ invdesk.Services = (*town.Town)(nil)

// cainOpenTTL is how long the quest answer is reused: the page polls every few seconds and the reader stats the save.
const cainOpenTTL = 5 * time.Second

// cainOpen says whether Deckard Cain is rescued: the Search for Cain quest reads Done for the hero's difficulty
// (memory first, then the save; V that the words exist, docs/saves.md), or Cain stands in the town the hero is in (a
// monster of his classes; U live). Either one opens the "Identify all (Cain)" command.
func cainOpen(ref *runnerRef) func(*d2.Snapshot) bool {
	var (
		mu   sync.Mutex
		at   time.Time
		diff = -1
		done bool
	)
	return func(s *d2.Snapshot) bool {
		if s == nil || s.Player == nil {
			return false
		}
		if s.Player.InTown() {
			for i := range s.Monsters {
				if m := &s.Monsters[i]; m.Alive() && town.IsCainClass(m.Class) {
					return true
				}
			}
		}
		r := ref.p.Load()
		if r == nil {
			return false
		}
		mu.Lock()
		defer mu.Unlock()
		if now := time.Now(); diff != int(s.Player.Difficulty) || now.Sub(at) > cainOpenTTL {
			diff, at, done = int(s.Player.Difficulty), now, false
			if v, ok := r.QuestView(diff); ok {
				done = v.Quests[task.QCain].Stage == task.StageDone
			}
			// the guided journey saw the Find Cain chapter done (a chapter's unlocks list, data/campaign/normal.json)
			done = done || (ref.jc != nil && ref.jc.Unlocked(journeyIdentifyCain))
		}
		return done
	}
}

// invAdapter lets the watch routes talk to the desk without importing it.
type invAdapter struct{ d *invdesk.Desk }

func (a invAdapter) View() any                         { return a.d.View() }
func (a invAdapter) SetMark(id uint32, m string) error { return a.d.SetMark(id, m) }
func (a invAdapter) Unpin(slot string) error           { return a.d.Unpin(slot) }
func (a invAdapter) RequestEquip(id uint32, slot string) (any, error) {
	return a.d.RequestEquip(id, slot)
}
func (a invAdapter) RequestTrash(id uint32, confirm bool) (any, error) {
	return a.d.RequestTrash(id, confirm)
}
func (a invAdapter) Cancel(id uint32, kind string) error { return a.d.Cancel(id, kind) }
func (a invAdapter) RequestIdentify(id uint32) (any, error) {
	return a.d.RequestIdentify(id)
}
func (a invAdapter) RequestBuyID(n int) (any, error) { return a.d.RequestBuyID(n) }
func (a invAdapter) RequestCain() (any, error)       { return a.d.RequestCain() }

func (a invAdapter) RequestStash(id uint32) (any, error) { return a.d.RequestStash(id) }
func (a invAdapter) RequestTake(id uint32) (any, error)  { return a.d.RequestTake(id) }
func (a invAdapter) RequestGold(deposit bool, n int) (any, error) {
	return a.d.RequestGold(deposit, n)
}

// The schema 7 part (docs/gearsets.md): unequip, the merc as a target and the player's gear sets.
func (a invAdapter) RequestUnequip(slot, target string) (any, error) {
	return a.d.RequestUnequipFor(slot, target)
}
func (a invAdapter) RequestEquipFor(id uint32, slot, target string) (any, error) {
	return a.d.RequestEquipFor(id, slot, target)
}
func (a invAdapter) SaveSet(name, target string) (any, error) { return a.d.SaveSet(name, target) }
func (a invAdapter) RenameSet(id, name string) error          { return a.d.RenameSet(id, name) }
func (a invAdapter) DeleteSet(id string) error                { return a.d.DeleteSet(id) }
func (a invAdapter) PreviewSet(id string) (any, error)        { return a.d.PreviewSet(id) }
func (a invAdapter) ApplySet(id string) (any, error)          { return a.d.ApplySet(id) }
func (a invAdapter) CancelApply(group string) error           { return a.d.CancelApply(group) }
func (a invAdapter) SetSetsGate(g func() (bool, string))      { a.d.SetSetsGate(g) }

var _ watch.InvGearDesk = invAdapter{}
var _ watch.InvDesk = invAdapter{}
var _ watch.InvStashDesk = invAdapter{}

// invAPI is what newFun mounts: nil when there is no desk.
func invAPI(d *invdesk.Desk) *watch.InvAPI {
	if d == nil {
		return nil
	}
	api := watch.NewInvAPI()
	api.Bind(invAdapter{d})
	return api
}

// playerGear is bot.Deps.PlayerGear: the desk, or a true nil (not a typed nil) when there is none.
func playerGear(ref *runnerRef) bot.PlayerGear {
	if ref == nil || ref.inv == nil {
		return nil
	}
	return ref.inv
}

// setsGate is the unlock of the gear sets (ladder row `gearsets`, docs/gearsets.md): open when the row is open, when there
// is no ladder at all, or on a DEV image / sandbox (DIDLE_DEV=1, the existing dev switch, docs/dev-update.md). A ladder
// that lacks the row keeps the feature off, like every other ui row.
func setsGate(eng *unlock.Engine, getenv func(string) string) func() (bool, string) {
	return func() (bool, string) {
		if getenv("DIDLE_DEV") == "1" || eng == nil {
			return true, ""
		}
		if eng.Allowed(invdesk.SetsLadderID) {
			return true, ""
		}
		for _, s := range eng.States("ui") {
			if s.ID == invdesk.SetsLadderID {
				return false, s.Teaser
			}
		}
		return false, "Gear sets are not part of this ladder"
	}
}

// bindSetsGate gives the inventory page's desk the unlock gate once the ladder exists.
func bindSetsGate(api *watch.InvAPI, fun *watch.Fun) {
	if api == nil {
		return
	}
	var eng *unlock.Engine
	if fun != nil && fun.Unlocks != nil {
		eng = fun.Unlocks.Engine
	}
	api.SetSetsGate(setsGate(eng, os.Getenv))
}

// wireSets reads the knob invdesk.maxSets at call time (a changed knob applies to the next save).
func wireSets(ref *runnerRef, e *env.Env) {
	if ref == nil || ref.inv == nil || e == nil {
		return
	}
	ref.inv.SetMaxSets(func() int {
		if e.Tune == nil {
			return 0
		}
		return e.Tune.Invdesk.MaxSets
	})
}
