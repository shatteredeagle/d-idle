package main

import (
	"context"
	"sort"
	"testing"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/questmem"
	"d2lod-bot/internal/task"
)

// recorder is a task.Actions that remembers which NPCs, rows, objects and bosses the nodes ask for.
type recorder struct {
	talks   map[[2]string]bool
	objects map[string]bool
	bosses  map[string]bool
}

func (r *recorder) Travel(context.Context, task.Area) error           { return nil }
func (r *recorder) ToTown(context.Context) error                      { return nil }
func (r *recorder) Clear(context.Context, task.Area) error            { return nil }
func (r *recorder) PickUp(context.Context, string) error              { return nil }
func (r *recorder) ActivateWaypoint(context.Context, task.Area) error { return nil }
func (r *recorder) Chores(context.Context) error                      { return nil }
func (r *recorder) LevelTo(context.Context, int) error                { return nil }
func (r *recorder) KillBoss(_ context.Context, n string) error        { r.bosses[n] = true; return nil }
func (r *recorder) Talk(_ context.Context, npc, pick string) error {
	r.talks[[2]string{npc, pick}] = true
	return nil
}
func (r *recorder) Interact(_ context.Context, o string) error  { r.objects[o] = true; return nil }
func (r *recorder) Transmute(context.Context, []string) error   { return nil }
func (r *recorder) HireMerc(context.Context, int, string) error { return nil }
func (r *recorder) UseItem(_ context.Context, i string) error {
	r.objects["item:"+i] = true
	return nil
}
func (r *recorder) UseForge(_ context.Context, _, _ string) error {
	r.objects[bot.ForgePlaceObject], r.objects[bot.ForgeSmashObject] = true, true
	return nil
}
func (r *recorder) SmashOrb(_ context.Context, w string) error {
	r.objects[w] = true
	return nil
}

// Every NPC row, object and boss the Act 2 and 3 nodes use must be understood by the adapter.
func TestActNodesResolveInTheAdapter(t *testing.T) {
	r := &recorder{talks: map[[2]string]bool{}, objects: map[string]bool{}, bosses: map[string]bool{}}
	c := task.Template("normal_all", true)
	for _, e := range c.Entries {
		for _, n := range task.Handler(e.Quest) {
			_ = n.Step.Do(context.Background(), r)
		}
	}
	// the optional Arcane pair and the Act 3 optionals are not in the default entries: run their handlers too
	for _, q := range []string{task.QArcane, task.QSummoner, task.QGolden, task.QBlade, task.QLamEsen, task.QHellforge, task.QPrison, task.QBetrayal} {
		for _, n := range task.Handler(q) {
			_ = n.Step.Do(context.Background(), r)
		}
	}
	for k := range r.talks {
		if _, ok := talkPaths[k]; !ok {
			t.Errorf("no talk path for %v", k)
		}
		if _, ok := npcByAct[k[0]]; !ok {
			t.Errorf("no NPC class for %s", k[0])
		}
	}
	for o := range r.objects {
		if o == task.ItemWill {
			continue // the weapon name SmashOrb gets, not an object
		}
		if _, ok := objSpecs[o]; !ok && o != CairnStones && o[:5] != "item:" {
			t.Errorf("object %q unknown to the adapter", o)
		}
	}
	sc := boss.AllActs()
	for b := range r.bosses {
		if b == task.BossAncients {
			continue // composite: boss.RunAncients over the Talic, Madawc and Korlic scripts
		}
		if b == task.BossFangskin {
			continue // optional guard, no script: the runner reports it as not found
		}
		if sc[b] == nil {
			t.Errorf("boss %q has no script", b)
		}
	}
	if len(r.talks) < 10 || len(r.objects) < 10 || len(r.bosses) < 5 {
		t.Errorf("handlers did less than expected: %d talks, %d objects, %d bosses", len(r.talks), len(r.objects), len(r.bosses))
	}
}

func TestEveryTaskBossHasAScriptAndTheCouncilRunsThree(t *testing.T) {
	sc := boss.AllActs()
	for _, n := range []string{task.BossBR, task.BossSmith, task.BossCount, task.BossAndy, task.BossRadament,
		task.BossSummoner, task.BossDuriel, task.BossCouncil, task.BossMephisto} {
		if n == task.BossBR {
			continue // Blood Raven is killed by the plain fallback, no script (boss.Act1 has none)
		}
		if sc[n] == nil {
			t.Errorf("no script for %q", n)
		}
	}
	if got := newBossRunner(&bossGame{}); got.Scripts[boss.NameCouncil] == nil || got.Scripts[boss.NameMephisto] == nil {
		t.Fatal("the runner must hold the Act 3 scripts")
	}
}

func TestNPCClassByAct(t *testing.T) {
	cases := []struct {
		npc  string
		act  int
		want int
	}{
		{task.NPCWarriv, 1, quest.Warriv}, {task.NPCWarriv, 2, quest.WarrivA2},
		{task.NPCCain, 2, quest.CainA2}, {task.NPCCain, 3, quest.CainA3},
		{task.NPCMeshif, 2, quest.Meshif}, {task.NPCMeshif, 3, quest.MeshifA3},
		{task.NPCTyrael, 4, quest.Tyrael}, {task.NPCTyrael, 2, quest.TyraelTomb}, {task.NPCLarzuk, 5, quest.Larzuk},
		{task.NPCAnya, 5, quest.AnyaNPC}, {task.NPCMalah, 5, quest.Malah}, {task.NPCQualKehk, 5, quest.QualKehk},
		{task.NPCAtma, 2, quest.Atma}, {npcAsheara, 3, quest.Asheara}, {task.NPCAkara, 1, quest.Akara},
	}
	for _, c := range cases {
		if got, ok := npcClassIn(c.npc, c.act); !ok || got != c.want {
			t.Errorf("%s in act %d: %d %v, want %d", c.npc, c.act, got, ok, c.want)
		}
	}
	if _, ok := npcClassIn("Nobody", 2); ok {
		t.Error("unknown NPC resolved")
	}
	if _, ok := npcClassIn(task.NPCMeshif, 1); ok {
		t.Error("Meshif has no act 1 class")
	}
}

// Every id of taskQuestIDs must be a quest of the saves table, or the reader would silently never see it.
func TestTaskQuestIDsExistInSaves(t *testing.T) {
	var bad []string
	for tid, sid := range taskQuestIDs {
		if _, ok := questmem.QuestIndex(sid); !ok {
			bad = append(bad, tid+"->"+sid)
		}
	}
	sort.Strings(bad)
	if len(bad) > 0 {
		t.Fatalf("unknown saves ids: %v", bad)
	}
}

func TestUnknownBossScriptIsNotFoundSoOptionalStepsSkip(t *testing.T) {
	r := newBossRunner(&bossGame{})
	err := r.KillBoss(context.Background(), task.BossFangskin)
	if task.Classify(mapBossErr(err)) != task.KindNotFound {
		t.Fatalf("got %v (%v)", err, task.Classify(mapBossErr(err)))
	}
}

// No paid Kashya hire: an Act 1 hire request is refused before any dialog is touched.
func TestMercHirerRefusesKashya(t *testing.T) {
	err := mercHirer(nil, nil)(context.Background(), 1, "")
	if err == nil || task.Classify(err) != task.KindPrereq {
		t.Fatalf("err %v kind %v", err, task.Classify(err))
	}
}
