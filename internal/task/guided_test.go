package task_test

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/task"
)

func guide(t *testing.T, class string, st *campaign.HeroState) *task.Guide {
	t.Helper()
	sp, err := campaign.Shipped(campaign.Validator{Quest: func(q string) bool { return task.Handler(q) != nil }})
	if err != nil {
		t.Fatal(err)
	}
	return &task.Guide{Spine: sp, State: func() *campaign.HeroState { return st }, Class: func() string { return class },
		Opts: func() campaign.Opts { return campaign.Opts{Now: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)} }}
}

func guidedWorld(lvl int, done ...string) task.World {
	w := task.World{Act: 1, Difficulty: "normal", Level: lvl, Gold: 2000, Potions: 10, HasTP: true,
		Quests: map[string]task.QuestState{}, WPs: map[task.Area]bool{task.ColdPlains: true}}
	for _, q := range done {
		w.Quests[q] = task.QuestState{Stage: task.StageDone, Ev: task.EvMemory}
		w.MercKnown = w.MercKnown || q == task.QBurialMerc // the Rogue was claimed and is dead now (the live hero)
	}
	return w
}

type blockedSession map[string]bool

func (b blockedSession) Finished(string) bool { return false }
func (b blockedSession) Blocked(g string) (string, bool) {
	return "test", b[g]
}

// The live Paladin of 2026-11-11: the floor planner grinds level.16 for Andariel (Cain off, the optionals skipped);
// Guided mode sends it to Cain.
func TestGuidedPaladinClvl12(t *testing.T) {
	w := guidedWorld(12, task.QDen, task.QBurial, task.QBurialMerc)
	w.AndarielMinLevel, w.ImbueMinLevel, w.ImbueUseful, w.TowerMinLevel = 16, 20, true, 15
	if p := task.NormalAct1(task.Act1Options{}).Plan(w, nil); p.ID != "level.16" {
		t.Fatalf("floor planner: %q (%s), want level.16", p.ID, p.Why)
	}
	st := campaign.NewHeroState()
	c := task.NormalAct1(task.Act1Options{}).WithGuide(guide(t, "paladin", st))
	if p := c.Plan(w, nil); p.Group != task.QCain || len(p.Nodes) == 0 {
		t.Fatalf("guided: %q group %q why %q", p.ID, p.Group, p.Why)
	}
	// Cain blocked this session (the executor gave up): the guide moves on and the persisted state stays clean.
	p := c.Plan(w, blockedSession{task.QCain: true})
	if p.Group == task.QCain || len(st.Intents) != 0 {
		t.Fatalf("blocked Cain: %q, intents %v", p.ID, st.Intents)
	}
	if p.ID != "level.13" || !strings.Contains(p.Nodes[0].Step.Title, "for Charsi's hammer") {
		t.Errorf("after Cain: %q %q", p.ID, p.Nodes[0].Step.Title)
	}
}

func TestGuidedSorceressFarmsWithChapterTitle(t *testing.T) {
	st := campaign.NewHeroState()
	c := task.NormalAct1(task.Act1Options{}).WithGuide(guide(t, "sorceress", st))
	p := c.Plan(guidedWorld(7, task.QDen), nil)
	if p.ID != "level.10" || p.Nodes[0].Step.Title != "Level up to 10 for Blood Raven" || !strings.Contains(p.Why, "to unlock Blood Raven") {
		t.Fatalf("%q %q %q", p.ID, p.Nodes[0].Step.Title, p.Why)
	}
	_ = st.SetGoAt(campaign.GoFloor)
	st.SetAnswer("blood_raven", true, 9)
	if p := c.Plan(guidedWorld(9, task.QDen), nil); p.Group != task.QBurial {
		t.Fatalf("floor policy, said yes: %q %q", p.ID, p.Why)
	}
}

// A safety wall of the floor planner (the Monastery route guard) raises the band: no walk the guard refuses.
func TestGuidedWallRaisesFloor(t *testing.T) {
	st := campaign.NewHeroState()
	_ = st.SetIntent(nil0(t), "find_cain", campaign.Skip, time.Time{})
	c := task.NormalAct1(task.Act1Options{}).WithGuide(guide(t, "paladin", st))
	w := guidedWorld(13, task.QDen, task.QBurial, task.QBurialMerc)
	w.RouteMinLevel = 14
	p := c.Plan(w, nil)
	if p.ID != "level.14" || !strings.Contains(p.Nodes[0].Step.Title, "for The Countess") || !strings.Contains(p.Why, "for level 14") {
		t.Fatalf("%q %q %q", p.ID, p.Nodes[0].Step.Title, p.Why)
	}
}

func nil0(t *testing.T) *campaign.Spine { return guide(t, "paladin", nil).Spine }

func TestGuidedFreeModeAndGoNow(t *testing.T) {
	st := campaign.NewHeroState()
	c := task.NormalAct1(task.Act1Options{}).WithGuide(guide(t, "paladin", st))
	w := guidedWorld(5)
	if p := c.Plan(w, nil); p.Group != task.QDen {
		t.Fatalf("guided clvl 5: %q", p.ID)
	}
	_ = st.SetMode(campaign.Free)
	if got, want := c.Plan(w, nil), task.NormalAct1(task.Act1Options{}).Plan(w, nil); got.ID != want.ID {
		t.Errorf("free mode %q, floor planner %q", got.ID, want.ID)
	}
	_ = st.SetMode(campaign.Guided)
	_ = st.SetGoNow(nil0(t), "andariel")
	if p := c.Plan(w, nil); p.Group != task.QAndariel || !strings.Contains(p.Why, "too_early") {
		t.Errorf("go now: %q %q", p.ID, p.Why)
	}
	_ = st.SetMode(campaign.Orders)
	if p := c.Plan(w, nil); p.ID != "level.6" {
		t.Errorf("orders mode: %q", p.ID)
	}
}

func TestGuidedHandsOverAfterAct1(t *testing.T) {
	c := task.NormalActs12(task.Act1Options{}, task.Act2Options{Merc: true}).WithGuide(guide(t, "paladin", campaign.NewHeroState()))
	w := guidedWorld(20, task.QDen, task.QBurial, task.QBurialMerc, task.QCain, task.QTools, task.QTower, task.QAndariel)
	w.Act = 2
	if p := c.Plan(w, nil); !strings.HasPrefix(p.ID, "quest.") || p.Group == task.QAndariel {
		t.Errorf("act 2: %q", p.ID)
	}
}

// Blood Raven done, no merc ever employed: the Rogue chapter runs the claim; a claimed Rogue that died does not.
func TestGuidedClaimsTheRogue(t *testing.T) {
	c := task.NormalAct1(task.Act1Options{}).WithRogueClaim().WithGuide(guide(t, "paladin", campaign.NewHeroState()))
	w := guidedWorld(12, task.QDen, task.QBurial)
	if p := c.Plan(w, nil); p.Group != task.QBurialMerc || len(p.Nodes) == 0 {
		t.Fatalf("no merc: %q %q", p.ID, p.Why)
	}
	w.MercKnown = true
	if p := c.Plan(w, nil); p.Group == task.QBurialMerc {
		t.Fatalf("dead Rogue claimed again: %q", p.ID)
	}
}
