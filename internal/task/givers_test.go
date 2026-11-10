package task

import (
	"context"
	"strings"
	"testing"
)

type fakeSession struct{ finished, blocked map[string]bool }

func (f fakeSession) Finished(g string) bool          { return f.finished[g] }
func (f fakeSession) Blocked(g string) (string, bool) { return "blocked", f.blocked[g] }

func giverCampaign(t *testing.T, class string, onErr func(error)) *Campaign {
	t.Helper()
	gt, err := ShippedGivers()
	if err != nil {
		t.Fatal(err)
	}
	return NormalAct1(Act1Options{}).WithQuestGivers(gt, func() string { return class }, onErr)
}

func campWorld(level int) World {
	return World{Act: 1, Difficulty: "normal", Area: RogueEncampment, InTown: true, Level: level, Potions: 6, HasTP: true,
		WPs: map[Area]bool{ColdPlains: true}, Quests: map[string]QuestState{QDen: {Stage: StageDone, Ev: EvSave}}}
}

func TestShippedGiversParse(t *testing.T) {
	gt, err := ShippedGivers()
	if err != nil {
		t.Fatal(err)
	}
	if len(gt.Givers) != 2 {
		t.Fatalf("givers %d", len(gt.Givers))
	}
	for class, want := range map[string]string{"Paladin": "melee", "sorceress": "caster", "Necromancer": "summoner"} {
		if f, err := gt.Family(class); err != nil || f != want {
			t.Errorf("%s: %q %v, want %q", class, f, err, want)
		}
	}
	if _, err := gt.Family("Wizard"); err == nil || !strings.Contains(err.Error(), "Wizard") {
		t.Fatalf("unknown class must be named: %v", err)
	}
}

func TestParseGiversErrorsNameTheQuest(t *testing.T) {
	hints := []byte(`{"families":{"paladin":{"family":"melee"}},"quests":[{"id":"q1","giver":"Akara","clvl":{"melee":1},"available":{}}]}`)
	ok := `{"schema":"quests.givers/1","givers":[{"quest":"q1","npc":"Akara","accept":{"pick":"quest","title":"t","why":"w"}}]}`
	if _, err := ParseGivers([]byte(ok), hints); err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string]string{
		"unknown quest": strings.Replace(ok, `"q1"`, `"nope"`, 1),
		"wrong giver":   strings.Replace(ok, `"Akara"`, `"Kashya"`, 1),
		"no action":     `{"schema":"quests.givers/1","givers":[{"quest":"q1","npc":"Akara"}]}`,
		"no title":      strings.Replace(ok, `"title":"t",`, ``, 1),
		"bad schema":    strings.Replace(ok, "givers/1", "givers/9", 1),
	} {
		_, err := ParseGivers([]byte(src), hints)
		if err == nil {
			t.Errorf("%s: want an error", name)
		} else if name == "unknown quest" && !strings.Contains(err.Error(), "nope") {
			t.Errorf("error must name the quest: %v", err)
		}
	}
}

// Kashya's task is taken at the class's own level (melee 7, caster 8) once Cold Plains was reached, for two classes.
func TestGiverVisitByClassFamily(t *testing.T) {
	for _, tc := range []struct {
		class string
		level int
		visit bool
	}{
		{"Paladin", 6, false}, {"Paladin", 7, true},
		{"Sorceress", 7, false}, {"Sorceress", 8, true},
	} {
		p := giverCampaign(t, tc.class, nil).Plan(campWorld(tc.level), nil)
		got := strings.HasPrefix(p.ID, "givers.sisters_burial")
		if got != tc.visit {
			t.Errorf("%s clvl %d: plan %q, visit want %v", tc.class, tc.level, p.ID, tc.visit)
		}
		if got && (len(p.Nodes) != 1 || p.Why == "" || !p.Nodes[0].Step.Optional) {
			t.Errorf("%s: the visit is one optional talk step with a reason: %+v", tc.class, p)
		}
	}
}

func TestGiverVisitConditions(t *testing.T) {
	c := giverCampaign(t, "Paladin", nil)
	w := campWorld(10)
	visit := func(w World, s Session) bool {
		return strings.HasPrefix(c.Plan(w, s).ID, "givers.")
	}
	if !visit(w, nil) {
		t.Fatal("level and area met: visit")
	}
	off := campWorld(10)
	off.WPs = nil
	if visit(off, nil) {
		t.Error("Cold Plains never reached: no visit")
	}
	field := campWorld(10)
	field.InTown, field.Area = false, ColdPlains
	if visit(field, nil) {
		t.Error("not in town: no visit")
	}
	nm := campWorld(10)
	nm.Difficulty = "nightmare"
	if visit(nm, nil) {
		t.Error("Normal only")
	}
	given := campWorld(10)
	given.Quests[QBurial] = QuestState{Stage: StageStarted, Ev: EvSave}
	if visit(given, nil) {
		t.Error("already given: no visit")
	}
	if visit(w, fakeSession{finished: map[string]bool{giverGroup(QBurial, ErrandAccept): true}}) {
		t.Error("finished this session: no repeat")
	}
	if visit(w, fakeSession{blocked: map[string]bool{giverGroup(QBurial, ErrandAccept): true}}) {
		t.Error("blocked: no visit")
	}
	// the objective is met and the reward is not claimed: the turn-in visit
	done := campWorld(10)
	done.Quests[QBurial] = QuestState{Stage: StageCleared, Ev: EvSave}
	if p := c.Plan(done, nil); p.ID != "givers.sisters_burial.turnin" {
		t.Errorf("turn-in plan %q", p.ID)
	}
	// without the opt-in the plan is today's plan
	plain := NormalAct1(Act1Options{}).Plan(w, nil)
	if strings.HasPrefix(plain.ID, "givers.") {
		t.Error("a campaign without WithQuestGivers must not visit givers")
	}
	// a quest the campaign skips is not given either (Tools of the Trade skipped by SkipTools)
	nt := NormalAct1(Act1Options{SkipTools: true})
	gt, _ := ShippedGivers()
	nt.WithQuestGivers(gt, func() string { return "Paladin" }, nil)
	tw := campWorld(30)
	tw.Quests[QBurial] = QuestState{Stage: StageDone, Ev: EvSave}
	tw.WPs[27] = true
	if p := nt.Plan(tw, nil); strings.Contains(p.ID, "tools_of_the_trade") {
		t.Errorf("Tools is not in this campaign: %q", p.ID)
	}
}

func TestGiverUnknownClassIsReported(t *testing.T) {
	var got error
	c := giverCampaign(t, "Wizard", func(err error) { got = err })
	p := c.Plan(campWorld(20), nil)
	if got == nil || !strings.Contains(got.Error(), "Wizard") {
		t.Fatalf("error must name the class: %v", got)
	}
	if strings.HasPrefix(p.ID, "givers.") {
		t.Error("an unknown class makes no visit")
	}
}

func TestGiverStepAsksTheGiver(t *testing.T) {
	c := giverCampaign(t, "Paladin", nil)
	p := c.Plan(campWorld(12), nil)
	if len(p.Nodes) != 1 {
		t.Fatalf("%+v", p)
	}
	var npc, pick string
	if err := p.Nodes[0].Step.Do(context.Background(), recordTalk{npc: &npc, pick: &pick}); err != nil {
		t.Fatal(err)
	}
	if npc != NPCKashya || pick != "quest" {
		t.Fatalf("talked to %q with %q", npc, pick)
	}
}

type recordTalk struct {
	Actions
	npc, pick *string
}

func (r recordTalk) Talk(_ context.Context, npc, pick string) error {
	*r.npc, *r.pick = npc, pick
	return nil
}
