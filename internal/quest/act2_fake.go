package quest

import (
	"context"
	"fmt"
)

// FakeAct2Env is an Act2Env for tests. Objects per area drive HasObject; Recipes maps a sorted-join of input codes
// to the outputs a Transmute yields; Placed records PlaceItem calls.
type FakeAct2Env struct {
	*FakeEnv
	Objects map[int][]int       // area -> object classes present
	Recipes map[string][]string // "msf+vip" -> ["hst"]
	Chests  map[int][]string    // chest class -> item codes that drop on OpenChest
	Dropped []string            // items on the ground, taken by the next Pickup of that code
	Placed  []string
	Hired   []string
}

func NewFakeAct2Env(area int) *FakeAct2Env {
	f := &FakeAct2Env{FakeEnv: NewFakeEnv(area), Objects: map[int][]int{}, Chests: map[int][]string{},
		Recipes: map[string][]string{ItemShaft + "+" + ItemViperAmulet: {ItemHoradricStaff}}}
	return f
}

func (f *FakeAct2Env) Pickup(ctx context.Context, item string) error {
	if err := f.op("pickup", item); err != nil {
		return err
	}
	for i, d := range f.Dropped {
		if d == item {
			f.Dropped = append(f.Dropped[:i], f.Dropped[i+1:]...)
			f.Pack[item] = true
			return nil
		}
	}
	if !f.NoPickup {
		f.Pack[item] = true
	}
	return nil
}

func (f *FakeAct2Env) UseItem(_ context.Context, code string) error {
	if err := f.op("useitem", code); err != nil {
		return err
	}
	delete(f.Pack, code)
	return nil
}

func (f *FakeAct2Env) PlaceItem(_ context.Context, code string, obj int) error {
	if err := f.op("place", code, "@", obj); err != nil {
		return err
	}
	if !f.hasObj(obj) {
		return fmt.Errorf("fake: no object %d in area %d", obj, f.CurArea)
	}
	delete(f.Pack, code)
	f.Placed = append(f.Placed, code)
	return nil
}

func (f *FakeAct2Env) OpenChest(_ context.Context, obj int) error {
	if err := f.op("openchest", obj); err != nil {
		return err
	}
	f.Dropped = append(f.Dropped, f.Chests[obj]...)
	delete(f.Chests, obj)
	return nil
}

func (f *FakeAct2Env) hasObj(class int) bool {
	for _, o := range f.Objects[f.CurArea] {
		if o == class {
			return true
		}
	}
	return false
}

func (f *FakeAct2Env) HasObject(_ context.Context, class int) bool { return f.hasObj(class) }

func (f *FakeAct2Env) HireMerc(_ context.Context, act int, aura string, _ int) error {
	if err := f.op("hire", act, aura); err != nil {
		return err
	}
	f.Hired = append(f.Hired, aura)
	return nil
}

func (f *FakeAct2Env) Transmute(_ context.Context, in []string) ([]string, error) {
	if err := f.op("transmute", in); err != nil {
		return nil, err
	}
	key := ""
	for i, c := range in {
		if i > 0 {
			key += "+"
		}
		key += c
	}
	out, ok := f.Recipes[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCubeNoResult, key)
	}
	for _, c := range in {
		delete(f.Pack, c)
	}
	for _, o := range out {
		f.Pack[o] = true
	}
	return out, nil
}

var _ Act2Env = (*FakeAct2Env)(nil)

// ToTown goes to Lut Gholein (the FakeEnv one returns to the Rogue Encampment).
func (f *FakeAct2Env) ToTown(_ context.Context) error {
	if err := f.op("totown"); err != nil {
		return err
	}
	f.CurArea = AreaLutGholein
	return nil
}

// FarTombEnv is a FakeAct2Env whose orifice only exists after enough ProbeFar steps in its tomb (the far room
// streams in when the player gets there), with an optional hint and tombs that cannot be entered.
type FarTombEnv struct {
	*FakeAct2Env
	Tomb     int         // the tomb with the orifice
	Steps    int         // probe steps needed before the orifice is visible
	Hint     int         // 0: no hint
	Unenter  map[int]int // tomb -> failed GoArea attempts left
	Entered  []int
	probed   int
	revealed bool
}

func NewFarTombEnv(canyon, tomb, steps int) *FarTombEnv {
	f := &FarTombEnv{FakeAct2Env: NewFakeAct2Env(canyon), Tomb: tomb, Steps: steps, Unenter: map[int]int{}}
	return f
}

func (f *FarTombEnv) GoArea(ctx context.Context, a int) error {
	if n := f.Unenter[a]; n > 0 {
		f.Unenter[a] = n - 1
		return fmt.Errorf("fake: exit to %d did not fire", a)
	}
	if err := f.FakeAct2Env.GoArea(ctx, a); err != nil {
		return err
	}
	f.Entered = append(f.Entered, a)
	f.probed, f.revealed = 0, false
	return nil
}

func (f *FarTombEnv) HasObject(ctx context.Context, class int) bool {
	if class == ObjOrifice && f.CurArea == f.Tomb {
		return f.revealed
	}
	return f.FakeAct2Env.HasObject(ctx, class)
}

func (f *FarTombEnv) ProbeFar(context.Context) (bool, error) {
	f.probed++
	f.revealed = f.CurArea == f.Tomb && f.probed >= f.Steps
	return f.probed < maxTombProbes, nil
}

func (f *FarTombEnv) StaffTombLevel(context.Context) (int, bool) { return f.Hint, f.Hint != 0 }
