// Package levelsim simulates Act 1 Normal leveling strategies for the d.idle bot:
// a per-area kill-rate model fitted on logged first-hour runs, a policy runner
// (today's plan is one policy) and a beam-search optimiser. docs/levelsim.md has
// the model, the tags and the limits.
package levelsim

import (
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
)

//go:embed data/areas.csv data/graph.json data/runs.json
var dataFS embed.FS

// expTable[l] is the total experience to reach level l (internal/telemetry/exp.go,
// Experience.txt Amazon column; the same thresholds as docs/research/xp-gold-trace.md Q6).
var expTable = [...]float64{0, 0, 500, 1500, 3750, 7875, 14175, 22680, 32886, 44396, 57715,
	72144, 90180, 112725, 140906, 176132, 220165, 275207, 344008, 430010, 537513,
	671891, 839864, 1049830, 1312287, 1640359, 2050449, 2563061, 3203826, 3902260, 4663553}

// MaxLevel is the highest level the table knows.
const MaxLevel = len(expTable) - 1

// XPForLevel returns the total XP to reach level l.
func XPForLevel(l int) float64 {
	if l < 1 {
		return 0
	}
	if l > MaxLevel {
		l = MaxLevel
	}
	return expTable[l]
}

// LevelFor returns the character level for a total XP.
func LevelFor(xp float64) int {
	l := 1
	for l < MaxLevel && xp >= expTable[l+1] {
		l++
	}
	return l
}

// Area is one row of data/areas.csv (derived from docs/research/areamon-table.csv).
type Area struct {
	ID        int
	Name      string
	Units     float64 // monsters per full clear (mean of 12 seeds, V run)
	MLvl      float64 // mean monster level
	Rate      float64 // fresh kills per minute
	RateSrc   string  // "obs" (a logged hour) or "areamon" (the I time model)
	ClearMin  float64
	XK        [6]float64 // XP per kill at clvl 1,3,5,8,12,16 (V run)
	UniqFloor int        // minimum safe clvl of the boss met here (boss1-trace, I); 0 = none
	PlanEnter int
	PlanMelee int
	PlanLeave int
	PlanFarm  bool
}

var xkAnchor = [6]float64{1, 3, 5, 8, 12, 16}

// XPPerKill is the mean XP of one kill (all units) at character level clvl.
func (a *Area) XPPerKill(clvl int) float64 {
	c := float64(clvl)
	if c <= xkAnchor[0] {
		return a.XK[0]
	}
	for i := 1; i < 6; i++ {
		if c <= xkAnchor[i] {
			f := (c - xkAnchor[i-1]) / (xkAnchor[i] - xkAnchor[i-1])
			return a.XK[i-1] + f*(a.XK[i]-a.XK[i-1])
		}
	}
	// beyond 16: continue the last slope, never below zero
	slope := (a.XK[5] - a.XK[4]) / 4
	return math.Max(0, a.XK[5]+slope*(c-16))
}

// Edge is a walk between neighbouring areas.
type Edge struct {
	A, B int
	Min  float64
}

// Graph is the area graph (data/graph.json).
type Graph struct {
	Edges     [][3]float64 `json:"edges"`
	Waypoints []int        `json:"waypoints"`
}

// RunArrival is one logged area entry.
type RunArrival struct {
	Min  float64
	Area int
}

// Run is one logged first-hour run reduced to derived rows.
type Run struct {
	ID       int     `json:"id"`
	EndMin   float64 `json:"end_min"`
	Observed struct {
		Clvl  int     `json:"clvl"`
		XP    float64 `json:"xp"`
		Kills float64 `json:"kills"`
	} `json:"observed"`
	Levelups map[string]float64 `json:"levelups"`
	Minutes  [][]float64        `json:"minutes"` // minute, area, clvl, xp, kills
	Arrivals [][]float64        `json:"arrivals"`
	Restarts []float64          `json:"restarts"`
	Dead     [][]float64        `json:"dead"`
	Build    string             `json:"build"`
	Tag      string             `json:"tag"`
}

type runsFile struct {
	Runs []Run `json:"runs"`
}

// Data is everything the simulator reads.
type Data struct {
	Areas map[int]*Area
	Order []int // area ids in file order
	Graph Graph
	Runs  []Run
}

// Load reads the embedded tables.
func Load() (*Data, error) {
	d := &Data{Areas: map[int]*Area{}}
	f, err := dataFS.Open("data/areas.csv")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := d.readAreas(f); err != nil {
		return nil, err
	}
	b, err := dataFS.ReadFile("data/graph.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &d.Graph); err != nil {
		return nil, fmt.Errorf("graph.json: %w", err)
	}
	b, err = dataFS.ReadFile("data/runs.json")
	if err != nil {
		return nil, err
	}
	var rf runsFile
	if err := json.Unmarshal(b, &rf); err != nil {
		return nil, fmt.Errorf("runs.json: %w", err)
	}
	d.Runs = rf.Runs
	return d, nil
}

func (d *Data) readAreas(r io.Reader) error {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return err
	}
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) != 18 {
			return fmt.Errorf("areas.csv line %d: %d columns", i+1, len(row))
		}
		num := func(c int) float64 {
			v, e := strconv.ParseFloat(row[c], 64)
			if e != nil && err == nil {
				err = fmt.Errorf("areas.csv line %d column %d: %v", i+1, c, e)
			}
			return v
		}
		a := &Area{Name: row[1], RateSrc: row[5]}
		a.ID = int(num(0))
		a.Units, a.MLvl, a.Rate, a.ClearMin = num(2), num(3), num(4), num(6)
		for k := 0; k < 6; k++ {
			a.XK[k] = num(7 + k)
		}
		a.UniqFloor, a.PlanEnter, a.PlanMelee, a.PlanLeave = int(num(13)), int(num(14)), int(num(15)), int(num(16))
		a.PlanFarm = row[17] == "1"
		d.Areas[a.ID] = a
		d.Order = append(d.Order, a.ID)
	}
	return err
}
