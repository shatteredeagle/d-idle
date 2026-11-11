package main

import (
	"path/filepath"
	"sync"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town"
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
		d.SetCainOpen(cainOpen(ref))
	}
	if lv != nil {
		d.SetEquipper(lv)
		d.SetTrasher(lv)
	}
	if lt != nil {
		lt.SetIgnore(d.Ignored) // items the player trashed stay on the ground
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

var _ watch.InvDesk = invAdapter{}

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
