package main

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
)

func TestBackendFlags(t *testing.T) {
	t.Setenv("D2BOT_BACKEND", "")
	t.Setenv("D2D_SOCK", "")
	t.Setenv("D2D_BIN", "")
	o, err := parseFlags([]string{"--attach"})
	if err != nil || o.d2d.on() {
		t.Fatalf("default must stay the Game.exe backend: %+v %v", o.d2d, err)
	}
	o, err = parseFlags([]string{"--attach", "--d2d-attach", "--backend", "d2d", "--d2d-sock", "/x/s", "--seed", "7", "--instance", "i3"})
	if err != nil || !o.d2d.on() || d2dSock(o) != "/x/s" || o.d2d.seed != 7 || o.instance != "i3" {
		t.Fatalf("%+v %v", o.d2d, err)
	}
	t.Setenv("D2D_SOCK", "/env/sock")
	t.Setenv("D2BOT_INSTANCE", "i4")
	o, err = parseFlags([]string{"--attach", "--d2d-attach", "--backend", "d2d"})
	if err != nil || d2dSock(o) != "/env/sock" || o.instance != "i4" {
		t.Fatalf("env not honoured: %+v %v", o, err)
	}
	t.Setenv("D2D_SOCK", "")
	o, _ = parseFlags([]string{"--attach", "--d2d-attach", "--backend", "d2d"})
	if !strings.HasSuffix(d2dSock(o), "d2d-i4.sock") {
		t.Fatalf("per-instance default socket: %s", d2dSock(o))
	}
	for _, bad := range [][]string{
		{"--sim", "--backend", "d2d"}, {"--attach", "--dry", "--backend", "d2d"}, {"--managed", "--backend", "d2d"}, {"--attach", "--backend", "x"}, {"--attach", "--backend", "d2d"},
	} {
		if _, err := parseFlags(bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
	o, err = parseFlags([]string{"--managed", "--backend", "d2d", "--d2d-bin", "/b/d2d", "--no-save", "--toggle", "autoloot=off,autoparty=off", "--d2d-args", "--lockstep", "--char", "Booty", "--d2d-class", "1", "--seed", "3"})
	if err != nil {
		t.Fatal(err)
	}
	cfg := d2dConfig(o, slog.Default(), &env.Tuning{}, nil, "/home")
	if cfg.Bin != "/b/d2d" || cfg.Spec.Name != "Booty" || cfg.Spec.Class != 1 || cfg.Spec.Seed != 3 || !cfg.Spec.NoSave ||
		strings.Join(cfg.ExtraArgs, " ") != "--toggle autoloot=off,autoparty=off --lockstep" {
		t.Fatalf("%+v", cfg)
	}
}

// --d2d-dial: the client dials the tap, d2d (managed) still listens on --d2d-sock.
func TestBackendDialFlag(t *testing.T) {
	t.Setenv("D2D_DIAL", "")
	o, err := parseFlags([]string{"--managed", "--backend", "d2d", "--d2d-bin", "/b/d2d", "--d2d-sock", "/r/d2d.sock", "--d2d-dial", "/r/bot.sock"})
	if err != nil {
		t.Fatal(err)
	}
	cfg := d2dConfig(o, slog.Default(), &env.Tuning{}, nil, "/home")
	if cfg.Sock != "/r/d2d.sock" || cfg.Dial != "/r/bot.sock" {
		t.Fatalf("%+v", cfg)
	}
	o, _ = parseFlags([]string{"--managed", "--backend", "d2d", "--d2d-bin", "/b/d2d"})
	if d2dConfig(o, slog.Default(), &env.Tuning{}, nil, "/home").Dial != "" {
		t.Fatal("dial must default to empty")
	}
}

// The quest source factory picks the d2d source for a d2d reader, questmem for the memory reader, nothing otherwise.
func TestMemQuestSourceFactory(t *testing.T) {
	if memQuestSource(&env.Env{}) != nil {
		t.Fatal("no reader must give no source")
	}
	var nilReader *d2.Reader
	if memQuestSource(&env.Env{Read: nilReader}) != nil {
		t.Fatal("typed nil memory reader must give no source")
	}
	if memQuestSource(&env.Env{Read: d2.New(nil)}) == nil {
		t.Fatal("memory reader lost its questmem source")
	}
	c := d2dctl.New("/nonexistent", d2dctl.Options{})
	rd := d2denv.NewReader(d2dsrc.New(c, d2dsrc.Options{Logf: func(string, ...any) {}}))
	if _, ok := memQuestSource(&env.Env{Read: rd}).(*d2denv.QuestSource); !ok {
		t.Fatal("d2d reader must get the d2d quest source")
	}
}

// With the fake d2d the whole bot boots: reads state, serves the dashboard and API, runs goal-manager ticks with the
// combat and town stubs ("not yet") and shuts down cleanly without ever sending quit to an attached d2d.
func TestRunD2DBootsOnFakeAndStopsCleanly(t *testing.T) { bootOnFake(t, false) }

// Same with game tables: the real combat, town, leveler and task layers are built over the fake (the stubs say "not
// yet" in the first test, these must not), merc mount answers, a goal runs, cancel stops cleanly.
func TestRunD2DWholeStackOnFake(t *testing.T) {
	geartest.Need(t)
	bootOnFake(t, true)
}

func bootOnFake(t *testing.T, tables bool) {
	dir, err := os.MkdirTemp("", "d2dbot")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	sock := filepath.Join(dir, "d.sock")
	srv, err := fake.New(sock, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := l.Addr().String()
	l.Close()
	dataDir, _ := os.MkdirTemp("", "d2dbotdata") // not t.TempDir: late writers (goals file) race its cleanup
	defer os.RemoveAll(dataDir)
	o := options{attach: true, data: dataDir, addr: addr, instance: "t", goalSpec: "farm:Blood Moor",
		d2d: d2dOptions{backend: "d2d", sock: sock, class: -1, attach: true}}
	if tables {
		cache := synthTables(t)
		old := openTables
		openTables = func(string, string) (*data.DB, error) { return data.Load(cache) }
		defer func() { openTables = old }()
		o.gameDir = "synthetic"
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lb := &lockedBuf{}
	defer func() {
		if t.Failed() {
			t.Log(lb.String())
		}
	}()
	done := make(chan error, 1)
	go func() { done <- runD2D(ctx, o, slog.New(slog.NewTextHandler(lb, nil))) }()

	base := "http://" + addr
	var body string
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(base + "/api/v1/instances/t/status"); err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.Contains(string(b), "runner") {
				body = string(b)
			}
		}
		if body != "" && sawStateReads(srv) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	select {
	case err := <-done:
		t.Fatalf("runD2D returned early: %v", err)
	default:
	}
	if body == "" || !sawStateReads(srv) {
		t.Fatalf("bot did not come up: status %q lines %v", body, srv.Lines())
	}
	if c, b := get(t, base+"/healthz"); c != 200 || !strings.Contains(b, `"ok":true`) {
		t.Fatalf("healthz %d %s", c, b)
	}
	if c, b := get(t, base+"/api/v1/instances/t/knobs"); c != 200 || !strings.Contains(b, `"knobs"`) {
		t.Fatalf("knobs %d %.200s", c, b)
	}
	if c, _ := get(t, base+"/"); c != 200 {
		t.Fatalf("dashboard %d", c)
	}
	time.Sleep(2 * time.Second) // a few goal-manager and runner ticks
	if out := lb.String(); !strings.Contains(out, "to=traveling") || !strings.Contains(out, "travel to Blood Moor") {
		t.Fatal("the goal manager did not run a goal against the d2d backend")
	}
	if tables {
		if out := lb.String(); strings.Contains(out, "not yet") || strings.Contains(out, "no game tables") {
			t.Fatalf("stub behaviours still in use with tables:\n%s", out)
		}
		if c, b := get(t, base+"/api/v1/instances/t/merc/"); c != 200 || !strings.Contains(b, `"enabled":false`) {
			t.Fatalf("merc mount %d %s", c, b)
		}
	} else if !strings.Contains(lb.String(), "no game tables") {
		t.Fatal("stubs expected without tables")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("no graceful shutdown")
	}
	if srv.Quit() {
		t.Fatal("attach mode must leave the d2d running")
	}
}

func sawStateReads(s *fake.Server) bool {
	var st, pl bool
	for _, l := range s.Lines() {
		st = st || l == "state"
		pl = pl || l == "debug players"
	}
	return st && pl
}

type lockedBuf struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}
func (l *lockedBuf) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.b.String() }

// synthTables writes a cache dir with header-only tables: enough for data.Load, so the combat/town/leveler constructors
// run without the (unlicensed, not committed) game files.
func synthTables(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "d2dtables")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	for _, n := range []string{"monstats", "levels", "skills", "charstats", "experience", "itemtypes", "weapons", "armor", "misc"} {
		if err := os.WriteFile(filepath.Join(dir, n+".tsv"), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for n, c := range map[string]string{"stamp.json": "{}", "strings.tsv": ""} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

type stepClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(30 * time.Minute) // every read is half a virtual hour
	return c.t
}
func (c *stepClock) Sleep(time.Duration) {}

func TestStopAtVirtualUsesTheEnvClock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		stopAtVirtual(ctx, cancel, &stepClock{}, time.Hour, slog.New(slog.DiscardHandler))
		close(done)
	}()
	select {
	case <-ctx.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("virtual deadline never fired")
	}
	<-done
}

func TestSoakHours(t *testing.T) {
	t.Setenv("D2DSOAK_HOURS", "1.5")
	if soakHours() != 90*time.Minute {
		t.Fatal(soakHours())
	}
	t.Setenv("D2DSOAK_HOURS", "")
	if soakHours() != 0 {
		t.Fatal("unset must be 0")
	}
}
