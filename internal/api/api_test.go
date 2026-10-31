package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
)

type env struct {
	t   *testing.T
	m   *goals.Manager
	r   *sim.Runner
	srv *httptest.Server
}

func setup(t *testing.T, token string) *env {
	t.Helper()
	r := sim.New(sim.Options{Seed: 1})
	m, err := goals.New(goals.Options{Runner: r})
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(Config{Manager: m, Token: token, Assets: fstest.MapFS{"index.html": {Data: []byte("<h1>hi</h1>")}}})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return &env{t, m, r, ts}
}

func (e *env) do(method, path, body string, hdr ...string) (int, []byte) {
	e.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rd)
	if method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func (e *env) json(method, path, body string, want int, out any) {
	e.t.Helper()
	code, b := e.do(method, path, body)
	if code != want {
		e.t.Fatalf("%s %s = %d, want %d: %s", method, path, code, want, b)
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			e.t.Fatalf("%s: %v", b, err)
		}
	}
}

func TestGoalCRUDAndControl(t *testing.T) {
	e := setup(t, "")
	var g goals.Goal
	e.json("POST", Prefix+"/goals", `{"type":"reach_level","params":{"level":30}}`, 201, &g)
	if g.ID == "" || g.Title != "Reach level 30" || g.State != goals.StateQueued {
		t.Fatalf("%+v", g)
	}
	var g2 goals.Goal
	e.json("POST", Prefix+"/goals", `{"type":"kill_boss","params":{"boss":"Mephisto","count":3},"priority":1}`, 201, &g2)
	var list []goals.Goal
	e.json("GET", Prefix+"/goals", "", 200, &list)
	if len(list) != 2 || list[0].ID != g2.ID {
		t.Fatalf("priority order: %+v", list)
	}
	e.json("PATCH", Prefix+"/goals/"+g.ID, `{"params":{"level":40}}`, 200, &g)
	if g.Title != "Reach level 40" {
		t.Fatalf("title should follow the params when it was auto: %q", g.Title)
	}
	e.json("POST", Prefix+"/goals/"+g.ID+"/move", `{"position":0}`, 200, &g)
	e.json("GET", Prefix+"/goals", "", 200, &list)
	if list[0].ID != g.ID {
		t.Fatalf("%+v", list)
	}
	e.json("POST", Prefix+"/goals/reorder", `{"ids":["`+g2.ID+`","`+g.ID+`"]}`, 200, &list)
	if list[0].ID != g2.ID {
		t.Fatal("reorder")
	}

	var st struct {
		Mode   string       `json:"mode"`
		Active *goals.Goal  `json:"activeGoal"`
		Runner goals.Status `json:"runner"`
	}
	e.json("POST", Prefix+"/control/start", "", 200, &st)
	if st.Mode != "running" || st.Active == nil || st.Active.ID != g2.ID || st.Runner.Character.Name != "Testy" {
		t.Fatalf("%+v", st)
	}
	e.json("POST", Prefix+"/control/pause", "", 200, &st)
	e.json("POST", Prefix+"/control/pause", "", 409, nil)
	e.json("POST", Prefix+"/control/stop", "", 200, &st)
	if st.Mode != "stopped" {
		t.Fatal(st.Mode)
	}
	e.json("POST", Prefix+"/goals/"+g.ID+"/skip", "", 200, &g)
	if g.State != goals.StateSkipped {
		t.Fatal(g.State)
	}
	e.json("POST", Prefix+"/goals/"+g.ID+"/skip", "", 409, nil)
	e.json("DELETE", Prefix+"/goals/"+g.ID, "", 204, nil)
	e.json("GET", Prefix+"/goals/"+g.ID, "", 404, nil)
}

func TestValidationErrors(t *testing.T) {
	e := setup(t, "")
	tests := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/goals", `{"type":"reach_level","params":{"level":0}}`, 400},
		{"POST", "/goals", `{"type":"nope"}`, 400},
		{"POST", "/goals", `{"type":"reach_level","params":{"level":5},"bogus":1}`, 400},
		{"POST", "/goals", `not json`, 400},
		{"POST", "/goals", ``, 400},
		{"POST", "/goals", `{"type":"reach_level","params":{"level":5}} {}`, 400},
		{"PATCH", "/goals/g404", `{"title":"x"}`, 404},
		{"POST", "/goals/g404/pause", ``, 404},
		{"POST", "/goals/g404/bogus", ``, 404},
		{"POST", "/goals/reorder", `{"ids":["x"]}`, 400},
		{"PUT", "/settings", `{"maxDeaths":-1,"difficulty":"auto"}`, 400},
		{"GET", "/stats/series?window=banana", ``, 400},
		{"GET", "/stats/series?window=30d&step=1m", ``, 400},
		{"DELETE", "/status", ``, 405},
		{"GET", "/nothing", ``, 404},
		{"POST", "/control/explode", ``, 404},
	}
	for _, tc := range tests {
		if code, b := e.do(tc.method, Prefix+tc.path, tc.body); code != tc.want {
			t.Errorf("%s %s %q = %d (%s), want %d", tc.method, tc.path, tc.body, code, b, tc.want)
		}
	}
	var eb errBody
	e.json("POST", Prefix+"/goals", `{"type":"reach_level","params":{"level":0}}`, 400, &eb)
	if eb.Code != "invalid" || !strings.Contains(eb.Error, "level") {
		t.Fatalf("%+v", eb)
	}
}

func TestSettingsStatsCatalog(t *testing.T) {
	e := setup(t, "")
	var s goals.Settings
	e.json("GET", Prefix+"/settings", "", 200, &s)
	s.MaxDeaths = 9
	e.json("PUT", Prefix+"/settings", `{"autoResume":false,"maxDeaths":9,"chickenLifePct":25,"potionLifePct":60,"difficulty":"hell"}`, 200, &s)
	if s.MaxDeaths != 9 || s.Difficulty != "hell" {
		t.Fatalf("%+v", s)
	}
	e.m.Handle(goals.Event{Type: goals.EvKill, Monster: "x", XP: 50})
	e.m.Handle(goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: "Tir Rune", Quality: "normal", Type: "rune"}})
	var series struct {
		Step   int64               `json:"step"`
		Points []goals.SeriesPoint `json:"points"`
	}
	e.json("GET", Prefix+"/stats/series?window=1h", "", 200, &series)
	var xp int64
	for _, p := range series.Points {
		xp += p.XP
	}
	if series.Step != 300 || len(series.Points) != 12 || xp != 50 {
		t.Fatalf("%+v", series)
	}
	var sum goals.StatsSummary
	e.json("GET", Prefix+"/stats", "", 200, &sum)
	if sum.Totals.Kills != 1 || len(sum.Drops) != 1 || sum.Qualities["normal"] != 1 {
		t.Fatalf("%+v", sum)
	}
	var cat goals.CatalogData
	e.json("GET", Prefix+"/catalog", "", 200, &cat)
	if len(cat.Areas) == 0 || len(cat.Bosses) == 0 {
		t.Fatal("empty catalog")
	}
	var evs []goals.Event
	e.json("GET", Prefix+"/events?limit=1", "", 200, &evs)
	if len(evs) != 1 || evs[0].Type != goals.EvDrop {
		t.Fatalf("%+v", evs)
	}
}

func TestAuthHostOrigin(t *testing.T) {
	e := setup(t, "s3cret")
	if code, _ := e.do("GET", Prefix+"/status", ""); code != 401 {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := e.do("GET", Prefix+"/status", "", "Authorization", "Bearer nope"); code != 401 {
		t.Fatalf("wrong token: %d", code)
	}
	if code, _ := e.do("GET", Prefix+"/status", "", "Authorization", "Bearer s3cret"); code != 200 {
		t.Fatalf("bearer: %d", code)
	}
	if code, _ := e.do("GET", Prefix+"/status?token=s3cret", ""); code != 200 {
		t.Fatalf("query token: %d", code)
	}
	if code, _ := e.do("GET", Prefix+"/status", "", "Cookie", "d2token=s3cret"); code != 200 {
		t.Fatalf("cookie: %d", code)
	}
	// page load with token sets the cookie and strips the token from the URL
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(e.srv.URL + "/?token=s3cret&x=1")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 303 || resp.Header.Get("Location") != "/?x=1" || !strings.Contains(resp.Header.Get("Set-Cookie"), "HttpOnly") {
		t.Fatalf("%d %q %q", resp.StatusCode, resp.Header.Get("Location"), resp.Header.Get("Set-Cookie"))
	}
	if code, _ := e.do("GET", "/", ""); code != 401 {
		t.Fatalf("page without token: %d", code)
	}
	if code, b := e.do("GET", "/", "", "Cookie", "d2token=s3cret"); code != 200 || !bytes.Contains(b, []byte("<h1>")) {
		t.Fatalf("page: %d", code)
	}

	e = setup(t, "")
	auth := []string{"Authorization", ""}
	_ = auth
	if code, _ := e.do("POST", Prefix+"/control/start", "", "Origin", "http://evil.example"); code != 403 {
		t.Fatalf("cross-origin: %d", code)
	}
	if code, _ := e.do("POST", Prefix+"/control/start", "", "Origin", e.srv.URL); code != 200 {
		t.Fatalf("same-origin: %d", code)
	}
	if code, _ := e.do("POST", Prefix+"/control/stop", "", "Content-Type", "text/plain"); code != 415 {
		t.Fatalf("simple-request content type must be refused: %d", code)
	}
	// DNS rebinding: a loopback listener refuses foreign Host headers
	req, _ := http.NewRequest("GET", e.srv.URL+Prefix+"/status", nil)
	req.Host = "evil.example"
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		t.Fatalf("foreign host: %d", resp.StatusCode)
	}
}

func TestNonLoopbackNeedsToken(t *testing.T) {
	m, _ := goals.New(goals.Options{Runner: sim.New(sim.Options{})})
	if _, err := New(Config{Manager: m, Addr: "0.0.0.0:8420"}); err == nil {
		t.Fatal("expected refusal")
	}
	if _, err := New(Config{Manager: m, Addr: "0.0.0.0:8420", Token: "x"}); err != nil {
		t.Fatal(err)
	}
	for addr, want := range map[string]bool{"127.0.0.1:1": true, "[::1]:1": true, "localhost:1": true, ":8420": false, "192.0.2.5:1": false} {
		if isLoopbackAddr(addr) != want {
			t.Errorf("%s", addr)
		}
	}
}

type sseMsg struct{ id, name, data string }

func readSSE(t *testing.T, resp *http.Response, ch chan<- sseMsg) {
	sc := bufio.NewScanner(resp.Body)
	var cur sseMsg
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if cur.name != "" {
				ch <- cur
			}
			cur = sseMsg{}
		case strings.HasPrefix(line, "id: "):
			cur.id = line[4:]
		case strings.HasPrefix(line, "event: "):
			cur.name = line[7:]
		case strings.HasPrefix(line, "data: "):
			cur.data = line[6:]
		}
	}
	close(ch)
}

func TestSSEStream(t *testing.T) {
	e := setup(t, "")
	e.m.Add(goals.Goal{Type: goals.ReachLevel, Params: goals.Params{Level: 99}}, -1)
	e.m.Start()
	e.m.Handle(goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 3})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", e.srv.URL+Prefix+"/events/stream", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatal(ct)
	}
	ch := make(chan sseMsg, 100)
	go readSSE(t, resp, ch)

	seen := map[string]int{}
	var liveID string
	deadline := time.After(5 * time.Second)
	gotLive, fired := false, false
	for !(gotLive && seen["status"] > 0 && seen["goals"] > 0) {
		select {
		case m, ok := <-ch:
			if !ok {
				t.Fatal("stream closed")
			}
			seen[m.name]++
			switch m.name {
			case "status":
				var s goals.Snapshot
				if err := json.Unmarshal([]byte(m.data), &s); err != nil || s.Mode != goals.ModeRunning {
					t.Fatalf("status %v %s", err, m.data)
				}
			case "log":
				var ev goals.Event
				json.Unmarshal([]byte(m.data), &ev)
				if m.id == "" || m.id != itoa(ev.Seq) {
					t.Fatalf("event id %q vs seq %d", m.id, ev.Seq)
				}
				if ev.Type == goals.EvDeath {
					gotLive, liveID = true, m.id
				}
			}
			if seen["status"] > 0 && !fired && seen["log"] >= 4 {
				fired = true
				e.m.Handle(goals.Event{Type: goals.EvDeath, Area: "Blood Moor"}) // arrives live
			}
		case <-deadline:
			t.Fatalf("timed out; saw %v", seen)
		}
	}
	if seen["log"] < 5 { // goal_added, control, goal_started, kill (replayed) + the live death
		t.Fatalf("saw %v", seen)
	}

	// resume after a disconnect: only newer events are replayed
	cancel()
	e.m.Handle(goals.Event{Type: goals.EvPotion, Potion: "Healing Potion"})
	req2, _ := http.NewRequest("GET", e.srv.URL+Prefix+"/events/stream", nil)
	req2.Header.Set("Last-Event-ID", liveID)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	resp2, err := http.DefaultClient.Do(req2.WithContext(ctx2))
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()
	ch2 := make(chan sseMsg, 100)
	go readSSE(t, resp2, ch2)
	select {
	case m := <-ch2:
		var ev goals.Event
		json.Unmarshal([]byte(m.data), &ev)
		if m.name != "log" || ev.Type != goals.EvPotion {
			t.Fatalf("first message after resume: %+v", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no replay")
	}
}

func itoa(n uint64) string { b, _ := json.Marshal(n); return string(b) }
