package fleet

import "testing"

func TestGoalsArg(t *testing.T) {
	for in, want := range map[string]string{
		"":                                "",
		"campaign:normal_act1":            "campaign:normal_act1",
		"levelto:30":                      "reach:30",
		"farm:Blood_Moor":                 "farm:Blood Moor",
		"campaign:normal_all, levelto:12": "campaign:normal_all;reach:12",
	} {
		got, err := Spec{Goals: in}.GoalsArg()
		if err != nil || got != want {
			t.Errorf("%q: %q, %v (want %q)", in, got, err, want)
		}
	}
	for _, bad := range []string{"reach:8", "levelto:1", "levelto:x", "campaign:nope", "farm:a b", "farm:", "rookie"} {
		if _, err := (Spec{Goals: bad}).GoalsArg(); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	s := Spec{Character: "Abc", Profile: "reach:8"}
	if s.Validate() == nil {
		t.Error("goal string accepted as handicap")
	}
	a, _ := Spec{Goals: "levelto:5"}.BotArgs()
	if len(a) != 3 || a[0] != "--goals" || a[1] != "reach:5" {
		t.Errorf("args %v", a)
	}
	if a, _ := (Spec{}).BotArgs(); a != nil {
		t.Errorf("args %v", a)
	}
}
