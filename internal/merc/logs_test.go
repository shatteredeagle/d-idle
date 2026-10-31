package merc

import "testing"

func TestLogLines(t *testing.T) {
	m := MercState{Class: 271, Alive: true, LifePct: 87, Level: 9}
	for _, c := range []struct{ got, want, kind string }{
		{LogHired(338, "Holy Freeze", 360, 1500, 1140), "merc: hired class=338 aura=Holy_Freeze price=360 gold_before=1500 gold_after=1140", LineHired},
		{LogSeen(m), "merc: seen class=271 alive=true life=87 level=9 aura=-", LineSeen},
		{LogRevived(600, 9), "merc: revived price=600 level=9", LineRevived},
		{LogReviveSkipped(ReasonGold, 600, 900), "merc: revive skipped (gold) price=600 gold=900", LineReviveSkipped},
		{LogReviveSkipped(ReasonStopped, 600, 900), "merc: revive skipped (stopped by deaths) price=600 gold=900", LineReviveSkipped},
		{LogEquip(SlotHelm, "Skull Cap", "empty slot"), "merc: equip slot=helm item=Skull_Cap reason=empty_slot", LineEquip},
		{LogStrip(SlotWeapon, "Partizan", 8), "merc: strip slot=weapon item=Partizan cells=8", LineStrip},
		{LogReqcheck("Skull Cap", ReqResult{true, "ok"}, false, m), "merc: reqcheck item=Skull_Cap ok=true reason=ok rule=level_only str=0 dex=0 level=9", LineReqcheck},
		{LogReqcheck("", ReqResult{false, "stats unknown"}, true, m), "merc: reqcheck item=- ok=false reason=stats_unknown rule=str_dex str=0 dex=0 level=9", LineReqcheck},
	} {
		if c.got != c.want {
			t.Errorf("got  %q\nwant %q", c.got, c.want)
		}
		if KindOf(c.got) != c.kind {
			t.Errorf("KindOf(%q)=%q want %q", c.got, KindOf(c.got), c.kind)
		}
	}
	for _, l := range []string{"", "merc:", "merc: unknown x=1", "boss: held", "merc: hiredx"} {
		if KindOf(l) != "" {
			t.Errorf("KindOf(%q)", l)
		}
	}
}

func TestKnobsFromDefaults(t *testing.T) {
	k := DefaultKnobs()
	want := Knobs{2, 2, 3, 5, 3, false, true, 10, true, 40}
	if k != want {
		t.Errorf("%+v", k)
	}
}
