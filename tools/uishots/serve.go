package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/watch"
)

// serve runs the watch view on a frozen simulated runner (speed 0, so the clock and every list hold still) plus
// fixture answers for the routes the sim does not have: a game frame, the minimap, gear, tally and currency. Everything is
// deterministic, so two runs of the same tree give the same pictures.
func serve(addr, class string) error {
	sim, err := watch.NewSim(class, 0)
	if err != nil {
		return err
	}
	sim.Hub.SetFrameURL("/stream")
	sim.Step(30 * time.Second) // one step so the hero, the map and the feed have something, then the clock stays
	if _, err := sim.Hub.GiveOrder(watch.OrderGo, "Cold Plains", 30); err != nil {
		log.Printf("uishots: fixture order: %v", err)
	}
	sim.Hub.Whyline().Observe(goals.Why{Code: "fight", Params: map[string]string{"area": "Blood Moor", "life": "72"}}, "Fighting in Blood Moor", "g1", "")
	sim.Hub.SetGoldSource(func() (int, int, bool) { return 1240, 5300, true })
	view := sim.Hub.ViewHandler()
	m := http.NewServeMux()
	m.Handle("/", view)
	// no live event stream: headless Chrome's virtual time waits for the network to go idle, and an open SSE request
	// never does; the pages fall back to their polls
	m.HandleFunc("/api/events", http.NotFound)
	m.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(frame())
	})
	m.HandleFunc("/api/repair", func(w http.ResponseWriter, r *http.Request) { js(w, repairFixture) })
	m.HandleFunc("/api/tally", func(w http.ResponseWriter, r *http.Request) { js(w, tallyFixture) })
	if b, err := os.ReadFile("internal/watch/testdata/currency-" + class + ".json"); err == nil {
		m.HandleFunc("/api/currency", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write(b)
		})
	}
	m.HandleFunc("/api/senses", func(w http.ResponseWriter, r *http.Request) { js(w, sensesFixture()) })
	// the unit layer script of the map lives behind the senses routes; a Senses with no source still serves it
	units := (&watch.Senses{}).Handler()
	m.Handle("/watch/senses_units.js", units)
	log.Printf("uishots: fixture viewer on http://%s/watch (%s, frozen)", addr, class)
	return http.ListenAndServe(addr, m)
}

func js(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

// frame is a fixed 800x600 picture standing in for the game window: a dark field with a lit diamond.
func frame() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			d := abs(x-400) + abs(y-300)
			c := color.RGBA{20, 16, 12, 255}
			if d < 180 {
				v := uint8(140 - d/2)
				c = color.RGBA{v, v * 3 / 4, v / 3, 255}
			}
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	jpeg.Encode(&b, img, &jpeg.Options{Quality: 80})
	return b.Bytes()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// sensesFixture is an /api/senses reply: a 48x32 area of rooms and corridors, an exit, a waypoint, the hero and a
// few units (tier 2 open), so the map panel draws something recognisable.
func sensesFixture() map[string]any {
	const w, h = 48, 32
	cells := make([]int, w*h)
	room := func(x0, y0, x1, y1 int) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				cells[y*w+x] = 1
			}
		}
	}
	room(2, 2, 14, 10)
	room(14, 5, 30, 7)
	room(30, 2, 44, 14)
	room(36, 14, 38, 26)
	room(20, 20, 44, 29)
	room(4, 18, 16, 28)
	room(10, 10, 12, 18)
	for i, c := range cells { // walls around the floor
		if c != 0 {
			continue
		}
		x, y := i%w, i/w
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+d[0], y+d[1]
			if nx >= 0 && ny >= 0 && nx < w && ny < h && cells[ny*w+nx] == 1 {
				cells[i] = 2
				break
			}
		}
	}
	var rle []int
	for i := 0; i < len(cells); {
		j := i
		for j < len(cells) && cells[j] == cells[i] {
			j++
		}
		rle = append(rle, cells[i], j-i)
		i = j
	}
	return map[string]any{
		"schema": "watch.senses/1", "tiers_unlocked": []int{1, 2}, "version": 1,
		"map": map[string]any{"schema": "watch.senses/1", "areaId": 2, "area": "Blood Moor", "version": 1, "x0": 0, "y0": 0, "step": 1,
			"w": w, "h": h, "rle": rle, "exits": []any{map[string]int{"dest": 3, "x": 43, "y": 8}}, "waypoints": []any{map[string]int{"x": 8, "y": 6}}},
		"hero": map[string]int{"x": 24, "y": 6},
		"units": []any{
			map[string]any{"id": 1, "kind": "monster", "x": 34, "y": 6},
			map[string]any{"id": 2, "kind": "monster", "x": 36, "y": 9, "elite": "champion", "name": "Fallen Shaman"},
			map[string]any{"id": 3, "kind": "merc", "x": 22, "y": 6},
			map[string]any{"id": 4, "kind": "npc", "x": 6, "y": 4},
		},
	}
}

// Fixture replies in the shapes of /api/repair (docs/repair.md) and /api/tally (docs/tallystrip.md).
var (
	repairFixture = map[string]any{"mode": "manual", "estimate": 180, "worst": "Short Staff is broken.", "gear": []any{
		map[string]any{"slot": "weapon", "name": "Short Staff", "dur": 0, "max": 18, "pct": 0, "broken": true},
		map[string]any{"slot": "torso", "name": "Quilted Armor", "dur": 5, "max": 24, "pct": 21, "warn": true},
		map[string]any{"slot": "head", "name": "Cap", "dur": 8, "max": 12, "pct": 67},
		map[string]any{"slot": "boots", "name": "Boots", "dur": 10, "max": 12, "pct": 83},
	}}
	tallyFixture = map[string]any{"schema": "watch.tally/1", "enabled": true, "mode": "didle", "strip": "Tally 42 banked", "balance": 42,
		"recent": []any{}, "expedition": map[string]any{"open": true, "credited": 6, "spent": 0, "net": 6, "lines": []any{}},
		"next": []any{map[string]any{"id": "loot_markers", "name": "Loot markers on the map", "cost": 40, "affordable": true, "wired": true}}}
)
