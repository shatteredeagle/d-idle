package watch

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/goals"
)

type hallRig struct {
	desk   *HallDesk
	store  *HallStore
	facts  HallFacts
	stops  int
	stopFn func() error
	led    *currency.Ledger
}

func newHallRig(t *testing.T, class, name string) *hallRig {
	t.Helper()
	tt, err := ShippedHallTitles()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 3, 0, 0, 0, time.UTC)
	led, err := currency.Open(t.TempDir(), currency.Options{Mode: currency.EventsMode, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	r := &hallRig{store: &HallStore{Ledger: led, Titles: tt}, led: led,
		facts: HallFacts{ID: name + "-1", Name: name, Class: class, Level: 18, GameSecs: 7200, WallSecs: 3600}}
	r.desk, err = NewHallDesk(HallConfig{Store: r.store, Facts: func() HallFacts { return r.facts },
		Stop: func() error {
			r.stops++
			if r.stopFn != nil {
				return r.stopFn()
			}
			return nil
		}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *hallRig) kill(name string, n int) {
	for i := 0; i < n; i++ {
		r.desk.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: name})
	}
}

func TestHallRetireFlow(t *testing.T) {
	for _, c := range []struct{ class, name, title string }{{"sorceress", "Sorra", "Slayer of Andariel"}, {"paladin", "Paula", "Slayer of Andariel"}} {
		t.Run(c.class, func(t *testing.T) {
			r := newHallRig(t, c.class, c.name)
			r.kill("Fallen", 120)
			r.desk.OnGoalEvent(goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: "Skin of the Vipermagi", Quality: "unique"}})
			if r.desk.Pending() != nil {
				t.Fatal("question before Andariel")
			}
			r.kill("Andariel", 1)
			q := r.desk.Pending()
			if q == nil || q.Default != HallKeep || !strings.Contains(q.Text, "Retire "+c.name+" into the Hall of Fame, or keep playing?") {
				t.Fatalf("question %+v", q)
			}
			if r.stops != 0 {
				t.Fatal("stopped without an answer")
			}
			if ch, err := r.desk.Answer(HallRetire); err != nil || !ch || r.stops != 1 {
				t.Fatalf("retire: %v %v stops %d", ch, err, r.stops)
			}
			// replay: no second entry, no second stop
			if ch, err := r.desk.Answer(HallRetire); err != nil || ch || r.stops != 1 {
				t.Fatalf("replay: %v %v stops %d", ch, err, r.stops)
			}
			v, err := r.store.View()
			if err != nil || len(v.Heroes) != 1 {
				t.Fatalf("view %+v %v", v, err)
			}
			h := v.Heroes[0]
			if h.Status != "retired" || h.Title != c.title || h.Kills != 121 || h.BestItem != "unique Skin of the Vipermagi" || h.KillsBy["Fallen"] != 120 {
				t.Errorf("line %+v", h)
			}
			if got := r.desk.Title(); got != c.title {
				t.Errorf("header title %q", got)
			}
			if _, err := r.desk.Answer(HallKeep); !errors.Is(err, ErrBadHall) {
				t.Errorf("keep after retire: %v", err)
			}
		})
	}
}

func TestHallRetireStopFailureRetries(t *testing.T) {
	r := newHallRig(t, "paladin", "Paula")
	r.kill("Andariel", 1)
	r.stopFn = func() error { return errors.New("boom") }
	if _, err := r.desk.Answer(HallRetire); err == nil {
		t.Fatal("stop error swallowed")
	}
	r.stopFn = nil
	if _, err := r.desk.Answer(HallRetire); err != nil || r.stops != 2 {
		t.Fatalf("retry: %v stops %d", err, r.stops)
	}
	if v, _ := r.store.View(); len(v.Heroes) != 1 {
		t.Fatalf("entry doubled: %+v", v.Heroes)
	}
}

// Away rule: the question waits as long as it takes; nothing retires without an answer.
func TestHallNeverAutoRetires(t *testing.T) {
	r := newHallRig(t, "sorceress", "Sorra")
	r.kill("Andariel", 1)
	for i := 0; i < 500; i++ {
		r.facts.WallSecs += 3600 // days away
		r.kill("Fallen", 1)
		r.desk.Section().Build(&ReturnData{})
		r.desk.Title()
	}
	if r.stops != 0 {
		t.Fatal("auto-retired")
	}
	if v, _ := r.store.View(); len(v.Heroes) != 0 {
		t.Fatalf("hall entry without an answer: %+v", v.Heroes)
	}
	if r.desk.Pending() == nil {
		t.Fatal("question vanished")
	}
	if ch, err := r.desk.Answer(HallKeep); err != nil || !ch || r.desk.Pending() != nil || r.stops != 0 {
		t.Fatalf("keep: %v %v", ch, err)
	}
}

func TestHallDeath(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newHallRig(t, class, "Dead"+class[:2])
		r.desk.OnGoalEvent(goals.Event{Type: goals.EvDeath, Monster: "Mephisto"}) // not permanent: nothing booked
		if v, _ := r.store.View(); len(v.Heroes) != 0 {
			t.Fatal("a recoverable death entered the Hall")
		}
		r.facts.Permanent, r.facts.Cause = true, "killed by Andariel"
		r.desk.OnGoalEvent(goals.Event{Type: goals.EvDeath, Monster: "Andariel"})
		r.desk.OnGoalEvent(goals.Event{Type: goals.EvDeath, Monster: "Andariel"}) // replay
		v, _ := r.store.View()
		if len(v.Heroes) != 1 || v.Heroes[0].Status != "dead" || v.Heroes[0].Cause != "killed by Andariel" || v.Heroes[0].Deaths != 2 {
			t.Fatalf("%s: %+v", class, v.Heroes)
		}
		if !strings.Contains(v.Heroes[0].Title, "Paladin") && class == "paladin" || !strings.Contains(v.Heroes[0].Title, "Sorceress") && class == "sorceress" {
			t.Errorf("title %q", v.Heroes[0].Title)
		}
	}
}

func TestHallUnknownClassAndTitles(t *testing.T) {
	r := newHallRig(t, "witch", "Hexa")
	r.kill("Andariel", 1)
	if _, err := r.desk.Answer(HallRetire); err == nil || !strings.Contains(err.Error(), `"witch"`) {
		t.Fatalf("unknown class: %v", err)
	}
	if r.stops != 0 || r.desk.Title() != "" {
		t.Fatal("unknown class retired or titled")
	}
	for _, bad := range []string{
		`{"version":1,"classes":["paladin"],"titles":[{"id":"x","rank":1,"text":"T","classes":["druidd"],"when":{"status":"dead"}}]}`,
		`{"version":1,"classes":["paladin"],"titles":[{"id":"x","rank":1,"text":"T","when":{}}]}`,
		`{"version":1,"classes":["paladin"],"titles":[{"id":"x","rank":1,"text":"T","when":{"status":"lost"}}]}`,
		`{"version":1,"classes":["paladin"],"titles":[{"id":"x","rank":0,"text":"T","when":{"status":"dead"}}]}`,
	} {
		if _, err := ParseHallTitles([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	_, err := ParseHallTitles([]byte(`{"version":1,"classes":["paladin"],"titles":[{"id":"x","rank":1,"text":"T","classes":["druidd"],"when":{"status":"dead"}}]}`))
	if err == nil || !strings.Contains(err.Error(), `"druidd"`) || !strings.Contains(err.Error(), `"x"`) {
		t.Errorf("error must name the row and class: %v", err)
	}
	tt, _ := ShippedHallTitles()
	if n := len(tt.Titles); n < 8 || n > 12 {
		t.Errorf("%d titles", n)
	}
}

func TestHallTitleThreshold(t *testing.T) {
	tt, _ := ShippedHallTitles()
	for _, class := range []string{"sorceress", "paladin"} {
		es, _ := tt.Earned(class, "", map[string]int{"Fallen": 99})
		if len(es) != 0 {
			t.Errorf("%s: %v", class, es)
		}
		es, _ = tt.Earned(class, "", map[string]int{"Fallen": 100, "Corpsefire": 1})
		if len(es) != 2 || es[0].ID != "corpsefire" || es[1].ID != "fallen" {
			t.Errorf("%s: %v", class, es)
		}
	}
}

func TestHallStatePersists(t *testing.T) {
	r := newHallRig(t, "paladin", "Paula")
	path := t.TempDir() + "/hall-state.json"
	cfg := HallConfig{Store: r.store, Facts: func() HallFacts { return r.facts }, Stop: func() error { return nil }, StatePath: path}
	d, _ := NewHallDesk(cfg)
	for i := 0; i < 3; i++ {
		d.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen"})
	}
	d.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Andariel"}) // saves
	d2, err := NewHallDesk(cfg)
	if err != nil || d2.Pending() == nil || d2.st.Kills != 4 {
		t.Fatalf("restored %+v %v", d2.st, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func hallServer(r *hallRig, token string) http.Handler {
	view := tokenGate(token, r.desk.Handler())
	return view
}

func TestHallHTTP(t *testing.T) {
	r := newHallRig(t, "sorceress", "Sorra")
	r.kill("Andariel", 1)
	r.desk.Answer(HallRetire)
	// a second hero of another class, dead
	r.facts = HallFacts{ID: "Paula-1", Name: "Paula", Class: "paladin", Level: 31, GameSecs: 90000, WallSecs: 40000, Permanent: true, Cause: "killed by Mephisto"}
	r.desk.OnGoalEvent(goals.Event{Type: goals.EvDeath, Monster: "Mephisto"})

	h := hallServer(r, "s3cret")
	get := func(path, auth string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	for _, p := range HallRoutes {
		if w := get(p, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s without a token: %d", p, w.Code)
		}
		if w := get(p, "wrong"); w.Code != http.StatusUnauthorized {
			t.Errorf("%s wrong token: %d", p, w.Code)
		}
	}
	req := httptest.NewRequest("POST", HallRoute, nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST on the viewer: %d", w.Code)
	}
	js := get(HallRoute, "s3cret")
	if js.Code != 200 {
		t.Fatal(js.Code)
	}
	var pretty bytes.Buffer
	json.Indent(&pretty, js.Body.Bytes(), "", " ")
	golden(t, "hall", pretty.String())
	page := get(HallPage, "s3cret")
	if page.Code != 200 || !strings.Contains(page.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("page %d", page.Code)
	}
	goldenFile(t, "testdata/hall.html", page.Body.String())
}

func TestHallPostGate(t *testing.T) {
	r := newHallRig(t, "paladin", "Paula")
	r.kill("Andariel", 1)
	o := AnswerOptions{Token: "tok", Origins: []string{"http://ok"}}
	h := r.desk.PostHandler(o)
	do := func(method, ct, token, body string) int {
		req := httptest.NewRequest(method, HallPostRoute, strings.NewReader(body))
		req.Header.Set("Content-Type", ct)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	if c := do("POST", "application/json", "", `{"answer":"retire"}`); c != 401 {
		t.Errorf("no token: %d", c)
	}
	if c := do("GET", "application/json", "tok", ""); c != 405 {
		t.Errorf("GET: %d", c)
	}
	if c := do("POST", "text/plain", "tok", `{"answer":"retire"}`); c != 415 {
		t.Errorf("content type: %d", c)
	}
	if c := do("POST", "application/json", "tok", `{"answer":"quit"}`); c != 409 {
		t.Errorf("bad answer: %d", c)
	}
	if r.stops != 0 {
		t.Fatal("a refused request stopped the bot")
	}
	if c := do("POST", "application/json", "tok", `{"answer":"retire"}`); c != 200 || r.stops != 1 {
		t.Errorf("retire: %d stops %d", c, r.stops)
	}
}

func goldenFile(t *testing.T, path, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("%s differs from the golden file\n got: %s\nwant: %s", path, got, want)
	}
}
