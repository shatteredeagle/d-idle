package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/scenario"
)

// board is a fake watch server: the view JSON is swapped by the test, the POSTs are recorded.
type board struct {
	mu   sync.Mutex
	view string
	got  []string
}

func (b *board) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		b.mu.Lock()
		defer b.mu.Unlock()
		if r.Method == http.MethodPost {
			b.got = append(b.got, r.URL.Path)
			w.WriteHeader(202)
			return
		}
		io.WriteString(w, b.view)
	})
}

func (b *board) set(v string) { b.mu.Lock(); b.view = v; b.mu.Unlock() }

func TestTapperTapsOnlyWhatIsAskedFor(t *testing.T) {
	b := &board{}
	srv := httptest.NewServer(b.handler(t))
	defer srv.Close()
	var log strings.Builder
	tp := newTapper(srv.URL, srv.URL, "tok", "search_for_cain", []string{"read"}, &log)
	ctx := context.Background()

	b.set(`{"quests":[{"id":"search_for_cain","chunks":[{"id":"scroll","state":"x","action":{"verb":"start"}}]}]}`)
	if tap, _, err := tp.Step(ctx); err != nil || tap != "" {
		t.Fatalf("scroll is not on the list: %q %v", tap, err)
	}
	b.set(`{"quests":[{"id":"search_for_cain","chunks":[{"id":"read","action":null}]},{"id":"den_of_evil","chunks":[{"id":"read","action":{"verb":"start"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "" {
		t.Fatalf("not startable or another quest: %q", tap)
	}
	b.set(`{"quests":[{"id":"search_for_cain","chunks":[{"id":"read","state":"x","action":{"verb":"start"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "start read" {
		t.Fatalf("tap %q", tap)
	}
	b.set(`{"running":{"quest":"search_for_cain","chunk":"read","state":"running"},"quests":[{"id":"search_for_cain","chunks":[{"id":"read","state":"x","action":{"verb":"cancel"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "" {
		t.Fatal("a running chunk is not tapped again")
	}
	b.set(`{"awaiting":{"quest":"search_for_cain","chunk":"read","state":"awaiting"},"quests":[{"id":"search_for_cain","chunks":[{"id":"read","state":"x","action":{"verb":"handin"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "handin read" {
		t.Fatalf("tap %q", tap)
	}
	b.set(`{"awaiting":{"quest":"search_for_cain","chunk":"read"},"handIn":{"state":"confirming"},"quests":[{"id":"search_for_cain","chunks":[{"id":"read","state":"x","action":{"verb":"handin""disabled":true}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "" {
		t.Fatal("hand-in already in progress")
	}
	b.set(`{"blocked":{"quest":"search_for_cain","chunk":"read","why":"x"}}`)
	if _, blocked, _ := tp.Step(ctx); !blocked {
		t.Fatal("a parked quest stops the tapper")
	}
	b.mu.Lock()
	got := strings.Join(b.got, ",")
	b.mu.Unlock()
	if got != "/api/quests/search_for_cain/start,/api/quests/search_for_cain/handin" || strings.Join(tp.Taps, ",") != "start read,handin read" {
		t.Fatalf("posts %s taps %v", got, tp.Taps)
	}
}

func TestTapperRunStopsWithTheContext(t *testing.T) {
	b := &board{view: `{"quests":[{"id":"q","chunks":[{"id":"a","state":"x","action":{"verb":"start"}}]}]}`}
	srv := httptest.NewServer(b.handler(t))
	defer srv.Close()
	tp := newTapper(srv.URL, srv.URL, "tok", "q", nil, io.Discard)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() { tp.Run(ctx, 20*time.Millisecond); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run ignored the context")
	}
	if len(tp.Taps) == 0 {
		t.Fatal("no tap made")
	}
}

func TestShippedBuildFieldStartIsBlockedBeforeRunning(t *testing.T) {
	scs, err := loadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err := findScenario(scs, "cain.stones.shipped")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("D2D_BIN_SANDBOX", "/x")
	why := blockedBefore(s, scenario.Run{ID: s.ID, Class: "Paladin", Seed: 11})
	if !strings.Contains(why, "field start") {
		t.Fatalf("reason %q", why)
	}
}

func TestEveryRunnableScenarioIsBuildable(t *testing.T) {
	scs, err := loadAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scs {
		if s.Pending() || s.Build == "sandbox" {
			continue
		}
		for _, c := range s.Matrix.Classes {
			su, err := scenario.BuildSetup(s, c, s.Matrix.Seeds[0])
			if err != nil || len(su.Unsupported) > 0 {
				t.Errorf("%s/%s: not buildable: %v %v", s.ID, c, err, su.Unsupported)
			}
		}
	}
}

func TestAddTrailReadsTheBotLog(t *testing.T) {
	f := t.TempDir() + "/x.bot.log"
	if err := os.WriteFile(f, []byte(`{"msg":"zone entry","area":1}`+"\n"+`{"msg":"zone entry","area":5}`+"\n"+`{"msg":"zone entry","area":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var st scenario.State
	addTrail(&st, f)
	if st.AreaChanges == nil || *st.AreaChanges != 2 || len(st.AreaTrail) != 3 {
		t.Fatalf("%+v", st)
	}
	var none scenario.State
	addTrail(&none, f+".missing")
	if none.AreaChanges != nil {
		t.Fatal("no log: not observed")
	}
}

// vboard serves /api/thoughts with a virtual clock the test advances and a board that offers the chunk from a given
// virtual second on; it records the virtual second of each POST.
type vboard struct {
	mu      sync.Mutex
	now     float64
	offerAt float64
	posts   []float64
}

func (b *vboard) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.mu.Lock()
		defer b.mu.Unlock()
		switch {
		case r.Method == http.MethodPost:
			b.posts = append(b.posts, b.now)
			w.WriteHeader(202)
		case r.URL.Path == "/api/thoughts":
			fmt.Fprintf(w, `{"now":%v}`, b.now)
		case len(b.posts) > 0:
			io.WriteString(w, `{"running":{"quest":"q","chunk":"a","state":"running"}}`)
		case b.now >= b.offerAt:
			io.WriteString(w, `{"quests":[{"id":"q","chunks":[{"id":"a","state":"x","action":{"verb":"start"}}]}]}`)
		default:
			io.WriteString(w, `{"offers":[]}`)
		}
	})
}

func (b *vboard) advance(to float64) { b.mu.Lock(); b.now = to; b.mu.Unlock() }

// The tap lands on the first grid point of VIRTUAL time at or after the offer, however fast the wall poll runs and
// however far the clock jumped between two polls.
func TestRunVirtualTapsOnTheVirtualGrid(t *testing.T) {
	for _, poll := range []time.Duration{2 * time.Millisecond, 15 * time.Millisecond} {
		b := &vboard{offerAt: 47}
		srv := httptest.NewServer(b.handler())
		tp := newTapper(srv.URL, srv.URL, "tok", "q", nil, io.Discard)
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { tp.RunVirtual(ctx, 30, poll, time.Minute); close(done) }()
		for _, v := range []float64{5, 29, 46, 47, 55, 61, 62, 100} {
			b.advance(v)
			time.Sleep(60 * time.Millisecond)
		}
		cancel()
		<-done
		srv.Close()
		if len(tp.Taps) != 1 || tp.Taps[0] != "start a" {
			t.Fatalf("poll %v: taps %v", poll, tp.Taps)
		}
		if len(tp.TapsAt) != 1 || tp.TapsAt[0] < 60 || tp.TapsAt[0] >= 90 {
			t.Fatalf("poll %v: the tap must land on the grid point 60 (first read at or after it), got %v", poll, tp.TapsAt)
		}
	}
}

func TestRunVirtualFallsBackToTheWallClock(t *testing.T) {
	b := &board{view: `{"quests":[{"id":"q","chunks":[{"id":"a","state":"x","action":{"verb":"start"}}]}]}`}
	srv := httptest.NewServer(b.handler(t)) // its GET /api/thoughts reply has no "now"
	defer srv.Close()
	var log strings.Builder
	tp := newTapper(srv.URL, srv.URL, "tok", "q", nil, &log)
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	tp.RunVirtual(ctx, 30, 5*time.Millisecond, 50*time.Millisecond)
	if !strings.Contains(log.String(), "falling back") {
		t.Fatalf("log %q", log.String())
	}
}

// A quest that is on the board but never startable says why in the taps log, once per change (card andariel1).
func TestTapperLogsWhyAChunkIsNotStartable(t *testing.T) {
	b := &board{}
	srv := httptest.NewServer(b.handler(t))
	defer srv.Close()
	var log strings.Builder
	tp := newTapper(srv.URL, srv.URL, "tok", "sisters_to_the_slaughter", []string{"andariel"}, &log)
	b.set(`{"quests":[{"id":"sisters_to_the_slaughter","state":"available","chunks":[{"id":"andariel","state":"available","readiness":"too_hard","why":"You have no merc: too hard"}]}]}`)
	for i := 0; i < 3; i++ {
		if tap, _, err := tp.Step(context.Background()); err != nil || tap != "" {
			t.Fatalf("tap %q %v", tap, err)
		}
	}
	got := log.String()
	if strings.Count(got, "not startable") != 1 || !strings.Contains(got, `readiness "too_hard"`) || !strings.Contains(got, "no merc") {
		t.Fatalf("log %q", got)
	}
}

// The Kashya case (2026-11-21): the part reads awaiting_handin with no run, so the view has no `awaiting` object; the row's
// button is Start and that is the tap, then Hand in once the run waits for it.
func TestTapperPressesTheRowsButton(t *testing.T) {
	b := &board{}
	srv := httptest.NewServer(b.handler(t))
	defer srv.Close()
	tp := newTapper(srv.URL, srv.URL, "tok", "sisters_burial_grounds", nil, io.Discard)
	ctx := context.Background()
	b.set(`{"quests":[{"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"awaiting_handin","action":{"verb":"start","label":"Start"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "start blood_raven" {
		t.Fatalf("tap %q", tap)
	}
	b.set(`{"awaiting":{"quest":"sisters_burial_grounds","chunk":"blood_raven"},"quests":[{"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"awaiting_handin","action":{"verb":"handin","label":"Hand in"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "handin blood_raven" {
		t.Fatalf("tap %q", tap)
	}
}

// necro drill (blocker 2): "*" taps every quest of the board in its own order, the row's own button, one chunk at a time, and
// logs why each skipped chunk is not startable once per change. A list taps only the named quests.
func TestTapperCampaignDriver(t *testing.T) {
	b := &board{}
	srv := httptest.NewServer(b.handler(t))
	defer srv.Close()
	var log strings.Builder
	tp := newTapper(srv.URL, srv.URL, "tok", "*", nil, &log)
	ctx := context.Background()
	b.set(`{"quests":[{"id":"den_of_evil","chunks":[{"id":"clear_den","state":"offered","action":{"verb":"start","label":"Start"}}]},
	 {"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"locked","readiness":"too hard","why":"clvl 14 of 16","action":{"verb":"start","disabled":true}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "start clear_den" {
		t.Fatalf("first tap %q", tap)
	}
	if tap, _, _ := tp.Step(ctx); tap != "start clear_den" { // still offered: the fake does not change state
		t.Fatalf("second tap %q", tap)
	}
	b.set(`{"running":{"quest":"den_of_evil","chunk":"clear_den"},"quests":[{"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"locked","readiness":"too hard","why":"clvl 14 of 16","action":{"verb":"start","disabled":true}}]}]}`)
	b.set(`{"quests":[{"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"locked","readiness":"too hard","why":"clvl 14 of 16","action":{"verb":"start","disabled":true}}]}]}`)
	for i := 0; i < 3; i++ {
		if tap, _, _ := tp.Step(ctx); tap != "" {
			t.Fatalf("a disabled button must not be tapped: %q", tap)
		}
	}
	if n := strings.Count(log.String(), "not startable sisters_burial_grounds/blood_raven"); n != 1 {
		t.Fatalf("the reason is logged once per change, got %d:\n%s", n, log.String())
	}
	// A named list leaves the other quests alone (a stage of a campaign).
	b2 := &board{}
	srv2 := httptest.NewServer(b2.handler(t))
	defer srv2.Close()
	tl := newTapper(srv2.URL, srv2.URL, "tok", "search_for_cain,den_of_evil", nil, io.Discard)
	b2.set(`{"quests":[{"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"offered","action":{"verb":"start"}}]}]}`)
	if tap, _, _ := tl.Step(ctx); tap != "" {
		t.Fatalf("a quest outside the list was tapped: %q", tap)
	}
}

// necro.act1-stage1 (2026-11-23): the driver tapped Start on the next chunk the moment it tapped Hand in, which cancelled the
// hand-in dialogue and left the Den unfinished. The next Start waits until the board no longer shows the hand-in.
func TestTapperWaitsForTheHandInBeforeTheNextStart(t *testing.T) {
	b := &board{}
	srv := httptest.NewServer(b.handler(t))
	defer srv.Close()
	tp := newTapper(srv.URL, srv.URL, "tok", "den_of_evil,sisters_burial_grounds", nil, io.Discard)
	ctx := context.Background()
	hand := `{"quests":[{"id":"den_of_evil","chunks":[{"id":"clear_den","state":"awaiting_handin","action":{"verb":"handin"}}]},
	 {"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"ready","action":{"verb":"start"}}]}]}`
	b.set(hand)
	if tap, _, _ := tp.Step(ctx); tap != "handin clear_den" {
		t.Fatalf("first tap %q", tap)
	}
	for i := 0; i < 3; i++ { // the board still shows the hand-in: no Start for the other quest
		if tap, _, _ := tp.Step(ctx); tap != "" {
			t.Fatalf("tapped %q while the hand-in is pending", tap)
		}
	}
	b.set(`{"quests":[{"id":"den_of_evil","chunks":[{"id":"clear_den","state":"done"}]},
	 {"id":"sisters_burial_grounds","chunks":[{"id":"blood_raven","state":"ready","action":{"verb":"start"}}]}]}`)
	if tap, _, _ := tp.Step(ctx); tap != "start blood_raven" {
		t.Fatalf("after the hand-in: %q", tap)
	}
}

// --- scripted taps (card pgtaps step 2): the engine decides on the bot's virtual second, with no wall clock ---

func boardWith(quest, chunk, verb string) questboard.View {
	return questboard.View{Quests: []questboard.QuestRow{{ID: quest, Chunks: []questboard.ChunkRow{{ID: chunk, Action: &questboard.ActionView{Verb: verb}}}}}}
}

func TestTapEngineTapsOnTheVirtualGridPoint(t *testing.T) {
	// two builds' quests: a caster's den and a melee's blood raven chunk (the engine is data-driven, no class in it)
	for _, c := range []struct{ quest, chunk string }{{"den_of_evil", "den"}, {"sisters_burial_grounds", "scroll"}} {
		e := scenario.NewTapEngine(scenario.TapScript{Quest: c.quest, EveryS: 30})
		var pressed []string
		press := func(verb, q, ch string) { pressed = append(pressed, verb+" "+q+"/"+ch) }
		v := boardWith(c.quest, c.chunk, "start")
		if got := e.Poll(v, 29.9, press); got != "" || len(pressed) != 0 {
			t.Fatalf("%s: tapped before the first grid point: %q", c.quest, got)
		}
		if got := e.Poll(v, 41, press); got != "start "+c.chunk {
			t.Fatalf("%s: first grid point after the offer: %q", c.quest, got)
		}
		if e.Poll(v, 44, press) != "" {
			t.Fatalf("%s: tapped twice inside one grid cell", c.quest)
		}
		if len(pressed) != 1 || e.Taps[0].At != 41 {
			t.Fatalf("%s: %v %v", c.quest, pressed, e.Taps)
		}
	}
}

func TestTapEngineRulesMatchTheHTTPTapper(t *testing.T) {
	press := func(string, string, string) {}
	// another quest is not tapped; a chunk outside the list is not tapped; "*" taps every quest
	e := scenario.NewTapEngine(scenario.TapScript{Quest: "a", Chunks: []string{"x"}, EveryS: 10})
	if e.Poll(boardWith("b", "x", "start"), 10, press) != "" || e.Poll(boardWith("a", "y", "start"), 20, press) != "" {
		t.Fatal("tapped a quest or chunk that is not wanted")
	}
	if e.Poll(boardWith("a", "x", "start"), 30, press) != "start x" {
		t.Fatal("wanted chunk not tapped")
	}
	star := scenario.NewTapEngine(scenario.TapScript{Quest: "*", EveryS: 10})
	if star.Poll(boardWith("zz", "q", "start"), 10, press) != "start q" {
		t.Fatal("* must tap any quest")
	}
	// a Start waits while a chunk runs; a hand-in is held until its button is gone
	run := boardWith("a", "x", "start")
	run.Running = &questboard.RunView{Quest: "a", Chunk: "x"}
	e = scenario.NewTapEngine(scenario.TapScript{Quest: "a", EveryS: 10})
	if e.Poll(run, 10, press) != "" {
		t.Fatal("Start tapped while a chunk runs")
	}
	hv := boardWith("a", "x", "handin")
	if e.Poll(hv, 20, press) != "handin x" {
		t.Fatal("hand-in not tapped")
	}
	next := boardWith("a", "x", "start")
	next.Quests[0].Chunks[0].State = "awaiting_handin"
	if e.Poll(next, 30, press) != "" {
		t.Fatal("Start tapped while the hand-in dialogue still waits")
	}
	next.Quests[0].Chunks[0].State = "done"
	if e.Poll(next, 40, press) != "start x" {
		t.Fatal("hold not released after the hand-in")
	}
}

func TestParseTapScript(t *testing.T) {
	s, err := scenario.ParseTapScript([]byte(`{"quest":"a,b","chunks":["x"]}`))
	if err != nil || s.EveryS != 30 || s.Quest != "a,b" {
		t.Fatalf("%+v %v", s, err)
	}
	for _, bad := range []string{`{}`, `{"quest":"a","every_s":-1}`, `{"quest":"a","nope":1}`, `x`} {
		if _, err := scenario.ParseTapScript([]byte(bad)); err == nil {
			t.Errorf("%s accepted", bad)
		}
	}
}

func TestReadScriptedTaps(t *testing.T) {
	f := t.TempDir() + "/x.taps.json"
	if taps, at := readScriptedTaps(f); taps != nil || at != nil {
		t.Fatal("a missing file must read as no taps")
	}
	if err := os.WriteFile(f, []byte(`[{"tap":"start scroll","at":30},{"tap":"handin read","at":90}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	taps, at := readScriptedTaps(f)
	if len(taps) != 2 || taps[1] != "handin read" || at[0] != 30 || at[1] != 90 {
		t.Fatalf("%v %v", taps, at)
	}
}
