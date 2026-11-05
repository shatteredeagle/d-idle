package watch

import (
	"bufio"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// golden compares a body with testdata/<name>.json; -update rewrites it.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := "testdata/" + name + ".json"
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Errorf("%s differs from the golden file\n got: %s\nwant: %s", name, got, want)
	}
}

// goldenRig is a fixed scenario on the virtual clock: a Sorceress, a fight order, the first question open, one
// answer, a second question open, an idle line.
func goldenRig(t *testing.T) *rig {
	r := newRig(t, "sorceress")
	r.fight()
	th := r.nextThought()
	r.must("/api/answer", map[string]any{"thought": th.ID, "answer": "drink"})
	r.nextThought()
	r.sim.Hub.Say("It is quiet in camp.")
	return r
}

func TestGoldenViews(t *testing.T) {
	r := goldenRig(t)
	for _, c := range []struct{ name, path string }{
		{"thoughts", "/api/thoughts"}, {"journal", "/api/journal"}, {"rules", "/api/rules"}, {"uptime", "/api/uptime"},
		{"orders", "/api/orders"}, {"config", "/api/config"},
	} {
		golden(t, c.name, r.get(c.path, nil).Body.String())
	}
	var j JournalView
	r.get("/api/journal", &j)
	for _, l := range j.Lines {
		if l.Kind == "answer" {
			golden(t, "why", r.get("/api/why?line="+l.ID, nil).Body.String())
		}
	}
}

func TestGoldenRuleViews(t *testing.T) {
	r := newRig(t, "paladin")
	r.fight()
	answerUntilRule(r, "portal")
	golden(t, "thoughts-confirm", r.get("/api/thoughts", nil).Body.String())
}

func TestSSEStream(t *testing.T) {
	old := SSEKeepAlive
	SSEKeepAlive = 40 * time.Millisecond
	defer func() { SSEKeepAlive = old }()
	r := newRig(t, "sorceress")
	srv := httptest.NewServer(r.view)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	next := func(prefix string) string {
		t.Helper()
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					t.Fatalf("stream ended before %q", prefix)
				}
				if strings.HasPrefix(l, prefix) {
					return l
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("no %q line", prefix)
			}
		}
	}
	seq := r.sim.Hub.Seq()
	if got := next("data:"); got != `data: {"schema":"watch.event/1","seq":`+itoa(seq)+`}` {
		t.Errorf("hello: %s", got)
	}
	next(": keepalive")
	r.must("/api/orders", map[string]any{"kind": "fight", "minutes": 5}) // changes the Hub: order given, order active
	if got := next("event:"); got != "event: change" {
		t.Errorf("event line: %s", got)
	}
	if got := next("data:"); !strings.HasPrefix(got, `data: {"schema":"watch.event/1","seq":`) {
		t.Errorf("change data: %s", got)
	}
	if r.sim.Hub.Seq() <= seq {
		t.Error("seq did not advance")
	}
	// Polling works without the stream: the seq in /api/uptime moves with every change.
	before := r.uptime().Seq
	r.sim.Hub.Say("poll me")
	if r.uptime().Seq <= before {
		t.Error("polling seq did not move")
	}
}

// TestConcurrentUseDoesNotDeadlock: readers, answerers and the stepping sim run together (store lock then Hub lock).
func TestConcurrentUseDoesNotDeadlock(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				for _, p := range []string{"/api/thoughts", "/api/journal", "/api/rules", "/api/uptime", "/api/orders"} {
					w := httptest.NewRecorder()
					r.view.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
				}
				var v ThoughtsView
				w := httptest.NewRecorder()
				r.view.ServeHTTP(w, httptest.NewRequest("GET", "/api/thoughts", nil))
				if json.Unmarshal(w.Body.Bytes(), &v) == nil && len(v.Thoughts) > 0 {
					b, _ := json.Marshal(map[string]any{"thought": v.Thoughts[0].ID, "answer": "drink"})
					r.postRaw(r.ans, "/api/answer", string(b), map[string]string{"Origin": origin})
				}
			}
		}()
	}
	finished := make(chan struct{})
	go func() {
		for i := 0; i < 400; i++ {
			r.sim.Step(5 * time.Second)
		}
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(20 * time.Second):
		t.Fatal("deadlock: the sim did not finish")
	}
	close(done)
	wg.Wait()
}

func itoa(u uint64) string {
	var b [20]byte
	i := len(b)
	for {
		i--
		b[i] = byte('0' + u%10)
		u /= 10
		if u == 0 {
			return string(b[i:])
		}
	}
}
