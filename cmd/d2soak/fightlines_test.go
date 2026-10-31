package main

import (
	"os"
	"strings"
	"testing"
)

func TestParseFightLines(t *testing.T) {
	log := `time=2026-10-29T08:01:02.5+02:00 level=INFO msg="combat fight area=3 diff=0 danger=2 kills=4 damage=120 minlife=35 potions=1 dodges=2 sec=20"
time=2026-10-29T08:40:00+02:00 level=INFO msg="combat kill" monster=x
time=2026-10-29T08:50:00+02:00 level=INFO msg="combat fight area=3 diff=0 danger=0 kills=2 damage=30 minlife=80 potions=0 dodges=0 sec=10"
time=2026-10-29T09:05:00+02:00 level=INFO msg="combat fight area=4 diff=1 danger=1 kills=1 damage=5 minlife=95 potions=0 dodges=1 sec=5"
fight area=oops
`
	hs, err := ParseFightLines(strings.NewReader(log))
	if err != nil || len(hs) != 2 {
		t.Fatalf("%v %+v", err, hs)
	}
	a := hs[0]
	if a.Fights != 2 || a.Kills != 6 || a.Damage != 150 || a.Potions != 1 || a.Dodges != 2 || a.Sec != 30 || a.MinLife != 35 {
		t.Fatalf("%+v", a)
	}
	if hs[1].Kills != 1 || !strings.Contains(FightText(hs), "09:00") {
		t.Fatalf("%+v", hs[1])
	}
}

func TestParseFightLinesJSONSample(t *testing.T) {
	f, err := os.Open("testdata/botlog_fight.jsonl") // 20 real lines from a bot run (job livebot/20261029-104840-d2bot)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	hs, err := ParseFightLines(f)
	if err != nil || len(hs) != 2 {
		t.Fatalf("%v %+v", err, hs)
	}
	if h := hs[0]; h.Hour.Hour() != 10 || h.Fights != 15 || h.Kills != 65 || h.Potions != 1 || h.MinLife != 85 {
		t.Errorf("%+v", h)
	}
	if h := hs[1]; h.Hour.Hour() != 11 || h.Fights != 5 || h.Kills != 21 || h.MinLife != 76 {
		t.Errorf("%+v", h)
	}
}
