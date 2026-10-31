package d2dcombat

import (
	"context"
	"reflect"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/input"
)

func newInput(t *testing.T) (*Input, *fake.Server) {
	t.Helper()
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	srv, err := fake.New(path, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	t.Cleanup(func() { c.Close() })
	ctl := d2dinput.New(c)
	ctl.Skills = map[string]int{"sk36": 36}
	return &Input{Controller: ctl, ctx: context.Background()}, srv
}

func TestBeltKeysAndCasts(t *testing.T) {
	in, srv := newInput(t)
	// Sample monster #5 stands at cell (13.00, 29.50) = subtile (65, 147)
	if err := in.CastAt("sk36", input.Point{X: 65, Y: 147}); err != nil {
		t.Fatal(err)
	}
	if err := in.CastAt("sk36", input.Point{X: 400, Y: 400}); err != nil { // nobody there: ground cast
		t.Fatal(err)
	}
	if err := in.Key("2"); err != nil {
		t.Fatal(err)
	}
	want := []string{"monsters", "cmd skill 36 13.1 29.5 5", "monsters", "cmd skill 36 80.1 80.1", "cmd belt 1"}
	if got := srv.Lines(); !reflect.DeepEqual(got, want) {
		t.Fatalf("lines\n got %q\nwant %q", got, want)
	}
	if s := in.Stats(); s.Potions != 1 || s.Casts[36] != 2 {
		t.Fatalf("stats %+v", s)
	}
}

func TestUnmappedHotkeyFallsBack(t *testing.T) {
	in, srv := newInput(t)
	_ = in.CastAt("f9", input.Point{X: 65, Y: 147}) // no id: key press then right click (d2dinput behaviour)
	if got := srv.Lines(); len(got) == 0 || got[0] != "key F9" {
		t.Fatalf("lines %q", got)
	}
}
