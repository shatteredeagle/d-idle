package main

import (
	"context"
	"d2lod-bot/internal/fleet"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSlotProcsUseSlotIdentity(t *testing.T) {
	ps := slotProcs(slotInfo{ID: "i2", Prefix: "/data/prefixes/i2", HelperPort: 47014, Display: ":102"})
	m := map[string]string{}
	for _, p := range ps {
		m[p.Name] = p.Spec
	}
	for name, want := range map[string]string{
		"bot":        "cmd=--instance i2",
		"game":       "env=WINEPREFIX=/data/prefixes/i2/pfx",
		"wineserver": "env=WINEPREFIX=/data/prefixes/i2/pfx",
		"helper":     "-addr 127.0.0.1:47014",
		"xvfb":       "Xvfb :102 ",
	} {
		if !strings.Contains(m[name], want) {
			t.Errorf("%s spec %q lacks %q", name, m[name], want)
		}
	}
	if _, err := parseProcSpecs(specStrings(ps)); err != nil {
		t.Fatal(err)
	}
}

func specStrings(ps []ProcSource) []string {
	var o []string
	for _, p := range ps {
		o = append(o, p.Name+"="+p.Spec)
	}
	return o
}

func TestFleetSlotsParsesAdminAPI(t *testing.T) {
	vs := []fleet.View{{ID: "i1", Alloc: fleet.Alloc{Prefix: "/p/i1", HelperPort: 47013, Display: ":101"},
		Bot: &fleet.BotStatus{Addr: "127.0.0.1:4000", Log: "/p/i1/logs/d2bot.log"}}, {ID: "i2"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/fleet-admin/slots" || r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no", 401)
			return
		}
		json.NewEncoder(w).Encode(vs)
	}))
	defer srv.Close()
	got, err := fleetSlots(context.Background(), srv.Client(), srv.URL, "tok")
	if err != nil || len(got) != 2 || got[0].BotAddr != "127.0.0.1:4000" || got[0].Prefix != "/p/i1" || got[1].BotAddr != "" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := fleetSlots(context.Background(), srv.Client(), srv.URL, "bad"); err == nil {
		t.Fatal("want an error for HTTP 401")
	}
}

// A fake flat bot; the sidecar must start one run for the slot, write the dashboard files, and follow the bot to a new address.
func fakeBot(hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch {
		case strings.HasSuffix(r.URL.Path, "/health"):
			w.Write([]byte(`{"state":"running","progressCounter":5}`))
		case strings.HasSuffix(r.URL.Path, "/status"):
			w.Write([]byte(`{"version":"d2bot abc1234"}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestSidecarRunsPerSlotAndFollowsBot(t *testing.T) {
	var h1, h2 atomic.Int32
	b1, b2 := fakeBot(&h1), fakeBot(&h2)
	defer b1.Close()
	defer b2.Close()
	addr := atomic.Value{}
	addr.Store(strings.TrimPrefix(b1.URL, "http://"))
	out := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	cfg := sidecarCfg{
		Slots: func(context.Context) ([]slotInfo, error) {
			return []slotInfo{{ID: "i1", Prefix: filepath.Join(out, "pfxdir"), BotAddr: addr.Load().(string)}}, nil
		},
		Out: out, Poll: 20 * time.Millisecond, Every: 20 * time.Millisecond, For: time.Hour, Limits: DefaultLimits(),
		Rules: DefaultAlertRules(), Now: time.Now, Log: t.Logf,
	}
	go func() { done <- runSidecar(ctx, cfg) }()
	waitFor(t, func() bool { return h1.Load() > 3 })
	addr.Store(strings.TrimPrefix(b2.URL, "http://"))
	waitFor(t, func() bool { return h2.Load() > 3 })
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	runs, _ := filepath.Glob(filepath.Join(out, "i1-*"))
	if len(runs) != 1 {
		t.Fatalf("want one run dir, got %v", runs)
	}
	for _, f := range []string{"samples.jsonl", "alerts.json", "status.txt", "version.txt"} {
		if _, err := os.Stat(filepath.Join(runs[0], f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}
	if v := loadVersion(runs[0]); v != "abc1234" {
		t.Errorf("version %q", v)
	}
	// d2soak report/status accept the directory of runs and pick the newest run
	if p, _ := samplesPath(out); filepath.Dir(p) != runs[0] {
		t.Errorf("samplesPath(root) = %s", p)
	}
}

func TestSidecarEndsWhenParentGone(t *testing.T) {
	var alive atomic.Bool
	alive.Store(true)
	done := make(chan error, 1)
	go func() {
		done <- runSidecar(context.Background(), sidecarCfg{
			Slots: func(context.Context) ([]slotInfo, error) { return nil, nil },
			Poll:  10 * time.Millisecond, Now: time.Now, Log: t.Logf, Alive: alive.Load,
		})
	}()
	time.Sleep(50 * time.Millisecond)
	alive.Store(false)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sidecar kept running")
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	for i := 0; i < 300; i++ {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout")
}
