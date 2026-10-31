package d2dctl_test

import (
	"os"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// TestRealD2D pings a running d2d. Skipped unless D2D_SOCK names its control socket (see docs/d2dctl.md).
// It only reads: ping, info, state, npcs, monsters, ground.
func TestRealD2D(t *testing.T) {
	sock := os.Getenv("D2D_SOCK")
	if sock == "" {
		t.Skip("D2D_SOCK not set")
	}
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 5 * time.Second})
	defer c.Close()
	if err := c.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	in, err := c.Info(ctx())
	if err != nil || in.W == 0 {
		t.Fatalf("info %+v %v", in, err)
	}
	if _, _, err := c.State(ctx()); err != nil {
		t.Fatal(err)
	}
	for _, f := range []func() (*d2dctl.Reply, error){
		func() (*d2dctl.Reply, error) { l, err := c.NPCs(ctx()); return l.Reply, err },
		func() (*d2dctl.Reply, error) { l, err := c.Monsters(ctx()); return l.Reply, err },
		func() (*d2dctl.Reply, error) { l, err := c.Ground(ctx()); return l.Reply, err },
	} {
		r, err := f()
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Bad) > 0 {
			t.Errorf("%s: unparsed lines (layout drift?): %q", r.Cmd, r.Bad)
		}
	}
}
