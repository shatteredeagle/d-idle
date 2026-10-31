package d2

import (
	"testing"

	"d2lod-bot/internal/mem"
)

// Fixtures recorded from the live game by cmd/d2snap (record=...): level 1 Sorceress "Testy" in the
// Rogue Encampment. They pin the decoders to real 1.14d memory.

func loadFixture(t *testing.T, name string) *Snapshot {
	t.Helper()
	src, err := mem.LoadSnapshot("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.VerifyBuild(src); err != nil {
		t.Logf("build header not in fixture (expected, header page not recorded): %v", err)
	}
	s, err := New(src).Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestLiveFixtureTown(t *testing.T) {
	s := loadFixture(t, "town.snap")
	if s.State != StateInGame || s.Player == nil {
		t.Fatalf("state %v", s.State)
	}
	p := s.Player
	if p.Name != "Testy" || p.Class != Sorceress || p.Area != 1 || !p.InTown() || p.Level != 1 {
		t.Fatalf("%+v", p)
	}
	if p.MaxLife != 40 || p.MaxMana != 35 || p.MaxStamina != 74 || p.Dexterity != 25 || p.Energy != 35 {
		t.Fatalf("%+v", p)
	}
	if p.RightSkill != SkillFireBolt || p.LeftSkill != SkillAttack {
		t.Fatalf("skills L%d R%d", p.LeftSkill, p.RightSkill)
	}
	var fb *Skill
	for i := range p.Skills {
		if p.Skills[i].ID == SkillFireBolt {
			fb = &p.Skills[i]
		}
	}
	if fb == nil || fb.Level != 0 || fb.Effective() != 1 {
		t.Fatalf("fire bolt %+v", fb)
	}
	if p.Stats.Get(68) != 110 {
		t.Fatalf("merged stats must include the staff bonus, got %d", p.Stats.Get(68))
	}
	if s.Screen != (Point{800, 600}) {
		t.Fatalf("screen %v", s.Screen)
	}
	if len(s.Monsters) < 4 || len(s.Objects) < 5 {
		t.Fatalf("units %d/%d", len(s.Monsters), len(s.Objects))
	}
	for _, m := range s.Monsters {
		if m.Monster == nil || m.Monster.LifePct != 100 {
			t.Fatalf("npc %+v", m)
		}
	}
	inv := s.Inventory
	if len(inv.Items) != 7 || len(inv.Container(ContBelt)) != 4 || len(inv.Container(ContInventory)) != 2 {
		t.Fatalf("inventory %d belt %d pack %d", len(inv.Items), len(inv.Container(ContBelt)), len(inv.Container(ContInventory)))
	}
	if st := inv.Equipped(BodyRHand); st == nil || st.Class != 63 {
		t.Fatalf("staff %+v", st)
	}
	if s.UI.AnyPanelOpen() {
		t.Fatalf("panels open in town fixture: %v", s.UI)
	}
}

func TestLiveFixtureInventoryOpen(t *testing.T) {
	s := loadFixture(t, "town_inv.snap")
	if !s.UI.Open(UIInventory) || !s.UI.AnyPanelOpen() {
		t.Fatalf("inventory panel flag not seen: %v", s.UI)
	}
	if s.Player == nil || s.Player.Name != "Testy" {
		t.Fatal("player")
	}
}
