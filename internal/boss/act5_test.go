package boss

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShenkMatchesByClassAndTravelsFromTown(t *testing.T) {
	f := newFake()
	f.level, f.potions, f.area = 26, 4, 109
	f.addBoss(Mon{Class: ClassShenk, UniqueNo: uniqueShenk, Name: NameShenk, Pos: Pt{300, 300}}, AreaBloodyFoothills, 60)
	res, err := Run(context.Background(), AllActs()[NameShenk], f, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 110" {
		t.Fatalf("first %v", f.log[0])
	}
}

func TestNihlathakRouteAndAbortHazard(t *testing.T) {
	f := newFake()
	f.level, f.potions = 31, 5
	f.presets = []Preset{{Kind: PresetObject, ID: presetNihlathakPlatform, Pos: Pt{400, 400}}}
	f.addBoss(Mon{Class: ClassNihlathak, UniqueNo: uniqueNihlathak, Name: NameNihlathak, Pos: Pt{400, 400}}, AreaHallsOfVaught, 60).vis = 40
	if _, err := Run(context.Background(), nihlathak(), f, f, Opts{}); err != nil {
		t.Fatal(err)
	}
	if f.log[0] != "travel 123" || f.log[1] != "exit 124" {
		t.Fatalf("route %v", f.log[:2])
	}
	h := AbortOnClasses("vipers", []int{777}, 1)
	c := Ctx{V: View{Monsters: []Mon{{Class: 777, Hostile: true, Mode: 2}}}}
	if !h.Detect(c) || h.Do.Kind != RespAbort {
		t.Fatal("class present must abort")
	}
	if AbortOnClasses("none", nil, 1).Detect(c) {
		t.Fatal("empty class list must never fire")
	}
}

// ancientsWorld: three Ancients in the summit, an altar preset, and no loot until the last kill.
func ancientsWorld() *objFake {
	o := newObjFake()
	f := o.fake
	f.level, f.potions, f.area = 25, 8, AreaArreatSummit
	f.presets = []Preset{{Kind: PresetObject, ID: objAncientsAltar, Pos: Pt{200, 200}}}
	for i, c := range []struct {
		class, u int
		name     string
	}{{ClassTalic, uniqueTalic, NameTalic}, {ClassMadawc, uniqueMadawc, NameMadawc}, {ClassKorlic, uniqueKorlic, NameKorlic}} {
		f.addBoss(Mon{Class: c.class, UniqueNo: c.u, Name: c.name, Pos: Pt{300 + i*5, 300}}, AreaArreatSummit, 60)
	}
	for _, m := range f.mons {
		m.area = -AreaArreatSummit // not in the list until the altar is touched
	}
	o.onUse = func(o *objFake, class int) {
		if class == objAncientsAltar {
			for _, m := range o.mons {
				m.area = AreaArreatSummit
			}
		}
	}
	f.lootN = 2
	return o
}

func TestRunAncientsAltarThenThreeKillsNoTownPortal(t *testing.T) {
	o := ancientsWorld()
	res, err := RunAncients(context.Background(), o, o, o, nil, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, o.log)
	}
	if o.uses[0] != objAncientsAltar || len(o.uses) != 1 {
		t.Fatalf("uses %v", o.uses)
	}
	if o.did("town") != 0 {
		t.Fatal("the Ancients leg must never open a town portal")
	}
	if o.did("loot") != 1 || len(o.loots) != 1 {
		t.Fatalf("loot calls %d (only the last kill may loot)", o.did("loot"))
	}
	for _, m := range o.mons {
		if m.Alive() {
			t.Errorf("class %d alive", m.Class)
		}
	}
}

func TestRunAncientsSkipsAltarWhenAlreadyFighting(t *testing.T) {
	o := ancientsWorld()
	for _, m := range o.mons {
		m.area = AreaArreatSummit
	}
	if _, err := RunAncients(context.Background(), o, o, o, []int{ClassMadawc, ClassTalic, ClassKorlic}, Opts{}); err != nil {
		t.Fatal(err)
	}
	if len(o.uses) != 0 {
		t.Fatalf("altar touched again: %v", o.uses)
	}
}

func TestAncientsLowLifeChickensInsteadOfPortal(t *testing.T) {
	o := ancientsWorld()
	for _, m := range o.mons {
		m.area = AreaArreatSummit
		m.dmg, m.approach = 40, 50
		m.Pos = Pt{103, 100}
	}
	o.potions = 0
	_, err := RunAncients(context.Background(), o, o, o, nil, Opts{})
	if !errors.Is(err, ErrChicken) && !errors.Is(err, ErrDead) {
		t.Fatalf("want chicken or death, got %v", err)
	}
	if o.did("town") != 0 {
		t.Fatalf("town portal used: %v", o.log)
	}
}

func TestAncientsFloors(t *testing.T) {
	o := ancientsWorld()
	o.level = 19
	if _, err := RunAncients(context.Background(), o, o, o, nil, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
	o.level = 25
	if _, err := RunAncients(context.Background(), o, o, nil, nil, Opts{}); !errors.Is(err, ErrAborted) {
		t.Fatalf("want ErrAborted without ObjectOps, got %v", err)
	}
}

func TestAncientsFromTownGoesViaWaypoint(t *testing.T) {
	o := ancientsWorld()
	o.area = 109
	if _, err := RunAncients(context.Background(), o, o, o, nil, Opts{}); err != nil {
		t.Fatal(err)
	}
	if o.log[0] != "travel 118" || o.log[1] != "exit 120" {
		t.Fatalf("route %v", o.log[:2])
	}
}

func TestInThrone(t *testing.T) {
	for p, want := range map[Pt]bool{{15095, 5029}: true, {15097, 5099}: false, {15060, 5030}: false, {15118, 5002}: true, {15119, 5030}: false} {
		if InThrone(p) != want {
			t.Errorf("InThrone(%v) = %v", p, !want)
		}
	}
}

// throne builds a Throne world: the throne Baal stands until the scripted waves are dead; each wave is a set of classes
// that spawns when the previous one is dead.
func throne(waves [][]int) *objFake {
	o := newObjFake()
	f := o.fake
	f.level, f.potions, f.area = 34, 8, AreaThrone
	f.addBoss(Mon{Class: ClassBaalThrone, UniqueNo: -1, Name: "Baal", Pos: ThroneBaalSpot}, AreaThrone, 1000)
	spawn := func(i int) {
		for j, c := range waves[i] {
			f.addBoss(Mon{Class: c, UniqueNo: -1, Pos: Pt{15090 + j*3, 5040}}, AreaThrone, 40)
		}
	}
	next := 0
	spawn(0)
	next = 1
	f.onAttack = func(f *fake, m *fmon) {
		if m.hp > 30 {
			return // this attack does not kill
		}
		alive := 0
		for _, x := range f.mons {
			if x != m && x.Class != ClassBaalThrone && x.Alive() && x.area == AreaThrone {
				alive++
			}
		}
		if alive > 0 {
			return
		}
		if next < len(waves) {
			spawn(next)
			next++
			return
		}
		for _, x := range f.mons {
			if x.Class == ClassBaalThrone {
				x.area = 0 // leaves for the chamber
			}
		}
	}
	return o
}

func TestRunWavesFiveWavesThenBaalLeaves(t *testing.T) {
	o := throne([][]int{{101, 102}, {103}, {ClassCouncilA5}, {104, 105}, {ClassLister, 572}})
	res, err := RunWaves(context.Background(), o, o, WaveOpts{})
	if err != nil || !res.BaalGone {
		t.Fatalf("res %+v err %v log %v", res, err, o.log)
	}
	if res.Waves < 1 {
		t.Fatalf("waves %d", res.Waves) // the fake spawns the next wave in the same slice, so only transitions that show are counted
	}
	if !containsInt(res.Named, waveCouncil) || !containsInt(res.Named, waveLister) {
		t.Fatalf("named %v", res.Named)
	}
	if !gotoLogged(o.fake, ThroneWait) || !gotoLogged(o.fake, ThronePrePortal) {
		t.Fatalf("log %v", o.log)
	}
}

func TestRunWavesIdleSweepVisitsBusyNodesOnly(t *testing.T) {
	o := throne([][]int{{101}})
	// A straggler outside the rectangle's attack list is not possible; put a monster that is outside the throne but near a node.
	f := o.fake
	f.addBoss(Mon{Class: 110, UniqueNo: -1, Pos: Pt{15097, 5080}}, AreaThrone, 40) // y = 5080 is outside the rectangle, 26 from node 1
	f.exploreFn = nil
	f.onAttack = nil
	for _, m := range f.mons {
		if m.Class == 101 {
			m.Mode = 12 // wave one already dead
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := RunWaves(ctx, o, o, WaveOpts{Timeout: 70 * time.Second})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout with Baal still standing, got %v", err)
	}
	if o.did("clear") == 0 {
		t.Fatalf("no idle sweep: %v", o.log[:min(len(o.log), 10)])
	}
}

func TestRunWavesAbortOnSoulClass(t *testing.T) {
	o := throne([][]int{{101}})
	o.addBoss(Mon{Class: 999, UniqueNo: -1, Pos: Pt{15090, 5040}}, AreaThrone, 40)
	_, err := RunWaves(context.Background(), o, o, WaveOpts{AbortClasses: []int{999}})
	if !errors.Is(err, ErrAborted) {
		t.Fatalf("want ErrAborted, got %v", err)
	}
}

func TestRunWavesChickensAtLowLife(t *testing.T) {
	o := throne([][]int{{101}})
	o.life, o.potions = 5, 0
	_, err := RunWaves(context.Background(), o, o, WaveOpts{})
	if !errors.Is(err, ErrChicken) || o.did("chicken") != 1 {
		t.Fatalf("err %v log %v", err, o.log)
	}
}

func TestRunWavesDrinksWhenHurt(t *testing.T) {
	o := throne([][]int{{101}})
	o.life = 40
	if _, err := RunWaves(context.Background(), o, o, WaveOpts{}); err != nil {
		t.Fatal(err)
	}
	if o.did("drink") == 0 {
		t.Fatal("never drank")
	}
}

func TestRunWavesLeftArea(t *testing.T) {
	o := throne([][]int{{101}})
	o.area = AreaWorldstoneKeep3
	if _, err := RunWaves(context.Background(), o, o, WaveOpts{}); !errors.Is(err, ErrLeftArena) {
		t.Fatalf("want ErrLeftArena, got %v", err)
	}
}

func TestRunBaalWholeLeg(t *testing.T) {
	o := throne([][]int{{101}, {ClassLister}})
	o.area = AreaWorldstoneKeep2
	f := o.fake
	f.arriveFn = func(f *fake, area int) {}
	f.presets = []Preset{{Kind: PresetObject, ID: objBaalPortal, Pos: Pt{15090, 5010}}}
	o.onUse = func(o *objFake, class int) {
		if class == objBaalPortal {
			o.area = AreaWorldstoneChamb
			o.addBoss(Mon{Class: ClassBaal, UniqueNo: -1, Pos: Pt{15134, 5900}}, AreaWorldstoneChamb, 90)
			o.addBoss(Mon{Class: ClassBaalClone, UniqueNo: -1, Pos: Pt{15130, 5905}}, AreaWorldstoneChamb, 5000)
		}
	}
	o.lootN = 3
	res, err := RunBaal(context.Background(), o, o, o, WaveOpts{}, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, o.log)
	}
	if o.log[0] != "exit 130" || o.log[1] != "exit 131" {
		t.Fatalf("route %v", o.log[:2])
	}
	if len(o.uses) != 1 || o.uses[0] != objBaalPortal {
		t.Fatalf("uses %v", o.uses)
	}
	for _, m := range o.mons {
		if m.Class == ClassBaalClone && o.did("attack "+itoa(int(m.ID))) != 0 {
			t.Fatal("the clone must be ignored, not attacked")
		}
	}
	if res.Phases[0] != "waves" || res.Phases[1] != "portal" {
		t.Fatalf("phases %v", res.Phases)
	}
}

func TestRunBaalFloorAndNoPortalCapability(t *testing.T) {
	o := throne([][]int{{101}})
	o.level = 31
	if _, err := RunBaal(context.Background(), o, o, o, WaveOpts{}, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
	o.level = 34
	if _, err := RunBaal(context.Background(), o, o, nil, WaveOpts{}, Opts{}); !errors.Is(err, ErrAborted) {
		t.Fatalf("want ErrAborted, got %v", err)
	}
}

func TestAct45RunnerDispatch(t *testing.T) {
	o := chaos(true, true, true)
	r := &Act45Runner{Runner: Runner{W: o, A: o}, Ops: o}
	if err := r.KillBoss(context.Background(), NameDiablo); err != nil {
		t.Fatalf("diablo: %v", err)
	}
	if !r.Last.Killed || r.Last.Phases[0] != "seals" {
		t.Fatalf("last %+v", r.Last)
	}
	f := newFake()
	f.level, f.potions = 31, 5
	f.addBoss(Mon{Class: ClassIzual, UniqueNo: -1, Pos: Pt{500, 500}}, AreaPlainsOfDespair, 60)
	r2 := &Act45Runner{Runner: Runner{W: f, A: f}}
	if err := r2.KillBoss(context.Background(), NameIzual); err != nil {
		t.Fatalf("izual: %v", err)
	}
	if err := r2.KillBoss(context.Background(), "Nobody"); err == nil {
		t.Fatal("unknown boss must fail")
	}
}
