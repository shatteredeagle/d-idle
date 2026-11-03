package nav

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dworld"
	"d2lod-bot/internal/env"
)

// replay serves d2d replies recorded on the worker host from a real d2d, seed 17 (docs/navexit22.md): the Rogue
// Encampment (level 1) and the Blood Moor (level 2) of that seed, whose layout puts the Blood Moor WEST of the camp.
type replay struct {
	dir     string
	level   int
	blocked map[int]map[[2]int]bool // level -> (tile x*2, tile row) -> blocked, from the recorded probes
}

func (r *replay) file(name string) []string {
	b, err := os.ReadFile(filepath.Join(r.dir, fmt.Sprintf("L%d-%s", r.level, name)))
	if err != nil {
		if gz, gerr := os.Open(filepath.Join(r.dir, fmt.Sprintf("L%d-%s.gz", r.level, name))); gerr == nil {
			defer gz.Close()
			zr, _ := gzip.NewReader(gz)
			var out []string
			sc := bufio.NewScanner(zr)
			sc.Buffer(nil, 1<<20)
			for sc.Scan() {
				out = append(out, sc.Text())
			}
			return out
		}
		panic(err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func (r *replay) Level(context.Context) (d2dctl.LevelInfo, error) {
	var li d2dctl.LevelInfo
	f := strings.Fields(r.file("debug_level.txt")[0])
	_, err := fmt.Sscanf(strings.Join(f[1:], " "), "%d %g %g %d %d %d %d", &li.ID, &li.X, &li.Y, &li.W, &li.H, &li.OriginX, &li.OriginY)
	return li, err
}

func (r *replay) Grid(context.Context, int) ([]string, error) {
	var rows []string
	for _, l := range r.file("debug_grid_5.txt") {
		if l != "ok" {
			rows = append(rows, l)
		}
	}
	return rows, nil
}

func replayList[T any](lines []string, parse func(string) (T, error)) (d2dctl.List[T], error) {
	var l d2dctl.List[T]
	for _, x := range lines {
		if x == "ok" || strings.HasPrefix(x, "room") {
			continue
		}
		v, err := parse(x)
		if err != nil {
			return l, err
		}
		l.Items = append(l.Items, v)
	}
	return l, nil
}

func (r *replay) Warps(context.Context) (d2dctl.List[d2dctl.Warp], error) {
	return replayList(r.file("debug_warps.txt"), d2dctl.ParseWarp)
}
func (r *replay) Presets(context.Context) (d2dctl.List[d2dctl.Preset], error) {
	return replayList(r.file("debug_presets.txt"), d2dctl.ParsePreset)
}
func (r *replay) NPCs(context.Context) (d2dctl.List[d2dctl.NPC], error) {
	return d2dctl.List[d2dctl.NPC]{}, nil
}

func ints(lines []string, n int) [][]int {
	var out [][]int
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) != n {
			continue
		}
		v := make([]int, n)
		ok := true
		for i := range f {
			if _, err := fmt.Sscanf(f[i], "%d", &v[i]); err != nil {
				ok = false
			}
		}
		if ok {
			out = append(out, v)
		}
	}
	return out
}

func (r *replay) Layout(context.Context) ([]d2dworld.Placement, error) {
	var out []d2dworld.Placement
	for _, v := range ints(r.file("debug_layout.txt"), 5) {
		out = append(out, d2dworld.Placement{Level: v[0], X: v[1], Y: v[2], W: v[3], H: v[4]})
	}
	return out, nil
}

func (r *replay) Nearby(context.Context) ([]d2dworld.Near, error) {
	var out []d2dworld.Near
	for _, v := range ints(r.file("debug_nearby.txt"), 3) {
		out = append(out, d2dworld.Near{Level: v[0], DX: v[1], DY: v[2]})
	}
	return out, nil
}

// Blocked answers from the recorded probes (one per tile, columns .5 of a tile); a point never recorded is an error,
// which the world reads as closed.
func (r *replay) Blocked(_ context.Context, x, y float64) (bool, error) {
	if r.blocked == nil {
		r.blocked = map[int]map[[2]int]bool{}
	}
	if r.blocked[r.level] == nil {
		m := map[[2]int]bool{}
		for _, l := range r.file("blocked.txt") {
			var cx float64
			var cy int
			var ok string
			var v int
			if _, err := fmt.Sscanf(l, "%g %d %s %d", &cx, &cy, &ok, &v); err == nil {
				m[[2]int{int(math.Floor(cx)), cy}] = v != 0
			}
		}
		r.blocked[r.level] = m
	}
	b, known := r.blocked[r.level][[2]int{int(math.Floor(x)), int(math.Floor(y))}]
	if !known {
		return true, fmt.Errorf("not recorded: %g %g", x, y)
	}
	return b, nil
}

// townGate loads the camp then the Blood Moor the way the bot walks (arrive in town, then cross) and returns the
// navigator with the world standing in the Blood Moor.
func seed17World(t *testing.T) (*Navigator, *d2dworld.World) {
	t.Helper()
	r := &replay{dir: filepath.Join("..", "d2dworld", "testdata", "seed17"), level: 1}
	w := d2dworld.New(r, d2dworld.Options{})
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	r.level = 2
	if _, err := w.Update(); err != nil {
		t.Fatal(err)
	}
	n := New(&env.Env{})
	n.UseWorld(w)
	return n, w
}

// The Blood Moor of seed 17 has the camp on its east edge. The camp's wall holds a 13-subtile gap there, and the
// gate toward level 1 must survive the check against the camp's grid.
func TestSeed17BloodMoorHasTownGate(t *testing.T) {
	n, w := seed17World(t)
	lm := w.Current()
	t.Logf("adjacent: %d entries", len(lm.Adjacent))
	for _, a := range lm.Adjacent {
		t.Logf("level %d contact %+v rect %+v", a.LevelNo, a.Contact, a.Rect)
	}
	t.Logf("border cells to 1: %d", len(lm.BorderCells(1)))
	gates := n.borderGates(lm, 1)
	t.Logf("gates: %+v", gates)
	if len(gates) == 0 {
		t.Fatal("no gate from the Blood Moor to the Rogue Encampment")
	}
}
