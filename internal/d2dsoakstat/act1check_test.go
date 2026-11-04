package d2dsoakstat

import (
	"strings"
	"testing"
)

func a1(t *testing.T, log string) map[int]Act1Row {
	t.Helper()
	a, err := ParseAct1(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	m := map[int]Act1Row{}
	for _, r := range a.Rows() {
		m[r.N] = r
	}
	return m
}

// build A: JSON lines (caster build); build B: text lines (melee build). Same checklist, same verdicts.
const act1Pass = `{"time":"2026-01-01T00:00:00Z","msg":"milestone","name":"den","word":"0x0001","clvl":6,"virtual":"12.0"}
{"time":"2026-01-01T00:01:00Z","msg":"milestone","name":"bloodraven","word":"0x0001","clvl":9,"virtual":"40.5"}
{"time":"2026-01-01T00:02:00Z","msg":"boss run","boss":"Andariel","killed":true}
{"time":"2026-01-01T00:02:01Z","msg":"milestone","name":"andariel","word":"0x0001","clvl":19,"virtual":"300.0"}
{"time":"2026-01-01T00:02:02Z","msg":"act1: andariel killed words 6=0x0001 7=0x0000"}
{"time":"2026-01-01T00:03:00Z","msg":"milestone","name":"act2","word":"0x0001","clvl":19,"virtual":"310.0"}
{"time":"2026-01-01T00:04:00Z","msg":"save read","normalQuestWords":["sisters_to_the_slaughter=0x0001"]}
`

const act1Text = `time=2026-01-01T00:00:00Z level=INFO msg=milestone name=den word=0x0001 clvl=5 virtual=9.0
time=2026-01-01T00:02:00Z level=INFO msg="boss run" boss=Andariel killed=true
time=2026-01-01T00:02:02Z level=INFO msg="act1: andariel killed words 6=0x0000 7=0x0001"
time=2026-01-01T00:04:00Z level=INFO msg="memory quest words" normalQuestWords="[den_of_evil=0x0001 sisters_burial_grounds=0x000c]"
`

func TestAct1CheckPassAndUnknown(t *testing.T) {
	r := a1(t, act1Pass)
	if r[4].Verdict != "PASS" || r[8].Verdict != "PASS" {
		t.Errorf("rows 4/8: %+v %+v", r[4], r[8])
	}
	if !strings.Contains(r[4].Detail, "6=0x0001 7=0x0000") {
		t.Errorf("row 4 lacks the words: %s", r[4].Detail)
	}
	for _, n := range []int{2, 3, 5, 6, 7, 9} {
		if r[n].Verdict != "UNKNOWN" {
			t.Errorf("row %d: %+v (undecidable from the log)", n, r[n])
		}
	}
}

func TestAct1CheckFailAndNoT1(t *testing.T) {
	r := a1(t, act1Text)
	// killed=true and word 6 still 0 in the milestone set: no milestone for andariel -> FAIL, even though word 7 is set
	if r[4].Verdict != "FAIL" {
		t.Errorf("row 4: %+v", r[4])
	}
	if r[2].Verdict != "UNKNOWN" {
		t.Errorf("row 2: %+v", r[2])
	}
	if r[3].Verdict != "FAIL" || !strings.Contains(r[3].Detail, "0x000c") { // burial started, not Done
		t.Errorf("row 3: %+v", r[3])
	}
	if r[8].Verdict != "UNKNOWN" {
		t.Errorf("row 8: %+v", r[8])
	}
}

func TestAct1CheckKillWithWordButNoSave(t *testing.T) {
	r := a1(t, strings.Join(strings.Split(act1Pass, "\n")[:5], "\n"))
	if r[4].Verdict != "UNKNOWN" || !strings.Contains(r[4].Detail, "no T1") {
		t.Errorf("row 4: %+v", r[4])
	}
}

func TestAct1CheckNoNewLinesIsNA(t *testing.T) {
	r := a1(t, `{"msg":"boss run","boss":"Andariel","killed":true}`+"\n")
	for n, row := range r {
		if row.Verdict != "n/a" {
			t.Errorf("row %d: %+v", n, row)
		}
	}
}
