package sim

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
)

// Clock never sleeps: Sleep advances Now.
type Clock struct{ T time.Time }

func (c *Clock) Now() time.Time        { return c.T }
func (c *Clock) Sleep(d time.Duration) { c.T = c.T.Add(d) }

var (
	dbOnce sync.Once
	dbVal  *data.DB
	dbErr  error
)

// RealDB opens the game tables from /tmp/d2mpq (copied from the install, see docs/data.md); the test is
// skipped when they are absent.
func RealDB(t testing.TB) *data.DB {
	t.Helper()
	const inst = "/tmp/d2mpq"
	if _, err := os.Stat(inst + "/d2data.mpq"); err != nil {
		t.Skip("no game tables at " + inst)
	}
	dbOnce.Do(func() {
		dbVal, dbErr = data.Open(inst, filepath.Join(os.TempDir(), "d2lod-town-testcache"))
	})
	if dbErr != nil {
		t.Fatal(dbErr)
	}
	return dbVal
}

// Rig is a ready-to-use test setup.
type Rig struct {
	W      *World
	In     *Input
	Mv     *Mover
	Env    *env.Env
	Events []goals.Event
}

// NewRig builds a world plus an Env wired to it.
func NewRig(t testing.TB) *Rig {
	db := RealDB(t)
	w := NewWorld(db)
	tune := env.DefaultTuning()
	r := &Rig{W: w, In: NewInput(w), Mv: &Mover{W: w}}
	r.Env = &env.Env{ID: "test", Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Data: db, Tune: &tune,
		Clock: &Clock{T: time.Unix(1000, 0)}, In: r.In, Emit: func(e goals.Event) { r.Events = append(r.Events, e) }}
	return r
}
