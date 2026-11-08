package watch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/selfupdate"
)

type fakeUpdater struct {
	mu     sync.Mutex
	calls  []string
	status selfupdate.Status
	err    error
}

func (f *fakeUpdater) rec(s string)                     { f.mu.Lock(); f.calls = append(f.calls, s); f.mu.Unlock() }
func (f *fakeUpdater) Status() selfupdate.Status        { return f.status }
func (f *fakeUpdater) Check(context.Context) error      { f.rec("check"); return f.err }
func (f *fakeUpdater) Download(context.Context) error   { f.rec("download"); return f.err }
func (f *fakeUpdater) Apply(c string, force bool) error { f.rec("apply:" + c); return f.err }
func (f *fakeUpdater) Cancel()                          { f.rec("cancel") }
func (f *fakeUpdater) Rollback() error                  { f.rec("rollback"); return f.err }
func (f *fakeUpdater) Inbox() error                     { f.rec("inbox"); return f.err }
func (f *fakeUpdater) MarkSeen() error                  { f.rec("seen"); return f.err }

func serveUpdate(t *testing.T, token string, u Updater) (view, answer string) {
	t.Helper()
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s, err := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: token, Update: u})
	if err != nil {
		t.Fatal(err)
	}
	return "http://" + s.View.String(), "http://" + s.Answer.String()
}

func req(t *testing.T, method, url, token, body string, hdr map[string]string) (int, string) {
	t.Helper()
	rq, _ := http.NewRequest(method, url, strings.NewReader(body))
	if token != "" {
		rq.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		rq.Header.Set("Content-Type", "application/json")
	}
	for k, v := range hdr {
		rq.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(rq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestUpdateStatusIsGatedAndReadOnly(t *testing.T) {
	u := &fakeUpdater{status: selfupdate.Status{State: "idle", Running: "0.3.0", Available: &selfupdate.Available{Version: "0.4.0", Notes: "n"}}}
	view, _ := serveUpdate(t, "tok", u)
	if code, _ := req(t, "GET", view+"/api/update/status", "", "", nil); code != 401 {
		t.Fatalf("status without token: %d", code)
	}
	for _, p := range []string{"/api/update", "/api/update/status"} {
		code, body := req(t, "GET", view+p, "tok", "", nil)
		if code != 200 || !strings.Contains(body, `"version":"0.4.0"`) || !strings.Contains(body, `"running":"0.3.0"`) {
			t.Fatalf("%s: %d %s", p, code, body)
		}
	}
	if code, _ := req(t, "POST", view+"/api/update/apply", "tok", `{"confirm":"0.4.0"}`, nil); code == 200 || len(u.calls) != 0 {
		t.Fatalf("a write reached the viewer: %d %v", code, u.calls)
	}
	if code, body := req(t, "GET", view+"/watch/update.js", "tok", "", nil); code != 200 || !strings.Contains(body, "/api/update/status") {
		t.Fatalf("script: %d", code)
	}
}

func TestUpdateBannerInjectedIntoPages(t *testing.T) {
	view, _ := serveUpdate(t, "tok", &fakeUpdater{})
	code, body := req(t, "GET", view+"/watch", "tok", "", nil)
	if code != 200 || strings.Count(body, `<script src="/watch/update.js"></script>`) != 1 {
		t.Fatalf("watch page: %d, script tags %d", code, strings.Count(body, "update.js"))
	}
	// without Update the page is unchanged
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, _ := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: "tok"})
	_, plain := req(t, "GET", "http://"+s.View.String()+"/watch", "tok", "", nil)
	if strings.Contains(plain, "update.js") {
		t.Fatal("banner injected without an updater")
	}
}

func TestUpdateAnswerRoutes(t *testing.T) {
	u := &fakeUpdater{status: selfupdate.Status{State: "ready"}}
	_, ans := serveUpdate(t, "tok", u)
	post := func(path, tok, body string, hdr map[string]string) (int, string) {
		return req(t, "POST", ans+path, tok, body, hdr)
	}
	if code, _ := post("/api/update/apply", "", `{"confirm":"0.4.0"}`, nil); code != 401 {
		t.Fatalf("no token: %d", code)
	}
	if code, _ := post("/api/update/apply", "tok", `{"confirm":"0.4.0"}`, map[string]string{"Origin": "http://evil.example"}); code != 403 {
		t.Fatalf("foreign origin: %d", code)
	}
	if code, _ := req(t, "GET", ans+"/api/update/check", "tok", "", nil); code != 405 {
		t.Fatalf("GET on a write route: %d", code)
	}
	if code, body := post("/api/update/apply", "tok", `{}`, nil); code != 400 || !strings.Contains(body, "confirm") {
		t.Fatalf("apply without confirm: %d %s", code, body)
	}
	if code, _ := post("/api/update/apply", "tok", `{"confirm":"0.4.0","bogus":1}`, nil); code != 400 {
		t.Fatalf("unknown field: %d", code)
	}
	if len(u.calls) != 0 {
		t.Fatalf("refused requests reached the engine: %v", u.calls)
	}
	for _, p := range []string{"check", "cancel", "rollback", "inbox", "seen"} {
		if code, body := post("/api/update/"+p, "tok", `{}`, nil); code != 200 || !strings.Contains(body, `"ok":true`) {
			t.Fatalf("%s: %d %s", p, code, body)
		}
	}
	if code, _ := post("/api/update/apply", "tok", `{"confirm":"0.4.0"}`, nil); code != 200 {
		t.Fatalf("apply: %d", code)
	}
	want := "check cancel rollback inbox seen apply:0.4.0"
	if got := strings.Join(u.calls, " "); got != want {
		t.Fatalf("calls %q, want %q", got, want)
	}
	u.err = selfupdate.ErrNotSafe
	if code, body := post("/api/update/apply", "tok", `{"confirm":"0.4.0"}`, nil); code != 409 || !strings.Contains(body, "not-safe") {
		t.Fatalf("not safe: %d %s", code, body)
	}
	u.err = errors.New("boom")
	if code, _ := post("/api/update/check", "tok", `{}`, nil); code != 409 {
		t.Fatalf("error mapping: %d", code)
	}
}
