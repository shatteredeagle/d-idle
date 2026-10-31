package boss

import (
	"context"
	"errors"
	"testing"
)

func TestAllActsHoldsAct3(t *testing.T) {
	if n := len(AllActs()); n != 7+4+3+6 {
		t.Fatalf("AllActs has %d scripts", n)
	}
	if len(All()) != 7 {
		t.Fatal("All() must keep its Acts 1+2 contract")
	}
	for _, n := range append(CouncilNames(), NameMephisto) {
		if _, ok := AllActs()[n]; !ok {
			t.Errorf("missing %s", n)
		}
	}
}

func TestRunCouncilKillsAllThreeAndLootsFlail(t *testing.T) {
	f := newFake()
	f.level = 26
	f.lootN = 1
	f.addBoss(Mon{Class: ClassCouncil1, UniqueNo: 26, Name: "Ismail Vilehand", Pos: Pt{300, 300}}, AreaTravincal, 60)
	f.addBoss(Mon{Class: ClassCouncil2, UniqueNo: 28, Name: "Bremm Sparkfist", Pos: Pt{306, 300}}, AreaTravincal, 60)
	f.addBoss(Mon{Class: ClassCouncil3, UniqueNo: 30, Name: "Wyand Voidbringer", Pos: Pt{300, 306}}, AreaTravincal, 60)
	res, err := RunCouncil(context.Background(), f, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	for _, m := range f.mons {
		if m.Alive() {
			t.Errorf("class %d still alive", m.Class)
		}
	}
	if f.log[0] != "travel 83" {
		t.Fatalf("first %v", f.log[0])
	}
	if len(f.loots) == 0 || f.loots[0].Quest[0] != "Khalim's Flail" {
		t.Fatalf("loot spec %+v", f.loots)
	}
}

func TestRunCouncilToleratesMissingMembers(t *testing.T) {
	f := newFake()
	f.level = 26
	f.addBoss(Mon{Class: ClassCouncil1, UniqueNo: 26, Name: "Ismail Vilehand", Pos: Pt{300, 300}}, AreaTravincal, 60)
	res, err := RunCouncil(context.Background(), f, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v", res, err)
	}
	if len(res.Notes) < 2 {
		t.Fatalf("expected a note per missing member: %v", res.Notes)
	}
}

func TestRunCouncilRequiresFirstMember(t *testing.T) {
	f := newFake()
	f.level = 26
	f.area = AreaTravincal
	if _, err := RunCouncil(context.Background(), f, f, Opts{}); err == nil {
		t.Fatal("no members at all must fail")
	}
}

func TestCouncilFloor(t *testing.T) {
	f := newFake()
	f.level = 20
	if _, err := RunCouncil(context.Background(), f, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
}

func TestMephistoRoute(t *testing.T) {
	f := newFake()
	f.level, f.potions = 30, 5
	f.addBoss(Mon{Class: ClassMephisto, Name: NameMephisto, UniqueNo: -1, Pos: Pt{400, 400}}, AreaDurance3, 90)
	f.lootN = 1
	res, err := Run(context.Background(), AllActs()[NameMephisto], f, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 101" || f.log[1] != "exit 102" {
		t.Fatalf("route %v", f.log[:2])
	}
	if len(f.loots) != 1 || f.loots[0].Quest[0] != "Mephisto's Soulstone" {
		t.Fatalf("loot %+v", f.loots)
	}
}

func TestMephistoFloor(t *testing.T) {
	f := newFake()
	f.level = 27
	if _, err := Run(context.Background(), mephisto(), f, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
}

func TestSidestepOnModeDetect(t *testing.T) {
	h := SidestepOnMode(modeAttacking2, 18)
	boss := &Mon{Class: ClassMephisto, Mode: modeAttacking2, Pos: Pt{110, 100}, LifePct: 100}
	c := Ctx{V: View{Pos: Pt{100, 100}}, Boss: boss}
	if !h.Detect(c) {
		t.Fatal("should fire in Attacking2 within range")
	}
	boss.Mode = 5
	if h.Detect(c) {
		t.Fatal("other modes must not fire")
	}
	boss.Mode, boss.Pos = modeAttacking2, Pt{200, 100}
	if h.Detect(c) {
		t.Fatal("out of range must not fire")
	}
	if h.Do.Kind != RespStepAway {
		t.Fatalf("response %v", h.Do.Kind)
	}
}

func TestMephistoMoatOnlyWithTeleport(t *testing.T) {
	m := mephisto()
	n := 0
	for _, sp := range m.Stand {
		if sp.NeedsTeleport {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("moat points %d", n)
	}
}
