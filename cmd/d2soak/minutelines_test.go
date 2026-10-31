package main

import (
	"strings"
	"testing"
)

const minText = `time=2026-10-29T10:01:00Z level=INFO msg=minute inst=i1 area=108 clvl=70 xp=1000 kills=10 deaths=0 potions=2 gold=500 restarts=0 rung=0 goal=farm step="Clearing area"
time=2026-10-29T10:02:00Z level=INFO msg=status area=3
time=2026-10-29T10:03:00Z level=INFO msg=minute inst=i1 area=108 clvl=71 xp=3000 kills=25 deaths=1 potions=5 gold=900 restarts=1 rung=2 goal=farm step="Clearing area"
time=2026-10-29T10:04:00Z level=INFO msg=minute area=bad
`

func TestParseMinuteLinesText(t *testing.T) {
	ms, err := ParseMinuteLines(strings.NewReader(minText))
	if err != nil || len(ms) != 2 {
		t.Fatalf("got %d, %v", len(ms), err)
	}
	if ms[0].Step != "Clearing area" || ms[1].Kills != 25 || ms[1].At.IsZero() {
		t.Fatalf("%+v", ms)
	}
	out := MinuteText(ms)
	for _, w := range []string{"clvl 70 -> 71", "kills 15", "deaths 1", "potions used 3", "restarts 1", "rung 2", "farm/Clearing area"} {
		if !strings.Contains(out, w) {
			t.Fatalf("%q missing:\n%s", w, out)
		}
	}
}

func TestParseMinuteLinesJSON(t *testing.T) {
	j := `{"time":"2026-10-29T10:01:00Z","level":"INFO","msg":"minute","area":3,"clvl":5,"xp":77,"kills":1,"deaths":0,"potions":0,"gold":9,"restarts":0,"rung":0,"goal":"idle","step":"x"}` + "\nnot json\n"
	ms, _ := ParseMinuteLines(strings.NewReader(j))
	if len(ms) != 1 || ms[0].XP != 77 || ms[0].Goal != "idle" {
		t.Fatalf("%+v", ms)
	}
	if !strings.Contains(MinuteText(nil), "no summary") {
		t.Fatal("empty text")
	}
}

func TestParseMinuteLinesBeltFields(t *testing.T) {
	l := `time=2026-10-29T10:01:00Z level=INFO msg=minute area=3 clvl=5 xp=77 kills=40 killsGame=9 deaths=0 potions=4 mana=2 potionsUsed=11 gold=9 restarts=0 rung=0 goal=farm step=x state=farming rstate=clearing` + "\n"
	ms, _ := ParseMinuteLines(strings.NewReader(l))
	if len(ms) != 1 {
		t.Fatalf("%+v", ms)
	}
	m := ms[0]
	if m.Potions != 4 || m.Mana != 2 || m.PotionsUsed != 11 || m.KillsGame != 9 || m.State != "farming" || m.RState != "clearing" || !m.HasBelt {
		t.Fatalf("%+v", m)
	}
	old, _ := ParseMinuteLines(strings.NewReader(minText))
	if old[0].PotionsUsed != 2 || old[0].Potions != 0 || old[0].HasBelt || old[0].KillsGame != 10 {
		t.Fatalf("legacy %+v", old[0])
	}
}
