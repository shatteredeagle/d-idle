package watch

import (
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// slowInv is an inventory desk whose view takes as long as the test says and counts its builds.
type slowInv struct {
	fakeInv
	gate   chan struct{}
	builds atomic.Int32
}

func (s *slowInv) View() any {
	s.builds.Add(1)
	<-s.gate
	return map[string]any{"schema": 1, "gold": 77, "stashGold": 5}
}

// /api/hero is read by five panels every two seconds. It must answer at once whatever the inventory desk does: a
// request that waited for the desk's view (it judges every carried item) held a browser connection, and with the
// video and event streams open the whole page starved in its loading state (live regression 2026-11-11).
func TestHeroNeverWaitsForTheInventoryView(t *testing.T) {
	r := newRig(t, "paladin")
	inv := &slowInv{gate: make(chan struct{})}
	api := NewInvAPI()
	api.Bind(inv)
	r.sim.Hub.SetGoldSource(goldFromInventory(api))
	get := func() HeroView {
		t.Helper()
		done := make(chan HeroView, 1)
		go func() {
			var v HeroView
			r.get(HeroRoute, &v)
			done <- v
		}()
		select {
		case v := <-done:
			return v
		case <-time.After(2 * time.Second):
			t.Fatal("/api/hero waited for the inventory view")
			return HeroView{}
		}
	}
	for i := 0; i < 20; i++ { // the view is still being built: no gold yet, and no pile of builds
		if v := get(); v.Gold != nil {
			t.Fatalf("gold before the first view: %v", *v.Gold)
		}
	}
	if n := inv.builds.Load(); n != 1 {
		t.Errorf("%d inventory builds for 20 hero polls while one is in flight", n)
	}
	close(inv.gate) // the desk answers
	deadline := time.Now().Add(2 * time.Second)
	for {
		if v := get(); v.Gold != nil && *v.Gold == 77 && *v.StashGold == 5 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the gold never arrived")
		}
		time.Sleep(10 * time.Millisecond)
	}
	before := inv.builds.Load()
	for i := 0; i < 50; i++ {
		get()
	}
	if n := inv.builds.Load() - before; n > 1 {
		t.Errorf("%d inventory builds for 50 hero polls inside one TTL", n)
	}
	// the inventory page's own route still gets a fresh view after a write drops the cache
	api.drop()
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/inventory", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"gold":77`) {
		t.Errorf("/api/inventory: %d %s", w.Code, w.Body.String())
	}
}

// The page keeps its long-lived connections few (a browser has six per host and the video takes one): one event
// stream shared by app.js and the shell, and one request per address in the shell's scheduler.
func TestWatchPageConnectionBudget(t *testing.T) {
	app, _ := webFS.ReadFile("web/app.js")
	sh, _ := webFS.ReadFile("web/shell.js")
	if strings.Count(string(app), "new EventSource(") != 1 || !strings.Contains(string(app), "window.d2events = new EventSource('/api/events')") {
		t.Error("app.js must open the page's one event stream and publish it as window.d2events")
	}
	if strings.Count(string(sh), "new EventSource(") != 1 || !strings.Contains(string(sh), "window.d2events || (window.d2events = new EventSource('/api/events'))") {
		t.Error("shell.js must reuse window.d2events and only open a stream when there is none")
	}
	for _, name := range []string{"alerts.js", "unlockui.js", "update.js", "inventory.js", "layoutedit.js", "layout.js"} {
		if b, err := webFS.ReadFile("web/" + name); err == nil && strings.Contains(string(b), "EventSource(") {
			t.Errorf("web/%s opens another event stream", name)
		}
	}
	if !strings.Contains(string(sh), "flying.get(url)") || !strings.Contains(string(sh), "flying.delete(url)") {
		t.Error("the scheduler must share one request per address")
	}
}

// The scheduler in node: shared requests, every panel leaves the skeleton after the first poll cycle, a hanging
// endpoint does not starve the others.
func TestShellSchedulerJS(t *testing.T) {
	runNode(t, "testdata/shell_sched.js", "web/shell.js")
}
