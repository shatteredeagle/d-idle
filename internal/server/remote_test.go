package server

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeBot answers like a d2bot API: /api/v1/status, /api/v1/bot/health, an SSE stream and echoes what it received.
func fakeBot(t *testing.T, name string, seq uint64) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"mode":"running","runner":{"state":"running","activity":"fighting","character":{"name":"`+name+`","level":9}},"xpPerHour":1234,"lastSeq":`+
			strings.TrimSpace(strings.Repeat(" ", 0))+itoa(seq)+`,"totals":{"kills":7}}`)
	})
	mux.HandleFunc("/api/v1/bot/health", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{"ok":true,"bot":"`+name+`"}`) })
	mux.HandleFunc("/api/v1/events/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "event: status\ndata: {\"bot\":\""+name+"\"}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	})
	mux.HandleFunc("/api/v1/echo", func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		json.NewEncoder(w).Encode(map[string]string{"method": r.Method, "query": r.URL.RawQuery, "auth": r.Header.Get("Authorization"),
			"cookie": r.Header.Get("Cookie"), "body": string(b), "bot": name})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts
}

func itoa(n uint64) string { b, _ := json.Marshal(n); return string(b) }

type remoteSet struct {
	addr  atomic.Value // string
	state atomic.Value
}

func newRemoteFixture(t *testing.T, rs *remoteSet) *fixture {
	return newFixture(t, "tok", func(c *Config) {
		c.Remotes = func() []Remote {
			a, _ := rs.addr.Load().(string)
			st, _ := rs.state.Load().(string)
			return []Remote{{ID: "i1", Title: "Tcont", State: st, Addr: a,
				Mounts: map[string]http.Handler{"viewport": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "vp:"+r.URL.Path) })}}}
		}
		c.Slots = func() []Slot { return []Slot{{ID: "i1", Title: "Tcont", State: "running"}} }
	})
}

func (f *fixture) auth(path string) (int, string) {
	f.t.Helper()
	c, b, _ := f.do("GET", path, "", "Authorization", "Bearer tok")
	return c, b
}

func TestRemoteListFleetAndRestart(t *testing.T) {
	b1 := fakeBot(t, "one", 42)
	rs := &remoteSet{}
	rs.state.Store("running")
	rs.addr.Store(strings.TrimPrefix(b1.URL, "http://"))
	f := newRemoteFixture(t, rs)

	c, body := f.auth("/api/v1/instances")
	if c != 200 || !strings.Contains(body, `"id":"i1"`) || !strings.Contains(body, `"lastSeq":42`) || !strings.Contains(body, `"viewport"`) {
		t.Fatalf("list %d %s", c, body)
	}
	var fl struct {
		Instances []summary
		Failsafe  bool
	}
	_, body = f.auth("/api/v1/fleet")
	if err := json.Unmarshal([]byte(body), &fl); err != nil {
		t.Fatal(err)
	}
	var i1 *summary
	n := 0
	for k := range fl.Instances {
		if fl.Instances[k].ID == "i1" {
			i1, n = &fl.Instances[k], n+1
		}
	}
	if n != 1 || i1.Runner.Character.Name != "one" || i1.LastSeq != 42 || i1.Totals.Kills != 7 || i1.Slot != "" || !i1.OK || i1.XPPerHour != 1234 {
		t.Fatalf("fleet %+v (n=%d)", i1, n)
	}
	if c, body = f.auth("/api/v1/instances/i1"); c != 200 || !strings.Contains(body, `"name":"one"`) {
		t.Fatalf("summary %d %s", c, body)
	}
	// health maps to the bot's /bot/health, other paths go straight through
	if c, body = f.auth("/api/v1/instances/i1/health"); c != 200 || !strings.Contains(body, `"bot":"one"`) {
		t.Fatalf("health %d %s", c, body)
	}
	if c, body = f.auth("/api/v1/instances/i1/viewport/frame.jpg"); c != 200 || body != "vp:/frame.jpg" {
		t.Fatalf("viewport %d %s", c, body)
	}
	if c, _ = f.auth("/api/v1/instances/zz/health"); c != 404 {
		t.Fatalf("unknown id: %d", c)
	}

	// bot restarts on another port: the next request follows it (the status cache is keyed by address)
	b2 := fakeBot(t, "two", 5)
	rs.addr.Store(strings.TrimPrefix(b2.URL, "http://"))
	if c, body = f.auth("/api/v1/instances/i1/health"); c != 200 || !strings.Contains(body, `"bot":"two"`) {
		t.Fatalf("after restart %d %s", c, body)
	}
	if _, body = f.auth("/api/v1/instances/i1"); !strings.Contains(body, `"lastSeq":5`) {
		t.Fatalf("summary after restart %s", body)
	}

	// bot down, first with no address, then with a dead one
	rs.state.Store("restarting")
	rs.addr.Store("")
	c, body = f.auth("/api/v1/instances/i1/status")
	if c != 503 || !strings.Contains(body, "bot_starting") || !strings.Contains(body, "slot i1") {
		t.Fatalf("starting %d %s", c, body)
	}
	b2.Close()
	rs.addr.Store("127.0.0.1:1")
	if c, body = f.auth("/api/v1/instances/i1/status"); c != 503 || !strings.Contains(body, "did not answer") {
		t.Fatalf("dead bot %d %s", c, body)
	}
	_, body = f.auth("/api/v1/fleet")
	if !strings.Contains(body, `"slot":"restarting"`) || !strings.Contains(body, `"live":false`) {
		t.Fatalf("fleet with dead bot %s", body)
	}
}

func TestRemoteProxyPostStripsCredentials(t *testing.T) {
	b := fakeBot(t, "one", 1)
	rs := &remoteSet{}
	rs.state.Store("running")
	rs.addr.Store(strings.TrimPrefix(b.URL, "http://"))
	f := newRemoteFixture(t, rs)
	c, body, _ := f.do("POST", "/api/v1/instances/i1/echo?a=1&token=tok", `{"x":1}`, "Authorization", "Bearer tok", "Cookie", "d2token=tok")
	if c != 200 {
		t.Fatalf("%d %s", c, body)
	}
	var e map[string]string
	json.Unmarshal([]byte(body), &e)
	if e["method"] != "POST" || e["query"] != "a=1" || e["auth"] != "" || e["cookie"] != "" || e["body"] != `{"x":1}` {
		t.Fatalf("echo %v", e)
	}
	if c, _, _ = f.do("GET", "/api/v1/instances/i1/echo", ""); c != 401 {
		t.Fatalf("proxy must be behind the token: %d", c)
	}
}

func TestRemoteSSEProxyAndReconnect(t *testing.T) {
	b := fakeBot(t, "one", 1)
	rs := &remoteSet{}
	rs.state.Store("running")
	rs.addr.Store(strings.TrimPrefix(b.URL, "http://"))
	f := newRemoteFixture(t, rs)
	req, _ := http.NewRequest("GET", f.srv.URL+"/api/v1/instances/i1/events/stream", nil)
	req.Header.Set("Authorization", "Bearer tok")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	got := make(chan string, 2)
	go func() {
		br := bufio.NewReader(res.Body)
		for {
			l, err := br.ReadString('\n')
			if err != nil {
				return
			}
			if strings.HasPrefix(l, "data:") {
				got <- l
			}
		}
	}()
	select {
	case l := <-got:
		if !strings.Contains(l, `"bot":"one"`) {
			t.Fatal(l)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no SSE event through the proxy (flush)")
	}
}
