package replay

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/input"
)

// Input implements input.Controller by recording every call as canonical text ("click L 10 20
// shift", "castat F 5 6"). With Inner nil it simulates like input.Fake (the player teleports to the
// last ClickWorld/MoveTo target); otherwise it forwards, which is how the recorder captures live runs.
type Input struct {
	Inner input.Controller
	// Now returns the stream time stamped on each call (default: zero).
	Now func() time.Duration
	// OnCall, if set, receives every call as it happens (the recorder uses it).
	OnCall func(Call)

	mu    sync.Mutex
	calls []Call
	at    input.Point
	Err   error // returned by the next call (Inner is not called)
}

// NewInput returns a simulating Input with the player at start.
func NewInput(start input.Point) *Input { return &Input{at: start} }

// Calls returns the recorded call texts in order.
func (in *Input) Calls() []string {
	in.mu.Lock()
	defer in.mu.Unlock()
	out := make([]string, len(in.calls))
	for i, c := range in.calls {
		out[i] = c.Text
	}
	return out
}

// Log returns the recorded calls with their times.
func (in *Input) Log() []Call {
	in.mu.Lock()
	defer in.mu.Unlock()
	return append([]Call(nil), in.calls...)
}

// do records the call and returns the injected error, if any.
func (in *Input) do(text string) error {
	in.mu.Lock()
	c := Call{Text: text}
	if in.Now != nil {
		c.T = in.Now()
	}
	in.calls = append(in.calls, c)
	err := in.Err
	in.Err = nil
	cb := in.OnCall
	in.mu.Unlock()
	if cb != nil {
		cb(c)
	}
	return err
}

func (in *Input) MoveMouse(x, y int) error {
	if err := in.do(fmt.Sprintf("move %d %d", x, y)); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.MoveMouse(x, y)
	}
	return nil
}

func (in *Input) Click(b input.Button, x, y int, mods ...input.Mod) error {
	name := "L"
	if b == input.Right {
		name = "R"
	}
	t := fmt.Sprintf("click %s %d %d", name, x, y)
	for _, m := range mods {
		t += " " + string(m)
	}
	if err := in.do(t); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.Click(b, x, y, mods...)
	}
	return nil
}

func (in *Input) Key(name string) error {
	if err := in.do("key " + name); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.Key(name)
	}
	return nil
}

func (in *Input) Hold(key string) error {
	if err := in.do("hold " + key); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.Hold(key)
	}
	return nil
}

func (in *Input) Release(key string) error {
	if err := in.do("release " + key); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.Release(key)
	}
	return nil
}

func (in *Input) Pos() (input.Point, error) {
	if in.Inner != nil {
		return in.Inner.Pos()
	}
	in.mu.Lock()
	defer in.mu.Unlock()
	return in.at, nil
}

func (in *Input) setAt(w input.Point) {
	in.mu.Lock()
	in.at = w
	in.mu.Unlock()
}

func (in *Input) ClickWorld(w input.Point) error {
	if err := in.do(fmt.Sprintf("clickworld %d %d", w.X, w.Y)); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.ClickWorld(w)
	}
	in.setAt(w)
	return nil
}

func (in *Input) MoveTo(w input.Point, o input.MoveOpts) error {
	if err := in.do(fmt.Sprintf("moveto %d %d", w.X, w.Y)); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.MoveTo(w, o)
	}
	in.setAt(w)
	return nil
}

func (in *Input) SelectSkill(hotkey string) error {
	if err := in.do("skill " + hotkey); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.SelectSkill(hotkey)
	}
	return nil
}

func (in *Input) CastAt(hotkey string, w input.Point) error {
	if err := in.do(fmt.Sprintf("castat %s %d %d", hotkey, w.X, w.Y)); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.CastAt(hotkey, w)
	}
	return nil
}

func (in *Input) Interact(t input.Target) error {
	if err := in.do(fmt.Sprintf("interact %d %d", t.Type, t.ID)); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.Interact(t)
	}
	return nil
}

func (in *Input) CloseAll() error {
	if err := in.do("closeall"); err != nil {
		return err
	}
	if in.Inner != nil {
		return in.Inner.CloseAll()
	}
	return nil
}

var _ input.Controller = (*Input)(nil)

// FormatCalls joins call texts one per line (the golden file format).
func FormatCalls(calls []string) string {
	if len(calls) == 0 {
		return ""
	}
	return strings.Join(calls, "\n") + "\n"
}
