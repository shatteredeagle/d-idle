package quest

import (
	"os"
	"testing"

	"d2lod-bot/internal/data"
)

// Checks the V-tagged ids of act3.go against the real tables. Skips where the install is absent (the worker has it).
// The expected names are the ones ACT3PLAN read on 2026-10-29 (docs/act3.md); a mismatch means the plan or the const is wrong.
func TestAct3ConstantsAgainstRealTables(t *testing.T) {
	const install = "/tmp/d2mpq"
	if _, err := os.Stat(install + "/d2data.mpq"); err != nil {
		t.Skip("no real install at " + install)
	}
	db, err := data.Open(install, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]string{
		AreaKurastDocks: "Kurast Docktown", AreaSpiderForest: "Spider Forest", AreaGreatMarsh: "Great Marsh", AreaFlayerJungle: "Flayer Jungle",
		AreaLowerKurast: "Lower Kurast", AreaKurastBazaar: "Kurast Bazaar", AreaUpperKurast: "Upper Kurast", AreaCauseway: "Kurast Causeway",
		AreaTravincal: "Travincal", AreaDurance1: "Durance of Hate Level 1", AreaDurance2: "Durance of Hate Level 2", AreaDurance3: "Durance of Hate Level 3",
	} {
		if l := db.AreaByID(id); l == nil || l.Name != want {
			t.Errorf("area %d = %+v, want %q", id, l, want)
		}
	}
	// Level names of the relic areas are not asserted by name (data names differ from the in-game ones); only existence.
	for _, id := range []int{AreaSpiderCavern, AreaFlayerDung3, AreaSewers3Kurast, AreaRuinedTemple} {
		if db.AreaByID(id) == nil {
			t.Errorf("area %d missing", id)
		}
	}
	for id, key := range map[int]string{Hratli: "hratli", Alkor: "alkor", Ormus: "ormus", Asheara: "asheara", MeshifA3: "meshif2", 242: "mephisto"} {
		if m := db.MonsterByClassID(id); m == nil || m.Key != key {
			t.Errorf("npc %d = %+v, want %s", id, m, key)
		}
	}
	for code, sz := range map[string]Size{ItemJadeFigurine: {1, 2}, ItemGoldenBird: {1, 2}, ItemLamTome: {2, 2}, ItemKhalimEye: {1, 1},
		ItemKhalimHeart: {1, 1}, ItemKhalimBrain: {1, 1}, ItemKhalimFlail: {2, 3}, ItemKhalimWill: {2, 3}, ItemSoulstone: {1, 1}} {
		it := db.ItemByCode(code)
		if it == nil || !it.Quest || it.Width != sz.W || it.Height != sz.H {
			t.Errorf("item %s = %+v, want quest %v", code, it, sz)
		}
	}
	for _, id := range []int{ObjLamTomeAltar, ObjGidbinnAltar, ObjGidbinn, ObjCompellingOrb, ObjRelicChestA, ObjRelicChestB, ObjRelicChestC, ObjRedPortalA3} {
		found := false
		for _, o := range db.Objects {
			found = found || o.ID == id
		}
		if !found {
			t.Errorf("object %d missing", id)
		}
	}
	recipe := false
	for _, c := range db.Cube {
		if c.Output == ItemKhalimWill && len(c.Inputs) == 4 {
			recipe = true
		}
	}
	if !recipe {
		t.Error("cubemain has no 4-input recipe for qf2")
	}
}
