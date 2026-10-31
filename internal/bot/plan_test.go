package bot

import (
	"d2lod-bot/internal/env"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPlan(t *testing.T) {
	p, err := LoadPlan("")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Areas) < 100 || len(p.Steps["normal"]) < 10 || len(p.Steps["nightmare"]) == 0 || len(p.Steps["hell"]) == 0 || len(p.Runs) < 5 {
		t.Fatalf("plan looks empty: areas=%d steps=%v runs=%d warnings=%v", len(p.Areas), len(p.Steps["normal"]), len(p.Runs), p.Warnings)
	}
	if !p.IsTown(1) || !p.IsTown(40) || p.IsTown(2) || p.Act(37) != 1 || !p.Waypoints[3] || p.Towns[2] != 40 {
		t.Fatal("area data wrong")
	}
	if r := p.Runs["andariel"]; r.FinalArea != 37 || r.MinLvl["normal"] != env.DefaultAndarielMinLevel || r.Kill[0] != "Andariel" {
		t.Fatalf("andariel run %+v", r)
	}
}

func TestLoadPlanIsTolerant(t *testing.T) {
	dir := t.TempDir()
	// aliases, string numbers, unknown fields, one broken entry; areas.json falls back to the embedded copy
	os.WriteFile(filepath.Join(dir, "progression-normal.json"), []byte(`{"future":1,"normal":[
		{"area_id":"2","min_clvl":1,"target_clvl":5,"name":"Blood Moor","extra":true},
		{"name":"no area","enter_clvl":1,"leave_clvl":3},
		{"area":3,"enter_clvl":4,"leave_clvl":2}]}`), 0o644)
	p, err := LoadPlan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st := p.Steps["normal"]; len(st) != 1 || st[0].Area != 2 || st[0].LeaveLvl != 5 {
		t.Fatalf("steps %+v", p.Steps["normal"])
	}
	if len(p.Warnings) != 2 || len(p.Areas) < 100 {
		t.Fatalf("warnings %v areas %d", p.Warnings, len(p.Areas))
	}
	os.WriteFile(filepath.Join(dir, "areas.json"), []byte(`{not json`), 0o644)
	if _, err := LoadPlan(dir); err == nil {
		t.Fatal("corrupt areas.json accepted")
	}
}

func TestPick(t *testing.T) {
	p, _ := LoadPlan("")
	act1 := map[int]bool{1: true}
	for _, c := range []struct {
		diff  string
		level int
		o     PickOptions
		want  int
	}{
		{"normal", 1, PickOptions{Acts: act1}, 2},                                  // Blood Moor
		{"normal", 5, PickOptions{Acts: act1}, 3},                                  // Cold Plains is the only window at its entry level
		{"normal", 6, PickOptions{Acts: act1}, 9},                                  // last window containing 6: Cave Level 1
		{"normal", 6, PickOptions{Acts: act1, Exclude: map[int]bool{9: true}}, 17}, // then Burial Grounds
		{"normal", 6, PickOptions{Acts: act1, Exclude: map[int]bool{9: true, 17: true}}, 3},
		{"normal", 5, PickOptions{Acts: act1, Margin: 4}, 2}, // margin 4: only Blood Moor qualifies
		{"normal", 60, PickOptions{Acts: act1}, 37},          // past every window: the last act 1 step
	} {
		s, ok := p.Pick(c.diff, c.level, c.o)
		if !ok || s.Area != c.want {
			t.Errorf("%s level %d %+v: got %d (%v), want %d", c.diff, c.level, c.o, s.Area, ok, c.want)
		}
	}
	if s, ok := p.Pick("nightmare", 36, PickOptions{Acts: act1}); !ok || s.EnterLvl > 36 || s.LeaveLvl < 36 {
		t.Errorf("nightmare pick %+v", s)
	}
	if _, ok := p.Pick("normal", 5, PickOptions{Acts: map[int]bool{9: true}}); ok {
		t.Error("picked an area in an unreachable act")
	}
	if d := p.DifficultyFor(10, 0, nil); d != "normal" {
		t.Errorf("difficulty %s", d)
	}
	if d := p.DifficultyFor(40, 0, nil); d != "nightmare" {
		t.Errorf("difficulty %s", d)
	}
}

func TestResolve(t *testing.T) {
	p, _ := LoadPlan("")
	for _, c := range []struct {
		name string
		id   int
		want int
		bad  bool
	}{
		{"", 12, 12, false}, {"cold plains", 0, 3, false}, {"run:Countess", 0, 25, false}, {"Run:Andariel", 0, 37, false},
		{"Pit Level 1", 0, 12, false}, {"nonsense", 0, 0, true}, {"run:nonsense", 0, 0, true},
	} {
		got, err := p.Resolve(c.name, c.id, nil, func(n string) int {
			if n == "Pit Level 1" {
				return 12
			}
			return 0
		})
		if (err != nil) != c.bad || got != c.want {
			t.Errorf("%q/%d: got %d, %v", c.name, c.id, got, err)
		}
	}
}

// Booti at clvl 12 (live 2026-10-29): with Stony Field cleared and the waypoint areas avoided, the fallback used to be
// Den of Evil, a monster level 2-ish area worth about 8 XP in 12 minutes. With a floor Pick reports nothing instead.
func TestPickFloorRefusesWorthlessFallback(t *testing.T) {
	p, _ := LoadPlan("")
	act1 := map[int]bool{1: true}
	ex := map[int]bool{}
	for _, a := range []int{3, 4, 5, 6, 7, 9, 13, 17, 18, 19, 20, 27, 29, 35, 36, 37} {
		ex[a] = true
	}
	if s, ok := p.Pick("normal", 12, PickOptions{Acts: act1, Exclude: ex}); !ok || s.Area != 8 {
		t.Fatalf("without a floor the old fallback is Den of Evil, got %+v %v", s, ok)
	}
	if s, ok := p.Pick("normal", 12, PickOptions{Acts: act1, Exclude: ex, Floor: 4}); ok {
		t.Fatalf("worthless fallback chosen: %+v", s)
	}
	// nothing excluded: past every window is still the last step, the floor only guards the excluded-window case
	if s, ok := p.Pick("normal", 60, PickOptions{Acts: act1, Floor: 4}); !ok || s.Area != 37 {
		t.Errorf("level 60: %+v %v", s, ok)
	}
	// a still-worthwhile area remains: Cold Plains leaves at 10, level 12 is within the floor of 4
	delete(ex, 3)
	if s, ok := p.Pick("normal", 12, PickOptions{Acts: act1, Exclude: ex, Floor: 4}); !ok || s.Area != 3 {
		t.Errorf("got %+v %v, want Cold Plains", s, ok)
	}
}

// Avoided (not cleared) areas must not trigger the floor refusal: the old fallback is kept and no fresh game is asked for.
func TestPickFloorIgnoresAvoidedAreas(t *testing.T) {
	p, _ := LoadPlan("")
	act1 := map[int]bool{1: true}
	ex := map[int]bool{}
	for _, a := range []int{3, 4, 5, 6, 7, 9, 13, 17, 18, 19, 20, 27, 29, 35, 36, 37} {
		ex[a] = true
	}
	if s, ok := p.Pick("normal", 12, PickOptions{Acts: act1, Exclude: ex, Cleared: map[int]bool{}, Floor: 4}); !ok || s.Area != 8 {
		t.Fatalf("got %+v %v, want the old fallback", s, ok)
	}
	if _, ok := p.Pick("normal", 12, PickOptions{Acts: act1, Exclude: ex, Cleared: ex, Floor: 4}); ok {
	}
}

// Soaky at clvl 3 (live 2026-10-29): Blood Moor and the Den cleared, the old fallback walked it into Burial Grounds.
func TestPickDoesNotSendAWeakCharacterFarAhead(t *testing.T) {
	p, _ := LoadPlan("")
	act1 := map[int]bool{1: true}
	ex := map[int]bool{2: true, 8: true}
	if s, ok := p.Pick("normal", 3, PickOptions{Acts: act1, Exclude: ex, Floor: 4}); ok {
		t.Fatalf("clvl 3 sent to %+v", s)
	}
	if s, ok := p.Pick("normal", 3, PickOptions{Acts: act1, Floor: 4}); !ok || s.Area != 8 {
		t.Fatalf("fresh clvl 3 should stay in the first window, got %+v %v", s, ok)
	}
}

// WP3 (Soaky soak 2026-10-29): with Blood Moor and the Den held back, a kit-less clvl 1-3 character used to be sent to
// Cold Plains. Cold Plains needs clvl 4 or a full kit; Den and Blood Moor stay as they were.
func TestPickColdPlainsNeedsKitOrLevel4(t *testing.T) {
	p, _ := LoadPlan("")
	act1 := map[int]bool{1: true}
	held := map[int]bool{2: true, 8: true}
	for _, c := range []struct {
		level int
		kit   bool
		want  int // 0 = nothing eligible
	}{
		{1, false, 0}, {2, false, 0}, {3, false, 0}, {4, false, 3},
		{1, true, 3}, {3, true, 3}, // (kit rows run without Floor: with it the old earlyGap rule still sends a clvl 1-3 to a fresh game)
	} {
		s, ok := p.Pick("normal", c.level, PickOptions{Acts: act1, Exclude: held, FullKit: c.kit, Floor: map[bool]int{false: 4}[c.kit]})
		if c.want == 0 && ok || c.want != 0 && (!ok || s.Area != c.want) {
			t.Errorf("clvl %d kit %v: got %+v %v, want area %d", c.level, c.kit, s, ok, c.want)
		}
	}
	// normal play is unchanged: Blood Moor at clvl 1, the Den at clvl 2-4 without any kit
	for lvl, want := range map[int]int{1: 2, 2: 8, 3: 8, 4: 8} {
		if s, ok := p.Pick("normal", lvl, PickOptions{Acts: act1}); !ok || s.Area != want {
			t.Errorf("clvl %d: got %+v %v, want %d", lvl, s, ok, want)
		}
	}
}

// Between clvl 1 and 30 the Act 1-only picker always finds an area, also with any single area excluded or cleared
// (docs/plan/plan-v2.md M2: the 14-22 band used to rely on Catacombs 4 alone). The knob default and runs.json agree.
func TestPickNeverEmptyInAct1(t *testing.T) {
	p, err := LoadPlan("")
	if err != nil {
		t.Fatal(err)
	}
	act1 := map[int]bool{1: true}
	if r := p.Runs["andariel"]; r.MinLvl["normal"] != env.DefaultAndarielMinLevel {
		t.Fatalf("runs.json andariel floor %d != knob default %d", r.MinLvl["normal"], env.DefaultAndarielMinLevel)
	}
	for lvl := 1; lvl <= 30; lvl++ {
		if _, ok := p.Pick("normal", lvl, PickOptions{Acts: act1, FullKit: true}); !ok {
			t.Errorf("clvl %d: no area", lvl)
		}
		for _, s := range p.Steps["normal"] {
			ex := map[int]bool{s.Area: true}
			if _, ok := p.Pick("normal", lvl, PickOptions{Acts: act1, FullKit: true, Exclude: ex}); !ok {
				t.Errorf("clvl %d without area %d (%s): no area", lvl, s.Area, s.Name)
			}
		}
	}
	// the XP band: 12 to 18 stays in Act 1 areas whose monster level is 9 to 12 (V levels.txt: Outer Cloister 9, Jail 1 10,
	// Catacombs 2 11, Catacombs 3 and 4 12), never a low field
	for lvl := 12; lvl <= 18; lvl++ {
		st, _ := p.Pick("normal", lvl, PickOptions{Acts: act1, FullKit: true})
		if st.Area != 27 && st.Area != 35 && st.Area != 36 && st.Area != 37 {
			t.Errorf("clvl %d picked %d %s", lvl, st.Area, st.Name)
		}
	}
}

func TestLevelPlanLine(t *testing.T) {
	st := Step{Area: 37, Name: "Catacombs 4", EnterLvl: 14, LeaveLvl: 18}
	for _, c := range []struct {
		lvl  int
		want string
	}{
		{15, "level plan: clvl 15 -> area 37 Catacombs 4 (in window 14-18)"},
		{20, "level plan: clvl 20 -> area 37 Catacombs 4 (past every window, last eligible (leaves at 18))"},
		{10, "level plan: clvl 10 -> area 37 Catacombs 4 (below every entry level, first eligible (enters at 14))"},
	} {
		if got := levelPlanLine(c.lvl, st, 0, 0); got != c.want {
			t.Errorf("got %q want %q", got, c.want)
		}
	}
}

// BUILDV2: Stony Field opens at clvl 7 for Barbarian and Paladin (Melee), 8 for everyone else.
func TestPickStonyFieldMeleeEnter(t *testing.T) {
	p, err := LoadPlan("")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		lvl   int
		melee bool
		stony bool
	}{{7, true, true}, {7, false, false}, {8, false, true}, {8, true, true}} {
		st, ok := p.Pick("normal", c.lvl, PickOptions{Melee: c.melee})
		if !ok || (st.Area == 4) != c.stony {
			t.Errorf("clvl %d melee %v: area %d ok %v, want Stony Field %v", c.lvl, c.melee, st.Area, ok, c.stony)
		}
	}
}
