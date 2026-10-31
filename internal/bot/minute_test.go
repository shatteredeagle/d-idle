package bot

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type lockedBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}
func (l *lockedBuf) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

func TestMinuteLine(t *testing.T) {
	g := newRig(t)
	buf := &lockedBuf{}
	g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
	g.mover.exploreFn = blockExplore
	g.run()
	defer g.finish()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })
	g.waitFor("minute line", func() bool {
		g.clock.Sleep(30 * time.Second)
		return strings.Contains(buf.String(), "msg=minute")
	})
	for _, k := range []string{"area=", "clvl=", "xp=", "kills=", "deaths=", "potions=", "mana=", "potionsUsed=", "killsGame=", "state=", "gold=", "restarts=", "rung=", "goal=", "step="} {
		if !strings.Contains(buf.String(), " "+k) {
			t.Fatalf("minute line lacks %s: %s", k, buf.String())
		}
	}
}
