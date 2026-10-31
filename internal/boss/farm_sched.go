package boss

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/plandata"
)

// The farm run table and the scheduler that picks the best run (docs/farm.md). Data: plandata farm-runs.json for the
// scores and runs.json for the clvl floors and acts. Every number is U except the xp points (V arithmetic).

// Goals a scheduler can rank by.
const (
	GoalXP      = "xp"
	GoalRunes   = "runes"
	GoalUniques = "uniques"
	GoalAuto    = "auto" // same as xp until a gold/revive gate exists (U)
)

var diffKeys = [3]string{"n", "nm", "h"}

// Point is one [clvl, xp per kill] sample.
type Point [2]int

// FarmRun is one row of the farm table.
type FarmRun struct {
	Name      string                        `json:"name"`
	Steps     []string                      `json:"steps"` // KillBoss names, in order, inside one game
	Secs      int                           `json:"secs"`  // one run, town to town (U estimate)
	TPMult    float64                       `json:"tp_mult"`
	Needs     []string                      `json:"needs"`
	MinClvl   map[string]int                `json:"min_clvl"`
	XP        map[string][]Point            `json:"xp"`
	Drops     map[string]float64            `json:"drops"`
	ClassMult map[string]map[string]float64 `json:"class_mult"`
	Act       int                           `json:"-"`
}

// FarmTable is the loaded table in file order.
type FarmTable []FarmRun

// LoadFarmTable reads farm-runs.json and runs.json from fsys (plandata.FS in the bot; a dir FS in tests).
func LoadFarmTable(fsys fs.FS) (FarmTable, error) {
	var farm struct {
		Runs []FarmRun `json:"runs"`
	}
	if err := readJSON(fsys, "farm-runs.json", &farm); err != nil {
		return nil, err
	}
	var base struct {
		Runs []struct {
			Name    string         `json:"name"`
			Act     int            `json:"act"`
			MinClvl map[string]int `json:"min_clvl"`
		} `json:"runs"`
	}
	if err := readJSON(fsys, "runs.json", &base); err != nil {
		return nil, err
	}
	for i := range farm.Runs {
		r := &farm.Runs[i]
		if r.Name == "" || len(r.Steps) == 0 || r.Secs <= 0 {
			return nil, fmt.Errorf("farm-runs.json: run %d needs name, steps and secs", i)
		}
		for _, b := range base.Runs {
			if !strings.EqualFold(b.Name, r.Name) {
				continue
			}
			r.Act = b.Act
			if r.MinClvl == nil {
				r.MinClvl = b.MinClvl
			}
		}
		if r.TPMult <= 0 {
			r.TPMult = 1
		}
	}
	return farm.Runs, nil
}

// DefaultFarmTable loads the embedded table.
func DefaultFarmTable() (FarmTable, error) { return LoadFarmTable(plandata.FS) }

func readJSON(fsys fs.FS, name string, v any) error {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Find looks a run up by name, ignoring case, spaces and "+" ("eldritch shenk" finds "Eldritch+Shenk").
func (t FarmTable) Find(name string) (FarmRun, bool) {
	k := runKey(name)
	for _, r := range t {
		if runKey(r.Name) == k {
			return r, true
		}
	}
	return FarmRun{}, false
}

func runKey(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '+' || r == '_' || r == '-' {
			return -1
		}
		return r
	}, strings.ToLower(s))
}

// Floor is the clvl floor of the run in difficulty d (0 Normal, 1 Nightmare, 2 Hell); 0 = none.
func (r FarmRun) Floor(d int) int { return r.MinClvl[diffKeys[clampDiff(d)]] }

// ExpectedXP is the experience of one run at clvl in difficulty d: linear interpolation of the table points. Below
// the first point it is 0 (the run is not an XP run there), above the last the last value holds (U).
func (r FarmRun) ExpectedXP(d, clvl int) float64 {
	pts := r.XP[diffKeys[clampDiff(d)]]
	if len(pts) == 0 || clvl < pts[0][0] {
		return 0
	}
	for i := 1; i < len(pts); i++ {
		if clvl <= pts[i][0] {
			a, b := pts[i-1], pts[i]
			if b[0] == a[0] {
				return float64(b[1])
			}
			f := float64(clvl-a[0]) / float64(b[0]-a[0])
			return float64(a[1]) + f*float64(b[1]-a[1])
		}
	}
	return float64(pts[len(pts)-1][1])
}

// FarmChar is what the scheduler needs to know about the character.
type FarmChar struct {
	Level      int
	Class      string // "Sorceress", ... as the class table spells it
	Difficulty int
	Teleport   bool
	Has        map[string]bool // unlock flags: anya_portal ...
}

// Choice is a ranked run.
type Choice struct {
	Run   FarmRun
	Score float64 // value per hour, in the goal's unit (xp per hour, or drop weight per hour)
}

// Rank lists the runs that are allowed for c, best first. goal is GoalXP, GoalRunes, GoalUniques or GoalAuto; skip
// names runs to leave out (failed ones). A run is dropped when the clvl floor, a needed flag, or the goal's value
// (zero xp or zero drop weight) rules it out.
func (t FarmTable) Rank(c FarmChar, goal string, skip map[string]bool) []Choice {
	var out []Choice
	for _, r := range t {
		if skip[r.Name] || c.Level < r.Floor(c.Difficulty) || !hasAll(c.Has, r.Needs) {
			continue
		}
		var value float64
		switch goal {
		case "", GoalAuto, GoalXP:
			value = r.ExpectedXP(c.Difficulty, c.Level)
		default:
			value = r.Drops[goal]
		}
		if value <= 0 {
			continue
		}
		secs := float64(r.Secs)
		if !c.Teleport {
			secs /= r.TPMult
		}
		if m, ok := r.ClassMult[c.Class][diffKeys[clampDiff(c.Difficulty)]]; ok {
			secs /= m
		}
		out = append(out, Choice{Run: r, Score: value * 3600 / secs})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// Pick is the best run, or false when none is allowed.
func (t FarmTable) Pick(c FarmChar, goal string, skip map[string]bool) (FarmRun, bool) {
	rk := t.Rank(c, goal, skip)
	if len(rk) == 0 {
		return FarmRun{}, false
	}
	return rk[0].Run, true
}

func hasAll(have map[string]bool, need []string) bool {
	for _, n := range need {
		if !have[n] {
			return false
		}
	}
	return true
}

// Limits are the stop conditions of a farm loop; zero means no limit. At least one of Runs and Duration should be
// set, or the loop only stops on Deaths, ConsecFails, an empty ranking or ctx.
type Limits struct {
	Runs        int           // completed or failed runs started
	Duration    time.Duration // wall time; checked between runs
	Deaths      int
	ConsecFails int // consecutive runs without a kill (default 4)
}

func (l Limits) consec() int {
	if l.ConsecFails <= 0 {
		return 4
	}
	return l.ConsecFails
}
