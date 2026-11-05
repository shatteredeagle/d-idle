package task_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// towerLifeWorld is floorWorld with the tower life defaults; caster false is the melee build (no mana in the kit).
func towerLifeWorld(lvl int, merc string, life int, caster bool, mut func(*task.World)) task.World {
	rt := env.DefaultRouteTuning()
	return floorWorld(lvl, merc, func(w *task.World) {
		w.MaxLife, w.TowerMinLife, w.TowerMinLevelNoLife = life, rt.TowerMinLife, rt.TowerMinLevelNoLife
		w.Caster = caster
		if caster {
			w.TowerKitMana = rt.TowerKitMana
		}
		if mut != nil {
			mut(w)
		}
	})
}

// TestTowerFloorLifeAndKit: SME rule (U numbers): no merc needs clvl 15, max life 150 and the kit; a living merc clvl 14;
// an unreadable max life falls back to 16 and says so. Sorceress (mana column) and Barbarian (none).
func TestTowerFloorLifeAndKit(t *testing.T) {
	cases := []struct {
		name   string
		lvl    int
		merc   string
		life   int
		caster bool
		mut    func(*task.World)
		want   string // "" = open, else a substring of the verdict
	}{
		{"floor met", 15, "none", 150, true, nil, ""},
		{"floor met melee", 15, "none", 150, false, nil, ""},
		{"clvl 14 with merc", 14, "alive", 120, true, nil, ""},
		{"clvl 14 with merc melee", 14, "alive", 120, false, nil, ""},
		{"clvl 13 with merc", 13, "alive", 300, true, nil, "clvl 13 < 14 (with merc)"},
		{"clvl 14 no merc", 14, "none", 300, true, nil, "no merc, clvl 14 < 15"},
		{"life 149", 15, "none", 149, true, nil, "life 149 < 150"},
		{"life 149 melee", 15, "none", 149, false, nil, "life 149 < 150"},
		{"life 149 dead merc", 16, "dead", 149, true, nil, "life 149 < 150"},
		{"unreadable life clvl 15", 15, "none", 0, true, nil, "clvl 15 < 16 (max life unreadable, fallback floor)"},
		{"unreadable life clvl 16", 16, "none", 0, false, nil, ""},
		{"no mana potions", 15, "none", 150, true, func(w *task.World) { w.ManaPotions = 2 }, "2 mana potions < 3"},
		{"no mana potions melee unaffected", 15, "none", 150, false, func(w *task.World) { w.ManaPotions = 0 }, ""},
		{"thin healing", 15, "none", 150, false, func(w *task.World) { w.Potions = 3 }, "3 healing potions < 4"},
		{"no portal", 15, "none", 150, true, func(w *task.World) { w.HasTP = false }, "no town portal"},
		{"no portal charges", 15, "none", 150, false, func(w *task.World) { w.TPKnown, w.TPCharges = true, 1 }, "portal charges"},
		{"life rule off", 15, "none", 100, true, func(w *task.World) { w.TowerMinLife = 0 }, ""}, // knob 0: no life rule, TowerMinLevel 15 alone
	}
	for _, c := range cases {
		w := towerLifeWorld(c.lvl, c.merc, c.life, c.caster, c.mut)
		got := task.TowerWhy(w)
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: verdict %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTowerLifeLogLine(t *testing.T) {
	w := towerLifeWorld(15, "none", 149, true, nil)
	if got := task.TowerLine(w); got != "route guard: blocked tower (no merc, life 149 < 150)" {
		t.Errorf("line %q", got)
	}
	if got := task.TowerLifeNote(towerLifeWorld(15, "none", 0, true, nil)); !strings.Contains(got, "unreadable") {
		t.Errorf("fallback note %q", got)
	}
	if got := task.TowerLifeNote(w); got != "" {
		t.Errorf("note with a readable life %q", got)
	}
}
