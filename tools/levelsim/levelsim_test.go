package levelsim

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func load(t *testing.T) *Data {
	t.Helper()
	d, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestThresholds(t *testing.T) {
	cases := map[float64]int{0: 1, 499: 1, 500: 2, 1500: 3, 3750: 4, 7874: 4, 7875: 5, 14175: 6, 57715: 10, 176132: 15}
	for xp, want := range cases {
		if got := LevelFor(xp); got != want {
			t.Errorf("LevelFor(%v) = %d, want %d", xp, got, want)
		}
	}
	for l := 2; l <= MaxLevel; l++ {
		if XPForLevel(l) <= XPForLevel(l-1) {
			t.Fatalf("table not increasing at %d", l)
		}
	}
}

func TestFarmMonotoneAndCapped(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	s := NewState()
	a := d.Areas[2]
	last := 0.0
	e.Farm(&s, a, Exit{MaxMin: 40}, func(_, xp, _ float64) {
		if xp < last {
			t.Fatalf("xp fell: %v after %v", xp, last)
		}
		last = xp
	})
	if s.Kills[2] > a.Units+1e-6 {
		t.Errorf("killed %v of %v units", s.Kills[2], a.Units)
	}
	if s.XP <= 0 || s.Clvl() < 2 {
		t.Errorf("no progress: xp %v clvl %d", s.XP, s.Clvl())
	}
	if s.LvlT[2] <= 0 || s.LvlT[3] < s.LvlT[2] {
		t.Errorf("level times not recorded in order: %v", s.LvlT[:5])
	}
}

func TestRestartResetsSpawnsButKeepsQuests(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	s := NewState()
	e.Travel(&s, 3) // Cold Plains: activates its waypoint on the way
	e.Farm(&s, d.Areas[3], Exit{MinF: 0.3}, nil)
	s.Den = true
	if e.Remaining(&s, d.Areas[3]) > 0.31 {
		t.Fatalf("remaining %v", e.Remaining(&s, d.Areas[3]))
	}
	t0 := s.T
	e.Restart(&s)
	if e.Remaining(&s, d.Areas[3]) != 1 || s.Loc != 1 || !s.Den || s.WP&(1<<3) == 0 {
		t.Errorf("restart state wrong: remaining %v loc %d den %v wp %b", e.Remaining(&s, d.Areas[3]), s.Loc, s.Den, s.WP)
	}
	if got := s.T - t0; math.Abs(got-e.P.RestartMin) > 1e-9 {
		t.Errorf("restart cost %v, want %v", got, e.P.RestartMin)
	}
}

func TestTravelUsesWaypointOnceActive(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	s := NewState()
	first := e.Travel(&s, 4) // walk Town > BM > CP > Stony, activating both waypoints
	s.Loc = 1
	again := e.Travel(&s, 4)
	if again >= first || math.Abs(again-e.P.HopMin) > 1e-9 {
		t.Errorf("waypoint hop %v, first walk %v", again, first)
	}
}

func TestDenQuestFlag(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	s := NewState()
	e.Farm(&s, d.Areas[8], Exit{MinF: 0.5}, nil)
	if s.Den {
		t.Error("Den done at half")
	}
	e.Farm(&s, d.Areas[8], Exit{MinF: 0.01}, nil)
	if !s.Den {
		t.Error("Den not done after a full clear")
	}
}

func TestCalibrationRegression(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModeStrict)
	limit := map[int]float64{1: 0.12, 4: 0.10}
	for _, r := range d.Runs {
		rp := e.ReplayRun(r)
		if rp.Skipped != "" {
			if _, _, ok := e.KillsCheck(r); !ok {
				t.Errorf("run %d: kills x xp/kill misses the observed level band", r.ID)
			}
			continue
		}
		if math.Abs(rp.XPErr) > limit[r.ID] || math.Abs(rp.KillsErr) > limit[r.ID] {
			t.Errorf("run %d: xp err %+.1f%% kills err %+.1f%% over %.0f%%", r.ID, 100*rp.XPErr, 100*rp.KillsErr, 100*limit[r.ID])
		}
		if rp.PredClvl != rp.ObsClvl {
			t.Errorf("run %d: clvl %d, observed %d", r.ID, rp.PredClvl, rp.ObsClvl)
		}
		if rp.HasSeries && rp.MinuteMAE > 0.08 {
			t.Errorf("run %d: per-minute error %.1f%%", r.ID, 100*rp.MinuteMAE)
		}
	}
}

func TestPlanRowsMatchPlandata(t *testing.T) {
	b, err := os.ReadFile("../../internal/plandata/progression-normal.json")
	if err != nil {
		t.Skip("plandata not readable:", err)
	}
	var f struct {
		Normal []struct {
			Area   int  `json:"area"`
			Enter  int  `json:"enter_clvl"`
			Melee  *int `json:"melee_enter_clvl"`
			Leave  int  `json:"leave_clvl"`
			Farm   bool `json:"farm"`
			MeleeM *int `json:"melee_min_clvl"`
		} `json:"normal"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	d := load(t)
	for _, r := range f.Normal {
		a, ok := d.Areas[r.Area]
		if !ok {
			continue
		}
		melee := r.Enter
		if r.Melee != nil {
			melee = *r.Melee
		}
		if r.MeleeM != nil {
			melee = *r.MeleeM
		}
		if a.PlanEnter != r.Enter || a.PlanLeave != r.Leave || a.PlanMelee != melee || a.PlanFarm != r.Farm {
			t.Errorf("%s: data/areas.csv %d/%d/%d/%v, plandata %d/%d/%d/%v (re-derive data/areas.csv)", a.Name, a.PlanEnter, a.PlanMelee, a.PlanLeave, a.PlanFarm, r.Enter, melee, r.Leave, r.Farm)
		}
	}
}

func TestPlanTodayIsOrderedAndLegal(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	pol := PlanToday(d)
	if pol.Rows[0].Area != 2 || pol.Rows[1].Area != 8 || pol.Rows[2].Area != 3 {
		t.Fatalf("rows out of file order: %+v", pol.Rows[:3])
	}
	r := e.RunPolicy(pol, 60)
	if len(r.Violations) != 0 {
		t.Errorf("plan breaks its own floors: %v", r.Violations)
	}
	if r.Clvl() < 5 || r.Clvl() > 7 {
		t.Errorf("plan at 60 min: clvl %d, the logged hours were 5 to 6", r.Clvl())
	}
	if !r.End.Den {
		t.Error("plan skipped the Den quest")
	}
	// the same plan under stricter floors reports the entries below them
	if v := NewEnv(d, DefaultParams(), ModeStrict).RunPolicy(pol, 60).Violations; len(v) == 0 {
		t.Error("strict mode found no violation in a plan that enters Cold Plains at clvl 4")
	}
}

// toy tables: area 2 pays 20 per kill, area 3 pays 40 at the same rate; the optimum never leaves area 3.
func toy() *Data {
	mk := func(id int, name string, xk float64) *Area {
		return &Area{ID: id, Name: name, Units: 1000, MLvl: 1, Rate: 10, RateSrc: "toy", ClearMin: 100, XK: [6]float64{xk, xk, xk, xk, xk, xk}}
	}
	return &Data{
		Areas: map[int]*Area{2: mk(2, "Poor", 20), 3: mk(3, "Rich", 40)},
		Order: []int{2, 3},
		Graph: Graph{Edges: [][3]float64{{1, 2, 0.2}, {1, 3, 0.2}}},
	}
}

func TestOptimiserFindsToyOptimum(t *testing.T) {
	e := NewEnv(toy(), DefaultParams(), ModeBold)
	e.NoDenRequired = true
	b := e.Optimise(30, 20)
	if got := Distinct(b.Steps); len(got) != 1 || got[0] != 3 {
		t.Fatalf("optimum farms %v, want only the rich area", got)
	}
	// 30 minutes minus the walk, at 10 kills a minute with the town tax, 40 xp a kill
	want := (30 - 0.2*0 - 0.4) * (1 - e.P.TownTax) * 10 * 40
	if b.XP < want*0.97 || b.XP > want*1.03 {
		t.Errorf("xp %v, expected about %v", b.XP, want)
	}
}

func TestOptimiserBeatsOrMatchesPolicy(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	pl := e.RunPolicy(PlanToday(d), 60)
	b := e.Optimise(60, 20)
	if b.XP < pl.End.XP {
		t.Errorf("optimiser %v below today's plan %v", b.XP, pl.End.XP)
	}
	if !b.DenOK {
		t.Error("optimum did not do the Den quest")
	}
}

func TestOptimiserRespectsFloorsAndDeterminism(t *testing.T) {
	d := load(t)
	e := NewEnv(d, DefaultParams(), ModePlan)
	b1 := e.Optimise(60, 15)
	b2 := e.Optimise(60, 15)
	if b1.XP != b2.XP || b1.Signature() != b2.Signature() {
		t.Errorf("not deterministic: %v %q vs %v %q", b1.XP, b1.Signature(), b2.XP, b2.Signature())
	}
	for _, s := range b1.Steps {
		if s.Kind == "farm" && s.Clvl0 < e.Floor(s.Area) {
			t.Errorf("entered area %d at clvl %d below floor %d", s.Area, s.Clvl0, e.Floor(s.Area))
		}
	}
}

func TestExtract(t *testing.T) {
	log := strings.Join([]string{
		`{"time":"2026-11-14T10:00:00Z","msg":"d2bot start"}`,
		`{"time":"2026-11-14T10:00:30Z","msg":"arrival","area":2}`,
		`{"time":"2026-11-14T10:01:00Z","msg":"minute","area":2,"clvl":1,"xp":270,"kills":12}`,
		`{"time":"2026-11-14T10:05:00Z","msg":"d2d: Save and Exit requested (chicken / leave game)"}`,
		`not json`,
	}, "\n")
	r, err := Extract(strings.NewReader(log), 9, "b", time.Time{}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Minutes) != 1 || r.Minutes[0][3] != 270 || len(r.Arrivals) != 1 || r.Arrivals[0][0] != 0.5 || len(r.Restarts) != 1 || r.Restarts[0] != 5 {
		t.Errorf("extract: %+v", r)
	}
}

func TestSensitivityScenariosApply(t *testing.T) {
	d := load(t)
	rows := Sensitivity(d, DefaultParams(), ModePlan, []float64{30}, 8)
	if len(rows) != len(Scenarios()) {
		t.Fatalf("%d rows", len(rows))
	}
	base, slow := rows[0].Opt["30"], rows[1].Opt["30"]
	if !(slow < base) {
		t.Errorf("kill rate -20%% did not lower the optimum: %v vs %v", slow, base)
	}
}
