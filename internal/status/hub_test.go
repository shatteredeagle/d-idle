package status

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSnapshotThenDeltas(t *testing.T) {
	h := NewHub()
	var n atomic.Int64
	h.Register(Panel{ID: "b", Title: "B", Order: 2, Snapshot: func() any { return map[string]int64{"n": n.Load()} }})
	h.Register(Panel{ID: "a", Title: "A", Order: 1, Snapshot: func() any { return "static" }})
	mux := http.NewServeMux()
	h.Mount(mux, "/live/")
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/live/api/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	var snap struct {
		Panels []struct{ ID string }
	}
	json.NewDecoder(resp.Body).Decode(&snap)
	resp.Body.Close()
	if len(snap.Panels) != 2 || snap.Panels[0].ID != "a" {
		t.Fatalf("order: %+v", snap)
	}

	req, _ := http.NewRequest("GET", srv.URL+"/live/api/events", nil)
	ev, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer ev.Body.Close()
	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(ev.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	want := func(prefix string) string {
		t.Helper()
		for {
			select {
			case l := <-lines:
				if strings.HasPrefix(l, prefix) {
					return l
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("timeout waiting for %q", prefix)
			}
		}
	}
	want("event: snapshot")
	h.Poll() // nothing changed relative to first poll? first poll pushes both (no cached value yet)
	n.Store(5)
	h.Register(Panel{ID: "b", Title: "B", Order: 2, Every: time.Millisecond, Snapshot: func() any { return map[string]int64{"n": n.Load()} }})
	h.Poll()
	want("event: update")
	h.Append("a", map[string]int{"x": 1})
	want("event: append")

	// unchanged data is not re-sent
	h.mu.Lock()
	before := h.seq
	h.mu.Unlock()
	time.Sleep(5 * time.Millisecond)
	h.Poll()
	h.mu.Lock()
	after := h.seq
	h.mu.Unlock()
	if after != before {
		t.Fatalf("unchanged poll bumped seq %d -> %d", before, after)
	}

	// static assets and manifest are served under the prefix
	m, _ := http.Get(srv.URL + "/live/manifest.json")
	var mf struct{ Modules []string }
	json.NewDecoder(m.Body).Decode(&mf)
	m.Body.Close()
	if len(mf.Modules) < 6 {
		t.Fatalf("manifest %v", mf)
	}
	if r, _ := http.Get(srv.URL + "/live/app.js"); r.StatusCode != 200 {
		t.Fatal("app.js")
	}
}
