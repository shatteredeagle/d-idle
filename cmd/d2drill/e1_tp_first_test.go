package main

import "testing"

func TestE1TPFirstEvals(t *testing.T) {
	buy := `{"msg":"shop: bought","item":"Scroll of Town Portal"}`
	mark := `{"msg":"gear: scroll before gear"}`
	gear := `{"msg":"leveler: buying","item":"Cap"}`
	for _, c := range []struct {
		name  string
		lines []string
		want  string
	}{
		{"scroll first", []string{mark, buy, gear}, "PASS"},
		{"gear first", []string{mark, gear, buy}, "FAIL"},
		{"no restock", []string{gear}, "FAIL"},
		{"restock bought nothing", []string{mark, gear}, "FAIL"},
	} {
		if v, why := e1TPFirstEvalA(c.lines); v != c.want {
			t.Errorf("%s: %s (%s)", c.name, v, why)
		}
	}
	if v, _ := e1TPFirstEvalB([]string{mark}); v != "FAIL" {
		t.Errorf("control with a restock: %s", v)
	}
	if v, _ := e1TPFirstEvalB([]string{`{"msg":"gear shop"}`}); v != "PASS" {
		t.Errorf("control: %s", v)
	}
}

func TestE1EquipEval(t *testing.T) {
	fail := `{"msg":"leveler: equip failed","swap":"Cap 12.0 -> head (was Cap 4.0)"}`
	eq := `{"msg":"gear equip"}`
	for _, c := range []struct {
		name  string
		lines []string
		want  string
	}{
		{"no failure", []string{eq}, "INCONCLUSIVE"},
		{"failure, no leave", []string{fail, eq, `{"msg":"task step"}`}, "PASS"},
		{"failure then fresh game", []string{fail, eq, `{"msg":"leveling: everything worth farming is cleared, starting a fresh game"}`}, "FAIL"},
		{"unbounded", []string{fail, fail, fail, eq}, "FAIL"},
	} {
		if v, why := e1EquipEval(c.lines); v != c.want {
			t.Errorf("%s: %s (%s)", c.name, v, why)
		}
	}
	if e1Swap(fail) != "Cap 12.0 -> head (was Cap 4.0)" {
		t.Error("swap text")
	}
}
