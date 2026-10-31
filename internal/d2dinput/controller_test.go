package d2dinput

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/input"
)

type fclock struct{ t time.Time }

func (f *fclock) Now() time.Time        { return f.t }
func (f *fclock) Sleep(d time.Duration) { f.t = f.t.Add(d) }

func rig(t *testing.T, fix fake.Fixtures) (*Controller, *fake.Server) {
	t.Helper()
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	srv, err := fake.New(path, fix, fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	t.Cleanup(func() { c.Close() })
	ctl := New(c)
	ctl.Ctx = context.Background()
	ctl.Clock = &fclock{t: time.Unix(0, 0)}
	return ctl, srv
}

// Each case: one call, the exact lines the channel must see (priming click and "debug level" included).
func TestVerbLines(t *testing.T) {
	prime := []string{"debug level", "click 400 300"}
	cases := []struct {
		name string
		call func(c *Controller) error
		want []string
	}{
		{"hover", func(c *Controller) error { return c.MoveMouse(10, 20) }, []string{"move 10 20"}},
		{"click", func(c *Controller) error { return c.Click(input.Left, 11, 22) }, []string{"click 11 22"}},
		{"rclick", func(c *Controller) error { return c.Click(input.Right, 11, 22) }, []string{"rclick 11 22"}},
		{"key esc", func(c *Controller) error { return c.Key("esc") }, []string{"key Escape"}},
		{"key enter", func(c *Controller) error { return c.Key("enter") }, []string{"key Return"}},
		{"key f3", func(c *Controller) error { return c.Key("f3") }, []string{"key F3"}},
		{"key potion", func(c *Controller) error { return c.Key("2") }, []string{"key 2"}},
		{"key panel", func(c *Controller) error { return c.Key("I") }, []string{"key i"}},
		{"text", func(c *Controller) error { return c.TypeChar('7') }, []string{"text 7"}},
		{"clickworld", func(c *Controller) error { return c.ClickWorld(input.Point{X: 175, Y: 70}) },
			append(append([]string{}, prime...), "cmd move 35.1 14.1")},
		{"select mapped", func(c *Controller) error { c.Skills = map[string]int{"f1": 54}; return c.SelectSkill("F1") },
			[]string{"cmd select 54 0"}},
		{"select unmapped", func(c *Controller) error { return c.SelectSkill("f2") }, []string{"key F2"}},
		{"cast mapped", func(c *Controller) error {
			c.Skills = map[string]int{"f1": 54}
			return c.CastAt("f1", input.Point{X: 160, Y: 70})
		},
			[]string{"cmd skill 54 32.1 14.1"}},
		{"cast unmapped", func(c *Controller) error { return c.CastAt("f4", input.Point{X: 153, Y: 68}) },
			[]string{"key F4", "debug players", "rclick 400 292"}},
		{"cast unit", func(c *Controller) error { return c.Cast(54, input.Point{X: 160, Y: 70}, 5) },
			[]string{"cmd skill 54 32.1 14.1 5"}},
		{"interact npc", func(c *Controller) error {
			return c.Interact(input.Target{Type: 1, ID: d2dsrc.NPCIDBase + 3})
		}, append(append([]string{}, prime...), "cmd interact 3")},
		{"interact item", func(c *Controller) error { return c.Interact(input.Target{Type: 4, ID: 17}) }, []string{"cmd pickup 17"}},
		{"interact monster", func(c *Controller) error {
			return c.Interact(input.Target{Type: 1, ID: 5, Pos: input.Point{X: 65, Y: 147}})
		}, []string{"cmd skill 0 13.1 29.5 5 left"}},
		{"belt", func(c *Controller) error { return c.Belt(1, false) }, []string{"cmd belt 1"}},
		{"belt merc", func(c *Controller) error { return c.Belt(2, true) }, []string{"cmd belt 2 merc"}},
		{"pickup", func(c *Controller) error { return c.Pickup(9) }, []string{"cmd pickup 9"}},
		{"choose menu", func(c *Controller) error { return c.ChooseMenu("TALK") }, []string{"menu", "click 812 468"}},
		{"closeall", func(c *Controller) error { return c.CloseAll() },
			[]string{"state", "dump ui", "menu", "click 812 498", "state", "dump ui", "menu", "click 812 498", "state", "dump ui", "menu", "click 812 498", "state", "dump ui", "menu", "click 812 498"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := rig(t, fake.Sample())
			if err := tc.call(c); err != nil && tc.name != "closeall" {
				t.Fatal(err)
			}
			if got := srv.Lines(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("lines\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestUnsupported(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	for name, err := range map[string]error{
		"shift click": c.Click(input.Left, 1, 2, input.ModShift),
		"ctrl click":  c.Click(input.Right, 1, 2, input.ModCtrl),
		"hold":        c.Hold("shift"),
		"release":     c.Release("shift"),
		"key shift":   c.Key("shift"),
		"merc":        c.Interact(input.Target{Type: 1, ID: d2dsrc.MercID}),
		"object":      c.Interact(input.Target{Type: 2, ID: 3}),
	} {
		if !errors.Is(err, input.ErrUnsupported) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(srv.Lines()) != 0 {
		t.Errorf("unsupported calls sent %q", srv.Lines())
	}
	if err := c.Key("pgup"); err == nil || errors.Is(err, input.ErrUnsupported) {
		t.Errorf("unknown key: %v", err)
	}
}

func TestPos(t *testing.T) {
	c, _ := rig(t, fake.Sample())
	p, err := c.Pos()
	if err != nil || p != (input.Point{X: 153, Y: 68}) {
		t.Fatalf("%v %v", p, err)
	}
	c.Origin = input.Point{X: 100, Y: 200}
	if p, _ = c.Pos(); p != (input.Point{X: 253, Y: 268}) {
		t.Fatalf("origin: %v", p)
	}
}

func TestPrimeOncePerLevel(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	for i := 0; i < 3; i++ {
		if err := c.Move(input.Point{X: 160, Y: 70}); err != nil {
			t.Fatal(err)
		}
	}
	n := 0
	for _, l := range srv.Lines() {
		if strings.HasPrefix(l, "click") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("kicks %d, lines %q", n, srv.Lines())
	}
	// A level change (as "debug level" reports it) primes again.
	srv.Fix.Level = "2 30.70 13.70 57 41 864 912"
	if err := c.Move(input.Point{X: 160, Y: 70}); err != nil {
		t.Fatal(err)
	}
	if ls := srv.Lines(); ls[len(ls)-2] != "click 400 300" {
		t.Fatalf("no kick after level change: %q", ls)
	}
}

func TestMoveToArrived(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	if err := c.MoveTo(input.Point{X: 154, Y: 68}, input.MoveOpts{}); err != nil {
		t.Fatal(err)
	}
	if got := srv.Lines(); len(got) != 1 || got[0] != "debug players" {
		t.Fatalf("%q", got)
	}
}

// The fake player never moves: MoveTo kicks once more after the first stall, then gives up with ErrMoveFailed.
func TestMoveToStuck(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	err := c.MoveTo(input.Point{X: 253, Y: 68}, input.MoveOpts{Timeout: time.Minute})
	if !errors.Is(err, input.ErrMoveFailed) {
		t.Fatalf("%v", err)
	}
	kicks, moves := 0, 0
	for _, l := range srv.Lines() {
		switch {
		case l == "click 400 300":
			kicks++
		case strings.HasPrefix(l, "cmd move"):
			moves++
		}
	}
	if kicks != 2 || moves < 3 { // priming click + stall kick
		t.Fatalf("kicks %d moves %d", kicks, moves)
	}
}

func TestMoveToTimeout(t *testing.T) {
	c, _ := rig(t, fake.Sample())
	err := c.MoveTo(input.Point{X: 253, Y: 68}, input.MoveOpts{Timeout: time.Second})
	if !errors.Is(err, input.ErrMoveFailed) {
		t.Fatalf("%v", err)
	}
}

func TestChooseMenuMissing(t *testing.T) {
	c, _ := rig(t, fake.Sample())
	if err := c.ChooseMenu("gamble"); err == nil {
		t.Fatal("want error")
	}
}

func TestUsePortal(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	if err := c.UsePortal(1); err != nil {
		t.Fatal(err) // fake player at 153,68, portal 1 at 30.7,14.3 -> (153,71): within tolerance
	}
	ls := srv.Lines()
	if !strings.HasPrefix(ls[len(ls)-1], "click ") {
		t.Fatalf("%q", ls)
	}
	if err := c.UsePortal(7); err == nil {
		t.Fatal("bad row accepted")
	}
}

func TestAsSemantic(t *testing.T) {
	c, _ := rig(t, fake.Sample())
	if _, ok := input.AsSemantic(c); !ok {
		t.Fatal("not semantic")
	}
	if _, ok := input.AsSemantic(input.NewFake(input.Point{})); ok {
		t.Fatal("fake claims semantic")
	}
}

func TestKeyNameTable(t *testing.T) {
	for in, want := range map[string]string{"ESC": "Escape", "return": "Return", "f8": "F8", "q": "q", "4": "4", "Space": "Space"} {
		if got, err := KeyName(in); err != nil || got != want {
			t.Errorf("%s: %q %v", in, got, err)
		}
	}
}
