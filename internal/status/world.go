package status

import (
	"encoding/json"
	"net/http"
	"sync"
)

// MapView is what the world panel needs from the map reader (internal/d2map).
type MapView interface {
	AreaID() int
	// Bounds is the known grid in game subtiles: x0,y0 inclusive, x1,y1 exclusive.
	Bounds() (x0, y0, x1, y1 int)
	Walkable(x, y int) bool
	Player() (x, y int)
	Units() []MapUnit
	Exits() []MapExit
	// Revision changes whenever the walkable grid or bounds change (new rooms
	// loaded, area switch). It lets the display cache terrain.
	Revision() int
}

type MapUnit struct {
	ID      uint32 `json:"id"`
	Kind    string `json:"kind"` // monster, npc, merc, item, object
	Name    string `json:"name,omitempty"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Dead    bool   `json:"dead,omitempty"`
	Hostile bool   `json:"hostile,omitempty"`
}

type MapExit struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Kind   string `json:"kind"` // exit, waypoint, portal
	Name   string `json:"name,omitempty"`
	ToArea int    `json:"toArea,omitempty"`
}

type WorldData struct {
	Area        int       `json:"area"`
	AreaName    string    `json:"areaName"`
	Rev         int       `json:"rev"`
	X0          int       `json:"x0"`
	Y0          int       `json:"y0"`
	X1          int       `json:"x1"`
	Y1          int       `json:"y1"`
	PX          int       `json:"px"`
	PY          int       `json:"py"`
	Units       []MapUnit `json:"units"`
	Exits       []MapExit `json:"exits"`
	TerrainPath string    `json:"terrainPath"` // relative to the display root
}

type terrain struct {
	Rev  int     `json:"rev"`
	X0   int     `json:"x0"`
	Y0   int     `json:"y0"`
	W    int     `json:"w"`
	H    int     `json:"h"`
	Rows [][]int `json:"rows"` // per row, run lengths alternating blocked, walkable, blocked, ...
}

func NewWorldPanel(v MapView, areaName func(int) string) Panel {
	if areaName == nil {
		areaName = AreaName
	}
	var mu sync.Mutex
	var cache *terrain
	build := func() *terrain {
		mu.Lock()
		defer mu.Unlock()
		rev := v.Revision()
		if cache != nil && cache.Rev == rev {
			return cache
		}
		x0, y0, x1, y1 := v.Bounds()
		t := &terrain{Rev: rev, X0: x0, Y0: y0, W: x1 - x0, H: y1 - y0}
		for y := y0; y < y1; y++ {
			var runs []int
			cur, n := false, 0
			for x := x0; x < x1; x++ {
				w := v.Walkable(x, y)
				if w != cur {
					runs = append(runs, n)
					cur, n = w, 0
				}
				n++
			}
			runs = append(runs, n)
			t.Rows = append(t.Rows, runs)
		}
		cache = t
		return t
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/terrain", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(build())
	})
	return Panel{ID: "world", Title: "World", Order: 40, Every: 500e6, Routes: mux, Snapshot: func() any {
		x0, y0, x1, y1 := v.Bounds()
		px, py := v.Player()
		d := WorldData{Area: v.AreaID(), AreaName: areaName(v.AreaID()), Rev: v.Revision(),
			X0: x0, Y0: y0, X1: x1, Y1: y1, PX: px, PY: py, Units: v.Units(), Exits: v.Exits(),
			TerrainPath: "api/panel/world/terrain"}
		if d.Units == nil {
			d.Units = []MapUnit{}
		}
		if d.Exits == nil {
			d.Exits = []MapExit{}
		}
		return d
	}}
}
