package town

import (
	"errors"
	"os"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

func realTables(t *testing.T) *data.DB {
	t.Helper()
	if _, err := os.Stat("/tmp/d2mpq/d2data.mpq"); err != nil {
		t.Skip("no real install at /tmp/d2mpq (run on the worker)")
	}
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMercCandidatesNormalAct2(t *testing.T) {
	db := realTables(t)
	for aura, id := range map[string]int{"Blessed Aim": 8, "Defiance": 7, "Prayer": 6} {
		c := MercCandidates(db, MercWant{Act: 2, Diff: data.Normal, Aura: aura})
		if len(c) != 1 || c[0].Row.ID != id || c[0].Row.Seller != 198 || c[0].Row.Name != "Desert Mercenary" {
			t.Fatalf("%s: %+v", aura, c)
		}
		if MercTypeLabel(aura) == "" {
			t.Fatalf("no screen label for %s", aura)
		}
	}
	if all := MercCandidates(db, MercWant{Act: 2, Diff: data.Normal}); len(all) != 3 {
		t.Fatalf("want 3 types, got %d", len(all))
	}
	if c := MercCandidates(db, MercWant{Act: 2, Diff: data.Normal, Aura: "Might"}); len(c) != 0 {
		t.Fatalf("Might is not a Normal Act 2 aura: %+v", c)
	}
}

func TestHireNeedsARowAndRefusesUnverifiedNPCs(t *testing.T) {
	if a := ActByNo(2); a.Merc[0] != 198 {
		t.Fatal("Greiz class")
	}
	if _, ok := MercMenu[198]; !ok {
		t.Fatal("Greiz dialog missing")
	}
	if _, ok := MercMenu[252]; ok {
		t.Fatal("Act 3 hireling dialog is unverified and must not be listed")
	}
	if !errors.Is(ErrMercRowUnknown, ErrMercRowUnknown) {
		t.Fatal("unreachable")
	}
}

func TestHirePolicy(t *testing.T) {
	alive := d2.Merc{Present: true, Alive: true}
	for _, c := range []struct {
		name  string
		diff  data.Difficulty
		merc  d2.Merc
		gold  int
		aura  string
		hires bool
	}{
		{"normal none", data.Normal, d2.Merc{}, 1000, "Defiance", true},
		{"normal poor", data.Normal, d2.Merc{}, 100, "Defiance", false},
		{"normal has merc", data.Normal, alive, 5000, "", false},
		{"normal dead merc still hires", data.Normal, d2.Merc{Present: true}, 5000, "Defiance", true},
		{"nightmare might", data.Nightmare, d2.Merc{}, 9000, "Might", true},
		{"nightmare poor", data.Nightmare, d2.Merc{}, 5000, "Might", false},
		{"hell never might", data.Hell, d2.Merc{}, 20000, "Defiance", true},
	} {
		w, ok := HirePolicy(c.diff, c.merc, c.gold)
		if ok != c.hires || (ok && (w.Aura != c.aura || w.Act != 2 || w.Row != -1)) {
			t.Errorf("%s: %+v %v", c.name, w, ok)
		}
	}
}

func TestReviveReady(t *testing.T) {
	dead := d2.Merc{Present: true}
	for _, c := range []struct {
		name  string
		act   int
		m     d2.Merc
		known bool
		gold  int
		lvl   int
		want  error
	}{
		{"ok", 2, dead, true, 7000, 30, nil},
		{"vanished but known", 4, d2.Merc{}, true, 7000, 30, nil},
		{"alive", 2, d2.Merc{Present: true, Alive: true}, true, 9999, 30, ErrMercAlive},
		{"never hired", 2, d2.Merc{}, false, 9999, 30, ErrNoMerc},
		{"poor", 2, dead, true, 6000, 30, ErrMercGold},
		{"bad act", 9, dead, true, 9999, 30, ErrNoMercNPC},
	} {
		if err := ReviveReady(c.act, c.m, c.known, c.gold, c.lvl); !errors.Is(err, c.want) {
			t.Errorf("%s: %v want %v", c.name, err, c.want)
		}
	}
}

func TestHasMercByClass(t *testing.T) {
	s := &d2.Snapshot{Player: &d2.Player{X: 100, Y: 100}, Monsters: []d2.Unit{{Class: 338, Mode: d2.MonModeStand, X: 400, Y: 400}}}
	if !HasMerc(s) {
		t.Fatal("a living merc far away is still alive")
	}
	s.Monsters[0].Mode = d2.MonModeDead
	if HasMerc(s) {
		t.Fatal("dead merc")
	}
}
