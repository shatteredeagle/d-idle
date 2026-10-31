package launch

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// fakeSys records every effect; spawned games appear in Procs with their marker.
type fakeSys struct {
	procs   []Proc
	marker  map[int]string
	exists  map[string]bool
	helper  map[int]bool
	signals []string
	runs    []string
	spawns  []spawn
	nextPID int
	// spawnGame: a Spawn of a Diablo II.exe command makes a game process appear.
	spawnGame bool
	runHook   func(name string, args []string) error
}

type spawn struct {
	env  []string
	dir  string
	log  string
	argv []string
}

func newFake() *fakeSys {
	return &fakeSys{marker: map[int]string{}, exists: map[string]bool{}, helper: map[int]bool{}, nextPID: 1000, spawnGame: true}
}

func (f *fakeSys) Procs() ([]Proc, error)               { return f.procs, nil }
func (f *fakeSys) Marker(pid int) string                { return f.marker[pid] }
func (f *fakeSys) Exists(p string) bool                 { return f.exists[p] }
func (f *fakeSys) HelperUp(port int) bool               { return f.helper[port] }
func (f *fakeSys) Sleep(context.Context, time.Duration) {}
func (f *fakeSys) Signal(pid int, s syscall.Signal) error {
	f.signals = append(f.signals, s.String()+" "+strconv.Itoa(pid))
	var keep []Proc
	for _, p := range f.procs {
		if p.PID != pid {
			keep = append(keep, p)
		}
	}
	f.procs = keep
	return nil
}
func (f *fakeSys) Run(_ context.Context, _ []string, _ string, name string, args ...string) ([]byte, error) {
	f.runs = append(f.runs, name+" "+strings.Join(args, " "))
	if f.runHook != nil {
		return nil, f.runHook(name, args)
	}
	return nil, nil
}
func (f *fakeSys) Spawn(env []string, dir, log, name string, args ...string) error {
	f.spawns = append(f.spawns, spawn{env, dir, log, append([]string{name}, args...)})
	f.nextPID++
	switch {
	case name == "Xvfb":
		f.procs = append(f.procs, Proc{PID: f.nextPID, Argv: append([]string{"Xvfb"}, args...)})
		f.exists["/tmp/.X11-unix/X"+strings.TrimPrefix(args[0], ":")] = true
	case f.spawnGame && len(args) > 1 && strings.HasSuffix(args[1], "Diablo II.exe"):
		f.procs = append(f.procs, Proc{PID: f.nextPID, Exe: "/x/wine-preloader", Argv: []string{`C:\Program Files (x86)\Diablo II\Game.exe`, "-w"}})
		for _, kv := range env {
			if v, ok := strings.CutPrefix(kv, "D2BOT_INSTANCE="); ok {
				f.marker[f.nextPID] = v
			}
		}
	case len(args) > 1 && strings.HasSuffix(args[1], "d2input.exe"):
		f.procs = append(f.procs, Proc{PID: f.nextPID, Argv: append([]string{"d2input.exe"}, args[2:]...)})
		f.helper[atoiPort(args[len(args)-1])] = true
	}
	return nil
}

func atoiPort(addr string) int {
	return atoi(addr[strings.LastIndex(addr, ":")+1:])
}

func testLife(t *testing.T) (*Life, *fakeSys, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	proton := filepath.Join(root, "runner", "proton")
	os.MkdirAll(filepath.Dir(proton), 0o755)
	os.WriteFile(proton, []byte("#!/bin/sh\n"), 0o755)
	f := newFake()
	var out bytes.Buffer
	l := &Life{Sys: f, Out: &out, Environ: []string{"HOME=/h", "WAYLAND_DISPLAY=wayland-0", "XAUTHORITY=/x", "DISPLAY=:1"},
		StartWait: 3, HelperWait: 3,
		Base: Config{Root: filepath.Join(root, "Games", "d2lod"), SteamRoot: root, ProtonScript: proton}}
	return l, f, &out
}

func env(e []string, k string) []string {
	var r []string
	for _, kv := range e {
		if v, ok := strings.CutPrefix(kv, k+"="); ok {
			r = append(r, v)
		}
	}
	return r
}

func TestStartMarkerEnvAndIdempotent(t *testing.T) {
	l, f, out := testLife(t)
	ctx := context.Background()
	os.MkdirAll(filepath.Join(l.Base.Root, "logs"), 0o755)
	// prefix present so start does not mkprefix
	cfg, _ := l.config("i2")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	if err := l.Start(ctx, "i2"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "i2: Game.exe up after 1s (pid ") {
		t.Fatalf("out %q", out.String())
	}
	var game spawn
	for _, s := range f.spawns {
		if strings.HasSuffix(s.argv[2], "Diablo II.exe") {
			game = s
		}
	}
	if m := env(game.env, "D2BOT_INSTANCE"); len(m) != 1 || m[0] != "i2" {
		t.Fatalf("marker env %v", m)
	}
	if d := env(game.env, "DISPLAY"); len(d) != 1 || d[0] != ":102" {
		t.Fatalf("display %v", d)
	}
	if len(env(game.env, "WAYLAND_DISPLAY")) != 0 || len(env(game.env, "XAUTHORITY")) != 0 {
		t.Fatal("desktop session leaked into an Xvfb instance")
	}
	if got := strings.Join(game.argv[3:], " "); got != "-w -ns" {
		t.Fatalf("args %q (must be muted)", got)
	}
	n := len(f.spawns)
	out.Reset()
	if err := l.Start(ctx, "i2"); err != nil || out.String() != "i2 already running\n" || len(f.spawns) != n {
		t.Fatalf("not idempotent: %v %q spawns %d->%d", err, out.String(), n, len(f.spawns))
	}
}

func TestStartCreatesPrefixAndXvfb(t *testing.T) {
	l, f, _ := testLife(t)
	os.MkdirAll(filepath.Join(l.Base.Root, "proton"), 0o755)
	if err := l.Start(context.Background(), "i3"); err != nil {
		t.Fatal(err)
	}
	if len(f.runs) == 0 || !strings.HasPrefix(f.runs[0], "rsync -a --link-dest=") {
		t.Fatalf("runs %v", f.runs)
	}
	if f.spawns[0].argv[0] != "Xvfb" || f.spawns[0].argv[1] != ":103" {
		t.Fatalf("spawns %+v", f.spawns)
	}
}

func TestStartNotUpFails(t *testing.T) {
	l, f, _ := testLife(t)
	f.spawnGame = false
	cfg, _ := l.config("i1")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	err := l.Start(context.Background(), "i1")
	if err == nil || err.Error() != "i1: Game.exe did not start" {
		t.Fatal(err)
	}
}

func TestStopCleansWineserverAndXvfb(t *testing.T) {
	l, f, out := testLife(t)
	cfg, _ := l.config("i1")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	if err := l.Start(context.Background(), "i1"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	f.runs = nil
	if err := l.Stop(context.Background(), "i1"); err != nil {
		t.Fatal(err)
	}
	if len(f.runs) != 1 || !strings.HasSuffix(f.runs[0], "files/bin/wineserver -k") {
		t.Fatalf("runs %v", f.runs)
	}
	o := out.String()
	if !strings.Contains(o, "i1: sent SIGTERM to ") || !strings.Contains(o, "i1: Xvfb :101 stopped") {
		t.Fatalf("out %q", o)
	}
	if len(l.GamePIDs("i1")) != 0 {
		t.Fatal("game still listed")
	}
	// stopping again is clean and says so
	out.Reset()
	l.Stop(context.Background(), "i1")
	if !strings.HasPrefix(out.String(), "i1 not running\n") || strings.Contains(out.String(), "Xvfb") {
		t.Fatalf("out %q", out.String())
	}
}

func TestStopI0OnlySignals(t *testing.T) {
	l, f, out := testLife(t)
	f.procs = []Proc{{PID: 77, Exe: "/w/wine-preloader", Argv: []string{`C:\Game.exe`}}} // unmarked = i0
	if err := l.Stop(context.Background(), "i0"); err != nil {
		t.Fatal(err)
	}
	if len(f.runs) != 0 || out.String() != "i0: sent SIGTERM to 77\n" {
		t.Fatalf("runs %v out %q", f.runs, out.String())
	}
}

func TestGamePIDsMarkerSeparation(t *testing.T) {
	l, f, _ := testLife(t)
	g := func(pid int) Proc { return Proc{PID: pid, Exe: "/w/wine-preloader", Argv: []string{`C:\x\Game.exe`}} }
	f.procs = []Proc{g(5), g(6), g(7), {PID: 8, Exe: "/bin/bash", Argv: []string{"Game.exe"}}}
	f.marker[6], f.marker[7] = "i1", "i2"
	if got := l.GamePIDs("i1"); len(got) != 1 || got[0] != 6 {
		t.Fatal(got)
	}
	if got := l.GamePIDs("i0"); len(got) != 1 || got[0] != 5 {
		t.Fatal(got)
	}
}

func TestMkPrefixRefusesWhileRunning(t *testing.T) {
	l, f, _ := testLife(t)
	f.procs = []Proc{{PID: 9, Exe: "/w/wine-preloader", Argv: []string{`C:\Game.exe`}}}
	f.marker[9] = "i4"
	err := l.MkPrefix(context.Background(), "i4")
	if err == nil || err.Error() != "i4 is running; stop it first" || len(f.runs) != 0 {
		t.Fatalf("%v %v", err, f.runs)
	}
	if _, ok := l.MkPrefix(context.Background(), "i0").(UsageError); !ok {
		t.Fatal("i0 mkprefix must be a usage error")
	}
}

func TestMkPrefixBreaksLinks(t *testing.T) {
	l, f, out := testLife(t)
	src := filepath.Join(l.Base.Root, "proton")
	gd := filepath.Join(src, "pfx", "drive_c", "Program Files (x86)", "Diablo II")
	os.MkdirAll(gd, 0o755)
	os.WriteFile(filepath.Join(src, "pfx", "user.reg"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(src, "tracked_files"), []byte("t"), 0o644)
	os.WriteFile(filepath.Join(gd, "d2.log"), []byte("l"), 0o644)
	os.WriteFile(filepath.Join(gd, "Game.exe"), []byte("exe"), 0o644)
	cfg, _ := l.config("i5")
	f.runHook = func(name string, _ []string) error { // stand-in for rsync --link-dest: hardlink the tree
		return filepath.Walk(src, func(p string, fi os.FileInfo, err error) error {
			rel, _ := filepath.Rel(src, p)
			d := filepath.Join(cfg.PrefixDir, rel)
			if fi.IsDir() {
				return os.MkdirAll(d, 0o755)
			}
			return os.Link(p, d)
		})
	}
	if err := l.MkPrefix(context.Background(), "i5"); err != nil {
		t.Fatal(err)
	}
	same := func(a, b string) bool {
		x, _ := os.Stat(a)
		y, _ := os.Stat(b)
		return os.SameFile(x, y)
	}
	dgd := filepath.Join(cfg.PrefixDir, "pfx", "drive_c", "Program Files (x86)", "Diablo II")
	if same(filepath.Join(src, "pfx", "user.reg"), filepath.Join(cfg.PrefixDir, "pfx", "user.reg")) ||
		same(filepath.Join(src, "tracked_files"), filepath.Join(cfg.PrefixDir, "tracked_files")) ||
		same(filepath.Join(gd, "d2.log"), filepath.Join(dgd, "d2.log")) {
		t.Fatal("rewritten files must be private copies")
	}
	if !same(filepath.Join(gd, "Game.exe"), filepath.Join(dgd, "Game.exe")) {
		t.Fatal("game files stay hardlinked")
	}
	if b, _ := os.ReadFile(filepath.Join(cfg.PrefixDir, "pfx", "user.reg")); string(b) != "a" {
		t.Fatal("content lost")
	}
	if !strings.HasPrefix(out.String(), "prefix "+cfg.PrefixDir+" ready") {
		t.Fatal(out.String())
	}
	if !strings.Contains(f.runs[0], "--exclude=Saved Games/Diablo II/*") {
		t.Fatal("saves must be excluded: ", f.runs[0])
	}
}

func TestEnsureHelper(t *testing.T) {
	l, f, out := testLife(t)
	cfg, _ := l.config("i2")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	if _, ok := l.EnsureHelper(context.Background(), "i0").(UsageError); !ok {
		t.Fatal("i0 helper is a usage error")
	}
	if err := l.EnsureHelper(context.Background(), "i2"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(out.String(), "d2input for i2 listening on 47014\n") {
		t.Fatal(out.String())
	}
	n := len(f.spawns)
	out.Reset()
	l.EnsureHelper(context.Background(), "i2")
	if out.String() != "d2input for i2 already running\n" || len(f.spawns) != n {
		t.Fatalf("not idempotent: %q", out.String())
	}
}

func TestStatusFormat(t *testing.T) {
	l, f, out := testLife(t)
	f.procs = []Proc{{PID: 11, Exe: "/w/wine-preloader", Argv: []string{`C:\Game.exe`}}}
	f.marker[11] = "i3"
	f.runHook = func(string, []string) error { return nil }
	l.Status(context.Background(), "i3", "i4")
	want := "i3: Game.exe pid 11  cpu=%% rss=0MB display=:103 helper=127.0.0.1:47015\ni4: stopped\n"
	want = strings.ReplaceAll(want, "%%", "%")
	if out.String() != want {
		t.Fatalf("got %q want %q", out.String(), want)
	}
}

// A helper process that exists but does not listen (wedged, bind failed) is replaced, not adopted.
func TestEnsureHelperReplacesWedged(t *testing.T) {
	l, f, out := testLife(t)
	cfg, _ := l.config("i2")
	f.exists[filepath.Join(cfg.PrefixDir, "pfx")] = true
	f.procs = append(f.procs, Proc{PID: 77, Argv: []string{"d2input.exe", "-addr", cfg.HelperAddr}})
	if err := l.EnsureHelper(context.Background(), "i2"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "replacing") || len(f.spawns) == 0 {
		t.Fatalf("wedged helper not replaced: %q spawns=%d", out.String(), len(f.spawns))
	}
}
