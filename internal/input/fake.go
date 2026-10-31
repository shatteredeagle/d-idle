package input

import (
	"fmt"
	"sync"
	"time"
)

// Fake is a Controller for tests: it records calls and simulates a player that teleports to the
// last ClickWorld/MoveTo target. Set Err to make the next call fail.
type Fake struct {
	mu    sync.Mutex
	Calls []string
	At    Point
	Err   error
	X     Transform
}

func NewFake(start Point) *Fake { return &Fake{At: start, X: Calibrated} }

func (f *Fake) rec(format string, a ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, fmt.Sprintf(format, a...))
	err := f.Err
	f.Err = nil
	return err
}

func (f *Fake) MoveMouse(x, y int) error { return f.rec("move %d %d", x, y) }
func (f *Fake) Click(b Button, x, y int, mods ...Mod) error {
	return f.rec("click %s %d %d %v", b.letter(), x, y, mods)
}
func (f *Fake) Key(name string) error      { return f.rec("key %s", name) }
func (f *Fake) TypeChar(ch rune) error     { return f.rec("char %c", ch) }
func (f *Fake) Hold(key string) error      { return f.rec("hold %s", key) }
func (f *Fake) Release(key string) error   { return f.rec("release %s", key) }
func (f *Fake) Pos() (Point, error)        { f.mu.Lock(); defer f.mu.Unlock(); return f.At, nil }
func (f *Fake) SelectSkill(k string) error { return f.rec("skill %s", k) }
func (f *Fake) CloseAll() error            { return f.rec("closeall") }
func (f *Fake) Interact(t Target) error    { return f.rec("interact %d/%d", t.Type, t.ID) }
func (f *Fake) CastAt(k string, w Point) error {
	return f.rec("cast %s %v", k, w)
}
func (f *Fake) ClickWorld(w Point) error {
	err := f.rec("clickworld %v", w)
	if err == nil {
		f.mu.Lock()
		f.At = w
		f.mu.Unlock()
	}
	return err
}
func (f *Fake) MoveTo(w Point, _ MoveOpts) error {
	err := f.rec("moveto %v", w)
	if err == nil {
		f.mu.Lock()
		f.At = w
		f.mu.Unlock()
	}
	return err
}

var _ Controller = (*Fake)(nil)
var _ Controller = (*Game)(nil)

// HoldClick records a held click.
func (f *Fake) HoldClick(b Button, x, y int, d time.Duration) error {
	return f.rec("hold %s %d %d %v", b.letter(), x, y, d)
}
