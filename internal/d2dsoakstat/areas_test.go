package d2dsoakstat

import (
	"strings"
	"testing"
)

const areaNew = `time=2026-01-01T00:00:00Z level=INFO msg=minute area=2 clvl=5 xp=1000 kills=10 deaths=0 gold=0 restarts=0 goldPickup=0
time=2026-01-01T00:01:00Z level=INFO msg=minute area=2 clvl=5 xp=1600 kills=20 deaths=0 gold=0 restarts=0 goldPickup=30
time=2026-01-01T00:01:10Z level=INFO msg="combat kill" monster=Fallen class=19 boss=false xp=20
time=2026-01-01T00:02:00Z level=INFO msg=minute area=2 clvl=5 xp=2200 kills=30 deaths=0 gold=0 restarts=0 goldPickup=90
time=2026-01-01T00:02:30Z level=INFO msg="area summary" area=2 minutes=2 xp=1200 kills=20 goldPickup=120
`

const areaOld = `time=2026-01-01T00:00:00Z level=INFO msg=minute area=3 clvl=5 xp=1000 kills=10 deaths=0 gold=0 restarts=0
time=2026-01-01T00:01:00Z level=INFO msg=minute area=3 clvl=5 xp=1300 kills=15 deaths=0 gold=0 restarts=0
`

func TestAreasNewLog(t *testing.T) {
	s, err := ParseAreas(strings.NewReader(areaNew))
	if err != nil {
		t.Fatal(err)
	}
	out := s.Text("new")
	for _, want := range []string{"2 ", "36000", "20", "60.0", "3600", "area summary lines: 1", "Fallen", "class=19", "xp/kill=20.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
}

func TestAreasOldLogNA(t *testing.T) {
	s, _ := ParseAreas(strings.NewReader(areaOld))
	out := s.Text("old")
	if !strings.Contains(out, "18000") || !strings.Contains(out, "n/a") || !strings.Contains(out, "xp per kill by monster: n/a") {
		t.Fatalf("\n%s", out)
	}
	e, _ := ParseAreas(strings.NewReader("garbage\n"))
	if !strings.Contains(e.Text("e"), "no minute lines: n/a") {
		t.Fatal("empty")
	}
}
