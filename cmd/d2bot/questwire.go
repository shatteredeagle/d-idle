package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/ether"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/watch"
)

// The quest board wiring (slice S4, docs/questboard.md "API"): the margin query behind the readiness words, the per-hero
// plan store, the desk behind GET and POST /api/quests and the glance rows. Everything here is inert while the knob
// quests.board is 0: the Exec answers "off" and boardStep never runs.

// questStoreDir is where the plan records live. The files are named <hero>.quests.json, so a Character Select switch needs
// no parking (internal/charsel/perchar.json lists only files that carry no hero name); each hero keeps its own file and its
// .bak next to it.
func questStoreDir(stateDir string) string { return filepath.Join(stateDir, "quests") }

// survShadow is the survival model the margin query reads (set by newSurvivalShadow; nil without game tables).
var survShadow atomic.Pointer[survivalShadow]

// questMargin is bot.Deps.QuestMargin: read only, never writes the ledger and never touches the zone choice.
func questMargin(s *d2.Snapshot, boss, diff string, potions int) (questboard.Hero, *questboard.Encounter, bool) {
	w := survShadow.Load()
	if w == nil {
		return questboard.Hero{}, nil, false
	}
	return w.questMargin(s, boss, diff, potions)
}

// wireQuestBoard gives the runner the board maker and the margin query, and returns the desk the watch routes use (nil
// without a state directory: the plan records need a place to live). Safe to call with the knob at 0.
func wireQuestBoard(stateDir string, ref *runnerRef, r *bot.Runner, logf func(msg string, kv ...any)) *questboard.Desk {
	if ref == nil || r == nil || stateDir == "" {
		return nil
	}
	b, err := questboard.Load()
	if err != nil {
		logf("quest board off", "err", err)
		return nil
	}
	dir := questStoreDir(stateDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		logf("quest board off: no plan directory", "err", err)
		return nil
	}
	r.SetQuestBoardMaker(func(hero string) (*questboard.Board, *questboard.Store, error) {
		st, err := questboard.Open(b, dir, hero)
		if err == nil {
			importJourney(stateDir, hero, st, logf)
		}
		return b, st, err
	})
	r.SetQuestMargin(questMargin)
	r.SetQuestLedger(questLedger(currencyDirFor(stateDir)))
	d := questboard.NewDesk(b, questboard.DeskConfig{
		Exec:      r.QuestBoard,
		Paused:    func() bool { return r.PlayerPauseView().State != bot.PauseRunning },
		Explain:   r.QuestReadiness,
		Class:     r.QuestClass,
		Level:     r.QuestLevel,
		Meter:     r.QuestMeter,
		Queued:    r.QuestQueued,
		HandInWhy: r.QuestHandInWhy,
		Refusal:   r.QuestRefusal,
	})
	ref.qd = d
	startOperator(stateDir, d, r, logf)
	return d
}

// bindQuests mounts the quest board routes on the watch screen.
func bindQuests(f *watch.Fun, d *questboard.Desk) {
	if f == nil || d == nil {
		return
	}
	api := watch.NewQuestsAPI()
	api.Bind(d)
	f.QuestBoard = api
}

func init() {
	// The video card strip and the Needs-you rows of the quest board (design 4.6, slice S4).
	glanceExtra = append(glanceExtra, func(src *watch.GlanceSources, d glanceDeps) {
		if d.Quests == nil {
			return
		}
		src.QuestOn = func() bool {
			qd := d.Quests()
			return qd != nil && qd.View().Enabled
		}
		src.Quest = func() (watch.GlanceQuest, bool) {
			qd := d.Quests()
			if qd == nil {
				return watch.GlanceQuest{}, false
			}
			s := qd.View().Strip
			if s == nil {
				return watch.GlanceQuest{}, false
			}
			return watch.GlanceQuest{Quest: s.Quest, Title: s.QuestTitle, Chunk: s.Chunk, Part: s.Title, State: string(s.State), Step: s.Step, NPC: s.NPC}, true
		}
		prev := src.Needs
		src.Needs = func() []watch.GlanceNeed {
			var out []watch.GlanceNeed
			if prev != nil {
				out = prev()
			}
			if qd := d.Quests(); qd != nil {
				out = append(out, questNeeds(qd.View().Needs)...)
			}
			return out
		}
	})
}

func questNeeds(rows []questboard.NeedRow) []watch.GlanceNeed {
	var out []watch.GlanceNeed
	for _, n := range rows {
		g := watch.GlanceNeed{ID: n.ID, Kind: n.Kind, Sev: n.Sev, Text: n.Text, Sub: n.Sub, Actions: []watch.GlanceVerb{}}
		for _, a := range n.Actions {
			body, _ := json.Marshal(a.Body)
			g.Actions = append(g.Actions, watch.GlanceVerb{Label: a.Label, Post: a.Post, Body: body})
		}
		out = append(out, g)
	}
	return out
}

// questLedger is the board's view of the account wallet (slice S10). The wallet is looked up on every call: it appears when
// loot.vault turns on and is nil (the whole hook inert) while it is 0. Quest keys are quest/<quest>/<chunk or ->/<difficulty>.
func questLedger(currencyDir string) func() questboard.Ledger {
	return func() questboard.Ledger {
		w := etherFor(currencyDir, "", nil, nil).Wallet()
		if w == nil {
			return nil
		}
		return walletLedger{w}
	}
}

type walletLedger struct{ w *ether.Wallet }

func chunkWord(chunk string) string {
	if chunk == "" {
		return "-"
	}
	return chunk
}

func (l walletLedger) Earn(quest, chunk, diff string, amount int) (questboard.LedgerResult, error) {
	r, err := l.w.Earn("quest", quest+"/"+chunkWord(chunk), diff, amount)
	return questboard.LedgerResult{Key: r.Key, Credited: r.Credited, Already: r.Already}, err
}

func (l walletLedger) Paid(quest, chunk, diff string) (int, bool) {
	return l.w.Paid("quest/" + quest + "/" + chunkWord(chunk) + "/" + diff)
}

// importJourney takes the decisions of the retired Journey (state/campaign.json) into the hero's plan, once. The
// Journey's file is only read: a copy named campaign.json.pre-board is made next to it (kept if it exists) and the
// original is never changed or removed, so rolling back to an older build still finds it. A missing file is a hero
// that never used the Journey; an unreadable one is logged and the hero starts clean (the board derives progress from
// the game, so nothing is lost either way).
func importJourney(stateDir, hero string, st *questboard.Store, logf func(msg string, kv ...any)) {
	path := filepath.Join(stateDir, campaign.StateFile)
	if _, err := os.Stat(path); err != nil {
		return
	}
	if _, err := os.Stat(path + ".pre-board"); err != nil {
		if b, err := os.ReadFile(path); err == nil {
			_ = os.WriteFile(path+".pre-board", b, 0o600)
		}
	}
	cs, err := campaign.LoadStore(path)
	if err != nil {
		logf("journey import skipped: state file unreadable", "err", err)
		return
	}
	sp, err := campaign.Shipped(campaign.Validator{})
	if err != nil {
		logf("journey import skipped: no chapter table", "err", err)
		return
	}
	h, ok := cs.Heroes[hero]
	if !ok {
		return
	}
	var l questboard.Legacy
	for id, in := range h.Intents {
		if ch := sp.Chapter(id); ch != nil && in == campaign.Skip && ch.Quest != "" {
			l.SkippedQuests = append(l.SkippedQuests, ch.Quest)
		}
	}
	sort.Strings(l.SkippedQuests)
	l.OrdersMode = h.Mode == campaign.Orders
	n, err := st.ImportLegacy(l, 0)
	if err != nil {
		logf("journey import failed", "hero", hero, "err", err)
		return
	}
	logf("journey import: decisions taken into the quest plan", "hero", hero, "skips", n, "orders", l.OrdersMode)
}
