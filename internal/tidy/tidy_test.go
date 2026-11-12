package tidy

import "testing"

func pol(t *testing.T) *Policy {
	t.Helper()
	p, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func pack(f Facts) Facts { f.Place = "pack"; return f }

func TestTable(t *testing.T) {
	p := pol(t)
	cases := []struct {
		name string
		f    Facts
		c    Ctx
		want Action
		why  string
	}{
		{"rune", Facts{Code: "r01", Type: "rune"}, Ctx{}, Stash, "Will stash: rune"},
		{"chipped gem", Facts{Type: "gem0"}, Ctx{}, Stash, "Will stash: gem"},
		{"charm", Facts{Type: "mcha", Quality: "magic"}, Ctx{}, Stash, "Will stash: charm"},
		{"unique", Facts{Type: "swor", Quality: "unique", SellValue: 5}, Ctx{}, Stash, "Will stash: unique or set item"},
		{"magic ring", Facts{Type: "ring", Quality: "magic"}, Ctx{}, Stash, ""},
		{"tome", Facts{Code: "tbk"}, Ctx{}, Keep, "Kept: tome, scroll or key"},
		{"quest", Facts{Type: "misc", Quest: true}, Ctx{}, Keep, "Kept: quest item"},
		{"white sword worth 50", Facts{Type: "swor", Quality: "normal", SellValue: 50}, Ctx{}, Sell, "Will sell: vendor junk, value 50"},
		{"6 gold club, pack roomy", Facts{Type: "club", Quality: "normal", SellValue: 6}, Ctx{PackFullPct: 40}, Ignore, ""},
		{"6 gold club, pack full", Facts{Type: "club", Quality: "normal", SellValue: 6}, Ctx{PackFullPct: 90}, Drop, "Will drop: worth under 20 gold and the pack is full, value 6"},
		{"4 socket white", Facts{Type: "armo", Quality: "normal", Sockets: 4, SellValue: 30}, Ctx{}, Stash, ""},
	}
	for _, c := range cases {
		d := p.Decide(pack(c.f), c.c)
		if d.Action != c.want || c.why != "" && d.Why != c.why {
			t.Errorf("%s: got %s %q, want %s %q", c.name, d.Action, d.Why, c.want, c.why)
		}
	}
}

// Equipment and everything outside the pack is never touched, whatever the item is worth or says.
func TestNeverTouchesEquipmentOrOtherPlaces(t *testing.T) {
	p := pol(t)
	for _, place := range []string{"equipment", "belt", "stash", "cube", "cursor", "ground", ""} {
		for _, f := range []Facts{{Type: "rune"}, {Type: "club", SellValue: 1}, {Type: "swor", Quality: "unique"}} {
			f.Place = place
			if d := p.Decide(f, Ctx{PackFullPct: 100}); d.Action != Ignore {
				t.Errorf("%s %v: %s", place, f.Type, d.Action)
			}
		}
	}
}

func TestMarksWinOverEveryRow(t *testing.T) {
	p := pol(t)
	for _, f := range []Facts{{Type: "club", SellValue: 1}, {Type: "rune"}, {Type: "swor", SellValue: 99, Quality: "normal"}} {
		for _, m := range []string{"keep", "sell"} {
			f.Mark = m
			if d := p.Decide(pack(f), Ctx{PackFullPct: 100}); d.Action != Ignore {
				t.Errorf("%s %s: %s", m, f.Type, d.Action)
			}
		}
	}
}

func TestStartKitAndQuestNeverSoldOrDropped(t *testing.T) {
	p := pol(t)
	for _, f := range []Facts{{Type: "club", SellValue: 1, StartItem: true}, {Type: "club", SellValue: 1, Quest: true}} {
		if d := p.Decide(pack(f), Ctx{PackFullPct: 100}); d.Action != Keep {
			t.Errorf("%+v: %s", f, d.Action)
		}
	}
}

// Two classes: the Amazon override keeps javelins, the Sorceress sells them. The code names no class.
func TestPerClassOverride(t *testing.T) {
	p := pol(t)
	jav := pack(Facts{Type: "jave", Quality: "normal", SellValue: 40})
	if d := p.Decide(jav, Ctx{Class: "Amazon"}); d.Action != Keep {
		t.Errorf("Amazon: %s", d.Action)
	}
	if d := p.Decide(jav, Ctx{Class: "Sorceress"}); d.Action != Sell {
		t.Errorf("Sorceress: %s", d.Action)
	}
}

func TestFullStashKeepsInPack(t *testing.T) {
	p := pol(t)
	d := p.Decide(pack(Facts{Type: "rune"}), Ctx{StashFullPct: 95})
	if d.Action != Keep {
		t.Fatalf("%s", d.Action)
	}
}

func TestPlanCapAndOrder(t *testing.T) {
	p := pol(t)
	var items []Item
	for i := 1; i <= 10; i++ {
		items = append(items, Item{uint32(i), pack(Facts{Type: "rune", SellValue: i})})
	}
	items = append(items, Item{99, pack(Facts{Type: "club", Quality: "normal", SellValue: 3})})
	got := p.Plan(items, Ctx{PackFullPct: 95}, 4)
	if len(got) != 4 || got[0].ID != 10 || got[3].ID != 7 {
		t.Fatalf("%+v", got)
	}
	if got := p.Plan(items, Ctx{PackFullPct: 95}, 0); len(got) != 0 {
		t.Fatalf("max 0 must plan nothing: %+v", got)
	}
	all := p.Plan(items, Ctx{PackFullPct: 95}, 50)
	if all[len(all)-1].ID != 99 || all[len(all)-1].Action != Drop {
		t.Fatalf("drops come last: %+v", all)
	}
}

func TestParseRejects(t *testing.T) {
	for _, s := range []string{`{}`, `{"rows":[{"name":"a","action":"burn","reason":"x"}]}`, `{"rows":[{"name":"","action":"keep","reason":"x"}]}`} {
		if _, err := Parse([]byte(s)); err == nil {
			t.Errorf("%s accepted", s)
		}
	}
}
