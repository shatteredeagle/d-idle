package d2dworld

import (
	"context"
	"os"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// TestLive runs against a real d2d (private fork on the worker) when D2D_SOCK is set; skipped otherwise. It prints the
// current level as ASCII and checks that the exit count equals the number of "debug warps" lines.
func TestLive(t *testing.T) {
	sock := os.Getenv("D2D_SOCK")
	if sock == "" {
		t.Skip("D2D_SOCK not set")
	}
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 10 * time.Second})
	defer c.Close()
	w := New(Client{Client: c}, Options{})
	st, err := w.Update()
	if err != nil {
		t.Fatal(err)
	}
	lm := w.Current()
	t.Logf("level %d pos %v rect %+v known %d note %q", st.LevelNo, st.Pos, lm.Rect, lm.Grid.KnownCount(), w.LastGridNote)
	t.Log("\n" + ASCII(lm, 160, 60))
	ws, err := c.Warps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("borders: %s", w.LastBorderNote)
	for _, a := range lm.Adjacent {
		t.Logf("adjacent %d contact %+v cells %d", a.LevelNo, a.Contact, len(lm.BorderCells(a.LevelNo)))
	}
	if len(lm.Exits) != len(ws.Items) {
		t.Fatalf("exits %d, debug warps %d", len(lm.Exits), len(ws.Items))
	}
}
