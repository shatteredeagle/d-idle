package bot

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/logx"
)

// logFighter stands in for combat.Fighter.fight's logging: it writes the real FightTelemetry line through the
// runner's logger exactly as Fighter.info does ("combat " + line) and keeps the sums for the assertions.
type logFighter struct {
	Fighter
	log    *slog.Logger
	fights int
	kills  int
	sec    int
}

func (f *logFighter) Clear(ctx context.Context, o ClearOpts) (ClearResult, error) {
	res, err := f.Fighter.Clear(ctx, o)
	ft := combat.FightTelemetry{Area: 2, Diff: 0, Danger: 1, MinLifePct: 100}
	// Deliberately the same values every call: identical consecutive fight lines must never be folded.
	ft.Kills = 3
	ft.Potions = 1
	f.fights++
	f.kills += 3
	f.sec += 7
	f.log.Info("combat " + ft.Line(7*time.Second))
	return res, err
}

func TestE2ETelemetryChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bot.log")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// As cmd/d2bot newLogger builds it.
	h := logx.NewDedup(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo}), 0)
	log := slog.New(h).With("inst", "i0")

	var lf *logFighter
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		lf = &logFighter{Fighter: d.Fighter, log: log}
		d.Fighter = lf
	})
	g.e.Log = log
	g.mover.exploreFn = blockExplore
	g.w.set(func(w *world) { w.level = 3; w.area = 2; w.x, w.y = 300, 310 })
	g.run()
	g.r.Start(farm("Blood Moor"))
	g.waitFor("in field", func() bool { return g.w.count("explore") > 0 })

	// a few fights (identical lines in a row)
	for i := 0; i < 4; i++ {
		g.w.addMonster(uint32(10+i), 10, 5, 0)
		n := i + 1
		g.waitFor("fight", func() bool { return g.w.count("clear") >= n })
	}
	// the fighter logs its line after Clear returns: let the last one land before the nav exit run, or it can split that run
	g.waitFor("fight lines", func() bool { return strings.Count(readFile(t, path), "combat fight area=") >= 4 })
	// repeated nav exit lines, as a tight retry loop writes them
	for i := 0; i < 500; i++ {
		log.Info("nav exit", "to", 3, "gate", "10,10", "push", 2, "border", false)
	}
	minutes := func() int { return strings.Count(readFile(t, path), `"msg":"minute"`) }
	g.waitFor("minute lines", func() bool { g.clock.Sleep(30 * time.Second); return minutes() >= 2 })

	g.w.addMonster(99, 10, 5, 0)
	g.waitFor("seen", func() bool { time.Sleep(3 * time.Millisecond); return true })
	die(g)
	g.waitFor("death event", func() bool { return len(g.events(goals.EvDeath)) == 1 })
	respawn(g)
	base := minutes()
	g.waitFor("minute after death", func() bool { g.clock.Sleep(30 * time.Second); return minutes() >= base+2 })
	g.finish()
	h.Flush()
	file.Close()

	out := readFile(t, path)
	if n := strings.Count(out, `"msg":"nav exit"`); n != 1 {
		t.Errorf("nav exit lines written %d, want 1 folded run", n)
	}
	if !strings.Contains(out, "last line repeated") && !strings.Contains(out, "repeated") {
		t.Errorf("no counter line for the folded nav exit run")
	}
	if n := strings.Count(out, "combat fight area="); n != lf.fights || n < 4 {
		t.Errorf("fight lines %d, fights run %d (Dedup folded or lost some)", n, lf.fights)
	}
	if n := strings.Count(out, "death diff="); n != 1 {
		t.Errorf("death lines %d, want 1", n)
	}
	if !strings.Contains(out, " build=") {
		t.Errorf("death line lacks build=")
	}
	if lf.kills != lf.fights*3 || lf.sec != lf.fights*7 {
		t.Errorf("fake sums off: %+v", lf)
	}
	t.Logf("fights=%d kills=%d sec=%d", lf.fights, lf.kills, lf.sec)
	if p := os.Getenv("D2E2E_KEEP"); p != "" {
		_ = os.WriteFile(p, []byte(out), 0o644)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
