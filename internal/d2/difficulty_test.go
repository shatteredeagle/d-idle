package d2

import (
	"strings"
	"testing"
)

func TestDoctorLine(t *testing.T) {
	p := &Player{Difficulty: Hell, Level: 70, Area: 107, Resist: Resists{Fire: 90, Cold: 30, Lightning: 75, Poison: 0}}
	l := DoctorLine(p, 85)
	for _, want := range []string{"Hell (byte 2)", "penalty=-100", "res(fire/cold/ltng/pois)=-25/-70/-25/-100", "mlvl=85 gap=15"} {
		if !strings.Contains(l, want) {
			t.Fatalf("%q missing: %s", want, l)
		}
	}
	if strings.Contains(DoctorLine(p, 0), "gap=") || DoctorLine(nil, 0) == "" {
		t.Fatal("optional parts")
	}
	if Normal.ResistPenalty() != 0 || Nightmare.ResistPenalty() != -40 {
		t.Fatal("penalty")
	}
}
