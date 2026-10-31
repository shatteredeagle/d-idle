package quest

import (
	"errors"
	"strings"
	"testing"
)

func act2Reader(words map[string]uint16) Reader { return NewReader(memWith(0, words)) }

// scripted Normal Act 2 world: drops where the real game has them, the true tomb is area 69.
func act2World() *FakeAct2Env {
	e := NewFakeAct2Env(AreaLutGholein)
	e.Dropped = []string{ItemBookOfSkill, ItemViperAmulet} // Radament's drop, the altar's amulet
	e.Chests = map[int][]string{ObjScrollChest: {ItemHoradricScroll}, ObjCubeChest: {ItemCube}, ObjStaffChest: {ItemShaft}}
	e.Objects[69] = []int{ObjOrifice}
	e.Travel["obj100"] = AreaDurielsLair
	e.Travel["obj298"] = AreaArcane
	e.Travel["obj60"] = AreaCanyon
	e.Effect = func(int, string, string) {}
	return e
}

func TestAct2TablesConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, id := range Act2Order {
		h, ok := Act2Handler(id)
		if !ok {
			t.Fatalf("order names %q without a handler", id)
		}
		seen[id] = true
		if len(h.Steps) == 0 {
			t.Errorf("%s has no steps", id)
		}
		for i, s := range h.Steps {
			switch s.Kind {
			case KGoNPC, KTalk, KTurnIn:
				if _, ok := FindMenu(s.NPC, LevelMain); !ok {
					t.Errorf("%s step %d: npc %d has no dialog table", id, i, s.NPC)
				}
			case KGoArea, KWaitArea:
				if s.Area < 40 || s.Area > 74 {
					t.Errorf("%s step %d: area %d is not Act 2", id, i, s.Area)
				}
			case KTransmute:
				if len(s.Labels) < 2 || s.Item == "" {
					t.Errorf("%s step %d: transmute needs inputs and an output", id, i)
				}
			}
		}
	}
	for _, h := range Act2 {
		if !seen[h.Quest] {
			t.Errorf("handler %s is not in Act2Order", h.Quest)
		}
	}
	if len(TombAreas()) != 7 || TombAreas()[0] != 66 || TombAreas()[6] != 72 {
		t.Fatalf("tomb areas %v", TombAreas())
	}
}

func TestAct2FullCampaign(t *testing.T) {
	e := act2World()
	a := &Act2Runner{Env: e, R: act2Reader(map[string]uint16{"act1_done": 1}), Diff: 0}
	ran, err := a.RunAll(bg, true)
	if err != nil {
		t.Fatalf("%v\nlog %v", err, e.Log)
	}
	if len(ran) != len(Act2Order) {
		t.Fatalf("ran %v", ran)
	}
	if a.Tomb != 69 {
		t.Fatalf("tomb %d", a.Tomb)
	}
	if len(e.Placed) != 1 || e.Placed[0] != ItemHoradricStaff {
		t.Fatalf("placed %v", e.Placed)
	}
	if e.Pack[ItemShaft] || e.Pack[ItemViperAmulet] || e.Pack[ItemBookOfSkill] {
		t.Fatalf("consumed items still in pack: %v", e.Pack)
	}
	if !e.Pack[ItemCube] || !e.Pack[ItemHoradricScroll] {
		t.Fatalf("pack %v", e.Pack)
	}
	joined := strings.Join(e.Log, " ")
	for _, w := range []string{"killRadament", "killFangskin", "killThe Summoner", "killDuriel", "transmute"} {
		if !strings.Contains(joined, w) {
			t.Errorf("log lacks %q: %v", w, e.Log)
		}
	}
	// the tomb scan visited 66,67,68 empty before 69, each time going back to the canyon
	if got := strings.Count(joined, "goarea66"); got != 1 {
		t.Errorf("tomb 66 visited %d times", got)
	}
	if e.Area() != AreaLutGholein { // seven_tombs ends with ToTown + Jerhyn
		t.Errorf("ended in area %d", e.Area())
	}
}

func TestAct2ProgressionOnlySkipsOptional(t *testing.T) {
	e := act2World()
	a := &Act2Runner{Env: e, R: act2Reader(nil)}
	ran, err := a.RunAll(bg, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ran {
		if id == "summoner" || id == "arcane_sanctuary" {
			t.Fatalf("optional %s ran", id)
		}
	}
	if strings.Contains(strings.Join(e.Log, " "), "The Summoner") {
		t.Fatal("summoner fought in the progression campaign")
	}
}

func TestAct2SkipsOnSaveAndProxy(t *testing.T) {
	e := act2World()
	e.Pack[ItemCube] = true
	a := &Act2Runner{Env: e, R: act2Reader(map[string]uint16{"radament": 1})}
	h, _ := Act2Handler("radament")
	res, err := a.Run(bg, h)
	if err != nil || !res.Skipped || !res.Confirmed || len(e.Log) != 0 {
		t.Fatalf("radament: %+v %v %v", res, err, e.Log)
	}
	h, _ = Act2Handler("horadric_cube")
	res, err = a.Run(bg, h)
	if err != nil || !res.Skipped || res.Confirmed || len(e.Log) != 0 {
		t.Fatalf("cube proxy skip must be unconfirmed (T3): %+v %v", res, err)
	}
	// an assembled staff also satisfies the two fetch legs that fed it
	e.Pack = map[string]bool{ItemHoradricStaff: true}
	for _, q := range []string{"staff_of_kings", "tainted_sun"} {
		h, _ = Act2Handler(q)
		if res, _ := a.Run(bg, h); !res.Skipped {
			t.Errorf("%s not skipped with the staff in the pack", q)
		}
	}
}

func TestAct2StaffNeedsInputs(t *testing.T) {
	h, _ := Act2Handler("horadric_staff")
	e := act2World()
	e.Pack[ItemShaft] = true // no amulet, no cube
	a := &Act2Runner{Env: e, R: act2Reader(nil)}
	if _, err := a.Run(bg, h); !errors.Is(err, ErrCubeMissing) {
		t.Fatalf("want ErrCubeMissing, got %v", err)
	}
	e.Pack[ItemViperAmulet], e.Pack[ItemCube] = true, true
	e.Recipes = map[string][]string{} // the cube does nothing
	if _, err := a.Run(bg, h); !errors.Is(err, ErrCubeNoResult) {
		t.Fatalf("want ErrCubeNoResult, got %v", err)
	}
	if !e.Pack[ItemShaft] || !e.Pack[ItemViperAmulet] {
		t.Fatal("a failed transmute must keep the inputs")
	}
}

func TestAct2NoTomb(t *testing.T) {
	e := act2World()
	delete(e.Objects, 69)
	e.Pack[ItemShaft], e.Pack[ItemViperAmulet], e.Pack[ItemCube] = true, true, true
	a := &Act2Runner{Env: e, R: act2Reader(nil)}
	h, _ := Act2Handler("horadric_staff")
	if _, err := a.Run(bg, h); !errors.Is(err, ErrNoTomb) {
		t.Fatalf("want ErrNoTomb, got %v", err)
	}
	if len(e.Placed) != 0 {
		t.Fatal("staff placed without an orifice")
	}
}

func TestAct2StepFailureNamesStep(t *testing.T) {
	e := act2World()
	e.Fail["kill"] = errors.New("boom")
	a := &Act2Runner{Env: e, R: act2Reader(nil)}
	h, _ := Act2Handler("radament")
	_, err := a.Run(bg, h)
	if err == nil || !strings.Contains(err.Error(), "Radament's Lair: step 4") {
		t.Fatalf("err %v", err)
	}
}

func TestAct2PrereqBlocks(t *testing.T) {
	e := act2World()
	a := &Act2Runner{Env: e, R: act2Reader(nil)}
	h := Handler{Quest: "x", Name: "x", Pre: []Cond{{"radament", Done}}, Steps: []Step{{Kind: KClose}}}
	if _, err := a.Run(bg, h); !errors.Is(err, ErrPrereq) {
		t.Fatalf("want ErrPrereq, got %v", err)
	}
}

func TestPlanCubeLayout(t *testing.T) {
	cells, err := PlanCubeLayout([]Size{{1, 3}, {1, 1}}) // Shaft, Amulet
	if err != nil || len(cells) != 2 || cells[0] != (Cell{0, 0}) || cells[1] != (Cell{1, 0}) {
		t.Fatalf("cells %v %v", cells, err)
	}
	if _, err := PlanCubeLayout([]Size{{2, 2}, {2, 2}, {2, 2}}); !errors.Is(err, ErrCubeNoFit) {
		t.Fatalf("three 2x2 must not fit 3x4: %v", err)
	}
	if _, err := PlanCubeLayout([]Size{{1, 5}}); !errors.Is(err, ErrCubeNoFit) {
		t.Fatalf("tall item: %v", err)
	}
	if dx, dy := CellCentre(Cell{1, 0}, Size{1, 1}); dx != 43 || dy != 14 {
		t.Fatalf("centre %d,%d", dx, dy)
	}
}

func TestAct2MercHire(t *testing.T) {
	e := act2World()
	a := &Act2Runner{Env: e, R: act2Reader(nil)}
	if _, err := a.Run(bg, Act2Merc); err != nil || len(e.Hired) != 1 || e.Hired[0] != "Defiance" {
		t.Fatalf("hired %v err %v", e.Hired, err)
	}
	if mi, n, row, err := RowIndex(Greiz, LevelMain, "Hire", nil); err != nil || mi != 1 || n != 3 || row.Tag != "V" {
		t.Fatalf("Greiz Hire row %d/%d %+v %v", mi, n, row, err)
	}
}

func TestHopA1toA2AndA2toA3(t *testing.T) {
	e := NewFakeEnv(AreaRogueEncampment)
	e.Travel["Go East"] = AreaLutGholein
	r := act2Reader(map[string]uint16{"sisters_to_the_slaughter": 1, "act1_done": 1})
	h, _ := HopFor(1)
	e.Status = func(q string) Status { return r.Status(0, q) }
	if res, err := RunHop(bg, e, r, 0, h); err != nil || !res.Confirmed || e.Area() != AreaLutGholein {
		t.Fatalf("A1->A2: %+v %v log %v", res, err, e.Calls)
	}
	// Act 2 to 3: Seven Tombs and Act 2 complete done, Jerhyn leg skipped, Meshif sails.
	r = act2Reader(map[string]uint16{"seven_tombs": 1, "act2_done": 1})
	e = NewFakeEnv(AreaLutGholein)
	e.Travel["Sail East"] = AreaKurastDocks
	e.Status = func(q string) Status { return r.Status(0, q) }
	h, _ = HopFor(2)
	if res, err := RunHop(bg, e, r, 0, h); err != nil || !res.Confirmed || e.Area() != AreaKurastDocks {
		t.Fatalf("A2->A3: %+v %v calls %v", res, err, e.Calls)
	}
	for _, c := range e.Calls {
		if strings.Contains(c, " 201") {
			t.Fatalf("Jerhyn leg should be skipped: %v", e.Calls)
		}
	}
	// Duriel dead but Act 2 not complete: the hop is blocked until the quest reads Done.
	r = act2Reader(map[string]uint16{"act2_done": 1})
	e = NewFakeEnv(AreaLutGholein)
	if _, err := RunHop(bg, e, r, 0, h); !errors.Is(err, ErrPrereq) {
		t.Fatalf("want ErrPrereq, got %v", err)
	}
}

func TestFindTombProbesFarRooms(t *testing.T) {
	f := NewFarTombEnv(AreaCanyon, 71, 4)
	got, err := FindTomb(bg, f, AreaCanyon)
	if err != nil || got != 71 || f.CurArea != 71 {
		t.Fatalf("got %d err %v area %d", got, err, f.CurArea)
	}
}

func TestFindTombHintGoesFirst(t *testing.T) {
	f := NewFarTombEnv(AreaCanyon, 70, 2)
	f.Hint = 70
	got, err := FindTomb(bg, f, AreaCanyon)
	if err != nil || got != 70 || len(f.Entered) != 1 {
		t.Fatalf("got %d err %v entered %v", got, err, f.Entered)
	}
	// a wrong hint costs one tomb, not the answer
	f = NewFarTombEnv(AreaCanyon, 70, 2)
	f.Hint = 68
	got, err = FindTomb(bg, f, AreaCanyon)
	if err != nil || got != 70 || f.Entered[0] != 68 {
		t.Fatalf("wrong hint: got %d err %v entered %v", got, err, f.Entered)
	}
}

func TestFindTombRetriesUnenterable(t *testing.T) {
	f := NewFarTombEnv(AreaCanyon, 69, 1)
	f.Unenter[69] = 1 // the gate did not fire the first time (V live: exit 69)
	got, err := FindTomb(bg, f, AreaCanyon)
	if err != nil || got != 69 {
		t.Fatalf("got %d err %v", got, err)
	}
	f = NewFarTombEnv(AreaCanyon, 69, 1)
	f.Unenter[69] = 5
	if _, err := FindTomb(bg, f, AreaCanyon); !errors.Is(err, ErrNoTomb) {
		t.Fatalf("want ErrNoTomb, got %v", err)
	}
}

func TestFindTombNoProberJudgesArrivalRoom(t *testing.T) {
	f := NewFakeAct2Env(AreaCanyon)
	f.Objects[70] = []int{ObjOrifice}
	if got, err := FindTomb(bg, f, AreaCanyon); err != nil || got != 70 {
		t.Fatalf("got %d err %v", got, err)
	}
}
