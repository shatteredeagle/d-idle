package watch

import "testing"

func TestQuestSection(t *testing.T) {
	var lines []string
	sec := QuestSection(func() []string { return lines })
	if sec.Build(&ReturnData{}) != nil {
		t.Error("no lines must hide the section")
	}
	lines = []string{"Den of Evil is paused: the hero died 2 times on this part. Since 2026-11-25 22:00 UTC."}
	v := sec.Build(&ReturnData{})
	if v == nil || v.ID != QuestSectionID || len(v.Lines) != 1 || len(v.Buttons) != 0 {
		t.Errorf("%+v", v)
	}
}
