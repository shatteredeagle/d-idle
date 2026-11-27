package d2dcombat

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	capdata "d2lod-bot/data/capabilities"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/delivery"
	"d2lod-bot/internal/input"
)

// castInput is the combat input over a fake d2d whose monster list also holds a corpse (unit 8 at cell 20, 20).
func castInput(t *testing.T, p *input.Planner, mods ...func(*fake.Fixtures)) (*Input, *fake.Server) {
	t.Helper()
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	fix := fake.Sample()
	fix.Monsters = append(fix.Monsters, "fallen1\t20.00\t20.00\t900\t500\t0/2\tDD\t-\t-\tlvl1\tFallen\t#8\t-")
	for _, m := range mods {
		m(&fix)
	}
	srv, err := fake.New(path, fix, fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	t.Cleanup(func() { c.Close() })
	ctl := d2dinput.New(c)
	ctl.Planner = p
	return &Input{Controller: ctl, ctx: context.Background()}, srv
}

// rowOf is the first skill (by name) of class with that target ("" any but corpse) and press: the test names no
// skill, the table does.
func rowOf(t *testing.T, tab *delivery.Table, class, target string, repeat bool) (string, *delivery.Use) {
	t.Helper()
	var names []string
	for n, e := range tab.Skills {
		u := e.Delivery.Use
		if e.Class == class && u != nil && (u.Target == target || target == "" && u.Target != "corpse") && (u.Press != "tap") == repeat {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		t.Fatalf("%s has no %s row (repeat %v) in the table", class, target, repeat)
	}
	sort.Strings(names)
	return names[0], tab.Skills[names[0]].Delivery.Use
}

func tail(left bool, unit int) string {
	switch {
	case left:
		return fmt.Sprintf(" %d left", unit)
	case unit != 0:
		return fmt.Sprintf(" %d", unit)
	}
	return ""
}

// Two builds (Sorceress, Paladin): the delivery row picks hand and target, the cast verb goes out alone with the knob
// on, the old command alone with the knob off, and both send the same line.
func TestDeliverTwoBuilds(t *testing.T) {
	tab, err := delivery.Load(capdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"sorceress", "paladin"} {
		for _, knob := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/planner=%v", class, knob), func(t *testing.T) {
				p := &input.Planner{Enabled: func() bool { return knob },
					Caps: &input.BackendCaps{Verbs: []string{input.CapCastUnit, input.CapCastPoint, input.CapCastHold}}}
				in, srv := castInput(t, p)
				const id = 40 // any id: the fake does not check it
				_, unitUse := rowOf(t, tab, class, "unit", false)
				_, pointUse := rowOf(t, tab, class, "point", false)
				if err := in.Deliver(id, unitUse, input.Point{X: 65, Y: 147}, 0); err != nil { // monster #5
					t.Fatal(err)
				}
				if err := in.Deliver(id, pointUse, input.Point{X: 400, Y: 400}, 0); err != nil {
					t.Fatal(err)
				}
				want := []string{
					"monsters", "cmd skill 40 13.1 29.5" + tail(unitUse.Hand == "left", 5),
					"cmd skill 40 80.1 80.1" + tail(pointUse.Hand == "left", 0),
				}
				if got := srv.Lines(); !reflect.DeepEqual(got, want) {
					t.Fatalf("lines\n got %q\nwant %q", got, want)
				}
				v, k, _, f := p.Counts()
				if knob && (v != 2 || k != 0 || f != 0) || !knob && v+k+f != 0 {
					t.Fatalf("counts verb=%d key=%d failed=%d", v, k, f)
				}
				if in.Stats().Casts[id] != 2 {
					t.Fatalf("stats %+v", in.Stats())
				}
			})
		}
	}
}

// A repeat press asks for the hold verb. A build that lists the cap but refuses the start (here: an older fork without
// the verb) gets the old single command once; a build whose hold starts gets the hold alone, never a second cast.
func TestDeliverHold(t *testing.T) {
	tab, err := delivery.Load(capdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"paladin", "barbarian"} { // the Sorceress has no repeat or channel row
		for _, verb := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/verb=%v", class, verb), func(t *testing.T) {
				_, use := rowOf(t, tab, class, "", true)
				p := &input.Planner{Enabled: func() bool { return true }, Caps: &input.BackendCaps{Verbs: []string{input.CapCastHold}}}
				in, srv := castInput(t, p, func(f *fake.Fixtures) {
					if verb {
						f.Hold = []string{"held=15 casts=1 frames=15 state=done"}
					}
				})
				if err := in.Deliver(7, use, input.Point{X: 400, Y: 400}, 600*time.Millisecond); err != nil {
					t.Fatal(err)
				}
				got := srv.Lines()
				if len(got) > 0 && got[0] == "monsters" && use.Target == "unit" { // nobody near the aim: a point cast
					got = got[1:]
				}
				l := "0"
				if use.Hand == "left" {
					l = "1"
				}
				want := []string{"cmd skillhold 7 80.1 80.1 -1 15 " + l, "cmd skill 7 80.1 80.1" + tail(use.Hand == "left", 0)}
				wv, wk := 0, 1
				if verb {
					want, wv, wk = []string{want[0], "cmd skillhold status"}, 1, 0
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("lines\n got %q\nwant %q", got, want)
				}
				if v, k, _, _ := p.Counts(); v != wv || k != wk {
					t.Fatalf("verb=%d key=%d", v, k)
				}
				if in.Stats().Casts[7] != 1 {
					t.Fatalf("stats %+v", in.Stats())
				}
			})
		}
	}
}

func TestDeliverCorpseCarriesTheCorpseUnit(t *testing.T) {
	p := &input.Planner{Enabled: func() bool { return true }, Caps: &input.BackendCaps{Verbs: []string{input.CapCastUnit}}}
	in, srv := castInput(t, p)
	use := &delivery.Use{Target: "corpse", Press: "tap"}
	if err := in.Deliver(3, use, input.Point{X: 100, Y: 100}, 0); err != nil {
		t.Fatal(err)
	}
	if err := in.Deliver(3, use, input.Point{X: 400, Y: 400}, 0); !errors.Is(err, ErrNoCorpse) {
		t.Fatalf("no corpse near: %v", err)
	}
	want := []string{"monsters", "cmd skill 3 20.1 20.1 8", "monsters"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
	if err := in.Deliver(3, nil, input.Point{}, 0); err == nil {
		t.Fatal("a skill without a use row must be an error")
	}
}

// CastAt and Swing keep their lines with the knob on; they only count as verbs.
func TestCastAtAndSwingThroughVerbs(t *testing.T) {
	p := &input.Planner{Enabled: func() bool { return true }, Caps: &input.BackendCaps{Verbs: []string{input.CapCastUnit, input.CapCastPoint}}}
	in, srv := castInput(t, p)
	in.Skills = map[string]int{"sk36": 36}
	if err := in.CastAt("sk36", input.Point{X: 65, Y: 147}); err != nil {
		t.Fatal(err)
	}
	if err := in.CastAt("sk36", input.Point{X: 400, Y: 400}); err != nil {
		t.Fatal(err)
	}
	if err := in.Swing(0, input.Point{X: 65, Y: 147}); err != nil {
		t.Fatal(err)
	}
	want := []string{"monsters", "cmd skill 36 13.1 29.5 5", "monsters", "cmd skill 36 80.1 80.1", "monsters", "cmd skill 0 13.1 29.5 5 left"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
	if v, _, _, _ := p.Counts(); v != 3 {
		t.Fatalf("verbs %d", v)
	}
	if s := in.Stats(); s.Casts[36] != 2 || s.Casts[0] != 1 || s.Clicks != 1 {
		t.Fatalf("stats %+v", s)
	}
}
