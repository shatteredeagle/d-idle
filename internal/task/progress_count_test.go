package task

import (
	"testing"

	"d2lod-bot/internal/goals"
)

// goals cannot import task, so it carries the quest count of each template; keep the two equal.
func TestGoalsCampaignQuestCount(t *testing.T) {
	for _, cain := range []bool{false, true} {
		if got, want := goals.CampaignQuests(goals.CampaignNormalAct1, cain), len(NormalAct1(Act1Options{Cain: cain}).Entries); got != want {
			t.Errorf("cain=%v: goals says %d quests, task has %d", cain, got, want)
		}
	}
}

func TestGoalsTemplateCounts(t *testing.T) {
	for _, cain := range []bool{false, true} {
		for _, name := range goals.CampaignTemplates {
			want := goals.CampaignQuests(name, cain)
			c := Template(name, cain)
			if c == nil {
				if want != 0 {
					t.Errorf("%s: goals says %d quests but task has no builder", name, want)
				}
				continue
			}
			if len(c.Entries) != want {
				t.Errorf("%s cain=%v: goals says %d quests, task has %d", name, cain, want, len(c.Entries))
			}
		}
	}
}
