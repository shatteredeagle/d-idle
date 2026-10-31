package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/event"
	"d2lod-bot/internal/party"
)

type fakeProbe struct {
	cores, load float64
	mem, disk   int64
}

func (p *fakeProbe) Cores() float64          { return p.cores }
func (p *fakeProbe) Load1() float64          { return p.load }
func (p *fakeProbe) MemAvailMB() int64       { return p.mem }
func (p *fakeProbe) DiskFreeMB(string) int64 { return p.disk }

func bigProbe() *fakeProbe { return &fakeProbe{cores: 16, mem: 32000, disk: 100000} }

type fakeRT struct {
	mu       sync.Mutex
	log      []string
	alive    map[string]bool
	startErr map[string]int // remaining failures
	leaveErr error
}

func newRT() *fakeRT { return &fakeRT{alive: map[string]bool{}, startErr: map[string]int{}} }

func (r *fakeRT) rec(s string) { r.log = append(r.log, s) }
func (r *fakeRT) Start(_ context.Context, a Alloc, _ Spec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec("start " + a.ID)
	if r.startErr[a.ID] > 0 {
		r.startErr[a.ID]--
		return errors.New("spawn failed")
	}
	r.alive[a.ID] = true
	return nil
}
func (r *fakeRT) Prepare(_ context.Context, a Alloc, _ Spec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec("prepare " + a.ID)
	return nil
}
func (r *fakeRT) Login(_ context.Context, a Alloc, s Spec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec("login " + a.ID + " " + s.Character)
	return nil
}
func (r *fakeRT) Alive(_ context.Context, a Alloc) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.alive[a.ID], nil
}
func (r *fakeRT) LeaveGame(_ context.Context, a Alloc) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec("leave " + a.ID)
	return r.leaveErr
}
func (r *fakeRT) Stop(_ context.Context, a Alloc) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec("stop " + a.ID)
	r.alive[a.ID] = false
	return nil
}
func (r *fakeRT) kill(id string) { r.mu.Lock(); r.alive[id] = false; r.mu.Unlock() }
func (r *fakeRT) count(s string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, l := range r.log {
		if l == s {
			n++
		}
	}
	return n
}
func (r *fakeRT) index(s string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, l := range r.log {
		if l == s {
			return i
		}
	}
	return -1
}

func fastSleep(ctx context.Context, _ time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Millisecond):
		return nil
	}
}

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

func mk(t *testing.T, rt *fakeRT, p Probe, mod func(*Config)) (*Fleet, *clock) {
	t.Helper()
	clk := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	cfg := Config{Runtime: rt, Probe: p, Sleep: fastSleep, Clock: clk.now, DataDir: t.TempDir(), Bus: event.NewBus("host", nil)}
	if mod != nil {
		mod(&cfg)
	}
	f, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close(context.Background()) })
	return f, clk
}

func wait(t *testing.T, f *Fleet, id string, st State) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := f.WaitState(ctx, id, st); err != nil {
		t.Fatal(err)
	}
}

func TestAddAllocatesAndRuns(t *testing.T) {
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), nil)
	ctx := context.Background()
	v, err := f.Add(ctx, Spec{Character: "Navi", Class: "Sorceress", Role: party.Leader})
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != "i1" || v.Alloc.HelperPort != 47013 || v.Alloc.Display != ":101" || v.Spec.Class != "sorceress" {
		t.Fatalf("%+v", v)
	}
	v2, _ := f.Add(ctx, Spec{Character: "Combi"})
	if v2.ID != "i2" || v2.Alloc.HelperPort != 47014 || v2.Spec.Role != party.Solo {
		t.Fatalf("%+v", v2)
	}
	wait(t, f, "i1", Running)
	wait(t, f, "i2", Running)
	if rt.index("start i1") > rt.index("login i1 Navi") || rt.count("login i2 Combi") != 1 {
		t.Fatal(rt.log)
	}
	// removing i1 frees its number for the next add
	if err := f.Remove(ctx, "i1", RemoveOpts{}); err != nil {
		t.Fatal(err)
	}
	v3, _ := f.Add(ctx, Spec{Character: "Towni"})
	if v3.ID != "i1" {
		t.Fatalf("freed id reused, got %s", v3.ID)
	}
}

func TestInvariants(t *testing.T) {
	f, _ := mk(t, newRT(), bigProbe(), nil)
	ctx := context.Background()
	if _, err := f.Add(ctx, Spec{Character: "Navi"}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Add(ctx, Spec{Character: "navi"}); !errors.Is(err, ErrDuplicate) {
		t.Fatal("same character twice:", err)
	}
	if _, err := f.Add(ctx, Spec{Character: "Other", ID: "i1"}); !errors.Is(err, ErrDuplicate) {
		t.Fatal(err)
	}
	for _, bad := range []Spec{{Character: "x"}, {Character: "Ok", ID: "i0"}, {Character: "Ok", Class: "mage"}, {Character: "Ok", Role: "tank"}, {Character: "Ok", Difficulty: "easy"}} {
		if _, err := f.Add(ctx, bad); err == nil {
			t.Errorf("%+v accepted", bad)
		}
	}
}

func TestAdmission(t *testing.T) {
	cases := []struct {
		name  string
		p     *fakeProbe
		want  string
		after int // add this many first
	}{
		{"cpu", &fakeProbe{cores: 3, mem: 32000, disk: 100000}, "cpu", 1}, // 1 + 2*1.05 = 3.1 > 3
		{"load", &fakeProbe{cores: 8, load: 9.5, mem: 32000, disk: 100000}, "load", 0},
		{"ram", &fakeProbe{cores: 16, mem: 1200, disk: 100000}, "ram", 0},
		{"disk", &fakeProbe{cores: 16, mem: 32000, disk: 100}, "disk", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := mk(t, newRT(), c.p, nil)
			for i := 0; i < c.after; i++ {
				if _, err := f.Add(context.Background(), Spec{Character: "Pre" + string(rune('a'+i))}); err != nil {
					t.Fatal(err)
				}
			}
			_, err := f.Add(context.Background(), Spec{Character: "Newbie"})
			var ae *AdmissionError
			if !errors.As(err, &ae) || ae.Check != c.want || !errors.Is(err, ErrAdmission) || ae.Reason == "" {
				t.Fatalf("%v", err)
			}
			if len(f.List()) != c.after {
				t.Fatal("refused add must not register a slot")
			}
			st := f.Status()
			if st.Capacity.Admit || st.Capacity.Refusal == "" {
				t.Fatalf("%+v", st.Capacity)
			}
		})
	}
	// capacity on a 4-core host: reserve 1 core, 1.05 each -> 2 instances
	f, _ := mk(t, newRT(), &fakeProbe{cores: 4, mem: 32000, disk: 100000}, nil)
	n := 0
	for i := 0; i < 5; i++ {
		if _, err := f.Add(context.Background(), Spec{Character: "Bot" + string(rune('a'+i))}); err == nil {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("admitted %d on 4 cores", n)
	}
}

func TestMaxSlots(t *testing.T) {
	f, _ := mk(t, newRT(), &fakeProbe{cores: 64, mem: 1 << 20, disk: 1 << 20}, func(c *Config) { c.Limits.MaxSlots = 2 })
	for i := 0; i < 2; i++ {
		if _, err := f.Add(context.Background(), Spec{Character: "Bot" + string(rune('a'+i))}); err != nil {
			t.Fatal(err)
		}
	}
	var ae *AdmissionError
	_, err := f.Add(context.Background(), Spec{Character: "Botz"})
	if !errors.As(err, &ae) || ae.Check != "slots" {
		t.Fatal(err)
	}
}

func TestLeases(t *testing.T) {
	f, clk := mk(t, newRT(), bigProbe(), nil)
	ctx := context.Background()
	_, _ = f.Add(ctx, Spec{Character: "Navi"})
	l1, err := f.Lease("i1", "bot", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	var le *LeasedError
	if _, err := f.Lease("i1", "debug", time.Minute); !errors.As(err, &le) || le.Holder != "bot" || !errors.Is(err, ErrLeased) {
		t.Fatal(err)
	}
	if _, err := f.Lease("i1", "bot", time.Minute); err == nil {
		t.Fatal("same owner must renew, not re-lease")
	}
	if v, _ := f.Get("i1"); v.LeasedBy != "bot" {
		t.Fatal(v)
	}
	if err := f.Remove(ctx, "i1", RemoveOpts{}); !errors.Is(err, ErrLeased) {
		t.Fatal("remove of a leased slot needs force:", err)
	}
	clk.add(59 * time.Second)
	if err := l1.Renew(time.Minute); err != nil {
		t.Fatal(err)
	}
	clk.add(61 * time.Second) // expired
	if l1.Valid() || l1.Renew(time.Minute) == nil {
		t.Fatal("expired lease must be dead")
	}
	l2, err := f.Lease("i1", "debug", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	l1.Release() // stale release must not free l2
	if !l2.Valid() {
		t.Fatal("stale release freed the new lease")
	}
	l2.Release()
	if l2.Valid() {
		t.Fatal()
	}
	if _, err := f.Lease("i9", "x", time.Minute); !errors.Is(err, ErrNoSlot) {
		t.Fatal(err)
	}
	l3, _ := f.Lease("i1", "bot", time.Minute)
	if err := f.Remove(ctx, "i1", RemoveOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	if l3.Valid() {
		t.Fatal("lease dies with the slot")
	}
}

func TestRemoveDrainsInOrder(t *testing.T) {
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), nil)
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	wait(t, f, "i1", Running)
	if err := f.Remove(context.Background(), "i1", RemoveOpts{}); err != nil {
		t.Fatal(err)
	}
	if l, s := rt.index("leave i1"), rt.index("stop i1"); l < 0 || s < l {
		t.Fatalf("leave before stop: %v", rt.log)
	}
	if len(f.List()) != 0 {
		t.Fatal("slot must be gone")
	}
	if err := f.Remove(context.Background(), "i1", RemoveOpts{}); !errors.Is(err, ErrNoSlot) {
		t.Fatal(err)
	}
}

func TestRemoveStillStopsWhenLeaveFails(t *testing.T) {
	rt := newRT()
	rt.leaveErr = errors.New("menu stuck")
	f, _ := mk(t, rt, bigProbe(), nil)
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	wait(t, f, "i1", Running)
	err := f.Remove(context.Background(), "i1", RemoveOpts{})
	if err == nil || rt.count("stop i1") == 0 || len(f.List()) != 0 {
		t.Fatalf("%v %v", err, rt.log)
	}
}

func TestSupervisedRestartWithBackoff(t *testing.T) {
	rt := newRT()
	var mu sync.Mutex
	var sleeps []time.Duration
	f, _ := mk(t, rt, bigProbe(), func(c *Config) {
		c.Backoff = Backoff{Min: 2 * time.Second, Max: 8 * time.Second, StableAfter: time.Hour}
		c.Sleep = func(ctx context.Context, d time.Duration) error {
			if d >= time.Second && d != c.PollEvery {
				mu.Lock()
				sleeps = append(sleeps, d)
				mu.Unlock()
			}
			return fastSleep(ctx, d)
		}
		c.PollEvery = 10 * time.Millisecond // distinguishable from backoff values
		c.Stagger = 10 * time.Millisecond
	})
	rt.startErr["i1"] = 2 // first two starts fail: backoff 2 s, 4 s
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	wait(t, f, "i1", Running)
	rt.kill("i1") // then a crash: another restart
	deadline := time.Now().Add(5 * time.Second)
	for rt.count("login i1 Navi") < 2 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	wait(t, f, "i1", Running)
	v, _ := f.Get("i1")
	mu.Lock()
	defer mu.Unlock()
	if len(sleeps) < 3 || sleeps[0] != 2*time.Second || sleeps[1] != 4*time.Second || sleeps[2] != 8*time.Second {
		t.Fatalf("backoff sequence %v", sleeps)
	}
	if v.Restarts != 3 {
		t.Fatalf("restarts %d", v.Restarts)
	}
}

func TestGivesUpThenRestartRevives(t *testing.T) {
	rt := newRT()
	rt.startErr["i1"] = 1000
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.MaxRestarts = 2 })
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	wait(t, f, "i1", Failed)
	v, _ := f.Get("i1")
	if v.LastError == "" || v.Restarts != 3 {
		t.Fatalf("%+v", v)
	}
	if f.Status().Capacity.Running != 0 {
		t.Fatal("failed slot holds no capacity")
	}
	rt.mu.Lock()
	rt.startErr["i1"] = 0
	rt.mu.Unlock()
	if err := f.Restart(context.Background(), "i1", "operator"); err != nil {
		t.Fatal(err)
	}
	wait(t, f, "i1", Running)
	if err := f.Remove(context.Background(), "i1", RemoveOpts{}); err != nil {
		t.Fatal(err)
	}
}

func TestEvents(t *testing.T) {
	bus := event.NewBus("host", nil)
	sub := bus.Subscribe("t", 64)
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.Bus = bus })
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	wait(t, f, "i1", Running)
	_ = f.Remove(context.Background(), "i1", RemoveOpts{})
	got := map[string]bool{}
	for len(sub.C()) > 0 {
		e := <-sub.C()
		if e.Kind != event.KindLifecycle || e.Source != SrcFleet || e.Instance != "i1" {
			t.Fatalf("%+v", e)
		}
		got[e.Payload().(event.Lifecycle).State] = true
	}
	for _, s := range []string{"slot.starting", "slot.running", "slot.draining", "slot.removed"} {
		if !got[s] {
			t.Errorf("missing %s in %v", s, got)
		}
	}
}

func TestPersistAndRestore(t *testing.T) {
	sf := filepath.Join(t.TempDir(), "slots.json")
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.StateFile = sf })
	_, _ = f.Add(context.Background(), Spec{Character: "Navi", Role: party.Killer})
	_, _ = f.Add(context.Background(), Spec{Character: "Combi"})
	_ = f.Remove(context.Background(), "i2", RemoveOpts{})
	// a new host process with a smaller machine
	f2, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.StateFile = sf })
	if errs := f2.Restore(context.Background()); len(errs) != 0 {
		t.Fatal(errs)
	}
	l := f2.List()
	if len(l) != 1 || l[0].Spec.Character != "Navi" || l[0].ID != "i1" || l[0].Spec.Role != party.Killer {
		t.Fatalf("%+v", l)
	}
}

func TestCloseKeepsSlotsForNextStart(t *testing.T) {
	sf := filepath.Join(t.TempDir(), "slots.json")
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.StateFile = sf })
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	if err := f.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	f2, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.StateFile = sf })
	if errs := f2.Restore(context.Background()); len(errs) != 0 {
		t.Fatal(errs)
	}
	if l := f2.List(); len(l) != 1 || l[0].Spec.Character != "Navi" {
		t.Fatalf("%+v", l)
	}
}

func TestHTTP(t *testing.T) {
	f, _ := mk(t, newRT(), bigProbe(), nil)
	srv := httptest.NewServer(Handler(f, "tok"))
	defer srv.Close()
	do := func(method, path, body string, auth bool) (int, map[string]any) {
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
		if auth {
			req.Header.Set("Authorization", "Bearer tok")
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var m map[string]any
		_ = json.NewDecoder(res.Body).Decode(&m)
		return res.StatusCode, m
	}
	if c, _ := do("GET", "/slots", "", false); c != 401 {
		t.Fatal(c)
	}
	if c, m := do("POST", "/slots", `{"character":"Navi","class":"sorceress","role":"leader"}`, true); c != 201 || m["id"] != "i1" {
		t.Fatal(c, m)
	}
	if c, _ := do("POST", "/slots", `{"character":"Navi"}`, true); c != 409 {
		t.Fatal(c)
	}
	if c, _ := do("POST", "/slots", `{"character":"x"}`, true); c != 422 {
		t.Fatal(c)
	}
	if c, m := do("POST", "/slots/i1/lease", `{"owner":"a","ttlSeconds":60}`, true); c != 200 || m["owner"] != "a" {
		t.Fatal(c, m)
	}
	if c, m := do("POST", "/slots/i1/lease", `{"owner":"b","ttlSeconds":60}`, true); c != 409 || m["holder"] != "a" {
		t.Fatal(c, m)
	}
	if c, _ := do("DELETE", "/slots/i1", "", true); c != 409 {
		t.Fatal("leased", c)
	}
	if c, m := do("GET", "/status", "", true); c != 200 || m["capacity"] == nil {
		t.Fatal(c, m)
	}
	if c, _ := do("DELETE", "/slots/i1?force=1", "", true); c != 204 {
		t.Fatal(c)
	}
	if c, _ := do("GET", "/slots/i1", "", true); c != 404 {
		t.Fatal(c)
	}
	// admission refusal is 503 with the reason
	f.cfg.Probe = &fakeProbe{cores: 1, mem: 32000, disk: 100000}
	if c, m := do("POST", "/slots", `{"character":"Combi"}`, true); c != 503 || m["check"] != "cpu" {
		t.Fatal(c, m)
	}
}

func TestOSProbeSmoke(t *testing.T) {
	var p OSProbe
	if p.Cores() <= 0 || p.MemAvailMB() <= 0 || p.DiskFreeMB(t.TempDir()) <= 0 || p.Load1() < 0 {
		t.Fatal("probe")
	}
}

// barrierRT's Stop blocks until n stops are in flight: it only returns when slots drain in parallel.
type barrierRT struct {
	*fakeRT
	wg sync.WaitGroup
}

func (b *barrierRT) Stop(ctx context.Context, a Alloc) error {
	b.wg.Done()
	done := make(chan struct{})
	go func() { b.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		return errors.New("stops were sequential")
	}
	return b.fakeRT.Stop(ctx, a)
}

func TestCloseDrainsSlotsInParallel(t *testing.T) {
	b := &barrierRT{fakeRT: newRT()}
	b.wg.Add(2)
	f, _ := mk(t, b.fakeRT, bigProbe(), func(c *Config) { c.Runtime = b })
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	_, _ = f.Add(context.Background(), Spec{Character: "Combi"})
	wait(t, f, "i1", Running)
	wait(t, f, "i2", Running)
	if err := f.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(f.List()) != 0 {
		t.Fatal("slots left")
	}
}
