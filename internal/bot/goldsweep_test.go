package bot

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
)

// pileMover takes a ground pile when walked onto, like the game does (VERIFIED LIVE for the corpse pile).
type pileMover struct {
	*fakeMover
	amount int
}

func (m *pileMover) GoTo(ctx context.Context, p input.Point) error {
	m.w.call("goto %d,%d", p.X, p.Y)
	m.w.set(func(w *world) {
		var keep []d2.Unit
		for _, u := range w.ground {
			if u.X == p.X && u.Y == p.Y && u.Class == goldClass {
				w.gold0 += m.amount
				continue
			}
			keep = append(keep, u)
		}
		w.ground = keep
	})
	return nil
}

func sweepRig(t *testing.T) (*rig, *bytes.Buffer, string) {
	g, buf, dir := repeatRig(t)
	g.r.d.Mover = &pileMover{fakeMover: g.mover, amount: 500}
	g.w.set(func(w *world) { w.level, w.area, w.gold0 = 10, 1, 0 })
	g.r.poll()
	return g, buf, dir
}

func sweepRec(g *rig, carried int) *recovery {
	d := DeathRecord{Time: g.clock.Now(), Area: 2, AreaName: "Blood Moor", Pos: input.Point{X: 210, Y: 205}, Level: 10,
		Autopsy: &Autopsy{GoldCarried: carried}}
	g.r.obMu.Lock()
	g.r.dlog.add(d)
	g.r.obMu.Unlock()
	return &recovery{rec: d, area: 2, pos: d.Pos, decided: true}
}

func control1(g *rig) control {
	return control{goal: &goals.Goal{}, gen: g.r.gen}
}

func TestExpectedPile(t *testing.T) {
	cases := []struct{ lvl, gold, want int }{{10, 1100, 990}, {30, 1000, 800}, {1, 0, 0}, {5, 200, 190}}
	for _, c := range cases {
		if got := expectedPile(DeathRecord{Level: c.lvl, Autopsy: &Autopsy{GoldCarried: c.gold}}); got != c.want {
			t.Errorf("lvl %d gold %d: %d want %d", c.lvl, c.gold, got, c.want)
		}
	}
	if expectedPile(DeathRecord{Level: 10}) != 0 {
		t.Error("no autopsy must be 0")
	}
}

func TestGoldSweepPicksPileAfterAbandonedRun(t *testing.T) {
	g, buf, dir := sweepRig(t)
	g.w.set(func(w *world) { w.ground = []d2.Unit{{Class: goldClass, X: 220, Y: 210}} })
	rec := sweepRec(g, 1100) // expected pile 990, corpse run abandoned (retrieve false)
	rec.why = "corpse worth about 990 gold, threshold 1000"
	g.r.sweepDeathGold(context.Background(), control1(g), rec)
	if g.w.count("back 2") != 1 || g.w.count("goto 220,210") != 1 {
		t.Fatalf("calls: %s", g.w.log())
	}
	if !strings.Contains(buf.String(), "death: gold sweep area=2 picked=+500 gold (carried 0 -> 500)") {
		t.Fatalf("log:\n%s", buf.String())
	}
	au := g.r.dlog.Records[0].Autopsy
	if au.GoldRecovered == nil || *au.GoldRecovered != 500 {
		t.Fatalf("goldRecovered %v", au.GoldRecovered)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "deaths.jsonl"))
	if !strings.Contains(string(b), `"followUp":true`) || !strings.Contains(string(b), `"goldRecovered":500`) {
		t.Fatalf("jsonl: %s", b)
	}
}

func TestGoldSweepAfterCorpseRunStaysInField(t *testing.T) {
	g, _, _ := sweepRig(t)
	g.w.set(func(w *world) { w.area = 2; w.ground = []d2.Unit{{Class: goldClass, X: 222, Y: 210}} })
	g.r.poll()
	rec := sweepRec(g, 1100)
	rec.retrieve = true
	g.r.sweepDeathGold(context.Background(), control1(g), rec)
	if g.w.count("back") != 0 || g.w.count("goto 222,210") != 1 {
		t.Fatalf("calls: %s", g.w.log())
	}
}

func TestGoldSweepSkips(t *testing.T) {
	cases := map[string]func(g *rig, rec *recovery){
		"banned (repeat site or spiral)": func(g *rig, _ *recovery) {
			g.r.obMu.Lock()
			g.r.skipAreaLocked(2, 1e12, whyRepeat)
			g.r.obMu.Unlock()
		},
		"small purse":            func(_ *rig, rec *recovery) { rec.rec.Autopsy.GoldCarried = 100 },
		"second death":           func(_ *rig, rec *recovery) { rec.tries = 1 << 20 },
		"boss":                   func(_ *rig, rec *recovery) { rec.rec.KillerBoss = true },
		"pack still at the spot": func(_ *rig, rec *recovery) { rec.rec.Autopsy.HostilesNear = 24 },
		"failed corpse run": func(_ *rig, rec *recovery) {
			rec.retrieve, rec.why = true, whyRetrievalFailed
		},
	}
	for name, mut := range cases {
		t.Run(name, func(t *testing.T) {
			g, buf, _ := sweepRig(t)
			g.w.set(func(w *world) { w.ground = []d2.Unit{{Class: goldClass, X: 220, Y: 210}} })
			rec := sweepRec(g, 1100)
			mut(g, rec)
			g.r.sweepDeathGold(context.Background(), control1(g), rec)
			if g.w.count("back") != 0 || g.w.count("goto") != 0 {
				t.Fatalf("swept: %s\n%s", g.w.log(), buf.String())
			}
		})
	}
}

func TestCarriedGoldFeedsCorpseValue(t *testing.T) {
	g, _, _ := repeatRig(t)
	g.w.set(func(w *world) { w.level, w.area = 10, 1 })
	s, _ := g.w.snapshot()
	mk := func(carried int) *recovery {
		d := DeathRecord{Time: g.clock.Now(), Area: 2, AreaName: "Blood Moor", Level: 10, CorpseValue: 100, Autopsy: &Autopsy{GoldCarried: carried}}
		return &recovery{rec: d, area: 2}
	}
	if ok, why := g.r.retrievalDecision(mk(0), s); ok {
		t.Fatalf("cheap gear, empty purse must not run: %s", why)
	}
	if ok, why := g.r.retrievalDecision(mk(5000), s); !ok || !strings.Contains(why, "4600") {
		t.Fatalf("ok=%v why=%s", ok, why)
	}
}
