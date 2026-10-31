package nav

import (
	"os"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

type fixedState struct{ snap *d2.Snapshot }

func (f fixedState) Snapshot() (*d2.Snapshot, error) { return f.snap, nil }

// Recorded Rogue Encampments of two different games (different seeds: different origin and even a mirrored layout,
// the Blood Moor is west of the town in one and east in the other). Tests must derive every position from the
// loaded level data, never hardcode a coordinate.
var townFixtures = []string{"rogue_encampment.d2mi", "rogue_encampment_b.d2mi"}

func rogueNav(t *testing.T, file string) (*Navigator, *obs) {
	t.Helper()
	fh, err := os.Open("../d2map/testdata/" + file)
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	img, err := d2map.LoadImage(fh)
	if err != nil {
		t.Fatal(err)
	}
	tr := d2map.NewTracker(img)
	st, err := tr.Update()
	if err != nil {
		t.Fatal(err)
	}
	tune := env.DefaultTuning()
	fk := input.NewFake(input.Point{X: st.Pos.X, Y: st.Pos.Y})
	n := New(&env.Env{In: fk, Tune: &tune, Clock: env.RealClock{}})
	n.w = tr
	snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{X: st.Pos.X, Y: st.Pos.Y, Area: 1, Mode: 5, Life: 40, MaxLife: 40}}
	n.st = fixedState{snap}
	return n, &obs{snap: snap, state: st, lm: tr.Current(), pos: input.Point{X: st.Pos.X, Y: st.Pos.Y}, level: 1}
}

func presetPos(lm *d2map.LevelMap, kind, id int) (input.Point, bool) {
	for _, p := range lm.Presets {
		if p.Kind == kind && p.ID == id {
			return input.Point{X: p.Pos.X, Y: p.Pos.Y}, true
		}
	}
	return input.Point{}, false
}

func TestRogueEncampmentPaths(t *testing.T) {
	for _, file := range townFixtures {
		n, o := rogueNav(t, file)
		targets := map[string]input.Point{}
		for name, c := range map[string][2]int{"waypoint": {d2map.PresetObject, 119}, "stash": {d2map.PresetObject, 267},
			"Akara": {d2map.PresetNPC, 148}, "Gheed": {d2map.PresetNPC, 147}, "Charsi": {d2map.PresetNPC, 154}, "Kashya": {d2map.PresetNPC, 150}} {
			p, ok := presetPos(o.lm, c[0], c[1])
			if !ok {
				t.Fatalf("%s: no %s preset", file, name)
			}
			targets[name] = p
		}
		g, ok := n.findGate(o, 2)
		if !ok {
			t.Fatalf("%s: no gate", file)
		}
		targets["gate"] = g.at
		for name, to := range targets {
			p, err := n.PathTo(to)
			if err != nil {
				t.Errorf("%s %s: %v", file, name, err)
				continue
			}
			prev := o.pos
			f := n.fieldFor(o.lm)
			for _, q := range p {
				if !f.los(prev.X, prev.Y, q.X, q.Y, 0) {
					t.Errorf("%s %s: %v -> %v is not clear", file, name, prev, q)
				}
				prev = q
			}
		}
	}
}

func TestRogueEncampmentGateIsABorderGap(t *testing.T) {
	for _, file := range townFixtures {
		n, o := rogueNav(t, file)
		g, ok := n.findGate(o, 2)
		if !ok || !g.border {
			t.Fatalf("%s: no border gate to the Blood Moor", file)
		}
		in := false
		for _, c := range o.lm.BorderCells(2) {
			if c.X == g.at.X && c.Y == g.at.Y {
				in = true
			}
		}
		if !in {
			t.Fatalf("%s: gate %v is not a border cell", file, g.at)
		}
		// the push point lies inside the neighbouring level's rectangle side, away from the town
		var adj d2map.Rect
		for _, a := range o.lm.Adjacent {
			if a.LevelNo == 2 {
				adj = a.Rect
			}
		}
		cx, cy := adj.X+adj.W/2, adj.Y+adj.H/2
		if dist(g.push, input.Point{X: cx, Y: cy}) >= dist(g.at, input.Point{X: cx, Y: cy}) {
			t.Fatalf("%s: push %v does not lead toward the Blood Moor from %v", file, g.push, g.at)
		}
	}
}
