package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
)

func codexRig(t *testing.T, class string) (CodexReader, *decide.Tracker, *decide.ManualClock) {
	t.Helper()
	tbl, err := decide.ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	if cat, err = cat.WithFirsts(tbl); err != nil {
		t.Fatal(err)
	}
	clk := &decide.ManualClock{}
	st, err := decide.New(cat, "p1", clk)
	if err != nil {
		t.Fatal(err)
	}
	cx := decide.NewCodex("p1")
	tr, err := decide.NewTracker(tbl, cat, st, cx, clk, decide.Who{Class: class, Diff: "normal"})
	if err != nil {
		t.Fatal(err)
	}
	return CodexReader{Codex: cx, Table: tbl, Clock: clk}, tr, clk
}

func feed(t *testing.T, tr *decide.Tracker, clk *decide.ManualClock) {
	t.Helper()
	for _, s := range []struct {
		at time.Duration
		ev decide.FirstEvent
	}{
		{2 * time.Minute, decide.FirstEvent{Kind: "monster", Name: "Fallen Shaman", Area: "Blood Moor", Clvl: 2}},
		{13 * time.Minute, decide.FirstEvent{Kind: "monster", Name: "Corpsefire", Area: "Den of Evil", Clvl: 4}},
		{25 * time.Minute, decide.FirstEvent{Kind: "waypoint", Name: "Cold Plains", Clvl: 5}},
	} {
		clk.T = s.at
		if _, err := tr.Observe(s.ev); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCodexAPI_Golden(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		rd, tr, clk := codexRig(t, class)
		feed(t, tr, clk)
		clk.T = 30 * time.Minute
		h := CodexHandler(rd)
		get := func(path string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			return w
		}
		w := get("/api/codex")
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
		golden(t, "codex-"+class, w.Body.String())
		var v CodexView
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil || v.Filled != 3 || v.Total < 30 || len(v.Recent) != 3 {
			t.Fatalf("view %+v (%v)", v, err)
		}
		// window of 10 minutes: only the waypoint is within [20, 30]
		if v := CodexViewOf(rd, 10*time.Minute); len(v.Recent) != 1 || !strings.HasPrefix(v.Line, "1 new first: Cold Plains waypoint") {
			t.Fatalf("window view %+v", v)
		}
		for _, bad := range []string{"/api/codex?window=x", "/api/codex?window=-3"} {
			if get(bad).Code != 400 {
				t.Fatalf("%s accepted", bad)
			}
		}
		pw := httptest.NewRecorder()
		h.ServeHTTP(pw, httptest.NewRequest(http.MethodPost, "/api/codex", nil))
		if pw.Code != 405 {
			t.Fatalf("POST %d", pw.Code)
		}
	}
}

func TestCodexSection(t *testing.T) {
	rd, tr, clk := codexRig(t, "sorceress")
	if got := CodexSection(rd.Summary(0)); got != "0 of "+strconv.Itoa(rd.Summary(0).Total)+" Act 1 entries" {
		t.Fatalf("empty: %q", got)
	}
	feed(t, tr, clk)
	got := CodexSection(rd.Summary(0))
	if !strings.HasPrefix(got, "3 new firsts: Fallen Shaman, Corpsefire, Cold Plains waypoint (3 of ") {
		t.Fatalf("section %q", got)
	}
}
