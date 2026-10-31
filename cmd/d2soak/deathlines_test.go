package main

import (
	"strings"
	"testing"
)

func TestParseDeathLines(t *testing.T) {
	log := `time=2026-10-29T10:00:00Z level=WARN msg="death diff=Hell area=108 clvl=70 mlvl=85 gap=15 killer=Oblivion_Knight class=311 boss=false affixes=ManaBurn,Cursed dist=4 build=sorceress_blizzard_endgame90"
time=x level=INFO msg="fight area=1 kills=3"
time=2026-10-29T10:05:00Z level=WARN msg="death diff=Hell area=108 clvl=71 mlvl=0 gap=0 killer=- class=0 boss=false affixes=- dist=0 build=sorceress_blizzard_endgame90"
death diff=Normal area=2
`
	ds, err := ParseDeathLines(strings.NewReader(log))
	if err != nil || len(ds) != 2 {
		t.Fatalf("%v %+v", err, ds)
	}
	if d := ds[0]; d.Build != "sorceress_blizzard_endgame90" || d.Gap != 15 || d.Affixes != "ManaBurn,Cursed" || d.CLvl != 70 || d.Area != 108 {
		t.Errorf("%+v", d)
	}
	txt := DeathText(ds)
	if !strings.Contains(txt, "sorceress_blizzard_endgame90 Hell 2 +15.0 Cursed") {
		t.Errorf("\n%s", txt)
	}
}

func TestParseDeathLinesJSON(t *testing.T) {
	log := `{"time":"2026-10-29T10:00:00-04:00","level":"WARN","msg":"death diff=Hell area=108 clvl=70 mlvl=85 gap=15 killer=Oblivion_Knight class=311 boss=true affixes=ManaBurn,Cursed dist=4 build=sorc","inst":"i4"}
{"time":"x","level":"INFO","msg":"death report","total":0}
`
	ds, err := ParseDeathLines(strings.NewReader(log))
	if err != nil || len(ds) != 1 || ds[0].Build != "sorc" || ds[0].Gap != 15 || !ds[0].Boss || ds[0].Affixes != "ManaBurn,Cursed" {
		t.Fatalf("%v %+v", err, ds)
	}
}
