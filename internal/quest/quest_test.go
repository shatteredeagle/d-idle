package quest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/saves"
)

var bg = context.Background()

func memWith(diff int, words map[string]uint16) *MapSource {
	m := &MapSource{}
	for k, w := range words {
		m.Set(diff, k, w)
	}
	return m
}

func TestInterpretLiveWords(t *testing.T) {
	cases := []struct {
		w    uint16
		want Status
	}{
		{0x0000, Needed}, {0x0001, Done}, {0x8001, Done}, {0x000c, Started}, {0x0010, Started}, {0x0004, Started}, {0x1000, Needed},
	}
	for _, c := range cases {
		if got, _ := Interpret(c.w); got != c.want {
			t.Errorf("word %#04x = %s, want %s", c.w, got, c.want)
		}
	}
	if _, p := Interpret(0x0002); !p {
		t.Error("bit 1 should read as pending")
	}
}

func TestReaderSaveAndMemoryPrecedence(t *testing.T) {
	var p saves.Progress
	p.Difficulty[0].Quests = []saves.QuestState{{ID: "den_of_evil", Word: 0x0004}, {ID: "radament", Word: 0x0001}}
	r := NewReader(SaveSource{p})
	if got := r.Quest(0, "den_of_evil"); got.Status != Started || got.Evidence != EvSave {
		t.Fatalf("den = %+v", got)
	}
	if !r.Skippable(0, "radament") || r.Skippable(0, "den_of_evil") {
		t.Fatal("skippable wrong")
	}
	if got := r.Quest(1, "radament"); got.Status != Needed || got.Evidence != EvNone {
		t.Fatalf("nightmare radament = %+v", got)
	}
	mem := memWith(0, map[string]uint16{"den_of_evil": 0x8001})
	r = NewReader(mem, SaveSource{p})
	if got := r.Quest(0, "den_of_evil"); got.Status != Done || got.Evidence != EvMemory {
		t.Fatalf("memory should win: %+v", got)
	}
	// Proxy evidence never skips.
	px := &MapSource{Ev: EvProxy}
	px.Set(0, "forgotten_tower", 1)
	if NewReader(px).Skippable(0, "forgotten_tower") {
		t.Fatal("proxy evidence must not skip")
	}
}

func TestActOpen(t *testing.T) {
	r := NewReader(memWith(0, map[string]uint16{"act1_done": 1}))
	if !r.ActOpen(0, 1) || r.ActOpen(0, 2) || r.ActOpen(1, 1) || r.ActOpen(0, 5) {
		t.Fatal("ActOpen wrong")
	}
}

func TestMenuTableAkaraVerified(t *testing.T) {
	st := func(string) Status { return Needed }
	idx, n, row, err := RowIndex(Akara, LevelTalk, "Den of Evil", st)
	if err != nil || idx != 2 || n != 4 || row.Tag != "V" {
		t.Fatalf("Den row = %d/%d %+v %v", idx, n, row, err)
	}
	idx, n, _, _ = RowIndex(Akara, LevelTalk, "Cancel", st)
	if idx != 3 || n != 4 {
		t.Fatalf("cancel = %d/%d", idx, n)
	}
	// Once Den is Done the row is gone (U) and Cancel moves up.
	done := func(string) Status { return Done }
	if _, _, _, err := RowIndex(Akara, LevelTalk, "Den of Evil", done); !errors.Is(err, ErrNoRow) {
		t.Fatalf("want ErrNoRow, got %v", err)
	}
	if _, _, _, err := RowIndex(9999, LevelMain, "Talk", st); !errors.Is(err, ErrNoMenu) {
		t.Fatalf("want ErrNoMenu, got %v", err)
	}
	for _, m := range Menus {
		if last := m.Rows[len(m.Rows)-1]; last.Label != "Cancel" {
			t.Errorf("npc %d %s: last row %q, want Cancel", m.NPC, m.Level, last.Label)
		}
	}
}

func TestPickPathAkaraDen(t *testing.T) {
	d := &FakeDialog{Status: func(string) Status { return Needed }}
	var got string
	d.Effect = func(npc int, level, label string) { got = label }
	if err := d.Talk(bg, Akara); err != nil {
		t.Fatal(err)
	}
	if err := PickPath(bg, d, Akara, d.Status, "Talk", "Den of Evil"); err != nil {
		t.Fatal(err)
	}
	if got != "Den of Evil" {
		t.Fatalf("effect label %q", got)
	}
	want := "choose 0/3 Talk"
	if d.Calls[1] != want || d.Calls[2] != "choose 2/4 Den of Evil" {
		t.Fatalf("calls %v", d.Calls)
	}
}

func TestFakeDialogRejectsWrongCount(t *testing.T) {
	d := &FakeDialog{}
	_ = d.Talk(bg, Akara)
	if err := d.Choose(bg, 0, 5); err == nil {
		t.Fatal("wrong line count should fail")
	}
}

func TestTurnInByTalk(t *testing.T) {
	e := NewFakeEnv(AreaRogueEncampment)
	e.Pack[ItemMalus] = true
	e.Effect = func(int, string, string) {}
	// The item stays: expect ErrTurnIn.
	err := DoTurnIn(bg, e, e, nil, TurnIn{NPC: Charsi, Item: ItemMalus})
	if !errors.Is(err, ErrTurnIn) {
		t.Fatalf("want ErrTurnIn, got %v", err)
	}
	if e.Open {
		t.Fatal("dialog left open")
	}
	// A Talk that consumes the item succeeds with no row pick.
	e.Pack[ItemMalus] = true
	delete(e.Pack, ItemMalus)
	if err := DoTurnIn(bg, e, e, nil, TurnIn{NPC: Charsi, Item: ItemMalus}); err != nil {
		t.Fatal(err)
	}
}

func TestTurnInFallbackRow(t *testing.T) {
	e := NewFakeEnv(AreaRogueEncampment)
	e.Pack["xyz"] = true
	e.Effect = func(npc int, level, label string) {
		if label == "Gossip" {
			delete(e.Pack, "xyz")
		}
	}
	if err := DoTurnIn(bg, e, e, nil, TurnIn{NPC: Charsi, Item: "xyz", Path: []string{"Talk", "Gossip"}}); err != nil {
		t.Fatal(err)
	}
}

func warrivWorld() (*FakeEnv, Reader) {
	e := NewFakeEnv(AreaRogueEncampment)
	e.Travel["Go East"] = AreaLutGholein
	r := NewReader(memWith(0, map[string]uint16{"sisters_to_the_slaughter": 1, "act1_done": 1}))
	e.Status = r.statusFor(0)
	return e, r
}

func (r Reader) statusFor(diff int) func(string) Status { return statusFn(r, diff) }

func TestHopWarrivSuccessByArea(t *testing.T) {
	e, r := warrivWorld()
	h, _ := HopFor(1)
	res, err := RunHop(bg, e, r, 0, h)
	if err != nil || !res.Confirmed || e.Area() != AreaLutGholein {
		t.Fatalf("res %+v err %v area %d log %v", res, err, e.Area(), e.Log)
	}
	if e.Log[len(e.Log)-1] != "touchwp" {
		t.Fatalf("hop must end by touching the town waypoint: %v", e.Log)
	}
	// Already there: skipped.
	res, err = RunHop(bg, e, r, 0, h)
	if err != nil || !res.Skipped {
		t.Fatalf("second run %+v %v", res, err)
	}
}

func TestHopPreconditions(t *testing.T) {
	e, _ := warrivWorld()
	h, _ := HopFor(1)
	if _, err := RunHop(bg, e, NewReader(), 0, h); !errors.Is(err, ErrPrereq) {
		t.Fatalf("no Andariel: %v", err)
	}
	r := NewReader(memWith(0, map[string]uint16{"sisters_to_the_slaughter": 1}))
	e.CurArea = 3
	if _, err := RunHop(bg, e, r, 0, h); !errors.Is(err, ErrWrongPlace) {
		t.Fatalf("wrong area: %v", err)
	}
	// Row missing because act1_done is not set: ErrNoRow, area unchanged.
	e.CurArea = AreaRogueEncampment
	e.Status = r.statusFor(0)
	if _, err := RunHop(bg, e, r, 0, h); !errors.Is(err, ErrNoRow) {
		t.Fatalf("no Go East row: %v", err)
	}
}

func TestHopFailsWhenAreaDoesNotChange(t *testing.T) {
	e, r := warrivWorld()
	delete(e.Travel, "Go East")
	h, _ := HopFor(1)
	if _, err := RunHop(bg, e, r, 0, h); err == nil || !strings.Contains(err.Error(), "waitarea") && !strings.Contains(err.Error(), "waiting") {
		t.Fatalf("expected area wait failure, got %v", err)
	}
}

func TestAllHopsWellFormed(t *testing.T) {
	for _, h := range Hops {
		if len(h.Steps) == 0 || h.Steps[len(h.Steps)-1].Kind != KTouchWP {
			t.Errorf("%s: last step must touch the waypoint", h.Name)
		}
		var waits bool
		for _, s := range h.Steps {
			if s.Kind == KWaitArea && s.Area == h.To {
				waits = true
			}
			if s.Kind == KPick {
				if _, _, _, err := RowIndex(s.NPC, LevelMain, s.Labels[0], func(string) Status { return Done }); err != nil {
					t.Errorf("%s: %v", h.Name, err)
				}
			}
		}
		if !waits {
			t.Errorf("%s: no area wait for %d", h.Name, h.To)
		}
	}
}

func TestMeshifHopSkipsJerhynWhenActDone(t *testing.T) {
	e := NewFakeEnv(AreaLutGholein)
	e.Travel["Sail East"] = AreaKurastDocks
	r := NewReader(memWith(0, map[string]uint16{"seven_tombs": 1, "act2_done": 1}))
	e.Status = r.statusFor(0)
	h, _ := HopFor(2)
	if _, err := RunHop(bg, e, r, 0, h); err != nil {
		t.Fatal(err, e.Log)
	}
	for _, c := range e.FakeDialog.Calls {
		if c == "talk 201" {
			t.Fatal("talked to Jerhyn although act2_done")
		}
	}
}

func TestRedPortalHop(t *testing.T) {
	e := NewFakeEnv(102)
	e.Travel["obj342"] = AreaFortress
	r := NewReader(memWith(0, map[string]uint16{"guardian": 1}))
	h, _ := HopFor(3)
	if res, err := RunHop(bg, e, r, 0, h); err != nil || !res.Confirmed {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestHandlersTableSane(t *testing.T) {
	seen := map[string]bool{}
	for _, h := range Act1 {
		if seen[h.Quest] {
			t.Errorf("duplicate %s", h.Quest)
		}
		seen[h.Quest] = true
		found := false
		for _, q := range []string{"den_of_evil", "sisters_burial_grounds", "tools_of_the_trade", "search_for_cain", "forgotten_tower", "sisters_to_the_slaughter"} {
			found = found || q == h.Quest
		}
		if !found {
			t.Errorf("unknown quest id %s", h.Quest)
		}
		for _, s := range h.Steps {
			if s.Kind == KPick {
				if _, _, _, err := RowIndex(s.NPC, LevelMain, s.Labels[0], nil); err != nil {
					t.Errorf("%s: %v", h.Name, err)
				}
			}
		}
	}
	if !seen["sisters_to_the_slaughter"] {
		t.Fatal("Andariel handler missing")
	}
}

func TestHandlerSkipsWhenDone(t *testing.T) {
	e := NewFakeEnv(AreaRogueEncampment)
	r := NewReader(memWith(0, map[string]uint16{"den_of_evil": 0x8001}))
	h, _ := Handler1("den_of_evil")
	res, err := RunHandler(bg, e, r, 0, h)
	if err != nil || !res.Skipped || len(e.Log) != 0 {
		t.Fatalf("%+v %v %v", res, err, e.Log)
	}
}

func TestHandlerDenOfEvilRuns(t *testing.T) {
	e := NewFakeEnv(AreaRogueEncampment)
	var labels []string
	e.Effect = func(npc int, level, label string) { labels = append(labels, label) }
	src := memWith(0, nil)
	r := NewReader(src)
	e.Status = r.statusFor(0)
	h, _ := Handler1("den_of_evil")
	res, err := RunHandler(bg, e, r, 0, h)
	if err != nil {
		t.Fatal(err, e.Log)
	}
	if !res.Ran || res.Confirmed {
		t.Fatalf("no evidence yet, must be unconfirmed: %+v", res)
	}
	if len(labels) != 1 || labels[0] != "Den of Evil" {
		t.Fatalf("labels %v", labels)
	}
	if strings.Join(e.Log, " ") != "gonpc148 goarea8 clear8 totown gonpc148" {
		t.Fatalf("log %v", e.Log)
	}
	src.Set(0, "den_of_evil", 0x0001)
	res, _ = RunHandler(bg, e, r, 0, h)
	if !res.Skipped {
		t.Fatal("after the word reads Done the handler skips")
	}
}

func TestHandlerStepFailureStops(t *testing.T) {
	e := NewFakeEnv(AreaRogueEncampment)
	e.Fail["kill"] = errors.New("boss not found")
	r := NewReader()
	e.Status = r.statusFor(0)
	h, _ := Handler1("sisters_burial_grounds")
	_, err := RunHandler(bg, e, r, 0, h)
	if err == nil || !strings.Contains(err.Error(), "boss not found") {
		t.Fatalf("err %v", err)
	}
	for _, l := range e.Log {
		if l == "totown" {
			t.Fatal("ran past the failed kill")
		}
	}
}

func cainWorld() (*FakeEnv, Reader) {
	e := NewFakeEnv(AreaRogueEncampment)
	e.Travel["obj17"] = AreaTristram
	r := NewReader()
	e.Status = r.statusFor(0)
	return e, r
}

func TestHandlerSearchForCain(t *testing.T) {
	e, r := cainWorld()
	e.OnTalk = func(npc int) {
		if npc == Akara && e.Pack[ItemInifussScroll] {
			delete(e.Pack, ItemInifussScroll)
			e.Pack[ItemCairnKey] = true
		}
	}
	h, _ := Handler1("search_for_cain")
	res, err := RunHandler(bg, e, r, 0, h)
	if err != nil || !res.Ran {
		t.Fatalf("%+v %v log %v", res, err, e.Log)
	}
	if e.Pack[ItemInifussScroll] || !e.Pack[ItemCairnKey] {
		t.Fatalf("pack %v", e.Pack)
	}
}

func TestHandlerSearchForCainFailsLoudlyWithoutTurnIn(t *testing.T) {
	e, r := cainWorld()
	h, _ := Handler1("search_for_cain")
	if _, err := RunHandler(bg, e, r, 0, h); !errors.Is(err, ErrTurnIn) {
		t.Fatalf("want ErrTurnIn, got %v", err)
	}
}

func TestHandlerPrereqEnforced(t *testing.T) {
	h := Handler{Quest: "x", Name: "x", Pre: []Cond{{"den_of_evil", Done}}}
	if _, err := RunHandler(bg, NewFakeEnv(1), NewReader(), 0, h); !errors.Is(err, ErrPrereq) {
		t.Fatalf("got %v", err)
	}
}

// Live record (i4, 2026-10-28): Akara's Talk opens the topic list only while Den of Evil is Needed; with the reward
// pending (0x001e) the first Talk pays it and no submenu exists, so the claim is the single label "Talk".
func TestAkaraTalkSubmenuFollowsTheQuestState(t *testing.T) {
	cases := []struct {
		den     Status
		respec  bool
		wantRow int // line count of the main menu
		path    []string
		wantErr error
	}{
		{Needed, false, 3, []string{"Talk", "Den of Evil"}, nil},
		{Started, false, 3, []string{"Talk"}, nil}, // reward claim, quest running: a speech
		{Started, false, 3, []string{"Talk", "Den of Evil"}, ErrNoSubmenu},
		{Done, true, 4, []string{"Talk"}, nil},
	}
	for i, c := range cases {
		st := func(q string) Status {
			switch q {
			case "den_of_evil":
				return c.den
			case RespecRow:
				if c.respec {
					return Started
				}
			}
			return Needed
		}
		d := &FakeDialog{Status: st}
		_ = d.Talk(bg, Akara)
		err := PickPath(bg, d, Akara, st, c.path...)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("case %d: err %v, want %v", i, err, c.wantErr)
		}
		if want := fmt.Sprintf("choose 0/%d Talk", c.wantRow); d.Calls[1] != want {
			t.Errorf("case %d: first click %q, want %q", i, d.Calls[1], want)
		}
	}
}
