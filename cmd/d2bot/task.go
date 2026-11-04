package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/saves"
	"d2lod-bot/internal/talk"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town"
)

// Task-layer capabilities for campaign goals (docs/task.md section 3.5).

// taskQuestIDs maps the task layer's quest ids to the saves table ids.
var taskQuestIDs = map[string]string{
	task.QDen:      "den_of_evil",
	task.QBurial:   "sisters_burial_grounds",
	task.QCain:     "search_for_cain",
	task.QTower:    "forgotten_tower",
	task.QTools:    "tools_of_the_trade",
	task.QAndariel: "sisters_to_the_slaughter",
	task.QAct1Done: "act1_done",
	// Act 2 (word index and name V saves/progress.go). The fetch legs (cube, viper, shaft) have no word of their own and
	// run on the item proxies; "act2_complete" is the word set by Jerhyn's talk.
	task.QRadament: "radament",
	task.QStaff:    "horadric_staff",
	task.QArcane:   "arcane_sanctuary",
	task.QSummoner: "summoner",
	task.QDuriel:   "seven_tombs",
	task.QAct2Word: "act2_done",
	// Act 3. Eye, Brain and Heart have no word either; the Council is "The Blackened Temple" (word 21, V that the word
	// exists, U that it is set by the Flail pickup or the kill).
	task.QLamEsen:  "lam_esen",
	task.QKhalim:   "khalims_will",
	task.QBlade:    "blade_old_religion",
	task.QGolden:   "golden_bird",
	task.QCouncil:  "blackened_temple",
	task.QGuardian: "guardian",
	// Acts 4 and 5: the ids are the saves ids themselves (internal/saves/progress.go). The hop (QAct4Done) has no word
	// of its own: the world's act decides. Words 37 and 40 are V live, the others S (docs/saves.md).
	task.QIzual:     "izual",
	task.QHellforge: "hellforge",
	task.QDiablo:    "terrors_end",
	task.QSiege:     "siege_harrogath",
	task.QRescue:    "rescue_arreat",
	task.QPrison:    "prison_of_ice",
	task.QBetrayal:  "betrayal_harrogath",
	task.QAncients:  "rite_of_passage",
	task.QBaal:      "eve_of_destruction",
}

// saveQuestReader is the bot's QuestReader over the character's .d2s (T1 evidence, quest.SaveSource). The file only
// changes on Save and Exit, so it is PRIOR knowledge: a quest the save shows started or done is reported, a quest it
// shows untouched is left to the in-game proxies (a step that just succeeded would otherwise look failed until the
// next save). This is what keeps Den of Evil from being redone after a restart.
type saveQuestReader struct {
	path func() string // current .d2s path; "" when the character name is not known yet
	read func(string) (saves.Progress, error)
	log  func(msg string, args ...any) // optional: reports what the save says whenever it changes
	// mem is the in-game memory source (internal/questmem), nil without a game reader. It is listed BEFORE the save: its words
	// are exact and change at once (docs/questmem.md), the save is prior knowledge that only lags behind.
	mem memQuests

	mu    sync.Mutex
	mtime time.Time
	size  int64
	prog  saves.Progress
	have  bool

	memKey string // last logged memory words

	now        func() time.Time // clock of the milestone log (virtual under lockstep); nil = wall
	t0         time.Time
	msDone     map[string]bool // Act 1 milestones already logged (milestone.go)
	andyLogged bool
}

// memQuests is questmem.Source as the reader needs it.
type memQuests interface {
	quest.Source
	Waypoint(diff, area int) (active, known bool)
}

func newSaveQuestReader(dir, name func() string) *saveQuestReader {
	return &saveQuestReader{read: saves.ProgressFile, path: func() string {
		d, n := dir(), name()
		if d == "" || n == "" {
			return ""
		}
		return filepath.Join(d, n+".d2s")
	}}
}

func (r *saveQuestReader) load() (saves.Progress, bool) {
	p := r.path()
	if p == "" {
		return saves.Progress{}, false
	}
	st, err := os.Stat(p)
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		return r.prog, r.have
	}
	if r.have && st.ModTime().Equal(r.mtime) && st.Size() == r.size {
		return r.prog, true
	}
	prog, err := r.read(p)
	if err != nil { // a half-written file: keep the last good read
		return r.prog, r.have
	}
	r.prog, r.have, r.mtime, r.size = prog, true, st.ModTime(), st.Size()
	if r.log != nil {
		var words []string
		for _, q := range prog.Difficulty[0].Quests {
			if q.Word != 0 {
				words = append(words, fmt.Sprintf("%s=0x%04x", q.ID, q.Word))
			}
		}
		r.log("save read", "file", filepath.Base(p), "level", prog.Level, "normalQuestWords", words)
	}
	return prog, true
}

// status is the quest status the save shows (Needed when unknown); it decides which rows a dialog shows.
func (r *saveQuestReader) status(quest_ string) quest.Status {
	var srcs []quest.Source
	if r.mem != nil {
		srcs = append(srcs, r.mem)
	}
	if prog, ok := r.load(); ok {
		srcs = append(srcs, quest.SaveSource{P: prog})
	}
	return quest.NewReader(srcs...).Status(0, quest_)
}

// logMem reports the memory words of the Normal quests whenever they change (live confirmation of a claim or a kill).
func (r *saveQuestReader) logMem(diff int) {
	if r.mem == nil || r.log == nil || diff != 0 {
		return
	}
	var words []string
	known := 0
	for _, sid := range taskQuestIDs {
		if w, ok := r.mem.Word(diff, sid); ok {
			known++
			if w != 0 {
				words = append(words, fmt.Sprintf("%s=0x%04x", sid, w))
			}
		}
	}
	sort.Strings(words)
	key := strings.Join(words, " ")
	if known == 0 {
		key = "unavailable"
		if pr, ok := r.mem.(interface {
			Progress() (*d2.PlayerProgress, error)
		}); ok {
			_, err := pr.Progress()
			key += fmt.Sprint(err)
		}
	}
	r.mu.Lock()
	changed := key != r.memKey
	r.memKey = key
	r.mu.Unlock()
	if changed && known == 0 {
		r.log("memory quest words unavailable (the save is used)", "why", key)
	} else if changed {
		r.log("memory quest words", "normalQuestWords", words)
	}
}

func (r *saveQuestReader) Read(diff int) (bot.QuestView, bool) {
	if diff < 0 || diff > 2 {
		return bot.QuestView{}, false
	}
	prog, haveSave := r.load()
	// memory first, the save as prior knowledge (a source that does not know a quest, e.g. out of game, falls through)
	var srcs []quest.Source
	if r.mem != nil {
		srcs = append(srcs, r.mem)
	}
	if haveSave {
		srcs = append(srcs, quest.SaveSource{P: prog})
	}
	if len(srcs) == 0 {
		return bot.QuestView{}, false
	}
	rd := quest.NewReader(srcs...)
	r.logMem(diff)
	r.logMilestones(diff)
	v := bot.QuestView{Quests: map[string]task.QuestState{}, WPs: map[int]bool{}}
	for tid, sid := range taskQuestIDs {
		info := rd.Quest(diff, sid)
		if info.Evidence == quest.EvNone {
			continue
		}
		ev := task.EvSave
		if info.Evidence == quest.EvMemory {
			ev = task.EvMemory
		}
		switch {
		case info.Status == quest.Done:
			v.Quests[tid] = task.QuestState{Stage: task.StageDone, Ev: ev}
		case info.Status == quest.Started && info.Pending:
			v.Quests[tid] = task.QuestState{Stage: task.StageCleared, Ev: ev}
		case info.Status == quest.Started:
			v.Quests[tid] = task.QuestState{Stage: task.StageStarted, Ev: ev}
		}
	}
	for _, wp := range prog.Difficulty[diff].Waypoints {
		on := wp.Active
		if r.mem != nil {
			if a, known := r.mem.Waypoint(diff, wp.Level); known {
				on = a
			}
		}
		if on {
			v.WPs[wp.Level] = true
		}
	}
	return v, true
}

// saveDirOf is the folder with the instance's character saves.
func saveDirOf(ctx context.Context, o options) string {
	if o.managed {
		if mc, err := managedConfig(ctx, o); err == nil {
			return mc.SaveDir()
		}
	}
	return cheat.DefaultSaveDir(o.instance)
}

func newTaskDeps(ctx context.Context, o options, tw *town.Town, nv *nav.Navigator, e *env.Env, name func() string) *bot.TaskDeps {
	dir := saveDirOf(ctx, o)
	rd := newSaveQuestReader(func() string { return dir }, name)
	rd.log = e.Log.Info
	if e.Clock != nil {
		rd.now = e.Clock.Now
	}
	rd.mem = memQuestSource(e) // questmem needs the memory reader; d2d has its own source (d2dbackend.go)
	return &bot.TaskDeps{
		Dialog:   talk.Adapter{T: tw, E: e, Status: rd.status},
		Objects:  objectAdapter{e: e, nv: nv, clicked: &onceSet{}},
		Cube:     func(ctx context.Context, in []string) error { _, err := tw.Transmute(ctx, in); return err },
		HireMerc: mercHirer(tw, e),
		Waypoint: nv.ActivateWaypoint,
		Quests:   rd,
		// the memory words change at once; without a game reader the save is all there is
		QuestsLive: rd.mem != nil,
	}
}
