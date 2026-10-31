package quest

import (
	"os"
	"testing"

	"d2lod-bot/internal/data"
)

// Checks the V-tagged ids of act4.go and act5.go against the real tables. Skips where the install is absent (the worker has it).
func TestAct45ConstantsAgainstRealTables(t *testing.T) {
	const install = "/tmp/d2mpq"
	if _, err := os.Stat(install + "/d2data.mpq"); err != nil {
		t.Skip("no real install at " + install)
	}
	db, err := data.Open(install, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Level names are the data names (docs/act4.md, docs/act5.md section 3), not the in-game ones.
	for id, want := range map[int]string{
		AreaFortress: "The Pandemonium Fortress", AreaOuterSteppes: "Outer Steppes", AreaPlainsOfDespair: "Plains of Despair",
		AreaCityOfTheDamned: "City of the Damned", AreaRiverOfFlame: "River of Flame", AreaChaosSanctuary: "Chaos Sanctum",
		AreaHarrogath: "Harrogath", AreaBloodyFoothills: "Bloody Foothills", AreaFrigidHighlands: "Rigid Highlands",
		AreaArreatPlateau: "Arreat Plateau", AreaWorldstoneKeep1: "The Worldstone Keep Level 1", AreaWorldstoneKeep2: "The Worldstone Keep Level 2",
		AreaWorldstoneKeep3: "The Worldstone Keep Level 3", AreaThrone: "Throne of Destruction", AreaWorldstoneChamber: "The Worldstone Chamber",
		AreaNihlathakTemple: "Nihlathaks Temple", AreaHallsOfAnguish: "Halls of Anguish",
	} {
		if l := db.AreaByID(id); l == nil || l.Name != want {
			t.Errorf("area %d = %+v, want %q", id, l, want)
		}
	}
	// Existence only (data names differ from the in-game ones and were not all dumped).
	for _, id := range []int{AreaCrystalPassage, AreaFrozenRiver, AreaGlacialTrail, AreaFrozenTundra, AreaAncientsWay, AreaIcyCellar,
		AreaArreatSummit, AreaHallsOfPain, AreaHallsOfVaught} {
		if db.AreaByID(id) == nil {
			t.Errorf("area %d missing", id)
		}
	}
	for id, key := range map[int]string{
		Tyrael: "tyrael2", Jamella: "jamella", Halbu: "halbu", ClassIzual: "izual", IzualGhost: "izualghost", ClassHephas: "hephasto",
		ClassDiablo: "diablo", Larzuk: "larzuk", AnyaNPC: "drehya", Malah: "malah", NihlathakNPC: "nihlathak", QualKehk: "qual-kehk",
		ClassShenk: "overseer1", ClassNihlathak: "nihlathakboss", ClassBaal: "baalcrab", ClassBaalClone: "baalclone",
	} {
		if m := db.MonsterByClassID(id); m == nil || m.Key != key {
			t.Errorf("monster %d = %+v, want %s", id, m, key)
		}
	}
	for _, id := range []int{CainA4, CainA4b, CainA5, TyraelA5, ClassTalic, ClassMadawc, ClassKorlic, ClassBaalThrone, ClassLister, ClassCouncilA5, ClassPrisonDoor, ClassCaptive} {
		if db.MonsterByClassID(id) == nil {
			t.Errorf("monster %d missing", id)
		}
	}
	for code, sz := range map[string]Size{ItemHellforgeHammer: {2, 3}, ItemSoulstone: {1, 1}} {
		it := db.ItemByCode(code)
		if it == nil || !it.Quest || it.Width != sz.W || it.Height != sz.H {
			t.Errorf("item %s = %+v, want quest %v", code, it, sz)
		}
	}
	for _, code := range []string{ItemMalahPotion, ItemResistScroll} {
		if db.ItemByCode(code) == nil {
			t.Errorf("item %s missing", code)
		}
	}
	for _, id := range []int{ObjHellforge, ObjSealInfector1, ObjSealInfector2, ObjSealSeis, ObjSealVizier1, ObjSealVizier2, ObjSealGlow, ObjDiabloStar,
		ObjWaypointA4, ObjHarrogath, ObjWaypointA5, ObjAnyaPlatform, ObjNihlathakPlat, ObjCage, ObjAncientsAltar, ObjStatueTalic, ObjStatueMadawc,
		ObjStatueKorlic, ObjToWSK1, ObjBaalPortal, ObjThroneEntrance, ObjFrozenAnya} {
		found := false
		for _, o := range db.Objects {
			found = found || o.ID == id
		}
		if !found {
			t.Errorf("object %d missing", id)
		}
	}
}
