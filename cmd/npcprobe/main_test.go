package main

import "testing"

func TestParsePlan(t *testing.T) {
	p, err := parsePlan(defaultPlan)
	if err != nil || len(p) != 4 || p[2].class != 147 {
		t.Fatalf("%v %v", p, err)
	}
	for _, bad := range []string{"npc", "foo:1", "npc:x"} {
		if _, err := parsePlan(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
