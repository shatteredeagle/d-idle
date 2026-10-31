package boss

import (
	"context"
	"errors"
	"testing"
)

func run2(t *testing.T, name string, f *fake, o Opts) (Result, error) {
	t.Helper()
	s, ok := All()[name]
	if !ok {
		t.Fatal("no script", name)
	}
	return Run(context.Background(), s, f, f, o)
}

func TestAllHoldsBothActs(t *testing.T) {
	if n := len(All()); n != 7 {
		t.Fatalf("All() has %d scripts, want 4 + 3", n)
	}
	for _, n := range []string{NameRadament, NameSummoner, NameDuriel, NameAndariel} {
		if _, ok := All()[n]; !ok {
			t.Errorf("missing %s", n)
		}
	}
	if _, ok := Lookup(NameDuriel); ok {
		t.Error("Lookup is Act 1 only; Act 2 goes through All() (request to INTEGRATE2 in docs/act2.md)")
	}
}

func TestRadamentRouteAndQuestLoot(t *testing.T) {
	f := newFake()
	f.level = 20
	f.addBoss(Mon{Class: ClassRadament, Name: NameRadament, UniqueNo: uniqueNoRadament, Pos: Pt{300, 300}}, AreaSewers3, 90)
	f.lootN = 1
	res, err := run2(t, NameRadament, f, Opts{})
	if err != nil || !res.Killed || res.Loot != 1 {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 48" || f.log[1] != "exit 49" {
		t.Fatalf("route %v", f.log[:2])
	}
	if len(f.loots) != 1 || len(f.loots[0].Quest) != 1 || f.loots[0].Quest[0] != "Book of Skill" {
		t.Fatalf("loot spec %+v", f.loots)
	}
}

func TestRadamentMatchesBySuperUniqueNumber(t *testing.T) {
	f := newFake()
	f.level = 20
	f.addBoss(Mon{Class: 0, Name: "", UniqueNo: uniqueNoRadament, Pos: Pt{300, 300}}, AreaSewers3, 60)
	if res, err := run2(t, NameRadament, f, Opts{}); err != nil || !res.Killed {
		t.Fatalf("unique number match failed: %+v %v", res, err)
	}
}

func TestSummonerFindsJournalAnchor(t *testing.T) {
	f := newFake()
	f.level = 26
	f.presets = []Preset{{Kind: PresetObject, ID: presetJournal, Pos: Pt{500, 500}}}
	b := f.addBoss(Mon{Class: ClassSummoner, Name: NameSummoner, UniqueNo: -1, Pos: Pt{505, 505}}, AreaArcane, 60)
	b.vis = 20 // only visible when the player stands near the journal
	f.pos = Pt{100, 100}
	res, err := run2(t, NameSummoner, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 74" {
		t.Fatalf("first %v", f.log)
	}
	if f.did("goto 497,497") != 1 { // journal (500,500) + (-3,-3)
		t.Fatalf("anchor walk missing: %v", f.log)
	}
}

func TestSummonerFloor(t *testing.T) {
	f := newFake()
	f.level = 20
	if _, err := run2(t, NameSummoner, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
}

func TestDurielEntryHookAndColdAura(t *testing.T) {
	f := newFake()
	f.level, f.area = 25, AreaCanyon
	b := f.addBoss(Mon{Class: ClassDuriel, Name: NameDuriel, UniqueNo: -1, Pos: Pt{208, 208}}, AreaDurielsLair, 150)
	b.dmg = 25
	f.pos = Pt{205, 205}
	f.potions = 0 // nothing to drink: only the step-away rule keeps the character alive longer
	entered := 0
	enter := func(context.Context) error { entered++; f.arrive(AreaDurielsLair); return nil }
	s := duriel()
	s.Kite = nil // the hazard rule alone is under test
	res, err := RunWithEntry(context.Background(), s, f, f, Opts{}, enter)
	if entered != 1 {
		t.Fatalf("entered %d times", entered)
	}
	if !has(res, "hazard:cold-aura") {
		t.Fatalf("no cold aura step: phases %v err %v log %v", res.Phases, err, f.log)
	}
	_ = err // the fake's Duriel may out-damage the character; the aura reaction is what is tested
}

func TestDurielNotEnteredWhenAlreadyInside(t *testing.T) {
	f := newFake()
	f.level, f.area = 25, AreaDurielsLair
	f.addBoss(Mon{Class: ClassDuriel, Name: NameDuriel, UniqueNo: -1, Pos: Pt{210, 210}}, AreaDurielsLair, 60)
	called := false
	res, err := RunWithEntry(context.Background(), duriel(), f, f, Opts{}, func(context.Context) error { called = true; return nil })
	if err != nil || !res.Killed || called {
		t.Fatalf("res %+v err %v called %v", res, err, called)
	}
}

func TestDurielEntryFailure(t *testing.T) {
	f := newFake()
	f.level, f.area = 25, AreaCanyon
	boom := errors.New("no tomb")
	_, err := RunWithEntry(context.Background(), duriel(), f, f, Opts{}, func(context.Context) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("want wrapped entry error, got %v", err)
	}
	if f.did("travel") != 0 {
		t.Fatal("must not travel after a failed entry")
	}
}

func TestDurielFloorAndTombList(t *testing.T) {
	f := newFake()
	f.level, f.area = 20, AreaDurielsLair
	if _, err := run2(t, NameDuriel, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
	if tombs := TombAreas(); len(tombs) != 7 || tombs[0] != 66 || tombs[6] != 72 {
		t.Fatalf("tombs %v", tombs)
	}
}

// teleFake adds Teleporter to the fake: the hop lands the player on the target.
type teleFake struct{ *fake }

func (t teleFake) TeleportTo(ctx context.Context, p Pt) error {
	t.logf("teleport %d,%d", p.X, p.Y)
	t.pos = p
	return nil
}

func TestSummonerTeleportsToAnchorWhenAvailable(t *testing.T) {
	f := newFake()
	f.level = 26
	f.presets = []Preset{{Kind: PresetObject, ID: presetJournal, Pos: Pt{500, 500}}}
	b := f.addBoss(Mon{Class: ClassSummoner, Name: NameSummoner, UniqueNo: -1, Pos: Pt{505, 505}}, AreaArcane, 60)
	b.vis = 20
	f.pos = Pt{100, 100}
	res, err := Run(context.Background(), All()[NameSummoner], f, teleFake{f}, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.did("teleport 497,497") != 1 {
		t.Fatalf("no teleport: %v", f.log)
	}
}
