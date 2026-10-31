package server

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"d2lod-bot/internal/event"
	"d2lod-bot/internal/fleet"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
	"d2lod-bot/internal/status"
)

type fixture struct {
	t   *testing.T
	srv *httptest.Server
	s   *Server
	reg *StaticRegistry
	bus map[string]*event.Bus
}

func mgr(t *testing.T, seed int64) *goals.Manager {
	t.Helper()
	m, err := goals.New(goals.Options{Runner: sim.New(sim.Options{Seed: seed})})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// newFixture serves two simulated instances: a (with a Bus, hub, viewport mount) and b (flat manager log only).
func newFixture(t *testing.T, token string, extra ...func(*Config)) *fixture {
	t.Helper()
	reg := &StaticRegistry{}
	hub := status.NewHub()
	hub.Register(status.Panel{ID: "p", Title: "P", Snapshot: func() any { return map[string]int{"x": 1} }})
	bus := event.NewBus("a", nil)
	err := reg.Add(&Instance{ID: "a", Title: "Alpha", Manager: mgr(t, 1), Bus: bus, Hub: hub,
		Deaths: func() any { return []string{"d1"} },
		Mounts: map[string]http.Handler{"viewport": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "vp:"+r.URL.Path) })}})
	if err == nil {
		err = reg.Add(&Instance{ID: "b", Manager: mgr(t, 2), Health: func() (any, bool) { return map[string]bool{"ok": false}, false }})
	}
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Instances: reg, Token: token, Assets: fstest.MapFS{"index.html": {Data: []byte("<h1>dash</h1>")}},
		Sentinel: func() any { return map[string]string{"mode": "watch"} },
		Mounts:   []Mount{{Path: "/watch/", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "w:"+r.URL.Path) })}}}
	for _, f := range extra {
		f(&cfg)
	}
	s, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(func() { ts.Close(); s.Close() })
	return &fixture{t, ts, s, reg, map[string]*event.Bus{"a": bus}}
}

func (f *fixture) do(method, path, body string, hdr ...string) (int, string, http.Header) {
	f.t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, f.srv.URL+path, rd)
	if method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := (&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}).Do(req)
	if err != nil {
		f.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), resp.Header
}

func (f *fixture) get(path string, want int) string {
	f.t.Helper()
	c, b, _ := f.do("GET", path, "")
	if c != want {
		f.t.Fatalf("GET %s = %d, want %d: %s", path, c, want, b)
	}
	return b
}

func TestInstanceListAndFleet(t *testing.T) {
	f := newFixture(t, "")
	var l []summary
	if err := json.Unmarshal([]byte(f.get("/api/v1/instances", 200)), &l); err != nil || len(l) != 2 || l[0].ID != "a" || l[0].Title != "Alpha" || !l[0].Live || l[1].OK {
		t.Fatalf("instances: %v %+v", err, l)
	}
	var fl fleetBody
	json.Unmarshal([]byte(f.get("/api/v1/fleet", 200)), &fl)
	if len(fl.Instances) != 2 || !fl.Failsafe || fl.Sentinel == nil {
		t.Fatalf("fleet: %+v", fl)
	}
	f.get("/api/v1/instances/zz/status", 404)
	f.get("/api/v1/sentinel", 200)
}

func TestInstanceScopedGoals(t *testing.T) {
	f := newFixture(t, "")
	c, b, h := f.do("POST", "/api/v1/instances/b/goals", `{"type":"reach_level","params":{"level":5}}`)
	if c != 201 || !strings.HasPrefix(h.Get("Location"), "/api/v1/instances/b/goals/") {
		t.Fatalf("add: %d %s %v", c, b, h)
	}
	count := func(path string) int {
		var gs []goals.Goal
		json.Unmarshal([]byte(f.get(path, 200)), &gs)
		return len(gs)
	}
	if count("/api/v1/instances/b/goals") != 1 || count("/api/v1/instances/a/goals") != 0 {
		t.Fatal("goals leaked between instances")
	}
	// the flat shim aliases the first instance (a), not b
	if count("/api/v1/goals") != 0 {
		t.Fatal("flat /goals must be the first instance")
	}
	c, _, _ = f.do("POST", "/api/v1/goals", `{"type":"reach_level","params":{"level":7}}`)
	if c != 201 || count("/api/v1/instances/a/goals") != 1 {
		t.Fatal("flat POST did not reach the first instance")
	}
	f.get("/api/v1/instances/a/status", 200)
	f.get("/api/v1/status", 200)
	f.get("/api/v1/instances/a/settings", 200)
}

func TestHealthDeathsStateMounts(t *testing.T) {
	f := newFixture(t, "")
	f.get("/api/v1/instances/a/health", 200)
	f.get("/api/v1/instances/b/health", 503)
	if got := f.get("/api/v1/instances/a/deaths", 200); !strings.Contains(got, "d1") {
		t.Fatal(got)
	}
	f.get("/api/v1/instances/b/deaths", 404)
	f.get("/api/v1/bot/deaths", 200) // flat alias of the first instance
	f.get("/api/v1/bot/health", 200)
	f.get("/api/v1/bot/state", 200)
	if got := f.get("/api/v1/instances/a/viewport/frame.jpg", 200); got != "vp:/frame.jpg" {
		t.Fatalf("viewport mount: %q", got)
	}
	if got := f.get("/watch/x", 200); got != "w:/x" {
		t.Fatalf("global mount: %q", got)
	}
	// healthz is process-only; readyz is false while an instance says so
	f.get("/healthz", 200)
	f.get("/readyz", 200)
	f.reg.Get("b").Ready = func() bool { return false }
	f.get("/readyz", 503)
}

func TestEventsEnvelopeAndFlatCompat(t *testing.T) {
	f := newFixture(t, "")
	bus := f.bus["a"]
	bus.Publish(event.New(event.KindLevelUp, event.SrcObserver, event.LevelUp{Level: 5}))
	bus.Publish(event.New(event.KindKill, event.SrcObserver, event.Kill{Monster: "Andariel"}))
	var evs []event.Event
	json.Unmarshal([]byte(f.get("/api/v1/instances/a/events", 200)), &evs)
	if len(evs) != 2 || evs[0].Seq != 1 || evs[1].Kind != event.KindKill || evs[1].Instance != "a" {
		t.Fatalf("events: %+v", evs)
	}
	json.Unmarshal([]byte(f.get("/api/v1/instances/a/events?since=1", 200)), &evs)
	if len(evs) != 1 || evs[0].Seq != 2 {
		t.Fatalf("since: %+v", evs)
	}
	// instance b has no bus: envelopes are derived from the manager log, tagged with the instance
	f.do("POST", "/api/v1/instances/b/control/start", "")
	json.Unmarshal([]byte(f.get("/api/v1/instances/b/events", 200)), &evs)
	if len(evs) == 0 || evs[0].Instance != "b" || evs[0].Kind != event.KindControl {
		t.Fatalf("derived events: %+v", evs)
	}
	// the flat route keeps the old goals.Event shape
	var old []goals.Event
	json.Unmarshal([]byte(f.get("/api/v1/events", 200)), &old)
	if len(old) != 0 { // instance a manager has no events of its own
		t.Fatalf("flat: %+v", old)
	}
}

type sseReader struct {
	sc *bufio.Scanner
}

// next returns the next (event name, id, data).
func (r *sseReader) next(t *testing.T) (name, id, data string) {
	t.Helper()
	for r.sc.Scan() {
		l := r.sc.Text()
		switch {
		case l == "":
			if name != "" {
				return
			}
		case strings.HasPrefix(l, "event: "):
			name = l[7:]
		case strings.HasPrefix(l, "id: "):
			id = l[4:]
		case strings.HasPrefix(l, "data: "):
			data = l[6:]
		}
	}
	t.Fatal("stream ended")
	return
}

func openSSE(t *testing.T, f *fixture, path string, hdr ...string) (*sseReader, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	req, _ := http.NewRequestWithContext(ctx, "GET", f.srv.URL+path, nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		cancel()
		t.Fatalf("sse %s: %v %v", path, err, resp)
	}
	return &sseReader{bufio.NewScanner(resp.Body)}, func() { cancel(); resp.Body.Close() }
}

func TestSSEBusReplayAndLive(t *testing.T) {
	f := newFixture(t, "")
	bus := f.bus["a"]
	for i := 1; i <= 3; i++ {
		bus.Publish(event.New(event.KindLevelUp, event.SrcObserver, event.LevelUp{Level: i}))
	}
	r, stop := openSSE(t, f, "/api/v1/instances/a/events/stream", "Last-Event-ID", "1")
	defer stop()
	for _, want := range []string{"2", "3"} {
		n, id, d := r.next(t)
		var e event.Event
		if n != "log" || id != want || json.Unmarshal([]byte(d), &e) != nil || e.Instance != "a" || e.Kind != event.KindLevelUp {
			t.Fatalf("replay %s: %s %s %s", want, n, id, d)
		}
	}
	if n, _, _ := r.next(t); n != "status" {
		t.Fatalf("want status, got %s", n)
	}
	if n, _, _ := r.next(t); n != "goals" {
		t.Fatalf("want goals, got %s", n)
	}
	bus.Publish(event.New(event.KindDeath, event.SrcObserver, event.Death{}))
	for {
		n, id, _ := r.next(t)
		if n == "log" {
			if id != "4" {
				t.Fatalf("live id %s", id)
			}
			return
		}
	}
}

func TestFleetStreamTagsInstance(t *testing.T) {
	f := newFixture(t, "")
	r, stop := openSSE(t, f, "/api/v1/fleet/events/stream")
	defer stop()
	if n, _, _ := r.next(t); n != "fleet" {
		t.Fatalf("first event %s", n)
	}
	f.bus["a"].Publish(event.New(event.KindKill, event.SrcObserver, event.Kill{Monster: "x"}))
	for {
		n, _, d := r.next(t)
		if n == "log" {
			var e event.Event
			json.Unmarshal([]byte(d), &e)
			if e.Instance != "a" || e.Kind != event.KindKill {
				t.Fatalf("fleet log: %s", d)
			}
			return
		}
	}
}

func TestLiveHubPerInstance(t *testing.T) {
	f := newFixture(t, "")
	if got := f.get("/instances/a/live/api/snapshot", 200); !strings.Contains(got, `"id":"p"`) {
		t.Fatal(got)
	}
	f.get("/live/api/snapshot", 200) // first instance's hub
	f.get("/instances/b/live/api/snapshot", 404)
	if c, _, h := f.do("GET", "/instances/a/live", ""); c != 301 || h.Get("Location") != "/instances/a/live/" {
		t.Fatalf("redirect %d %v", c, h)
	}
}

func TestSecurityOnNewRoutes(t *testing.T) {
	f := newFixture(t, "sek")
	paths := []string{"/api/v1/instances", "/api/v1/instances/a/status", "/api/v1/instances/a/events", "/api/v1/instances/a/events/stream",
		"/api/v1/instances/a/health", "/api/v1/instances/a/viewport/x", "/api/v1/fleet", "/api/v1/sentinel", "/api/v1/status",
		"/api/v1/bot/health", "/instances/a/live/api/snapshot", "/live/api/snapshot", "/watch/x", "/"}
	for _, p := range paths {
		if c, _, _ := f.do("GET", p, ""); c != 401 {
			t.Errorf("%s without token: %d", p, c)
		}
		if c, _, _ := f.do("GET", p, "", "Authorization", "Bearer nope"); c != 401 {
			t.Errorf("%s bad token: %d", p, c)
		}
	}
	for _, h := range [][]string{{"Authorization", "Bearer sek"}, {"X-Token", "sek"}, {"Cookie", "d2token=sek"}} {
		if c, _, _ := f.do("GET", "/api/v1/instances/a/status", "", h...); c != 200 {
			t.Errorf("auth via %s: %d", h[0], c)
		}
	}
	if c, _, _ := f.do("GET", "/api/v1/fleet?token=sek", ""); c != 200 {
		t.Errorf("query token on API: %d", c)
	}
	// open health endpoints
	f.get("/healthz", 200)
	f.get("/readyz", 200)
	// page load with ?token= sets the cookie and strips the token
	c, _, h := f.do("GET", "/instances/a/live/?token=sek", "")
	if c != 303 || !strings.Contains(h.Get("Set-Cookie"), "d2token=sek") || strings.Contains(h.Get("Location"), "token") {
		t.Errorf("cookie bootstrap: %d %v", c, h)
	}
	// but a data feed under /live/api/ never redirects
	if c, _, _ := f.do("GET", "/instances/a/live/api/snapshot?token=sek", ""); c != 200 {
		t.Errorf("feed with query token: %d", c)
	}
}

func TestHostOriginContentTypeCSP(t *testing.T) {
	f := newFixture(t, "", func(c *Config) { c.AllowedHosts = []string{"d2.example"} })
	if c, _, _ := f.do("GET", "/api/v1/instances", "", "Host-Test", "x"); c != 200 {
		t.Fatal(c)
	}
	req, _ := http.NewRequest("GET", f.srv.URL+"/api/v1/instances/a/status", nil)
	req.Host = "evil.example"
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 403 {
		t.Errorf("bad Host: %d", resp.StatusCode)
	}
	req.Host = "d2.example"
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("allowed Host: %d", resp.StatusCode)
	}
	if c, _, _ := f.do("POST", "/api/v1/instances/a/control/start", "", "Origin", "http://evil.example"); c != 403 {
		t.Errorf("cross-origin POST: %d", c)
	}
	req, _ = http.NewRequest("POST", f.srv.URL+"/api/v1/instances/a/control/start", nil)
	req.Header.Set("Content-Type", "text/plain")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 415 {
		t.Errorf("content-type: %d", resp.StatusCode)
	}
	_, _, h := f.do("GET", "/api/v1/instances/a/status", "")
	if !strings.Contains(h.Get("Content-Security-Policy"), "default-src 'self'") || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("headers: %v", h)
	}
	// mutating a read-only route
	if c, _, _ := f.do("POST", "/api/v1/instances/a/events", ""); c != 405 {
		t.Errorf("POST events: %d", c)
	}
}

func TestNewValidation(t *testing.T) {
	if _, err := New(Config{Addr: "0.0.0.0:0"}); err == nil {
		t.Error("non-loopback without token must be refused")
	}
	if _, err := New(Config{Addr: "0.0.0.0:0", Token: "t"}); err != nil {
		t.Error(err)
	}
	if _, err := New(Config{Mounts: []Mount{{Path: "/", Handler: http.NotFoundHandler()}}}); err == nil {
		t.Error("mount at / must be refused")
	}
	r := &StaticRegistry{}
	m := mgr(t, 1)
	if r.Add(&Instance{ID: "Bad_ID", Manager: m}) == nil || r.Add(&Instance{ID: "ok", Manager: m}) != nil || r.Add(&Instance{ID: "ok", Manager: m}) == nil {
		t.Error("id validation / duplicates")
	}
}

func TestNoInstancesSentinelOnly(t *testing.T) {
	hub := status.NewHub()
	hub.Register(SentinelPanel(func() any { return "snap" }))
	f := newFixture(t, "", func(c *Config) { c.Instances = &StaticRegistry{}; c.Hub = hub })
	f.get("/api/v1/status", 404)
	f.get("/api/v1/instances", 200)
	if got := f.get("/live/api/snapshot", 200); !strings.Contains(got, "sentinel") {
		t.Fatal(got)
	}
}

func TestRingBounded(t *testing.T) {
	b := event.NewBus("x", nil)
	r := newRing(b, func(*event.Event) {})
	defer r.close()
	for i := 0; i < ringSize+50; i++ {
		b.Publish(event.New(event.KindInfo, "t", event.Message{Message: "m"}))
	}
	if got := r.since(0, 0); len(got) != ringSize || got[0].Seq != 51 {
		t.Fatalf("ring: %d first %d", len(got), got[0].Seq)
	}
	if got := r.since(0, 3); len(got) != 3 || got[2].Seq != ringSize+50 {
		t.Fatal("newest limit")
	}
}

func TestFlatForwardsInstanceMounts(t *testing.T) {
	f := newFixture(t, "")
	// the default instance (a) has a viewport mount; flat /api/v1/viewport/... reaches it, prefix stripped
	if got := f.get("/api/v1/viewport/x", 200); got != "vp:/x" {
		t.Fatalf("flat mount: %q", got)
	}
	if got := f.get("/api/v1/instances/a/viewport/x", 200); got != "vp:/x" {
		t.Fatalf("scoped mount: %q", got)
	}
	f.get("/api/v1/nonexistent", 404)
}

func TestExactMount(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "old:"+r.URL.Path) })
	f := newFixture(t, "tok", func(c *Config) { c.Mounts = append(c.Mounts, Mount{Path: "/api/old", Exact: true, Handler: h}) })
	if c, _, _ := f.do("GET", "/api/old", ""); c != 401 {
		t.Fatalf("exact mount must be guarded: %d", c)
	}
	if c, b, _ := f.do("GET", "/api/old", "", "Authorization", "Bearer tok"); c != 200 || b != "old:/api/old" {
		t.Fatalf("exact: %d %q", c, b)
	}
	if c, _, _ := f.do("GET", "/api/old/more", "", "Authorization", "Bearer tok"); c != 404 {
		t.Fatalf("exact mount must not match a subpath: %d", c)
	}
}

type nopRT struct{}

func (nopRT) Start(context.Context, fleet.Alloc, fleet.Spec) error { return nil }
func (nopRT) Login(context.Context, fleet.Alloc, fleet.Spec) error { return nil }
func (nopRT) Alive(context.Context, fleet.Alloc) (bool, error)     { return true, nil }
func (nopRT) LeaveGame(context.Context, fleet.Alloc) error         { return nil }
func (nopRT) Stop(context.Context, fleet.Alloc) error              { return nil }

func TestFleetAdminAndSlots(t *testing.T) {
	fl, err := fleet.New(fleet.Config{Runtime: nopRT{}, DataDir: t.TempDir(), Limits: fleet.Limits{MaxLoadFactor: -1}})
	if err != nil {
		t.Fatal(err)
	}
	defer fl.Close(context.Background())
	slots := func() []Slot {
		var out []Slot
		for _, v := range fl.List() {
			out = append(out, Slot{ID: v.ID, Title: v.Spec.Character, State: string(v.State)})
		}
		return out
	}
	f := newFixture(t, "tok", func(c *Config) { c.FleetAdmin = fleet.Handler(fl, ""); c.Slots = slots })
	auth := []string{"Authorization", "Bearer tok"}
	if c, _, _ := f.do("GET", "/api/v1/fleet-admin/slots", ""); c != 401 {
		t.Fatalf("admin must need the server token: %d", c)
	}
	c, b, _ := f.do("POST", "/api/v1/fleet-admin/slots", `{"character":"Navi","class":"sorceress"}`, auth...)
	if c != 201 || !strings.Contains(b, `"id":"i1"`) {
		t.Fatalf("add slot: %d %s", c, b)
	}
	if c, b, _ := f.do("GET", "/api/v1/fleet-admin/status", "", auth...); c != 200 || !strings.Contains(b, "capacity") {
		t.Fatalf("status: %d %s", c, b)
	}
	c, b, _ = f.do("GET", "/api/v1/fleet", "", auth...)
	var body fleetBody
	json.Unmarshal([]byte(b), &body)
	var found *summary
	for k := range body.Instances {
		if body.Instances[k].ID == "i1" {
			found = &body.Instances[k]
		}
	}
	if c != 200 || len(body.Slots) != 1 || found == nil || found.Title != "Navi" || found.Slot == "" || found.Runner.State == "" || len(body.Instances) != 3 {
		t.Fatalf("fleet slots: %d %s", c, b)
	}
	// the plain instance list does not contain slots without a manager
	var l []summary
	json.Unmarshal([]byte(func() string { _, b, _ := f.do("GET", "/api/v1/instances", "", auth...); return b }()), &l)
	if len(l) != 2 {
		t.Fatalf("instances: %+v", l)
	}
}

func TestEventsCarryActiveGoal(t *testing.T) {
	f := newFixture(t, "")
	m := f.reg.Get("a").Manager
	g, err := m.Add(goals.Goal{Type: "reach_level", Params: goals.Params{Level: 9}}, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if m.Snapshot().ActiveGoalID != g.ID {
		t.Skip("goal did not become active: " + m.Snapshot().ActiveGoalID)
	}
	r, stop := openSSE(t, f, "/api/v1/instances/a/events/stream")
	defer stop()
	f.bus["a"].Publish(event.New(event.KindKill, event.SrcObserver, event.Kill{Monster: "Fallen"}))
	for {
		n, _, d := r.next(t)
		if n != "log" {
			continue
		}
		var e event.Event
		json.Unmarshal([]byte(d), &e)
		if e.Kind != event.KindKill {
			continue
		}
		if e.Goal != g.ID {
			t.Fatalf("live event goal %q, want %q", e.Goal, g.ID)
		}
		break
	}
	var evs []event.Event
	json.Unmarshal([]byte(f.get("/api/v1/instances/a/events", 200)), &evs)
	for _, e := range evs {
		if e.Kind == event.KindKill && e.Goal != g.ID {
			t.Fatalf("history event goal %q, want %q", e.Goal, g.ID)
		}
	}
}

func TestSetupMountAndPage(t *testing.T) {
	var gotPath string
	setupH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { gotPath = r.URL.Path; io.WriteString(w, "setup") })
	f := newFixture(t, "tok", func(c *Config) {
		c.Setup = setupH
		c.Assets = fstest.MapFS{"index.html": {Data: []byte("d")}, "setup.html": {Data: []byte("<h1>wizard</h1>")}}
	})
	get := func(path, tok string) (int, string) {
		req, _ := http.NewRequest("GET", f.srv.URL+path, nil)
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	if c, b := get("/api/setup/state", "tok"); c != 200 || b != "setup" || gotPath != "/state" {
		t.Errorf("%d %q %q", c, b, gotPath)
	}
	if c, _ := get("/api/setup/state", ""); c != 401 && c != 403 {
		t.Errorf("setup API must need the token, got %d", c)
	}
	if c, b := get("/setup.html", "tok"); c != 200 || !strings.Contains(b, "wizard") {
		t.Errorf("%d %q", c, b)
	}
}

func TestSoakEndpoint(t *testing.T) {
	root := t.TempDir()
	write := func(run, name, body string, mt time.Time) {
		d := filepath.Join(root, run)
		os.MkdirAll(d, 0o755)
		p := filepath.Join(d, name)
		os.WriteFile(p, []byte(body), 0o644)
		os.Chtimes(p, mt, mt)
	}
	now := time.Now()
	write("old", "alerts.json", `{"level":"crit","alerts":[]}`, now.Add(-time.Hour))
	write("new", "alerts.json", `{"updated":"2026-10-29T10:00:00Z","level":"warn","alerts":[{"level":"warn","code":"slow","msg":"kills low"}],"status":"from json"}`, now.Add(-time.Minute))
	write("new", "status.txt", "\n  run new: 12 areas, 3 deaths \n", now.Add(-time.Minute))

	off := newFixture(t, "")
	off.get("/api/v1/soak", 404)
	f := newFixture(t, "", func(c *Config) { c.SoakDir = root })
	body := f.get("/api/v1/soak", 200)
	var v SoakView
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatal(err)
	}
	if v.Run != "new" || v.Level != "warn" || len(v.Alerts) != 1 || v.Alerts[0].Msg != "kills low" || v.Status != "run new: 12 areas, 3 deaths" || v.AgeSec < 50 || v.AgeSec > 120 {
		t.Fatalf("%+v", v)
	}

	// Broken json: unknown level with an error, never a failure.
	write("newer", "alerts.json", `{not json`, now)
	v, ok := LoadSoak(root, now)
	if !ok || v.Run != "newer" || v.Level != "unknown" || v.Error == "" {
		t.Fatalf("%+v", v)
	}
	if _, ok := LoadSoak(filepath.Join(root, "nope"), now); ok {
		t.Fatal("missing root found")
	}
}
