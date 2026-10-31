package quest

import (
	"os"
	"strings"
	"testing"

	"d2lod-bot/internal/data"
)

// Checks every V-tagged id of act2.go against the real tables. Skips where the install is absent (the worker has it).
func TestAct2ConstantsAgainstRealTables(t *testing.T) {
	const install = "/tmp/d2mpq"
	if _, err := os.Stat(install + "/d2data.mpq"); err != nil {
		t.Skip("no real install at " + install)
	}
	db, err := data.Open(install, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]string{
		AreaLutGholein: "Lut Gholein", AreaSewers2: "Sewers Level 2", AreaSewers3: "Sewers Level 3", AreaHallsDead2: "Halls of the Dead Level 2",
		AreaHallsDead3: "Halls of the Dead Level 3", AreaClawViper2: "Claw Viper Temple Level 2", AreaMaggotLair3: "Maggot Lair Level 3",
		AreaCanyon: "Canyon of the Magi", AreaTalTomb1: "Tal Rasha's Tomb", AreaTalTomb7: "Tal Rasha's Tomb",
		AreaDurielsLair: "Duriel's Lair", AreaArcane: "Arcane Sanctuary", AreaPalaceCellar3: "Palace Cellar Level 3", AreaKurastDocks: "Kurast Docktown",
	} {
		if l := db.AreaByID(id); l == nil || l.Name != want {
			t.Errorf("area %d = %+v, want %q", id, l, want)
		}
	}
	for id, key := range map[int]string{Atma: "atma", Drognan: "drognan", Fara: "fara", Greiz: "greiz", Elzix: "elzix", Geglash: "geglash",
		Jerhyn: "jerhyn", Lysander: "lysander", Meshif: "meshif1", MeshifA3: "meshif2", WarrivA2: "warriv2", TyraelTomb: "tyrael1"} {
		if m := db.MonsterByClassID(id); m == nil || m.Key != key {
			t.Errorf("npc %d = %+v, want %s", id, m, key)
		}
	}
	for code, sz := range map[string]Size{ItemCube: {2, 2}, ItemShaft: {1, 3}, ItemViperAmulet: {1, 1}, ItemHoradricStaff: {1, 4}, ItemBookOfSkill: {2, 2}, ItemHoradricScroll: {2, 2}} {
		it := db.ItemByCode(code)
		if it == nil || !it.Quest || it.Width != sz.W || it.Height != sz.H {
			t.Errorf("item %s = %+v, want quest %v", code, it, sz)
		}
	}
	for id, tok := range map[int]string{ObjOrifice: "HA", ObjTaintedSun: "za", ObjDurielDoor: "SJ", ObjTomeJournal: "TT"} {
		found := false
		for _, o := range db.Objects {
			if o.ID == id && o.Token == tok {
				found = true
			}
		}
		if !found {
			t.Errorf("object %d token %s missing", id, tok)
		}
	}
	recipe := false
	for _, c := range db.Cube {
		if c.Output == ItemHoradricStaff && len(c.Inputs) == 2 && strings.Join(c.Inputs, "+") == ItemShaft+"+"+ItemViperAmulet {
			recipe = true
		}
	}
	if !recipe {
		t.Error("cubemain has no msf+vip -> hst recipe")
	}
	for _, v := range db.Inventories {
		if v.Class == "Transmogrify Box Page 1" && (v.Cols != CubeCols || v.Rows != CubeRows || v.CellW != CubeCell) {
			t.Errorf("cube grid %dx%d cell %d", v.Cols, v.Rows, v.CellW)
		}
	}
	hr := db.HirelingsFor(2, data.Normal)
	if len(hr) == 0 || hr[0].Seller != Greiz {
		t.Errorf("act 2 normal hirelings %v", hr)
	}
}
