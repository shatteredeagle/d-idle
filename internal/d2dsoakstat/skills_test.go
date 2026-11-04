package d2dsoakstat

import (
	"strings"
	"testing"
)

const skillsJSON = `{"msg":"combat decision","act":"cast","skill":"Charged Bolt"}
{"msg":"combat decision","act":"cast","skill":"Charged Bolt"}
{"msg":"combat decision","act":"cast","skill":"Attack"}
{"msg":"combat decision","act":"nova","n":3}
{"msg":"combat decision","act":"nova","n":5}
{"msg":"combat buff","skill":"Frozen Armor"}
{"msg":"combat buff cut by a hit, will retry","skill":"Frozen Armor","try":1}
{"msg":"killtime","monster":"x"}
{"msg":"killtime","monster":"y"}
{"msg":"start","branch":"d2d-main","commit":"abc123"}
`

const skillsText = `msg="combat decision" act=cast skill="Fire Bolt" target=x
msg="combat decision" act=cast skill="Fire Bolt" target=y
msg="combat buff" skill="Thunder Storm"
msg=killtime monster=z
`

func TestSkillsJSONAndText(t *testing.T) {
	s, err := ParseSkills(strings.NewReader(skillsJSON))
	if err != nil {
		t.Fatal(err)
	}
	if s.Casts["Charged Bolt"] != 2 || s.Novas != 2 || s.NovaMonsters != 8 || s.Kills != 2 || s.BuffRetries["Frozen Armor"] != 1 || s.Commit != "abc123" {
		t.Fatalf("%+v", s)
	}
	out := s.Text("a", "Attack")
	for _, want := range []string{"Charged Bolt (attack): casts started 2", "casts/kill 1.0", "monsters per nova 4.0", "staff swings 1",
		"Frozen Armor (buff): casts started 1", "recasts after a cut 1", "d2d-main abc123", "mana spent n/a (not logged)"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	// second build/class: text log, other skills, no commit, no melee name
	s2, _ := ParseSkills(strings.NewReader(skillsText))
	out2 := s2.Text("b", "")
	for _, want := range []string{"Fire Bolt (attack): casts started 2", "casts/kill 2.0", "Thunder Storm (buff)", "novas 0", "d2d build n/a (not logged)", "staff swings n/a"} {
		if !strings.Contains(out2, want) {
			t.Errorf("missing %q in\n%s", want, out2)
		}
	}
}

const fieldsJSON = `{"msg":"build","branch":"agent/x","commit":"abc1234"}
{"msg":"combat decision","act":"cast","skill":"Charged Bolt"}
{"msg":"cast result","skill":"Charged Bolt","made":true,"cut":false,"cost":24,"ticks":13}
{"msg":"cast result","skill":"Charged Bolt","made":false,"cut":true,"cost":24,"ticks":13,"kills":1}
{"msg":"combat decision","act":"cast","skill":"Fire Ball"}
{"msg":"cast result","skill":"Fire Ball","made":true,"cut":false,"cost":10,"ticks":13,"targets":3}
{"msg":"cast result","skill":"Chain Lightning","made":true,"cut":false,"cost":30,"ticks":19}
{"msg":"combat kill","monster":"a","last_skill":"Charged Bolt"}
{"msg":"combat buff","skill":"Frozen Armor"}
{"msg":"minute","armor":"Frozen Armor"}
{"msg":"minute","armor":"none"}
{"msg":"minute","armor":"?"}
{"msg":"killtime","monster":"a"}
`

const fieldsText = `msg="cast result" skill="Fire Bolt" made=true cut=false cost=5 ticks=13
msg="combat decision" act=cast skill="Fire Bolt"
msg="combat kill" monster=z last_skill="Fire Bolt"
msg=killtime monster=z
`

func TestSkillsLogFields(t *testing.T) {
	s, _ := ParseSkills(strings.NewReader(fieldsJSON))
	out := s.Text("a", "")
	for _, want := range []string{"made effect 1 of 2", "cut before action frame 1 of 2", "mana spent 48 (24.0 per cast)",
		"kills 1 credited (last skill of the kill; 1 inside the cast window)", "casts/kill 1.0", "hits/cast 3.0 monsters in the area (1 casts, not confirmed hits)", "hits/cast n/a (not logged), no missile-hit signal", "seconds absent ~60 (1 of 2 minute samples without armor, 1 unreadable)",
		"agent/x abc1234", "Chain Lightning 19, Charged Bolt 13, Fire Ball 13 (modelled"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	s2, _ := ParseSkills(strings.NewReader(fieldsText))
	out2 := s2.Text("b", "")
	for _, want := range []string{"Fire Bolt (attack): casts started 1, made effect 1 of 1", "mana spent 5 (5.0 per cast)", "cast length ticks Fire Bolt 13 (modelled"} {
		if !strings.Contains(out2, want) {
			t.Errorf("missing %q in\n%s", want, out2)
		}
	}
}

func TestSkillsEmpty(t *testing.T) {
	s, _ := ParseSkills(strings.NewReader("garbage\n"))
	out := s.Text("e", "")
	if !strings.Contains(out, "no skill lines") || !strings.Contains(out, "n/a (no kills)") {
		t.Errorf("%s", out)
	}
}
