package d2dinput

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/input"
)

func TestCastVerbLines(t *testing.T) {
	c, srv := rig(t, fake.Sample())
	at := input.Point{X: 65, Y: 147} // cell 13.1, 29.5
	if err := c.CastPoint(9, false, at); err != nil {
		t.Fatal(err)
	}
	if err := c.CastPoint(9, true, at); err != nil {
		t.Fatal(err)
	}
	if err := c.CastUnit(9, false, at, 5); err != nil {
		t.Fatal(err)
	}
	if err := c.CastUnit(9, true, at, 5); err != nil {
		t.Fatal(err)
	}
	want := []string{"cmd skill 9 13.1 29.5", "cmd skill 9 13.1 29.5 0 left", "cmd skill 9 13.1 29.5 5", "cmd skill 9 13.1 29.5 5 left"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
	if _, ok := any(c).(input.HoldCaster); ok {
		t.Fatal("no fork build has a hold verb; the controller must not claim one")
	}
}
