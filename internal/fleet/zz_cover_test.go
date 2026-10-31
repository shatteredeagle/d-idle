package fleet

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/event"
)

func mkRT(t *testing.T, rt Runtime) *Fleet {
	t.Helper()
	clk := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	f, err := New(Config{Runtime: rt, Probe: bigProbe(), Sleep: fastSleep, Clock: clk.now, DataDir: t.TempDir(), Bus: event.NewBus("host", nil)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close(context.Background()) })
	return f
}

// hungRT never returns from Start until its ctx is cancelled: a wedged wine launch.
type hungRT struct{ *fakeRT }

func (h hungRT) Start(ctx context.Context, a Alloc, s Spec) error {
	<-ctx.Done()
	return ctx.Err()
}

// loginFailRT fails Login while n > 0.
type loginFailRT struct {
	*fakeRT
	n int
}

func (l *loginFailRT) Login(ctx context.Context, a Alloc, s Spec) error {
	l.mu.Lock()
	if l.n > 0 {
		l.n--
		l.mu.Unlock()
		return errors.New("oog: no character list")
	}
	l.mu.Unlock()
	return l.fakeRT.Login(ctx, a, s)
}

func TestCoverRemoveDoesNotHangOnWedgedStart(t *testing.T) {
	f := mkRT(t, hungRT{newRT()})
	if _, err := f.Add(context.Background(), Spec{Character: "Navi"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- f.Remove(ctx, "i1", RemoveOpts{Force: true}) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Remove hangs while Start is wedged")
	}
	if len(f.List()) != 0 {
		t.Fatal("slot survived Remove")
	}
}

func TestCoverLoginFailureIsRetriedAndRecorded(t *testing.T) {
	rt := &loginFailRT{fakeRT: newRT(), n: 2}
	f := mkRT(t, rt)
	_, _ = f.Add(context.Background(), Spec{Character: "Navi"})
	wait(t, f, "i1", Running)
	v, _ := f.Get("i1")
	if v.Restarts != 2 {
		t.Fatalf("restarts %d, want 2", v.Restarts)
	}
	if !strings.Contains(v.LastError, "login") {
		t.Fatalf("last error %q", v.LastError)
	}
	// every failed bring-up must clean leftovers before the retry
	if rt.count("stop i1") < 2 {
		t.Fatalf("stop calls %d", rt.count("stop i1"))
	}
}

func TestCoverRestartUnknownSlot(t *testing.T) {
	f, _ := mk(t, newRT(), bigProbe(), nil)
	if err := f.Restart(context.Background(), "i9", "x"); !errors.Is(err, ErrNoSlot) {
		t.Fatal(err)
	}
}

func TestCoverHTTPRestartAndUnlease(t *testing.T) {
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), nil)
	srv := httptest.NewServer(Handler(f, "tok"))
	defer srv.Close()
	do := func(method, path, body string) int {
		req, _ := http.NewRequest(method, srv.URL+path, bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer tok")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	if c := do("POST", "/slots/i7/restart", `{}`); c != 404 {
		t.Fatalf("restart unknown: %d", c)
	}
	if c := do("POST", "/slots", `{"character":"Navi","class":"sorceress","role":"leader"}`); c != 201 {
		t.Fatal(c)
	}
	wait(t, f, "i1", Running)
	if c := do("POST", "/slots/i1/restart", `{"reason":"test"}`); c != 202 {
		t.Fatalf("restart: %d", c)
	}
	wait(t, f, "i1", Running)
	if c := do("POST", "/slots/i1/lease", `{"owner":"a","ttlSeconds":60}`); c != 200 {
		t.Fatal(c)
	}
	if c := do("DELETE", "/slots/i1/lease", ``); c != 204 {
		t.Fatalf("unlease: %d", c)
	}
	if v, _ := f.Get("i1"); v.LeasedBy != "" {
		t.Fatalf("still leased by %q", v.LeasedBy)
	}
	if c := do("POST", "/slots/i1/lease", `{bad`); c != 400 {
		t.Fatalf("bad lease json: %d", c)
	}
}
