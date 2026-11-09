package watch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/senses"
)

func sensesFixture(t *testing.T) *senses.Publisher {
	t.Helper()
	lm := &d2map.LevelMap{No: 1, Grid: d2map.ParseASCII(0, 0, "#####", "#...#", "#####"),
		Exits: []d2map.Exit{{Dest: 2, Pos: d2map.Point{X: 2, Y: 1}}}}
	p := &senses.Publisher{}
	if err := p.Observe(lm, "Test", d2map.Point{X: 2, Y: 1}); err != nil {
		t.Fatal(err)
	}
	return p
}

func getReply(t *testing.T, h http.Handler, url string) (int, SensesReply) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
	var r SensesReply
	json.Unmarshal(w.Body.Bytes(), &r)
	return w.Code, r
}

func TestSensesTierGate(t *testing.T) {
	p := sensesFixture(t)
	for tier, wantMap := range map[int]bool{0: false, 1: true, 3: true} {
		h := (&Senses{Src: p, Lock: FixedTier(tier)}).Handler()
		code, r := getReply(t, h, "/api/senses")
		if code != 200 || (r.Map != nil) != wantMap || (r.Hero != nil) != wantMap {
			t.Fatalf("tier %d: %d %+v", tier, code, r)
		}
		if len(r.Tiers) != tier {
			t.Fatalf("tier %d unlocked %v", tier, r.Tiers)
		}
	}
	// a locked tier is absent from the bytes, not empty
	w := httptest.NewRecorder()
	(&Senses{Src: p, Lock: FixedTier(0)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	if strings.Contains(w.Body.String(), `"map"`) || strings.Contains(w.Body.String(), `"hero"`) {
		t.Fatal(w.Body.String())
	}
	// nil Unlocker is closed
	if _, r := getReply(t, (&Senses{Src: p}).Handler(), "/api/senses"); r.Map != nil {
		t.Fatal("nil unlocker opened the map")
	}
}

func TestSensesVersionCacheAndWaiting(t *testing.T) {
	p := sensesFixture(t)
	h := (&Senses{Src: p, Lock: FixedTier(1)}).Handler()
	_, a := getReply(t, h, "/api/senses")
	if a.Map == nil || a.Version == 0 {
		t.Fatal("no map")
	}
	_, b := getReply(t, h, "/api/senses?v="+strconv.FormatUint(a.Version, 10))
	if b.Map != nil || b.Hero == nil {
		t.Fatalf("cached caller got the map again: %+v", b)
	}
	_, w := getReply(t, (&Senses{Src: &senses.Publisher{}, Lock: FixedTier(1)}).Handler(), "/api/senses")
	if !w.Waiting || w.Map != nil {
		t.Fatalf("%+v", w)
	}
}

func TestSensesGETOnly(t *testing.T) {
	h := (&Senses{Src: sensesFixture(t), Lock: FixedTier(1)}).Handler()
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(m, "/api/senses", nil))
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: %d", m, w.Code)
		}
	}
}

func TestSensesServedBehindToken(t *testing.T) {
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sv, err := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: "tok",
		Senses: &Senses{Src: sensesFixture(t), Lock: FixedTier(1)}})
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + sv.View.String()
	for _, p := range SensesRoutes {
		if resp := mustGet(t, base+p); resp.StatusCode != 401 {
			t.Fatalf("%s without token: %d", p, resp.StatusCode)
		}
	}
	req, _ := http.NewRequest("GET", base+"/api/senses", nil)
	req.Header.Set("Authorization", "Bearer tok")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("with token: %v %v", resp, err)
	}
	var out SensesReply
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Map == nil || out.Map.AreaID != 1 || len(out.Map.Exits) != 1 {
		t.Fatalf("%+v", out)
	}
	if resp := mustGet(t, base+"/watch/senses_units.js?t=tok"); resp.StatusCode != 200 {
		t.Fatal("senses_units.js", resp.StatusCode)
	}
	if resp := mustGet(t, base+"/watch/panels/map/panel.js?t=tok"); resp.StatusCode != 200 {
		t.Fatal("map panel", resp.StatusCode)
	}
}
