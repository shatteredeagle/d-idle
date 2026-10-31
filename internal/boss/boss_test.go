package boss

import (
	"context"
	"d2lod-bot/internal/env"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func run(t *testing.T, name string, f *fake, o Opts) (Result, error) {
	t.Helper()
	s, ok := Lookup(name)
	if !ok {
		t.Fatal("no script", name)
	}
	return Run(context.Background(), s, f, f, o)
}

func has(res Result, p string) bool {
	for _, x := range res.Phases {
		if x == p {
			return true
		}
	}
	return false
}

func TestAndarielFullRunPoisonAndKite(t *testing.T) {
	f := newFake()
	f.antidotes, f.level = 2, 24
	b := f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{22549, 9514}, UniqueNo: -1}, AreaCatacombs4, 150)
	b.approach, b.dmg = 3, 5
	f.pos = Pt{22549, 9520}
	f.arriveFn = func(f *fake, a int) { f.pos = Pt{22549, 9500} }
	calls := 0
	f.onAttack = func(f *fake, m *fmon) {
		if calls++; calls == 1 {
			f.poisoned = true
		}
	}
	f.lootN = 3
	res, err := run(t, NameAndariel, f, Opts{})
	if err != nil || !res.Killed || res.Loot != 3 {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 35" || f.log[1] != "exit 36" || f.log[2] != "exit 37" {
		t.Fatalf("route %v", f.log[:3])
	}
	if f.did("antidote") < 2 { // pre-fight (poison res 0 < 75) and one for the poisoned state
		t.Fatalf("antidotes %d log %v", f.did("antidote"), f.log)
	}
	if !has(res, "kite") || !has(res, "attack") || !has(res, "loot") {
		t.Fatalf("phases %v", res.Phases)
	}
}

func TestAndarielPoisonCloudStepsAway(t *testing.T) {
	f := newFake()
	f.area = AreaCatacombs4
	f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 200}, UniqueNo: -1}, AreaCatacombs4, 60)
	f.pos = Pt{150, 150}
	f.dangers = []Danger{{Kind: DangerPoisonCloud, Pos: Pt{152, 150}}}
	res, err := run(t, NameAndariel, f, Opts{})
	if err != nil || !has(res, "hazard:poison-cloud") {
		t.Fatalf("%v %v", res.Phases, err)
	}
	// the first goto must move away from the cloud (x below 150)
	for _, l := range f.log {
		if strings.HasPrefix(l, "goto ") {
			var x, y int
			if _, e := sscan(l, &x, &y); e == nil && x >= 150 {
				t.Fatalf("stepped towards the cloud: %s", l)
			}
			break
		}
	}
}

func sscan(l string, x, y *int) (int, error) {
	var n int
	_, err := fmt.Sscanf(l, "goto %d,%d", x, y)
	return n, err
}

func TestAndarielTooWeak(t *testing.T) {
	f := newFake()
	f.level = env.DefaultAndarielMinLevel - 1
	if _, err := run(t, NameAndariel, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("%v", err)
	}
	f.level, f.diff = 24, 1
	if _, err := run(t, NameAndariel, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("nightmare floor: %v", err)
	}
}

func TestCountessRouteSpawnPresetAndRunes(t *testing.T) {
	f := newFake()
	f.level = 20
	f.presets = []Preset{{Kind: PresetNPC, ID: presetCountessSpawn, Pos: Pt{12534, 11040}}}
	f.addBoss(Mon{Class: 45, UniqueNo: 6, Name: "The Countess", Pos: Pt{12542, 11036}}, AreaTowerCellar5, 70).vis = 30
	f.pos = Pt{12400, 11000}
	f.arriveFn = func(f *fake, a int) {
		if a == AreaTowerCellar5 {
			f.pos = Pt{12400, 11000}
		}
	}
	res, err := run(t, NameCountess, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v %v", err, f.log)
	}
	want := []string{"travel 6", "exit 20", "exit 21", "exit 22", "exit 23", "exit 24", "exit 25", "goto 12534,11040"}
	for i, w := range want {
		if f.log[i] != w {
			t.Fatalf("log[%d]=%q want %q (%v)", i, f.log[i], w, f.log)
		}
	}
	if f.did("clear") == 0 {
		t.Fatal("no post-kill clear")
	}
	l := f.loots[0]
	if l.Kinds[0] != "rune" || l.Center == (Pt{}) {
		t.Fatalf("loot %+v", l)
	}
}

func TestCountessRouteLowLifeEscapes(t *testing.T) {
	f := newFake()
	f.level = 20
	f.life = 30
	f.arriveFn = func(f *fake, a int) {}
	_, err := run(t, NameCountess, f, Opts{})
	if !errors.Is(err, ErrEscaped) || f.townCalls != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestCountessNoTPChickens(t *testing.T) {
	f := newFake()
	f.level, f.life, f.hasTP = 20, 30, false
	_, err := run(t, NameCountess, f, Opts{})
	if !errors.Is(err, ErrChicken) || f.townCalls != 0 || f.did("chicken") != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestCountessPresetMissingFallsBackToHintThenExplore(t *testing.T) {
	f := newFake()
	f.level = 20
	f.area = AreaTowerCellar5
	b := f.addBoss(Mon{Class: 45, UniqueNo: 6, Name: "The Countess", Pos: Pt{9000, 9000}}, AreaTowerCellar5, 40)
	b.vis = 20
	f.exploreFn = func(f *fake) { f.pos = Pt{9005, 9005} }
	res, err := run(t, NameCountess, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v", err)
	}
	if f.did("goto 12578,11043") != 1 || f.did("explore") != 1 {
		t.Fatalf("%v", f.log)
	}
	if len(res.Notes) == 0 || !strings.Contains(res.Notes[0], "preset") {
		t.Fatalf("notes %v", res.Notes)
	}
}

func TestSmithQuestLootAndRetreat(t *testing.T) {
	f := newFake()
	f.level = 10
	f.area = AreaBarracks
	f.potions = 2
	sm := f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{500, 500}, UniqueNo: -1}, AreaBarracks, 90)
	sm.dmg = 0
	f.pos = Pt{480, 500}
	for i := 0; i < 7; i++ {
		m := f.addBoss(Mon{Class: 50, Pos: Pt{485 + i, 505}, UniqueNo: -1}, AreaBarracks, 10)
		m.Mon.ID = uint32(100 + i)
	}
	f.life = 48
	res, err := run(t, NameSmith, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v %v", err, res.Phases)
	}
	if !has(res, "hazard:crowded") {
		t.Fatalf("phases %v", res.Phases)
	}
	if f.loots[0].Quest[0] != "Horadric Malus" {
		t.Fatalf("%+v", f.loots[0])
	}
}

func TestBloodRavenLureAndSkeletons(t *testing.T) {
	f := newFake()
	f.level = 8
	f.area = AreaBurialGrounds
	f.pos = Pt{300, 300}
	br := f.addBoss(Mon{Class: ClassBloodRaven, Name: "Blood Raven", Pos: Pt{400, 300}, UniqueNo: -1}, AreaBurialGrounds, 60)
	br.approach = 40
	for i := 0; i < 3; i++ {
		f.addBoss(Mon{Class: 60, Name: "Skeleton", Minion: true, Pos: Pt{310 + i, 300}, UniqueNo: -1}, AreaBurialGrounds, 5)
	}
	res, err := run(t, NameBloodRaven, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v", err)
	}
	if !has(res, "hazard:raised-skeletons") {
		t.Fatalf("%v", res.Phases)
	}
	if res.Phases[0] != "outer-clear" || res.Phases[1] != "locate" || has(res, "lure") {
		t.Fatalf("Blood Raven is fought where she stands, no lure: %v", res.Phases)
	}
}

func TestLowLifeNoPotionEscapesToTown(t *testing.T) {
	f := newFake()
	f.area = AreaCatacombs4
	b := f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 200}, UniqueNo: -1}, AreaCatacombs4, 100)
	b.dmg, b.approach = 40, 50
	f.pos, f.potions, f.life = Pt{190, 200}, 0, 60
	_, err := run(t, NameAndariel, f, Opts{})
	if !errors.Is(err, ErrEscaped) || f.townCalls != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestChickenBelowLine(t *testing.T) {
	f := newFake()
	f.area = AreaCatacombs4
	f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 200}, UniqueNo: -1}, AreaCatacombs4, 100)
	f.life, f.potions = 8, 3
	if _, err := run(t, NameAndariel, f, Opts{}); !errors.Is(err, ErrChicken) {
		t.Fatal(err)
	}
}

func TestBossNotFound(t *testing.T) {
	f := newFake()
	f.area = AreaBarracks
	f.level = 10
	_, err := run(t, NameSmith, f, Opts{})
	if !errors.Is(err, ErrBossNotFound) || f.did("explore") != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestAlreadyDeadOnlyLoots(t *testing.T) {
	f := newFake()
	f.area = AreaBarracks
	f.level = 10
	sm := f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{5, 5}, UniqueNo: -1}, AreaBarracks, 0)
	sm.Mode = 12
	f.lootN = 1
	res, err := run(t, NameSmith, f, Opts{})
	if err != nil || !res.Killed || f.did("attack") != 0 || f.did("loot") != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestTimeoutAndAttackFailures(t *testing.T) {
	f := newFake()
	f.area = AreaBarracks
	f.level = 10
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{5, 5}, UniqueNo: -1}, AreaBarracks, 1_000_000)
	s, _ := Lookup(NameSmith)
	s.Timeout = 20 * time.Second
	if _, err := Run(context.Background(), s, f, f, Opts{}); !errors.Is(err, ErrTimeout) {
		t.Fatal(err)
	}
	f.attackErr = errors.New("boom")
	s.Timeout = time.Hour
	if _, err := Run(context.Background(), s, f, f, Opts{}); err == nil || errors.Is(err, ErrTimeout) {
		t.Fatalf("want attack failure, got %v", err)
	}
	f.attackErr = ErrChicken
	if _, err := Run(context.Background(), s, f, f, Opts{}); !errors.Is(err, ErrChicken) {
		t.Fatal(err)
	}
}

func TestLeftArena(t *testing.T) {
	f := newFake()
	f.level = 20
	f.exitFail = nil
	f.arriveFn = func(f *fake, a int) {
		if a == AreaTowerCellar1 {
			f.area = 1 // fell back to town
		}
	}
	if _, err := run(t, NameCountess, f, Opts{}); !errors.Is(err, ErrLeftArena) {
		t.Fatal(err)
	}
}

func TestRunnerKillBossAndUnknown(t *testing.T) {
	f := newFake()
	f.area, f.level = AreaBarracks, 10
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{5, 5}, UniqueNo: -1}, AreaBarracks, 30)
	r := &Runner{W: f, A: f}
	if err := r.KillBoss(context.Background(), NameSmith); err != nil || !r.Last.Killed {
		t.Fatal(err)
	}
	if err := r.KillBoss(context.Background(), "Nobody"); err == nil {
		t.Fatal("want error")
	}
}

func TestMatchAndPreset(t *testing.T) {
	m := Match{UniqueNo: 6, Name: "The Countess"}
	if !m.Is(Mon{UniqueNo: 6}) || !m.Is(Mon{UniqueNo: -1, Name: "the countess"}) || m.Is(Mon{UniqueNo: -1}) {
		t.Fatal("unique match")
	}
	// plain monsters of the Smith's class without a name are not the Smith
	sm := Match{Class: ClassSmith, UniqueNo: -1, Name: "The Smith"}
	if sm.Is(Mon{Class: ClassSmith, UniqueNo: -1}) || !sm.Is(Mon{Class: ClassSmith, Name: "The Smith", UniqueNo: -1}) {
		t.Fatal("class match needs the unique name")
	}
	s, _ := Lookup(NameCountess)
	if c := s.WithUniqueNo(7); c.Match.UniqueNo != 7 || s.Match.UniqueNo != -1 {
		t.Fatal("WithUniqueNo must copy")
	}
	ref := &PresetRef{Kind: 2, ID: 1, Off: Pt{1, 2}}
	if p, ok := resolvePreset(ref, []Preset{{Kind: 2, ID: 1, Pos: Pt{10, 10}}}); !ok || p != (Pt{11, 12}) {
		t.Fatal(p)
	}
	if _, ok := resolvePreset(ref, nil); ok {
		t.Fatal("no preset")
	}
}

func TestContextCancel(t *testing.T) {
	f := newFake()
	f.area, f.level = AreaBarracks, 10
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{5, 5}, UniqueNo: -1}, AreaBarracks, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s, _ := Lookup(NameSmith)
	if _, err := Run(ctx, s, f, f, Opts{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// A boss that drops out of the unit list (the player moved away, the client unloaded it) is walked back to, not given up on.
func TestBossOutOfViewIsWalkedBackTo(t *testing.T) {
	f := newFake()
	f.level = 10
	f.area = AreaBurialGrounds
	f.pos = Pt{300, 300}
	br := f.addBoss(Mon{Class: ClassBloodRaven, Name: "Blood Raven", Pos: Pt{340, 300}, UniqueNo: -1}, AreaBurialGrounds, 60)
	br.vis = 45
	// the first slice carries the player 100 subtiles off: she is out of the list until it walks back
	moved := false
	f.onAttack = func(f *fake, m *fmon) {
		if !moved {
			moved = true
			f.pos = Pt{440, 300}
		}
	}
	res, err := run(t, NameBloodRaven, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v %v", err, f.log)
	}
	if f.did("goto 340,300") == 0 {
		t.Fatalf("did not walk to her last position: %v", f.log)
	}
}

func TestCriticalLifeEscapesEvenWithPotions(t *testing.T) {
	f := newFake()
	f.area = AreaCatacombs4
	b := f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 200}, UniqueNo: -1}, AreaCatacombs4, 100)
	b.dmg, b.approach = 40, 50
	f.pos, f.potions, f.life = Pt{190, 200}, 5, 14
	_, err := run(t, NameAndariel, f, Opts{})
	if !errors.Is(err, ErrEscaped) || f.townCalls != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestPotionFloorIsBeltAware(t *testing.T) {
	for _, c := range []struct{ slots, want, got int }{
		{0, 3, 3}, {4, 3, 2}, {4, 1, 1}, {8, 3, 3}, {8, 12, 8}, {12, 4, 4},
	} {
		if g := (View{BeltSlots: c.slots}).PotionFloor(c.want); g != c.got {
			t.Errorf("belt %d want %d: floor %d, expected %d", c.slots, c.want, g, c.got)
		}
	}
}

// Blood Raven tactic (docs/sme/blood-raven-caster.md section 3): the ring around the entry is cleared before she is looked
// for, a first retreat walks back to the entry and drinks, and the second retreat of one attempt is a town portal.
func brFight(t *testing.T, o Opts) (*fake, Result, error) {
	t.Helper()
	f := newFake()
	f.level = 9
	f.area = AreaBurialGrounds
	f.pos = Pt{300, 300}
	f.addBoss(Mon{Class: ClassBloodRaven, Name: "Blood Raven", Pos: Pt{330, 300}, UniqueNo: -1}, AreaBurialGrounds, 600)
	n := 0
	f.onAttack = func(f *fake, _ *fmon) {
		if n++; n == 2 || n == 10 || n == 18 {
			f.life = 35 // below RetreatPct (40) and DrinkPct (45)
		}
	}
	res, err := run(t, NameBloodRaven, f, o)
	return f, res, err
}

func TestBloodRavenClearsOuterRingFirst(t *testing.T) {
	f, res, _ := brFight(t, Opts{})
	if f.did("clear 28") != 1 || f.log[0] != "clear 28" {
		t.Fatalf("outer clear not first: %v", f.log)
	}
	if !has(res, "outer-clear") {
		t.Fatal(res.Phases)
	}
}

func TestBloodRavenSecondRetreatEndsTheAttempt(t *testing.T) {
	f, res, err := brFight(t, Opts{})
	if !errors.Is(err, ErrEscaped) || f.townCalls != 1 {
		t.Fatalf("err %v town %d log %v", err, f.townCalls, f.log)
	}
	if !has(res, "retreat") || !has(res, "retreat-limit") || res.Killed {
		t.Fatalf("phases %v", res.Phases)
	}
	if f.did("drink") != 1 {
		t.Fatalf("the first retreat drinks, the second leaves: %v", f.log)
	}
}

func TestBloodRavenRetreatKnob(t *testing.T) {
	f, _, err := brFight(t, Opts{MaxRetreats: 3})
	if !errors.Is(err, ErrEscaped) || f.did("drink") != 2 || f.townCalls != 1 {
		t.Fatalf("err %v drinks %d town %d", err, f.did("drink"), f.townCalls)
	}
}
