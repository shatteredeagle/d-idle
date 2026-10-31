package quest

import (
	"context"
	"fmt"
)

// FakeEnv is an Env for tests: it records every call, tracks the area and a set of pack items, and lets a test
// script area changes and failures.
type FakeEnv struct {
	FakeDialog
	CurArea int
	Pack    map[string]bool
	Log     []string
	// OnArea maps a Warriv-style travel click or portal to an arrival area: label -> area.
	Travel map[string]int
	// Fail makes the named op ("kill", "goarea", ...) return an error once.
	Fail map[string]error
	// Gives lists items that appear in the pack when Pickup of that code runs (default: the item itself).
	NoPickup bool
}

func NewFakeEnv(area int) *FakeEnv {
	e := &FakeEnv{CurArea: area, Pack: map[string]bool{}, Travel: map[string]int{}, Fail: map[string]error{}}
	e.FakeDialog.Effect = func(npc int, level, label string) {
		if a, ok := e.Travel[label]; ok {
			e.CurArea = a
		}
	}
	return e
}

func (e *FakeEnv) op(name string, a ...any) error {
	e.Log = append(e.Log, name+fmt.Sprint(a...))
	if err, ok := e.Fail[name]; ok {
		delete(e.Fail, name)
		return err
	}
	return nil
}

func (e *FakeEnv) Has(code string) bool { return e.Pack[code] }
func (e *FakeEnv) Area() int            { return e.CurArea }
func (e *FakeEnv) GoNPC(_ context.Context, c int) error {
	return e.op("gonpc", c)
}
func (e *FakeEnv) GoArea(_ context.Context, a int) error {
	if err := e.op("goarea", a); err != nil {
		return err
	}
	e.CurArea = a
	return nil
}
func (e *FakeEnv) Clear(_ context.Context, a int) error { return e.op("clear", a) }
func (e *FakeEnv) Kill(_ context.Context, t string, a int) error {
	return e.op("kill", t, "@", a)
}
func (e *FakeEnv) Pickup(_ context.Context, item string) error {
	if err := e.op("pickup", item); err != nil {
		return err
	}
	if !e.NoPickup {
		e.Pack[item] = true
	}
	return nil
}
func (e *FakeEnv) UseObject(_ context.Context, class, a int) error {
	if err := e.op("useobject", class, "@", a); err != nil {
		return err
	}
	if to, ok := e.Travel[fmt.Sprint("obj", class)]; ok {
		e.CurArea = to
	}
	return nil
}
func (e *FakeEnv) TouchWaypoint(_ context.Context) error { return e.op("touchwp") }
func (e *FakeEnv) WaitArea(_ context.Context, a int) error {
	if err := e.op("waitarea", a); err != nil {
		return err
	}
	if e.CurArea != a {
		return fmt.Errorf("quest: area %d, waiting for %d", e.CurArea, a)
	}
	return nil
}
func (e *FakeEnv) ToTown(_ context.Context) error {
	if err := e.op("totown"); err != nil {
		return err
	}
	e.CurArea = AreaRogueEncampment
	return nil
}

var _ Env = (*FakeEnv)(nil)
