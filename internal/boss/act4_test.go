package boss

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// objFake adds the ObjectOps capability to the scripted world. Using a seal (or the altar/portal) runs onUse.
type objFake struct {
	*fake
	modes  map[int]int
	uses   []int
	useErr error
	onUse  func(o *objFake, class int)
}

func newObjFake() *objFake { return &objFake{fake: newFake(), modes: map[int]int{}} }

func (o *objFake) UseObject(ctx context.Context, class int, at Pt) error {
	o.logf("use %d", class)
	o.uses = append(o.uses, class)
	if o.useErr != nil {
		return o.useErr
	}
	o.modes[class] = 1
	if o.onUse != nil {
		o.onUse(o, class)
	}
	return nil
}

func (o *objFake) ObjectMode(ctx context.Context, class int, at Pt) (int, error) {
	return o.modes[class], nil
}

func TestAllActsHoldsAct45(t *testing.T) {
	m := AllActs()
	for _, n := range []string{NameIzual, NameHephasto, NameDiablo, NameShenk, NameNihlathak, NameTalic, NameMadawc, NameKorlic, NameBaal} {
		if _, ok := m[n]; !ok {
			t.Errorf("missing %s", n)
		}
	}
	for _, n := range []string{NameTalic, NameMadawc, NameKorlic} {
		if !m[n].NoTP {
			t.Errorf("%s must be NoTP", n)
		}
	}
}

func TestIzualRouteAndLoot(t *testing.T) {
	f := newFake()
	f.level, f.potions = 31, 5
	f.presets = []Preset{{Kind: PresetNPC, ID: ClassIzual, Pos: Pt{500, 500}}}
	f.addBoss(Mon{Class: ClassIzual, UniqueNo: -1, Pos: Pt{500, 500}}, AreaPlainsOfDespair, 90).vis = 40
	f.lootN = 1
	res, err := Run(context.Background(), AllActs()[NameIzual], f, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 106" || f.log[1] != "exit 105" {
		t.Fatalf("route %v", f.log[:2])
	}
}

func TestIzualFloor(t *testing.T) {
	f := newFake()
	f.level = 29
	if _, err := Run(context.Background(), izual(), f, f, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
}

func TestHephastoTakesTheHammer(t *testing.T) {
	f := newFake()
	f.level, f.potions = 30, 5
	f.addBoss(Mon{Class: ClassHephasto, UniqueNo: uniqueHephasto, Name: NameHephasto, Pos: Pt{300, 300}}, AreaRiverOfFlame, 60)
	f.lootN = 1
	res, err := Run(context.Background(), hephasto(), f, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v", res, err)
	}
	if f.log[0] != "travel 107" || len(f.loots) == 0 || f.loots[0].Quest[0] != "Hell Forge Hammer" {
		t.Fatalf("log %v loots %+v", f.log, f.loots)
	}
}

func TestFireWallHazard(t *testing.T) {
	h := FireWall(6, 14)
	c := Ctx{V: View{Pos: Pt{100, 100}, Dangers: []Danger{{Kind: DangerFireWall, Pos: Pt{103, 100}}}}}
	if !h.Detect(c) || h.Do.Kind != RespStepAway {
		t.Fatal("should step away from a close fire wall")
	}
	c.V.Dangers[0].Pos = Pt{200, 100}
	if h.Detect(c) {
		t.Fatal("a far fire wall must not fire")
	}
}

// chaos builds a Chaos Sanctuary world: seal presets of layout A or B, and boss/Diablo spawns on use.
func chaos(vizierA, seisA, infA bool) *objFake {
	o := newObjFake()
	f := o.fake
	f.level, f.potions, f.area = 32, 6, AreaChaosSanctuary
	pick := func(a bool, hit, other int) int {
		if a {
			return hit
		}
		return other
	}
	f.presets = []Preset{
		{Kind: PresetObject, ID: SealVizier1, Pos: Pt{7650, 5290}},
		{Kind: PresetObject, ID: SealVizier2, Pos: Pt{7660, pick(vizierA, 5275, 5295)}},
		{Kind: PresetObject, ID: SealSeis, Pos: Pt{pick(seisA, 7773, 7790), 5150}},
		{Kind: PresetObject, ID: SealInfector1, Pos: Pt{pick(infA, 7893, 7920), 5290}},
		{Kind: PresetObject, ID: SealInfector2, Pos: Pt{7930, 5300}},
	}
	o.onUse = func(o *objFake, class int) {
		open := func(cs ...int) bool {
			for _, c := range cs {
				if o.modes[c] == 0 {
					return false
				}
			}
			return true
		}
		switch {
		case class == SealVizier2 && open(SealVizier1, SealVizier2):
			o.addBoss(Mon{Class: 330, UniqueNo: uniqueVizier, Name: "Grand Vizier of Chaos", Pos: Pt{7660, 5300}}, AreaChaosSanctuary, 60)
		case class == SealSeis:
			o.addBoss(Mon{Class: 331, UniqueNo: uniqueSeis, Name: "Lord De Seis", Pos: Pt{7790, 5150}}, AreaChaosSanctuary, 60)
		case class == SealInfector2 && open(SealInfector1, SealInfector2):
			o.addBoss(Mon{Class: 332, UniqueNo: uniqueInfector, Name: "Infector of Souls", Pos: Pt{7920, 5300}}, AreaChaosSanctuary, 60)
		}
		if open(SealVizier1, SealVizier2, SealSeis, SealInfector1, SealInfector2) {
			o.addBoss(Mon{Class: ClassDiablo, UniqueNo: -1, Name: NameDiablo, Pos: Pt{7791, 5293}}, AreaChaosSanctuary, 90)
		}
	}
	return o
}

func gotoLogged(f *fake, p Pt) bool {
	want := "goto " + itoa(p.X) + "," + itoa(p.Y)
	for _, l := range f.log {
		if strings.HasPrefix(l, want) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func TestSealLayoutDetectionBothWays(t *testing.T) {
	for _, g := range SealGroups {
		for _, wantA := range []bool{true, false} {
			v := map[bool]int{true: g.Probe.Value, false: g.Probe.Value + 7}[wantA]
			pos := Pt{v, 5000}
			if g.Probe.Y {
				pos = Pt{7000, v}
			}
			path, a, ok := g.Layout([]Preset{{Kind: PresetObject, ID: g.Probe.Class, Pos: pos}})
			if !ok || a != wantA {
				t.Errorf("%s: ok %v a %v want %v", g.Name, ok, a, wantA)
			}
			if wantA && len(path) != len(g.A) || !wantA && len(path) != len(g.B) {
				t.Errorf("%s: wrong path for layout", g.Name)
			}
		}
		if _, a, ok := g.Layout(nil); ok || !a {
			t.Errorf("%s: missing probe must report !ok, A", g.Name)
		}
	}
}

func TestOpenSealsAllLayoutAOpensFiveSealsAndFindsDiablo(t *testing.T) {
	o := chaos(true, true, true)
	res, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if err != nil || !res.Diablo {
		t.Fatalf("res %+v err %v log %v", res, err, o.log)
	}
	if len(res.Opened) != 5 || len(res.Bosses) != 3 {
		t.Fatalf("opened %v bosses %v notes %v", res.Opened, res.Bosses, res.Notes)
	}
	if o.uses[0] != SealVizier1 || o.uses[2] != SealSeis || o.uses[3] != SealInfector1 {
		t.Fatalf("order %v", o.uses)
	}
	for _, p := range []Pt{{7794, 5517}, {7662, 5314}, {7788, 5153}, {7919, 5290}} {
		if !gotoLogged(o.fake, p) {
			t.Errorf("never walked to %v", p)
		}
	}
	if res.Layouts["vizier"] != "A" || res.Layouts["infector"] != "A" {
		t.Fatalf("layouts %v", res.Layouts)
	}
}

func TestOpenSealsLayoutBUsesBPaths(t *testing.T) {
	o := chaos(false, false, false)
	res, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if err != nil || !res.Diablo {
		t.Fatalf("res %+v err %v", res, err)
	}
	for _, p := range []Pt{{7653, 5284}, {7803, 5154}, {7923, 5313}} {
		if !gotoLogged(o.fake, p) {
			t.Errorf("never walked to %v", p)
		}
	}
	for _, p := range []Pt{{7662, 5314}, {7788, 5153}, {7919, 5290}} {
		if gotoLogged(o.fake, p) {
			t.Errorf("walked layout A node %v", p)
		}
	}
	if res.Layouts["seis"] != "B" {
		t.Fatalf("layouts %v", res.Layouts)
	}
}

func TestOpenSealsResumesAndSkipsOpenGroups(t *testing.T) {
	o := chaos(true, true, true)
	o.modes[SealVizier1], o.modes[SealVizier2] = 1, 1
	res, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if err != nil || !res.Diablo {
		t.Fatalf("res %+v err %v", res, err)
	}
	for _, c := range o.uses {
		if c == SealVizier1 || c == SealVizier2 {
			t.Fatalf("reopened a vizier seal: %v", o.uses)
		}
	}
	if gotoLogged(o.fake, Pt{7662, 5314}) {
		t.Fatal("walked into the open vizier room")
	}
}

func TestOpenSealsDiabloAlreadyThere(t *testing.T) {
	o := chaos(true, true, true)
	o.addBoss(Mon{Class: ClassDiablo, UniqueNo: -1, Name: NameDiablo, Pos: Pt{7791, 5293}}, AreaChaosSanctuary, 90)
	res, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if err != nil || !res.Diablo || len(o.uses) != 0 {
		t.Fatalf("res %+v err %v uses %v", res, err, o.uses)
	}
}

func TestOpenSealsStuckSeal(t *testing.T) {
	o := chaos(true, true, true)
	o.onUse = nil
	o.useErr = errors.New("not clickable")
	_, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if !errors.Is(err, ErrSealStuck) {
		t.Fatalf("want ErrSealStuck, got %v", err)
	}
	if len(o.uses) != 5 {
		t.Fatalf("tries %d, want 5", len(o.uses))
	}
}

func TestOpenSealsDiabloNeverSpawns(t *testing.T) {
	o := chaos(true, true, true)
	inner := o.onUse
	o.onUse = func(o *objFake, class int) {
		inner(o, class)
		for _, m := range o.mons {
			if m.Class == ClassDiablo {
				m.area = 0
			}
		}
	}
	_, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if !errors.Is(err, ErrBossNotFound) {
		t.Fatalf("want ErrBossNotFound, got %v", err)
	}
}

func TestOpenSealsDeathAborts(t *testing.T) {
	o := chaos(true, true, true)
	o.attackErr = ErrDead
	o.onUse = nil
	o.modes[SealVizier1], o.modes[SealVizier2] = 0, 0
	o.addBoss(Mon{Class: 330, UniqueNo: uniqueVizier, Name: "Grand Vizier of Chaos", Pos: Pt{7660, 5300}}, AreaChaosSanctuary, 60)
	_, err := OpenSeals(context.Background(), o, o, o, SealOpts{})
	if !errors.Is(err, ErrDead) {
		t.Fatalf("want ErrDead, got %v", err)
	}
}

func TestRunDiabloFromRiverOfFlame(t *testing.T) {
	o := chaos(true, false, true)
	f := o.fake
	f.area = AreaRiverOfFlame
	f.lootN = 1
	res, err := RunDiablo(context.Background(), o, o, o, SealOpts{}, Opts{})
	if err != nil || !res.Killed || res.Boss != NameDiablo {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "exit 108" {
		t.Fatalf("first %v", f.log[0])
	}
	if len(res.Phases) == 0 || res.Phases[0] != "seals" {
		t.Fatalf("phases %v", res.Phases)
	}
	for _, m := range f.mons {
		if m.Class == ClassDiablo && m.Alive() {
			t.Fatal("Diablo alive")
		}
	}
}

func TestRunDiabloFromTownTravelsToWaypoint(t *testing.T) {
	o := chaos(true, true, true)
	o.area = 103
	if _, err := RunDiablo(context.Background(), o, o, o, SealOpts{}, Opts{}); err != nil {
		t.Fatal(err)
	}
	if o.log[0] != "travel 107" || o.log[1] != "exit 108" {
		t.Fatalf("route %v", o.log[:2])
	}
}

func TestRunDiabloFloorAndCapability(t *testing.T) {
	o := chaos(true, true, true)
	o.level = 29
	if _, err := RunDiablo(context.Background(), o, o, o, SealOpts{}, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("want ErrTooWeak, got %v", err)
	}
	o.level = 32
	if _, err := RunDiablo(context.Background(), o, o, nil, SealOpts{}, Opts{}); !errors.Is(err, ErrAborted) {
		t.Fatalf("want ErrAborted without ObjectOps, got %v", err)
	}
	if len(o.log) != 0 {
		t.Fatalf("walked before refusing: %v", o.log)
	}
}
