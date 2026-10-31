package d2dctl

import "testing"

func TestParseSpikeLines(t *testing.T) {
	// whitespace-separated, exactly as printed in docs/d2d-spike.md section 4
	m, err := ParseMonster("fallen1  12.10 28.70 752 436 2/2 NU - - lvl1 Fallen #2 -")
	if err != nil || m.Name != "Fallen" || m.Unit != 2 || m.CellX != 12.1 || m.SY != 436 {
		t.Fatalf("%+v %v", m, err)
	}
	n, err := ParseNPC("Kashya  812 484 1 0 35.7 13.5")
	if err != nil || n.Name != "Kashya" || n.SX != 812 || n.CellY != 13.5 {
		t.Fatalf("%+v %v", n, err)
	}
	g, err := ParseGround("gld  8800 Gold  371 331 #1 40.9 11.5")
	if err != nil || g.Label != "8800 Gold" || g.SX != 371 || len(g.Extra) != 2 {
		t.Fatalf("%+v %v", g, err)
	}
	w, err := ParseWarp("0 67.0 28.0 8")
	if err != nil || w.To != 8 {
		t.Fatalf("%+v %v", w, err)
	}
	ml, err := ParseMenuLine("Aliza - Lvl: 5  Life: 63  Def: 31  Cost: 130   400 164")
	if err != nil || ml.X != 400 || ml.Y != 164 || ml.Text != "Aliza - Lvl: 5 Life: 63 Def: 31 Cost: 130" {
		t.Fatalf("%+v %v", ml, err)
	}
}

func TestParseTolerant(t *testing.T) {
	for _, bad := range []string{"", "garbage", "a\tb", "1 2 3"} {
		// must not panic; Raw is kept
		if m, _ := ParseMonster(bad); m.Raw != bad {
			t.Fatal("raw lost")
		}
		ParseNPC(bad)
		ParseGround(bad)
		ParseMenuLine(bad)
		ParsePlayer(bad)
		ParseCorpse(bad)
		ParseWarp(bad)
		ParsePreset(bad)
		ParseObject(bad)
		ParsePortal(bad)
		parseLevel(bad)
		parseQuest(bad)
	}
	// an extra trailing column and an unknown boss word survive
	m, err := ParseMonster("a 1 2 3 4 5/9 NU weird 7,x lvl3 Big Guy #11 A1:2 extra1 extra2")
	if m.Name != "Big Guy" || m.Boss != "weird" || len(m.Extra) != 2 || m.Row != "A1:2" || len(m.Mods) != 1 {
		t.Fatalf("%+v", m)
	}
	if err == nil {
		t.Fatal("bad mod id should be reported")
	}
	// layout drift
	if _, err := ParseMonster("a 1 2 3 4 5/9 NU - - level3 X #1 -"); err == nil {
		t.Fatal("expected layout error")
	}
}

func TestKV(t *testing.T) {
	kv := ParseKV("w=800 h=600 loose frame=9 w=1")
	if kv.M["w"] != "1" || len(kv.Order) != 3 || kv.Loose[0] != "loose" {
		t.Fatalf("%+v", kv)
	}
	s := parseState("difficulty=? life=4/45")
	if s.Difficulty != -1 || s.Life.Cur != 4 {
		t.Fatalf("%+v", s)
	}
	if !parseState("gamemenu=1:0").GameMenuOpen() || parseState("gamemenu=0:0").GameMenuOpen() {
		t.Fatal("gamemenu")
	}
}

func TestSubtile(t *testing.T) {
	if SubTile(30.7) != 153 || Cell(153) != 30.6 {
		t.Fatal(SubTile(30.7), Cell(153))
	}
}
