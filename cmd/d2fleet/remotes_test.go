package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/fleet"
	"d2lod-bot/internal/server"
	"d2lod-bot/internal/viewport"
)

// fakeBots reports a bot address per slot and lets the test move it (a bot restart on a new port).
type fakeBots struct {
	mu     sync.Mutex
	report func(fleet.BotStatus)
}

func (b *fakeBots) Run(ctx context.Context, a fleet.Alloc, s fleet.Spec, report func(fleet.BotStatus)) {
	b.mu.Lock()
	b.report = report
	b.mu.Unlock()
	<-ctx.Done()
}

func (b *fakeBots) ready() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.report != nil
}

func (b *fakeBots) set(addr string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	st := fleet.BotStatus{State: fleet.BotRunning, Addr: addr}
	if addr == "" {
		st.State = fleet.BotRestarting
	}
	b.report(st)
}

type jpegCap struct{ display chan string }

func (jpegCap) Persistent() bool { return false }
func (c jpegCap) Open(_ context.Context, s viewport.Spec) (viewport.Session, error) {
	select {
	case c.display <- s.Display:
	default:
	}
	return jpegSess{}, nil
}

type jpegSess struct{}

func (jpegSess) Close() error { return nil }
func (jpegSess) Next(context.Context) (viewport.Frame, error) {
	return viewport.Frame{Data: []byte{0xFF, 0xD8, 1, 0xFF, 0xD9}, Mime: "image/jpeg", At: time.Now()}, nil
}

func fakeBotServer(name string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status" {
			io.WriteString(w, `{"mode":"running","runner":{"state":"running","activity":"fighting","character":{"name":"`+name+`"}},"lastSeq":9}`)
			return
		}
		io.WriteString(w, name+":"+r.URL.Path)
	}))
}

func TestSlotsBecomeInstances(t *testing.T) {
	bots := &fakeBots{}
	f, err := fleet.New(fleet.Config{Runtime: nopRT{}, Bots: bots, Stagger: time.Millisecond, DataDir: t.TempDir(), Limits: fleet.Limits{MaxLoadFactor: -1}})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close(context.Background())
	if _, err := f.Add(context.Background(), fleet.Spec{Character: "Tcont"}); err != nil {
		t.Fatal(err)
	}
	disp := make(chan string, 1)
	vh := newViewHub(jpegCap{disp})
	defer vh.Close()
	srv, err := server.New(server.Config{Addr: "127.0.0.1:0", Token: "t", Remotes: func() []server.Remote { return remotesOf(f, vh) }})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	get := func(path string) (int, string) {
		rq, _ := http.NewRequest("GET", ts.URL+path, nil)
		rq.Header.Set("Authorization", "Bearer t")
		res, err := http.DefaultClient.Do(rq)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	waitFor := func(path, want string) {
		t.Helper()
		var c int
		var b string
		for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
			if c, b = get(path); strings.Contains(b, want) {
				return
			}
		}
		t.Fatalf("%s: want %q, last %d %s", path, want, c, b)
	}

	// the slot is listed at once; while its bot has no address the API says so
	_, b := get("/api/v1/instances")
	if !strings.Contains(b, `"id":"i1"`) || !strings.Contains(b, `"title":"Tcont"`) {
		t.Fatalf("instances: %s", b)
	}
	for end := time.Now().Add(5 * time.Second); !bots.ready() && time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
	}
	if c, b := get("/api/v1/instances/i1/goals"); c != 503 || !strings.Contains(b, "bot_starting") {
		t.Fatalf("starting: %d %s", c, b)
	}

	one, two := fakeBotServer("one"), fakeBotServer("two")
	defer one.Close()
	defer two.Close()
	bots.set(strings.TrimPrefix(one.URL, "http://"))
	waitFor("/api/v1/instances/i1/goals", "one:/api/v1/goals")
	waitFor("/api/v1/fleet", `"lastSeq":9`)
	if _, b := get("/api/v1/fleet"); !strings.Contains(b, `"name":"one"`) {
		t.Fatalf("fleet: %s", b)
	}
	bots.set(strings.TrimPrefix(two.URL, "http://"))
	waitFor("/api/v1/instances/i1/goals", "two:/api/v1/goals")

	// the game view comes from the slot's own display (Alloc.Display :101 for i1)
	if c, b := get("/api/v1/instances/i1/viewport/frame.jpg"); c != 200 || len(b) != 5 {
		t.Fatalf("frame: %d %q", c, b)
	}
	if d := <-disp; d != ":101" {
		t.Fatalf("captured display %q", d)
	}
	if c, _ := get("/api/v1/instances/i1/viewport/overlay"); c != 200 {
		t.Fatalf("overlay %d", c)
	}

	// removing the slot removes the instance and closes its capture
	if err := f.Remove(context.Background(), "i1", fleet.RemoveOpts{Force: true}); err != nil {
		t.Fatal(err)
	}
	waitFor("/api/v1/instances", "[]")
	if c, _ := get("/api/v1/instances/i1/viewport/frame.jpg"); c != 404 {
		t.Fatalf("removed slot viewport: %d", c)
	}
}
