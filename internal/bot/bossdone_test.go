package bot

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// bossWorld: the boss run and the dialog of the fake game, with the Seven Tombs word (record 14) the reader reports.
// fightKills makes the boss run end in "not found" after the general fight killed the boss (the act2drill5 sequence).
type bossWorld struct {
	g          *rig
	mu         sync.Mutex
	word       uint16
	runs       int
	talks      int
	fightKills bool
	talkSets   uint16 // bits a talk adds to the word
}

func (b *bossWorld) Boss(_ context.Context, name string) error {
	b.mu.Lock()
	b.runs++
	fk := b.fightKills
	b.mu.Unlock()
	if fk {
		b.g.r.noteKill(name)
		return boss.ErrBossNotFound
	}
	return nil
}

func (b *bossWorld) Talk(_ context.Context, _, _ string) error {
	b.mu.Lock()
	b.talks++
	b.word |= b.talkSets
	b.mu.Unlock()
	return nil
}

func (b *bossWorld) Read(int) (QuestView, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	v := QuestView{Quests: map[string]task.QuestState{}}
	if b.word != 0 {
		v.Quests[task.QDuriel] = task.QuestState{Stage: task.StageStarted, Ev: task.EvMemory, Raw: b.word}
	}
	return v, true
}

func bossRig(t *testing.T, class d2.Class, word uint16) (taskActs, *bossWorld) {
	g := newRig(t)
	g.w.class = class
	g.r.e.Tune.Quests.StepExec = 1
	bw := &bossWorld{g: g, word: word}
	g.r.d.Task = &TaskDeps{Boss: bw.Boss, Dialog: bw, Quests: bw}
	g.r.syncTaskGame()
	return taskActs{r: g.r}, bw
}

// A boss already dead is a done kill step without a boss run, whichever way the game says so: the kill the observer
// saw, the dead unit in the snapshot, or the quest word bit the data row names (record 14 bit 0x0020 for Duriel).
func TestKillBossAlreadyDeadIsDone(t *testing.T) {
	dead := map[string]func(a taskActs, bw *bossWorld){
		"seen kill": func(a taskActs, _ *bossWorld) { a.r.noteKill(task.BossDuriel) },
		"corpse": func(_ taskActs, bw *bossWorld) {
			bw.g.w.set(func(w *world) {
				w.mons = append(w.mons, d2.Unit{ID: 77, Type: d2.TypeMonster, Class: 211, Mode: d2.MonModeDead, Monster: &d2.MonsterInfo{Name: "Duriel"}})
			})
		},
		"quest bit": func(_ taskActs, bw *bossWorld) { bw.word = 0x0024 },
	}
	for _, cl := range execClasses {
		for how, kill := range dead {
			a, bw := bossRig(t, cl, 0x0004)
			kill(a, bw)
			if err := a.KillBoss(context.Background(), task.BossDuriel); err != nil || bw.runs != 0 {
				t.Fatalf("class %v %s: err %v runs %d", cl, how, err, bw.runs)
			}
			if !a.r.loop.task.killed[task.BossDuriel] {
				t.Fatalf("class %v %s: the campaign does not see the kill", cl, how)
			}
		}
	}
}

// Must fail: a living boss (word 0x0004, a live unit, no kill seen) is fought: the boss run starts.
func TestKillBossAliveIsFought(t *testing.T) {
	for _, cl := range execClasses {
		a, bw := bossRig(t, cl, 0x0004)
		bw.g.w.set(func(w *world) {
			w.mons = append(w.mons, d2.Unit{ID: 77, Type: d2.TypeMonster, Class: 211, Mode: 1, Monster: &d2.MonsterInfo{Name: "Duriel"}})
		})
		if err := a.KillBoss(context.Background(), task.BossDuriel); err != nil || bw.runs != 1 {
			t.Fatalf("class %v: err %v runs %d", cl, err, bw.runs)
		}
	}
}

// The act2drill5 stall: the general fight kills the boss while the boss run looks for him. "Not found" then reads the
// post and is done; with the boss still alive it stays the boss run's error.
func TestKillBossNotFoundButDead(t *testing.T) {
	for _, cl := range execClasses {
		a, bw := bossRig(t, cl, 0x0004)
		bw.fightKills = true
		if err := a.KillBoss(context.Background(), task.BossDuriel); err != nil || bw.runs != 1 {
			t.Fatalf("class %v: err %v runs %d", cl, err, bw.runs)
		}
		a, bw = bossRig(t, cl, 0x0004)
		bw.g.r.d.Task.Boss = func(context.Context, string) error { bw.runs++; return boss.ErrBossNotFound }
		err := a.KillBoss(context.Background(), task.BossDuriel)
		if !errors.Is(err, boss.ErrBossNotFound) || a.r.loop.task.killed[task.BossDuriel] {
			t.Fatalf("class %v: must fail while he lives: err %v", cl, err)
		}
	}
}

// The hand-in rows are data: Tyrael's talk is Done when record 14 gains bit 0x2000 (S d2d SevenTombsQuest), and a talk
// that moves nothing uses the row's two tries and fails.
func TestTyraelHandInRow(t *testing.T) {
	for _, cl := range execClasses {
		a, bw := bossRig(t, cl, 0x0024)
		bw.talkSets = 0x2008
		if err := a.Talk(context.Background(), task.NPCTyrael, "reward"); err != nil || bw.talks != 1 {
			t.Fatalf("class %v: err %v talks %d", cl, err, bw.talks)
		}
		a, bw = bossRig(t, cl, 0x0024)
		err := a.Talk(context.Background(), task.NPCTyrael, "reward")
		if err == nil || !strings.Contains(err.Error(), "not observed") || bw.talks != 2 {
			t.Fatalf("class %v: must fail: err %v talks %d", cl, err, bw.talks)
		}
	}
}
