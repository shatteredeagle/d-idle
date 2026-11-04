package d2dsoakstat

import (
	"strings"
	"testing"
)

// Two builds (a sorceress and a barbarian log) plus an old log without the new lines.
const lifeSorc = `time=2026-11-04T10:00:00Z level=INFO msg="zone entry" area=2 clvl=9 life=80/80 heal=4 mana=40/40 tome=1 merc=none
time=2026-11-04T10:01:00Z level=INFO msg="zone entry" area=3 clvl=9 life=50/80 heal=3 mana=10/40 tome=1 merc=none
time=2026-11-04T10:02:00Z level=INFO msg="safety decision" action=chicken area=3 life=20 maxLife=80 heal=3 beltMP=0 rejuv=0 tome=1
time=2026-11-04T10:03:00Z level=INFO msg="safety decision" action=town-portal area=3 life=30 maxLife=80 heal=0 beltMP=0 rejuv=0 tome=1
time=2026-11-04T10:04:00Z level=WARN msg="character died" area=3 killer=Fallen life=0 maxLife=80 heal=0
`

const lifeBarb = `{"time":"2026-11-04T10:00:00Z","msg":"zone entry","area":2,"clvl":20,"life":"300/310","heal":6,"mana":"20/20","tome":2,"merc":"alive"}
{"time":"2026-11-04T10:01:00Z","msg":"safety decision","action":"crowd-retreat","area":2,"life":150,"maxLife":310}
`

func TestParseLife(t *testing.T) {
	rows, err := ParseLife(strings.NewReader(lifeSorc))
	if err != nil || len(rows) != 2 {
		t.Fatalf("%v %v", rows, err)
	}
	r := rows[1]
	if r.Area != 3 || r.Entries != 1 || r.Chickens != 1 || r.Deaths != 1 || !r.HasRatio || r.MinRatio != 0.25 {
		t.Errorf("area 3: %+v", r)
	}
	b, _ := ParseLife(strings.NewReader(lifeBarb))
	if len(b) != 1 || b[0].Entries != 1 || b[0].Chickens != 0 || b[0].MinRatio < 0.48 || b[0].MinRatio > 0.49 {
		t.Errorf("barb: %+v", b)
	}
	txt := LifeText("x", rows)
	if !strings.Contains(txt, "3") || !strings.Contains(txt, "25") {
		t.Errorf("text:\n%s", txt)
	}
}

func TestParseLifeOldLog(t *testing.T) {
	rows, err := ParseLife(strings.NewReader(areaOld))
	if err != nil || len(rows) != 0 {
		t.Fatalf("%v %v", rows, err)
	}
	if !strings.Contains(LifeText("old", rows), "n/a") {
		t.Error("no n/a for an old log")
	}
}
