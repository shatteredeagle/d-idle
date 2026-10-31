package boss

import (
	"os"
	"testing"

	"d2lod-bot/internal/data"
)

// Checks the Act 1 ids against the real tables. Skips where the install is absent (the worker has it at /tmp/d2mpq).
func TestAct1IDsAgainstRealTables(t *testing.T) {
	const install = "/tmp/d2mpq"
	if _, err := os.Stat(install + "/d2data.mpq"); err != nil {
		t.Skip("no real install at " + install)
	}
	db, err := data.Open(install, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]string{AreaColdPlains: "Cold Plains", AreaBlackMarsh: "Black Marsh", AreaBurialGrounds: "Burial Grounds",
		AreaForgottenTower: "Forgotten Tower", AreaTowerCellar1: "Tower Cellar Level 1", 22: "Tower Cellar Level 2", 23: "Tower Cellar Level 3",
		24: "Tower Cellar Level 4", AreaTowerCellar5: "Tower Cellar Level 5", AreaOuterCloister: "Outer Cloister", AreaBarracks: "Barracks",
		AreaCatacombs2: "Catacombs Level 2", AreaCatacombs3: "Catacombs Level 3", AreaCatacombs4: "Catacombs Level 4"} {
		if l := db.AreaByID(id); l == nil || l.Name != want {
			t.Errorf("area %d = %+v, want %q", id, l, want)
		}
	}
	for id, key := range map[int]string{ClassAndariel: "andariel", ClassBloodRaven: "bloodraven", ClassSmith: "smith"} {
		m := db.MonsterByClassID(id)
		if m == nil {
			t.Errorf("class %d missing", id)
			continue
		}
		t.Logf("class %d = key %q name %q", id, m.Key, m.Name)
		if m.Key != key {
			t.Errorf("class %d key %q, want %q", id, m.Key, key)
		}
	}
	su := db.SuperUniqueByName("The Countess")
	if su == nil {
		t.Fatal("no super unique named The Countess")
	}
	t.Logf("countess: unique %d class %s", su.ID, su.Class)
	found := false
	for _, o := range db.Objects {
		if o.ID == presetCountessChest {
			found = true
			t.Logf("object %d token %q name %q", o.ID, o.Token, o.Name)
		}
	}
	if !found {
		t.Logf("object %d (Kolbot SuperChest) is not in objects.txt: the chest preset id stays U", presetCountessChest)
	}
}
