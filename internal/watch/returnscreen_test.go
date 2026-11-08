package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
)

var updateReturn = os.Getenv("UPDATE_RETURN") != ""

func retGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := "testdata/returnscreen/" + name
	if updateReturn {
		os.MkdirAll("testdata/returnscreen", 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Errorf("%s differs from the golden file (UPDATE_RETURN=1 rewrites it)\n got: %s\nwant: %s", name, got, want)
	}
}

// A recorded-style stream per build: two town trips, a death, a unique, a boss, a level. Area ids 1 and 40 are towns.
const evSorc = `{"t":10,"kind":"area_change","area":"Blood Moor","areaId":2}
{"t":20,"kind":"kill","monster":"Fallen","xp":30}
{"t":25,"kind":"kill","monster":"Fallen","xp":30}
{"t":30,"kind":"potion","potion":"healing"}
{"t":35,"kind":"drop","item":"Wirt's Leg","quality":"normal","type":"mace"}
{"t":40,"kind":"drop","item":"Tyrael's Might","quality":"unique","type":"armor"}
{"t":45,"kind":"death","monster":"Corpsefire","area":"Den of Evil"}
{"t":50,"kind":"area_change","area":"Rogue Encampment","areaId":1}
{"t":60,"kind":"area_change","area":"Cold Plains","areaId":3}
{"t":70,"kind":"kill","monster":"Blood Raven","boss":true,"xp":900}
{"t":75,"kind":"level_up","level":9}
{"t":80,"kind":"drop","item":"Ort Rune","quality":"normal","type":"rune"}
{"t":85,"kind":"quest","quest":"Sisters to the Slaughter","done":true}
{"t":90,"kind":"close","lifePct":9,"area":"Cold Plains"}
`
const evPala = `{"t":12,"kind":"area_change","area":"Stony Field","areaId":4}
{"t":22,"kind":"kill","monster":"Quill Rat","xp":50}
{"t":33,"kind":"drop","item":"Sigon's Gage","quality":"set","type":"gloves"}
{"t":44,"kind":"level_up","level":14}
{"t":55,"kind":"level_up","level":15}
`

const dropKey = "t0.retreat.drink|act=1,champbonus=none,diff=normal,drinklife=26-49,hc=normal,held=hp,kit=OK"

func fixtureJournal() []JournalLine {
	return []JournalLine{
		{ID: "L1", T: 31, Kind: "question", Key: dropKey, Text: "Question: Life is low. When: Life fell to 30% in Blood Moor, with 4 healing potions in the belt and 3 enemies close. Meanwhile I: drink a potion."},
		{ID: "L2", T: 46, Kind: "question", Key: dropKey, Text: "Question: Life is low. When: Life fell to 12% in Den of Evil, with 0 healing potions in the belt and 3 enemies close. Meanwhile I: drink a potion."},
		{ID: "L3", T: 47, Kind: "missed", Key: dropKey, Text: "Missed: Life is low."},
		{ID: "L4", T: 48, Kind: "missed", Key: "t0.retreat.drink|act=2,champbonus=none,diff=normal,drinklife=26-49,hc=normal,held=hp,kit=OK", Text: "Missed: Life is low."},
		{ID: "L5", T: 62, Kind: "missed", Key: "t0.retreat.repeat|act=1,diff=normal,hc=normal", Text: "Missed: retreat"},
		{ID: "L6", T: 20000, Kind: "missed", Key: dropKey, Text: "outside the window"},
		{ID: "X1", T: 45, Kind: "incident", Text: "The hero died in Den of Evil."},
	}
}

func input(t *testing.T, class, events string) ReturnInput {
	t.Helper()
	evs, err := ReadAwayEvents(strings.NewReader(events))
	if err != nil {
		t.Fatal(err)
	}
	h := newSimHub(t, strings.ToLower(class))
	return ReturnInput{Journal: fixtureJournal(), Events: evs, Since: 0, Now: 14400, Hero: HeroInfo{Class: class, Level: 9, Area: "Cold Plains"},
		Cat: h.cfg.Cat, OpenAsk: map[string]int{}, OpenRule: map[string]int{}}
}

func newSimHub(t *testing.T, build string) *Hub {
	t.Helper()
	s, err := NewSim(build, 1)
	if err != nil {
		t.Fatal(err)
	}
	return s.Hub
}

func TestReturnGoldenTwoBuilds(t *testing.T) {
	for _, c := range []struct{ class, events, file string }{{"Sorceress", evSorc, "sorceress"}, {"Paladin", evPala, "paladin"}} {
		t.Run(c.file, func(t *testing.T) {
			d := BuildReturn(input(t, c.class, c.events))
			j, _ := json.MarshalIndent(d, "", " ")
			retGolden(t, c.file+".json", append(j, '\n'))
			retGolden(t, c.file+".html", RenderReturn(d, false))
		})
	}
}

func TestReturnContent(t *testing.T) {
	d := BuildReturn(input(t, "Sorceress", evSorc))
	if !strings.Contains(d.Headline, "Sorceress is level 9 in Cold Plains") || !strings.Contains(d.Headline, "2 expeditions, 1 death") {
		t.Errorf("headline: %q", d.Headline)
	}
	if len(d.Trips) != 2 || d.Trips[0].Zones[0] != "Cold Plains" || d.Trips[1].Kills != 2 || d.Trips[1].XP != 60 || d.Trips[1].Potions != 1 {
		t.Errorf("trips: %+v", d.Trips)
	}
	kinds := map[string]string{}
	for _, m := range d.Moments {
		kinds[m.Kind] = m.Text
	}
	if !strings.Contains(kinds["close"], "9%") { // the close event (9) beats the journal's 12
		t.Errorf("closest call: %q", kinds["close"])
	}
	if !strings.Contains(kinds["drop"], "Tyrael's Might") || !strings.Contains(kinds["boss"], "Blood Raven") || kinds["level"] == "" || kinds["stats"] == "" {
		t.Errorf("moments: %v", kinds)
	}
	if len(d.Deaths) != 1 || !strings.Contains(d.Deaths[0], "Corpsefire") {
		t.Errorf("deaths: %v", d.Deaths)
	}
	if len(d.Missed) != 2 || d.Missed[0].Count != 2 || d.Missed[0].Point != "t0.retreat.drink" || len(d.Missed[0].Choices) < 2 {
		t.Errorf("missed (one group per point, outside the window dropped): %+v", d.Missed)
	}
	if len(d.Sections) != 2 || d.Sections[0].ID != "proposals" || d.Sections[1].ID != "tally" {
		t.Errorf("placeholders: %+v", d.Sections)
	}
}

func TestReturnJournalOnlyFallsBack(t *testing.T) {
	in := input(t, "Paladin", "")
	d := BuildReturn(in)
	if len(d.Deaths) != 1 || len(d.Trips) != 0 || !strings.Contains(strings.Join(d.Gaps, " "), "No event stream") {
		t.Errorf("fallback: %+v", d)
	}
	if !strings.Contains(d.Headline, "1 death") {
		t.Errorf("headline: %q", d.Headline)
	}
}

func TestReturnSectionRegistryReplaces(t *testing.T) {
	r := DefaultSections()
	r.Register("tally", SectionFunc(func(d *ReturnData) *SectionView {
		return &SectionView{Title: "Tally", After: true, Lines: []string{"plus 12 tally"}}
	}))
	r.Register("policy", SectionFunc(func(*ReturnData) *SectionView { return nil })) // hidden
	in := input(t, "Sorceress", evSorc)
	in.Sections = r
	d := BuildReturn(in)
	if len(d.Sections) != 2 || d.Sections[1].Lines[0] != "plus 12 tally" || d.Sections[1].ID != "tally" {
		t.Errorf("sections: %+v", d.Sections)
	}
	if h := string(RenderReturn(d, true)); !strings.Contains(h, "plus 12 tally") || strings.Contains(h, "Tally is not switched on") {
		t.Error("registered section not rendered")
	}
}

func TestReturnHTMLEscapes(t *testing.T) {
	d := BuildReturn(ReturnInput{Now: 100, Events: []AwayEvent{{T: 1, Kind: AwayArea, Area: "<script>x</script>", AreaID: 2}}})
	if h := string(RenderReturn(d, false)); strings.Contains(h, "<script>x") {
		t.Error("unescaped text")
	}
}

type retRig struct {
	hub  *Hub
	sim  *Sim
	cfg  ReturnConfig
	wall time.Time
	h    http.Handler
}

func newRetRig(t *testing.T, build, token string) *retRig {
	t.Helper()
	s, err := NewSim(build, 1)
	if err != nil {
		t.Fatal(err)
	}
	r := &retRig{hub: s.Hub, sim: s, wall: time.Unix(1_000_000, 0)}
	log := s.Hub.AwayLog()
	log.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2})
	log.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 30})
	log.OnGoalEvent(goals.Event{Type: goals.EvInfo}) // dropped
	r.cfg = ReturnConfig{Hub: s.Hub, Events: log, Visit: NewVisit("", func() time.Time { return r.wall }), Speed: 1}
	live := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("LIVE")) })
	r.h = tokenGate(token, r.cfg.ReturnHandler(live))
	return r
}

func (r *retRig) get(path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.h.ServeHTTP(w, req)
	return w
}

func TestReturnServedBehindToken(t *testing.T) {
	for _, b := range []string{"sorceress", "paladin"} {
		t.Run(b, func(t *testing.T) {
			r := newRetRig(t, b, strings.Repeat("w", 8))
			for _, p := range []string{"/watch/return", "/api/return", "/watch/return.js", "/watch/return.css", "/watch"} {
				if w := r.get(p, ""); w.Code != 401 {
					t.Errorf("%s without token: %d", p, w.Code)
				}
			}
			w := r.get("/watch/return", strings.Repeat("w", 8))
			if w.Code != 200 || !strings.Contains(w.Body.String(), "While you were away") || !strings.Contains(w.Header().Get("Content-Security-Policy"), "script-src 'self'") {
				t.Fatalf("page: %d %v", w.Code, w.Header())
			}
			if strings.Contains(w.Body.String(), "<script>") || strings.Contains(w.Body.String(), "style=") {
				t.Error("inline code breaks the CSP")
			}
			var d ReturnData
			w = r.get("/api/return", strings.Repeat("w", 8))
			if err := json.Unmarshal(w.Body.Bytes(), &d); err != nil || d.Schema != returnSchema || len(d.Trips) != 1 || d.Trips[0].Kills != 1 {
				t.Errorf("api: %v %+v", err, d)
			}
			post := httptest.NewRecorder()
			r.h.ServeHTTP(post, httptest.NewRequest("POST", "/api/return", nil))
			if post.Code != 401 && post.Code != 405 {
				t.Errorf("POST: %d", post.Code)
			}
			post = httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/return", nil)
			req.Header.Set("Authorization", "Bearer "+strings.Repeat("w", 8))
			r.h.ServeHTTP(post, req)
			if post.Code != 405 {
				t.Errorf("the viewer must be read-only: %d", post.Code)
			}
		})
	}
}

func TestReturnLandingAfterAbsence(t *testing.T) {
	r := newRetRig(t, "sorceress", "")
	if w := r.get("/watch", ""); w.Body.String() != "LIVE" { // never seen: a first start is not a return
		t.Errorf("first visit: %d %q", w.Code, w.Body.String())
	}
	r.get("/watch?live=1", "") // a look
	r.wall = r.wall.Add(19 * time.Minute)
	if w := r.get("/watch", ""); w.Code != 200 {
		t.Errorf("19 min: %d", w.Code)
	}
	r.wall = r.wall.Add(2 * time.Minute) // 21 min since the look
	w := r.get("/watch", "")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/watch/return" {
		t.Errorf("21 min: %d %v", w.Code, w.Header())
	}
	// the same absence redirects once: the next visits reach the live page until a look and a new absence (TASK-178)
	for i := 0; i < 3; i++ {
		r.wall = r.wall.Add(time.Minute)
		if w := r.get("/watch", ""); w.Code != 200 {
			t.Errorf("repeat %d in the same absence: %d", i, w.Code)
		}
	}
	if w := r.get("/watch?live=1", ""); w.Body.String() != "LIVE" {
		t.Error("?live=1 must reach the live screen")
	}
	if w := r.get("/watch", ""); w.Code != 200 {
		t.Error("the look resets the absence")
	}
	r.wall = r.wall.Add(21 * time.Minute) // a second absence lands again, once
	if w := r.get("/watch", ""); w.Code != http.StatusFound {
		t.Errorf("second absence: %d", w.Code)
	}
	if w := r.get("/watch", ""); w.Code != 200 {
		t.Errorf("second absence repeat: %d", w.Code)
	}
	if w := r.get("/watch/return?hours=4", ""); w.Code != 200 { // the explicit view never needs an absence
		t.Error("explicit view")
	}
}

func TestReturnPersistentVisit(t *testing.T) {
	p := t.TempDir() + "/visit.json"
	now := time.Unix(5000, 0)
	NewVisit(p, func() time.Time { return now }).Touch()
	now = now.Add(time.Hour)
	if d, ok := NewVisit(p, func() time.Time { return now }).Away(); !ok || d != time.Hour {
		t.Errorf("away: %v %v", d, ok)
	}
}

func TestReturnAnswerOnceForAll(t *testing.T) {
	for _, b := range []string{"sorceress", "paladin"} {
		t.Run(b, func(t *testing.T) {
			r := newRetRig(t, b, "")
			if _, err := r.hub.GiveOrder(OrderHere, "", 200); err != nil {
				t.Fatal(err)
			}
			var th []int
			for i := 0; i < 400 && len(th) == 0; i++ {
				r.sim.Step(5 * time.Second)
				for _, o := range r.hub.cfg.Store.OpenThoughts() {
					if !o.Confirm {
						th = append(th, o.ID)
					}
				}
			}
			if len(th) == 0 {
				t.Fatal("no thought opened")
			}
			info, _ := r.hub.cfg.Store.Thought(th[0])
			ans := r.cfg.ReturnAnswerHandler(AnswerOptions{Origins: []string{origin}})
			do := func(path, body string) (int, string) {
				req := httptest.NewRequest("POST", path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", origin)
				w := httptest.NewRecorder()
				ans.ServeHTTP(w, req)
				return w.Code, w.Body.String()
			}
			if c, _ := do("/api/return/answer-kind", `{"point":"`+info.Point+`","answer":"nonsense"}`); c != 400 {
				t.Errorf("bad choice: %d", c)
			}
			c, body := do("/api/return/answer-kind", `{"point":"`+info.Point+`","answer":"`+info.Choices[0].ID+`"}`)
			if c != 200 || !strings.Contains(body, `"answered":`) {
				t.Fatalf("answer-kind: %d %s", c, body)
			}
			if c, _ := do("/api/return/answer-kind", `{"point":"`+info.Point+`","answer":"`+info.Choices[0].ID+`"}`); c != 404 {
				t.Errorf("none open any more: %d", c)
			}
			if c, _ := do("/api/return/answer-kind", `{"point":"x","answer":"a","yes":true}`); c != 400 {
				t.Errorf("both: %d", c)
			}
			if c, _ := do("/api/return/notes", `{"text":"fine"}`); c != 501 {
				t.Errorf("no record: %d", c)
			}
			req := httptest.NewRequest("POST", "/api/return/notes", strings.NewReader(`{"text":"x"}`))
			req.Header.Set("Origin", "http://evil.example")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			ans.ServeHTTP(w, req)
			if w.Code != 403 {
				t.Errorf("foreign origin: %d", w.Code)
			}
		})
	}
}

type notesBox struct{ got string }

func (n *notesBox) SaveSessionNotes(t string) error { n.got = t; return nil }

func TestReturnNotesFeedTheRecord(t *testing.T) {
	r := newRetRig(t, "sorceress", "")
	nb := &notesBox{}
	r.cfg.Notes = nb
	ans := r.cfg.ReturnAnswerHandler(AnswerOptions{})
	req := httptest.NewRequest("POST", "/api/return/notes", strings.NewReader(`{"text":"Good day. Two deaths. Keep going."}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ans.ServeHTTP(w, req)
	if w.Code != 200 || nb.got != "Good day. Two deaths. Keep going." {
		t.Errorf("notes: %d %q", w.Code, nb.got)
	}
}

// The return and policy pages reuse app.css, whose `main` is a 12-column grid at >= 900 px; main.ret must reset it
// (screenshot 2026-11-08: ~13 columns 25 px wide). Visual check not possible here; this pins the rule and the class.
func TestReturnLayoutResetsMainGrid(t *testing.T) {
	if !strings.Contains(returnCSS, "main.ret{display:block;grid-template-columns:none;") {
		t.Fatal("return.css lost the main.ret grid reset")
	}
	for _, name := range []string{"sorceress.html", "paladin.html"} {
		b, err := os.ReadFile("testdata/returnscreen/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `<main class="ret"`) || !strings.Contains(string(b), "/watch/return.css") {
			t.Fatalf("%s: main.ret or return.css link missing", name)
		}
	}
	b, _ := os.ReadFile("testdata/policysheet/sheet-sorceress.html")
	if !strings.Contains(string(b), `<main class="ret"`) || !strings.Contains(string(b), "/watch/return.css") {
		t.Fatal("policy page does not use main.ret with return.css")
	}
}
