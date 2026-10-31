package launch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/mem"

	"d2lod-bot/internal/paths"
)

// Life is the game lifecycle of the instances (start, stop, status, helper, prefix): a Go port of what
// tools/game.sh did over ssh+bash, with no ssh and no bash. It runs on the machine that owns the game
// (the worker). cmd/d2game is its CLI, the sentinel calls it directly, and tools/game.sh is a thin wrapper.
//
// Output lines are the same as game.sh printed (the dev workflow and docs/multi.md depend on them).
// Life does NOT lock: the CLI takes the per-instance flock (LockInstance); the sentinel holds the lock
// itself around restarts.
//
// Tags: the logic is ported 1:1 from the script that is VERIFIED LIVE (docs/multi.md); the Go port itself
// is verified offline with a fake System only. UNVERIFIED live until one cycle runs on a spare instance.
type Life struct {
	Sys  System
	Base Config // Defaults(); Base.Root is ~/Games/d2lod (the main prefix parent)
	Out  io.Writer
	// Environ is the environment children inherit (default os.Environ()).
	Environ []string
	// StartWait and HelperWait are the poll counts (1 s each) for Game.exe and the helper port.
	StartWait, HelperWait int
	// Reap overrides the winedevice reaping after a stop (tests); the zero value is the real system.
	Reap ReapOpts
}

// NewLife returns the real lifecycle writing progress to out (nil = discard).
func NewLife(out io.Writer) *Life {
	if out == nil {
		out = io.Discard
	}
	return &Life{Sys: OSSystem{}, Base: Defaults(), Out: out, Environ: os.Environ(), StartWait: 40, HelperWait: 30}
}

// UsageError marks a refusal that is the caller's mistake (exit code 2 in the CLI).
type UsageError string

func (e UsageError) Error() string { return string(e) }

var idRe = regexp.MustCompile(`^i[0-9]$`)

// ValidID reports whether id is an instance id ("i0".."i9").
func ValidID(id string) bool { return idRe.MatchString(id) }

// DefaultIDs are the instances `status` lists when no id is given.
var DefaultIDs = []string{"i0", "i1", "i2", "i3", "i4", "i5"}

func (l *Life) say(f string, a ...any) { fmt.Fprintf(l.Out, f+"\n", a...) }

// instance builds the Instance of an id: private Xvfb :10N for every id but i0.
func (l *Life) instance(id string) Instance {
	in := Instance{ID: id}
	if in.Num() > 0 {
		in.Display = fmt.Sprintf(":%d", 100+in.Num())
	}
	return in
}

// config derives the instance Config. The error (no Proton runner) is returned separately because stop
// and mkprefix do not need a runner.
func (l *Life) config(id string) (Config, error) {
	c := l.instance(id).Config(l.Base)
	err := c.Resolve()
	return c, err
}

func (l *Life) env() []string {
	if l.Environ != nil {
		return l.Environ
	}
	return os.Environ()
}

func (l *Life) display(in Instance) Display {
	if in.Num() == 0 {
		rt := fmt.Sprintf("/run/user/%d", os.Getuid())
		xa, _ := filepath.Glob(filepath.Join(rt, "xauth_*"))
		d := HostDisplay{Display: ":1", Wayland: "wayland-0", RuntimeDir: rt}
		if len(xa) > 0 {
			d.XAuthority = xa[0]
		}
		return d
	}
	return XvfbDisplay{Sys: l.Sys, Num: 100 + in.Num(), LogDir: filepath.Join(l.Base.Root, "logs"), ID: in.ID}
}

// childEnv is the environment of the game and helper: the Proton variables, the marker D2BOT_INSTANCE
// (Wine children inherit it; mem.FindGamesFor tells the games apart by it) and the instance's display.
// Extra instances drop the desktop session variables so nothing leaks onto the user's desktop.
func (l *Life) childEnv(cfg Config, d Display, id string) []string {
	base := l.env()
	if id != "i0" {
		base = envWithout(base, "WAYLAND_DISPLAY", "XAUTHORITY", "DISPLAY")
	}
	e := append([]string(nil), base...)
	e = append(e,
		"STEAM_COMPAT_CLIENT_INSTALL_PATH="+cfg.SteamRoot,
		"STEAM_COMPAT_DATA_PATH="+cfg.PrefixDir,
		"PROTON_LOG=1", "PROTON_LOG_DIR="+cfg.LogDir,
		"D2_ROOT="+cfg.Root, "D2BOT_INSTANCE="+id)
	return append(e, d.Env()...)
}

// GamePIDs are the Game.exe processes carrying the marker of id (i0: no marker or i0), ascending.
// Same rule as mem.FindGamesFor: exe link contains "preloader" and the cmdline contains game.exe.
func (l *Life) GamePIDs(id string) []int {
	ps, err := l.Sys.Procs()
	if err != nil {
		return nil
	}
	var out []int
	for _, p := range ps {
		if !mem.IsWineExe(p.Exe) || !strings.Contains(strings.ToLower(strings.Join(p.Argv, " ")), "game.exe") {
			continue
		}
		m := l.Sys.Marker(p.PID)
		if m == id || (id == "i0" && m == "") {
			out = append(out, p.PID)
		}
	}
	sortInts(out)
	return out
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

func joinPIDs(p []int) string {
	s := make([]string, len(p))
	for i, v := range p {
		s[i] = strconv.Itoa(v)
	}
	return strings.Join(s, " ")
}

// Status prints one line per id (none given: all of DefaultIDs).
func (l *Life) Status(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		ids = DefaultIDs
	}
	for _, id := range ids {
		pp := l.GamePIDs(id)
		if len(pp) == 0 {
			l.say("%s: stopped", id)
			continue
		}
		n := l.instance(id).Num()
		disp := fmt.Sprintf(":%d", 100+n)
		if n == 0 {
			disp = ":1"
		}
		// game.sh printed "pid $pp  cpu=" with the trailing space of `tr`: kept for identical output.
		l.say("%s: Game.exe pid %s cpu=%s%% rss=%dMB display=%s helper=127.0.0.1:%d",
			id, joinPIDs(pp)+" ", l.ps(ctx, "pcpu", pp[0]), atoi(l.ps(ctx, "rss", pp[0]))/1024, disp, 47012+n)
	}
	return nil
}

func (l *Life) ps(ctx context.Context, field string, pid int) string {
	out, _ := l.Sys.Run(ctx, nil, "", "ps", "-o", field+"=", "-p", strconv.Itoa(pid))
	return strings.TrimSpace(string(out))
}

func atoi(s string) int { n, _ := strconv.Atoi(strings.Split(s, ".")[0]); return n }

// MkPrefix (re)creates the Wine prefix of an extra instance as a hardlink clone of the main one: MPQs and
// dlls cost no disk. It refuses while the instance runs. Files that Wine or the game rewrite in place are
// copied so a write never goes through the link into the main prefix (VERIFIED LIVE in game.sh).
func (l *Life) MkPrefix(ctx context.Context, id string) error {
	if id == "i0" {
		return UsageError("i0 uses the main prefix; mkprefix is for i1..i9")
	}
	if len(l.GamePIDs(id)) > 0 {
		return fmt.Errorf("%s is running; stop it first", id)
	}
	cfg, _ := l.config(id)  // paths only; a missing runner is irrelevant here
	src := l.Base.PrefixDir // the wizard's base prefix (d2fleet sets it); otherwise the main prefix
	if src == "" {
		src = filepath.Join(l.Base.Root, "proton")
	}
	dst := cfg.PrefixDir
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(cfg.Root, "logs"), 0o755); err != nil {
		return err
	}
	out, err := l.Sys.Run(ctx, nil, "", "rsync", "-a", "--link-dest="+src+"/",
		"--exclude=pfx/drive_c/users/steamuser/Desktop/*", "--exclude=Saved Games/Diablo II/*", src+"/", dst+"/")
	if err != nil {
		return fmt.Errorf("rsync: %w: %s", err, strings.TrimSpace(string(out)))
	}
	pfx := filepath.Join(dst, "pfx")
	var files []string
	for _, pat := range []string{
		filepath.Join(pfx, "*.reg"), filepath.Join(pfx, "*.reg.old"),
		filepath.Join(pfx, "drive_c", "Program Files (x86)", "Diablo II", "*.txt"),
		filepath.Join(pfx, "drive_c", "Program Files (x86)", "Diablo II", "*.log"),
	} {
		m, _ := filepath.Glob(pat)
		files = append(files, m...)
	}
	files = append(files, filepath.Join(dst, "tracked_files"))
	for _, f := range files {
		if err := breakLink(f); err != nil {
			return err
		}
	}
	l.say("prefix %s ready", dst)
	return nil
}

// breakLink replaces a hardlinked regular file by a private copy (cp --remove-destination).
func breakLink(path string) error {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tmp := path + ".t"
	if err := os.WriteFile(tmp, b, st.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (l *Life) ensurePrefix(ctx context.Context, id string, cfg Config) error {
	if l.Sys.Exists(filepath.Join(cfg.PrefixDir, "pfx")) {
		return nil
	}
	return l.MkPrefix(ctx, id)
}

func (l *Life) wait(ctx context.Context, n int, cond func() bool) (int, bool) {
	for i := 1; i <= n; i++ {
		if cond() {
			return i, true
		}
		l.Sys.Sleep(ctx, time.Second)
	}
	return n, false
}

// EnsureHelper starts the d2input helper of an extra instance if it is not running. i0's helper is
// tools/helper.sh.
func (l *Life) EnsureHelper(ctx context.Context, id string) error {
	if id == "i0" {
		return UsageError("i0's helper is tools/helper.sh")
	}
	cfg, err := l.config(id)
	if err != nil {
		return err
	}
	if err := l.ensurePrefix(ctx, id, cfg); err != nil {
		return err
	}
	in := l.instance(id)
	port := 47012 + in.Num()
	want := "d2input.exe -addr " + cfg.HelperAddr
	if pids := l.helperPIDs(want); len(pids) > 0 {
		// A process exists: fine when it answers. One that holds no listening port is a wedged leftover
		// (e.g. its listener failed); give it a few seconds, then kill it and start a fresh one.
		if _, ok := l.wait(ctx, 5, func() bool { return l.Sys.HelperUp(port) }); ok {
			l.say("d2input for %s already running", id)
			return nil
		}
		l.say("d2input for %s running but not listening: replacing", id)
		for _, p := range pids {
			l.Sys.Signal(p, syscall.SIGKILL)
		}
		l.Sys.Sleep(ctx, time.Second)
	}
	d := l.display(in)
	if err := d.Ensure(ctx); err != nil {
		return err
	}
	env := l.childEnv(cfg, d, id)
	log := filepath.Join(l.Base.Root, "logs", "d2input-"+id+".out")
	name, argv, err := cfg.Wrap("", cfg.ProtonScript, "run", cfg.HelperExe, "-addr", cfg.HelperAddr)
	if err != nil {
		return err
	}
	if err := l.Sys.Spawn(env, "", log, name, argv...); err != nil {
		return err
	}
	if _, ok := l.wait(ctx, l.HelperWait, func() bool { return l.Sys.HelperUp(port) }); ok {
		l.say("d2input for %s listening on %d", id, port)
		return nil
	}
	return fmt.Errorf("d2input for %s did not start", id)
}

// helperPIDs are the processes whose command line contains want (this instance's helper: the wine
// launcher, steam.exe and d2input.exe itself; the port in want keeps other instances' helpers out).
func (l *Life) helperPIDs(want string) []int {
	ps, _ := l.Sys.Procs()
	var out []int
	for _, p := range ps {
		if strings.Contains(strings.Join(p.Argv, " "), want) {
			out = append(out, p.PID)
		}
	}
	return out
}

// Start launches the game of id (idempotent: a running instance is left alone). Extra instances get their
// prefix and Xvfb created on demand. The game gets -w -ns (muted; run tools/mute-d2.sh after launches).
func (l *Life) Start(ctx context.Context, id string) error {
	if len(l.GamePIDs(id)) > 0 {
		l.say("%s already running", id)
		return nil
	}
	cfg, err := l.config(id)
	if err != nil {
		return err
	}
	in := l.instance(id)
	if in.Num() > 0 {
		if err := l.ensurePrefix(ctx, id, cfg); err != nil {
			return err
		}
	}
	d := l.display(in)
	if err := d.Ensure(ctx); err != nil {
		return err
	}
	logName := "game-" + id + ".out"
	if id == "i0" {
		logName = "game.out"
	}
	if err := os.MkdirAll(filepath.Join(l.Base.Root, "logs"), 0o755); err != nil {
		return err
	}
	args := append([]string{"run", cfg.GameExe()}, cfg.GameArgs...)
	name, argv, err := cfg.Wrap(cfg.GameDir, cfg.ProtonScript, args...)
	if err != nil {
		return err
	}
	if err := l.Sys.Spawn(l.childEnv(cfg, d, id), cfg.GameDir, filepath.Join(l.Base.Root, "logs", logName), name, argv...); err != nil {
		return err
	}
	if i, ok := l.wait(ctx, l.StartWait, func() bool { return len(l.GamePIDs(id)) > 0 }); ok {
		l.say("%s: Game.exe up after %ds (pid %d)", id, i, l.GamePIDs(id)[0])
		return nil
	}
	return fmt.Errorf("%s: Game.exe did not start", id)
}

// Stop SIGTERMs Game.exe; for extra instances it then runs `wineserver -k` of the prefix and stops the
// instance's Xvfb. i0: SIGTERM only (never touch the user's desktop session).
func (l *Life) Stop(ctx context.Context, id string) error {
	if err := l.stopGame(ctx, id); err != nil {
		return err
	}
	if id == "i0" {
		return nil
	}
	d := l.display(l.instance(id))
	if stopped, err := d.Stop(ctx); err != nil {
		return err
	} else if stopped {
		l.say("%s: Xvfb %s stopped", id, d.Name())
	}
	return nil
}

// StopGame is Stop without the Xvfb: it ends only Game.exe (and the prefix's wineserver), for cleaning up an
// orphan while the display and the d2input helper of the instance stay up.
func (l *Life) StopGame(ctx context.Context, id string) error { return l.stopGame(ctx, id) }

func (l *Life) stopGame(ctx context.Context, id string) error {
	pp := l.GamePIDs(id)
	if len(pp) == 0 {
		l.say("%s not running", id)
	} else {
		for _, p := range pp {
			if err := l.Sys.Signal(p, syscall.SIGTERM); err != nil {
				return err
			}
		}
		l.say("%s: sent SIGTERM to %s", id, joinPIDs(pp))
	}
	if id == "i0" {
		return nil
	}
	l.Sys.Sleep(ctx, 2*time.Second)
	cfg, _ := l.config(id)
	script := cfg.ProtonScript
	if script == "" {
		script = filepath.Join(cfg.SteamRoot, "compatibilitytools.d", cfg.RunnerName, "proton")
	}
	ws := filepath.Join(filepath.Dir(script), "files", "bin", "wineserver")
	env := append(append([]string(nil), l.env()...), "WINEPREFIX="+filepath.Join(cfg.PrefixDir, "pfx"))
	_, _ = l.Sys.Run(ctx, env, "", ws, "-k") // errors ignored: nothing running is fine
	ro := l.Reap
	if ro.Log == nil {
		ro.Log = func(s string) { l.say("%s", s) }
	}
	cfg.Instance = id
	reapAfterStop(cfg, ro)
	return nil
}

// ErrLocked: the instance lock is held by another run.
var ErrLocked = errors.New("instance lock held by another run")

// layout is the data-dir layout from the environment; without a home it falls back to the temp dir.
func layout() paths.Layout {
	l, err := paths.FromEnv()
	if err != nil {
		l = paths.Layout{Root: filepath.Join(os.TempDir(), "d2bot")}
	}
	return l
}

// LockPaths are the per-instance flock files shared with the dev tools (tools/live.sh takes the same
// files; tools/lib-paths.sh computes them in shell): the paths.Instance.GameLock first, then the legacy
// /tmp/d2lod-game[-<id>].lock that tools and services of the old layout still use. A run holds all of them.
func LockPaths(id string) []string { return layout().GameLocks(id) }

// HelperLockPaths are the helper-lock files of an instance: <run>/helper-<id>.lock, then the legacy
// /tmp/d2lod-helper-<id>.lock.
func HelperLockPaths(id string) []string {
	return []string{filepath.Join(layout().Run(), "helper-"+id+".lock"), "/tmp/d2lod-helper-" + id + ".lock"}
}

// LockHeld reports whether another process holds the lock of instance id right now (any of its files).
func LockHeld(id string) bool {
	for _, p := range LockPaths(id) {
		f, err := os.OpenFile(p, os.O_RDONLY|os.O_CREATE, 0o666)
		if err != nil {
			continue
		}
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		}
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return true
		}
	}
	return false
}

// LockInstance takes the instance flock files (see LockPaths), waiting up to wait for each. The returned
// func releases them. On failure nothing stays locked and the error wraps ErrLocked.
func LockInstance(id string, wait time.Duration) (func(), error) {
	return lockFiles(LockPaths(id), wait)
}

// LockHelper takes the small per-instance helper lock (never the game lock, which a live run holds for
// its whole duration), so concurrent helper (re)starts do not both spawn.
func LockHelper(id string, wait time.Duration) (func(), error) {
	return lockFiles(HelperLockPaths(id), wait)
}

func lockFiles(ps []string, wait time.Duration) (func(), error) {
	var held []*os.File
	release := func() {
		for i := len(held) - 1; i >= 0; i-- {
			syscall.Flock(int(held[i].Fd()), syscall.LOCK_UN)
			held[i].Close()
		}
	}
	deadline := time.Now().Add(wait)
	for _, p := range ps {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			release()
			return nil, err
		}
		f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o666)
		if err != nil {
			release()
			return nil, err
		}
		for {
			err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
			if err == nil {
				held = append(held, f)
				break
			}
			if time.Now().After(deadline) {
				f.Close()
				release()
				return nil, fmt.Errorf("%w: %s", ErrLocked, p)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	return release, nil
}
