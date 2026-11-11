package watch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/senses"
)

func sensesFixture(t *testing.T) *senses.Publisher {
	t.Helper()
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####"),
		Exits: []d2map.Exit{{Dest: 2, Pos: d2map.Point{X: 2, Y: 1}}}}
	p := &senses.Publisher{}
	if err := p.Observe(lm, "Test", d2map.Point{X: 2, Y: 1}); err != nil {
		t.Fatal(err)
	}
	return p
}

func getReply(t *testing.T, h http.Handler, url string) (int, SensesReply) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
	var r SensesReply
	json.Unmarshal(w.Body.Bytes(), &r)
	return w.Code, r
}

func TestSensesTierGate(t *testing.T) {
	p := sensesFixture(t)
	for tier, wantMap := range map[int]bool{0: false, 1: true, 3: true} {
		h := (&Senses{Src: p, Lock: FixedTier(tier)}).Handler()
		code, r := getReply(t, h, "/api/senses")
		if code != 200 || (r.Map != nil) != wantMap || (r.Hero != nil) != wantMap {
			t.Fatalf("tier %d: %d %+v", tier, code, r)
		}
		if len(r.Tiers) != tier {
			t.Fatalf("tier %d unlocked %v", tier, r.Tiers)
		}
	}
	// a locked tier is absent from the bytes, not empty
	w := httptest.NewRecorder()
	(&Senses{Src: p, Lock: FixedTier(0)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	if strings.Contains(w.Body.String(), `"map"`) || strings.Contains(w.Body.String(), `"hero"`) {
		t.Fatal(w.Body.String())
	}
	// nil Unlocker is closed
	if _, r := getReply(t, (&Senses{Src: p}).Handler(), "/api/senses"); r.Map != nil {
		t.Fatal("nil unlocker opened the map")
	}
}

func TestSensesVersionCacheAndWaiting(t *testing.T) {
	p := sensesFixture(t)
	h := (&Senses{Src: p, Lock: FixedTier(1)}).Handler()
	_, a := getReply(t, h, "/api/senses")
	if a.Map == nil || a.Version == 0 {
		t.Fatal("no map")
	}
	_, b := getReply(t, h, "/api/senses?v="+strconv.FormatUint(a.Version, 10))
	if b.Map != nil || b.Hero == nil {
		t.Fatalf("cached caller got the map again: %+v", b)
	}
	_, w := getReply(t, (&Senses{Src: &senses.Publisher{}, Lock: FixedTier(1)}).Handler(), "/api/senses")
	if !w.Waiting || w.Map != nil {
		t.Fatalf("%+v", w)
	}
}

func TestSensesGETOnly(t *testing.T) {
	h := (&Senses{Src: sensesFixture(t), Lock: FixedTier(1)}).Handler()
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(m, "/api/senses", nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: %d", m, w.Code)
		}
	}
}

func TestSensesServedBehindToken(t *testing.T) {
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sv, err := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: "tok",
		Senses: &Senses{Src: sensesFixture(t), Lock: FixedTier(1)}})
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + sv.View.String()
	for _, p := range SensesRoutes {
		if resp := mustGet(t, base+p); resp.StatusCode != 401 {
			t.Fatalf("%s without token: %d", p, resp.StatusCode)
		}
	}
	req, _ := http.NewRequest("GET", base+"/api/senses", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("with token: %v %v", resp, err)
	}
	var out SensesReply
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Map == nil || out.Map.AreaID != 1 || len(out.Map.Exits) != 1 {
		t.Fatalf("%+v", out)
	}
	if resp := mustGet(t, base+"/watch/senses_units.js?t=tok"); resp.StatusCode != 200 {
		t.Fatal("senses_units.js", resp.StatusCode)
	}
	if resp := mustGet(t, base+"/watch/panels/map/panel.js?t=tok"); resp.StatusCode != 200 {
		t.Fatal("map panel", resp.StatusCode)
	}
}

type trailSrc struct {
	SensesSource
	pts []senses.TrailPoint
	evs []senses.Event
	now time.Time
}

func (s trailSrc) EventsOf() []senses.Event { return s.evs }

func (s trailSrc) TrailOf() ([]senses.TrailPoint, int, time.Time, bool) { return s.pts, 1, s.now, true }

// The trail is additive: present with ages relative to ts, absent for a source without one and under a closed tier.
func TestSensesTrailPayload(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	src := trailSrc{SensesSource: sensesFixture(t), now: now, pts: []senses.TrailPoint{
		{At: now.Add(-30 * time.Second), X: 1, Y: 1, Area: 1, Entry: true},
		{At: now.Add(-1500 * time.Millisecond), X: 3, Y: 1, Area: 1, Act: senses.ActCombat},
	}, evs: []senses.Event{{At: now.Add(-1 * time.Second), X: 3, Y: 1, Area: 1, Kind: senses.EvKill, Text: "Fallen Shaman killed", Big: false,
		Fight: &senses.FightInfo{End: "clear", Kills: 1, MinLife: 88, Life: 92, Sec: 2.1}}}}
	w := httptest.NewRecorder()
	(&Senses{Src: src, Lock: FixedTier(1)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	golden(t, "senses-trail", w.Body.String())
	var m map[string]json.RawMessage
	json.Unmarshal(w.Body.Bytes(), &m)
	want := `[{"ago":30000,"x":1,"y":1,"e":true},{"ago":1500,"x":3,"y":1,"a":1}]`
	if string(m["trail"]) != want || string(m["trail_area"]) != "1" || string(m["ts"]) != strconv.FormatInt(now.UnixMilli(), 10) {
		t.Fatalf("trail %s area %s ts %s", m["trail"], m["trail_area"], m["ts"])
	}
	if _, r := getReply(t, (&Senses{Src: struct{ SensesSource }{sensesFixture(t)}, Lock: FixedTier(1)}).Handler(), "/api/senses"); r.Trail != nil || r.Ts != 0 {
		t.Fatal("a source without a trail adds no keys")
	}
	w = httptest.NewRecorder()
	(&Senses{Src: src, Lock: FixedTier(0)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	if strings.Contains(w.Body.String(), "trail") {
		t.Fatal("closed tier leaks the trail")
	}
}

type coverSrc struct {
	SensesSource
	c  senses.Coverage
	ok bool
}

func (s coverSrc) CoverageOf() (senses.Coverage, bool) { return s.c, s.ok }

// The coverage key is additive: golden for a roaming level, absent when there is none or the tier is closed.
func TestSensesCoveragePayload(t *testing.T) {
	c := senses.Coverage{Area: 1, Cell: 8, Cover: 5, Total: 3, Drawn: 3, Seen: 1, Failed: 1, Legs: 2, Held: 1, Target: &senses.Point{X: 3, Y: 1},
		Cells: [][3]int{{1, 1, senses.CoverSeen}, {2, 1, senses.CoverFailed}, {3, 1, senses.CoverUnseen}}}
	get := func(src SensesSource, tier int) string {
		w := httptest.NewRecorder()
		(&Senses{Src: src, Lock: FixedTier(tier)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
		return w.Body.String()
	}
	base := struct{ SensesSource }{sensesFixture(t)}
	golden(t, "senses-coverage", get(coverSrc{SensesSource: base, c: c, ok: true}, 1))
	if strings.Contains(get(coverSrc{SensesSource: base}, 1), "coverage") || strings.Contains(get(coverSrc{SensesSource: base, c: c, ok: true}, 0), "coverage") {
		t.Fatal("coverage leaked")
	}
	c.Explored = true
	if !strings.Contains(get(coverSrc{SensesSource: base, c: c, ok: true}, 1), `"state":"explored"`) {
		t.Fatal("explored state")
	}
}

type histSrc struct {
	SensesSource
	s   []senses.Sample
	now time.Time
}

func (h histSrc) HeatOf() ([]senses.HeatPoint, int) {
	return []senses.HeatPoint{{X: 0, Y: 0, Kills: 3, DwellS: 12, Dmg: 40}, {X: 7, Y: 0, DwellS: 2}}, 1
}

func (h histSrc) SamplesOf() ([]senses.Sample, time.Time) { return h.s, h.now }

func TestSensesHistoryEndpoint(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	src := histSrc{SensesSource: sensesFixture(t), now: now, s: []senses.Sample{
		{At: now.Add(-10 * time.Second), Life: 100, Mana: 80, Area: 1, Act: senses.ActWalk, Why: "walk", Level: 5, XP: 900},
		{At: now.Add(-2 * time.Second), Life: 62, Mana: 40, Area: 1, Act: senses.ActCombat, Why: "fight", Level: 5, XP: 960}}}
	get := func(tier int) string {
		w := httptest.NewRecorder()
		(&Senses{Src: src, Lock: FixedTier(tier)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/history", nil))
		return w.Body.String()
	}
	golden(t, "history", get(1))
	hw := httptest.NewRecorder()
	(&Senses{Src: src, Lock: FixedTier(1)}).Handler().ServeHTTP(hw, httptest.NewRequest("GET", "/api/history?heat=1", nil))
	golden(t, "history-heat", hw.Body.String())
	if strings.Contains(get(1), "heat") {
		t.Fatal("heat is only sent on request")
	}
	if strings.Contains(get(0), `"l":`) {
		t.Fatal("a closed tier must not leak samples")
	}
	if g := get(1); !strings.Contains(g, `"samples":[{`) {
		t.Fatal(g)
	}
}
