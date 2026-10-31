// d2container is the container entrypoint skeleton and the spike harness (WP-CONTAINER).
//
//	d2container run [-display :99] [-screen 832x640x24] -- CMD ARGS...   supervise CMD under a managed Xvfb
//	d2container spike -winexe /opt/probe/winprobe.exe                      run the container spike, print RESULT lines
//	d2container mkchar -name N -class C                                     create one character through the OOG flow (under run)
//	d2container hold                                                       start Xvfb and wait for SIGTERM (stop-time test)
//	d2container probe PID | fdprobe SOCK                                   helper readers used by spike
//
// It is a child subreaper (so Wine's detached processes stay readable through /proc/pid/mem without
// CAP_SYS_PTRACE), reaps orphans (it may be PID 1) and turns SIGTERM into wineserver -k plus a clean exit.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"d2lod-bot/internal/fleet"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/memfd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: d2container run|spike|hold|probe|fdprobe")
		os.Exit(2)
	}
	if err := launch.BecomeSubreaper(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var rc int
	switch os.Args[1] {
	case "run":
		rc = run(os.Args[2:])
	case "spike":
		rc = spike(os.Args[2:])
	case "mkchar":
		rc = mkchar(os.Args[2:])
	case "hold":
		rc = hold()
	case "probe":
		pid, _ := strconv.Atoi(os.Args[2])
		rc = probe(pid)
	case "fdprobe":
		rc = fdprobe(os.Args[2])
	default:
		rc = 2
	}
	os.Exit(rc)
}

// reaper collects orphans until ctx ends. launch.Reap skips exec.Cmd children only if they are
// registered with the Spawner; plain exec.Cmd children here are waited by Cmd.Wait, so we only reap
// zombies whose parent is us and that Cmd does not own (see launch.Reap).
func reaper(ctx context.Context) {
	reaperSkip(ctx, nil)
}

// reaperSkip is reaper that never waits for the pid in skip (the supervised command, whose exit status
// belongs to Cmd.Wait; launch.Reap cannot know it, so a tick between its death and Wait would eat the status).
func reaperSkip(ctx context.Context, skip *atomic.Int64) {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			reapOrphans(skip)
		}
	}
}

func reapOrphans(skip *atomic.Int64) {
	ps, err := launch.ListProcs()
	if err != nil {
		return
	}
	me := os.Getpid()
	for _, p := range ps {
		if p.PPID != me || p.State != 'Z' || (skip != nil && int64(p.PID) == skip.Load()) {
			continue
		}
		var ws syscall.WaitStatus
		syscall.Wait4(p.PID, &ws, syscall.WNOHANG, nil)
	}
}

// exitCode maps a Wait error to the shell convention: the real exit code, or 128+signal when killed.
func exitCode(err error) int {
	ee, ok := err.(*exec.ExitError)
	if !ok {
		if err != nil {
			return 1
		}
		return 0
	}
	if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return ee.ExitCode()
}

func startXvfb(display, screen string) (*exec.Cmd, error) {
	x := exec.Command("Xvfb", display, "-screen", "0", screen, "-nolisten", "tcp")
	x.Stderr = os.Stderr
	if err := x.Start(); err != nil {
		return nil, err
	}
	sock := "/tmp/.X11-unix/X" + strings.TrimPrefix(display, ":")
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(sock); err == nil {
			return x, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	x.Process.Kill()
	return nil, fmt.Errorf("Xvfb did not create %s", sock)
}

func envDuration(key string, def time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil && d > 0 {
		return d
	}
	return def
}

func wineKill() {
	exec.Command("wineserver", "-k").Run()
}

func run(args []string) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	display := fs.String("display", ":99", "X display for the managed Xvfb")
	screen := fs.String("screen", "832x640x24", "Xvfb screen (800x600 game window plus Wine frame; 800x600 clips the bottom 20 px)")
	grace := fs.Duration("grace", envDuration("D2CONTAINER_GRACE", fleet.ContainerGrace), "wait this long for the command to exit after SIGTERM before killing it (keep below the container stop_grace_period)")
	fs.Parse(args)
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "run: command required after --")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	var child atomic.Int64
	go reaperSkip(ctx, &child)
	x, err := startXvfb(*display, *screen)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() { x.Process.Kill(); x.Wait() }()
	os.Setenv("DISPLAY", *display)
	cmd := exec.Command(fs.Arg(0), fs.Args()[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	child.Store(int64(cmd.Process.Pid))
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		wineKill()
		return exitCode(err)
	case <-ctx.Done():
		cmd.Process.Signal(syscall.SIGTERM)
		var err error
		select {
		case err = <-done:
		case <-time.After(*grace):
			cmd.Process.Kill()
			err = <-done
		}
		wineKill()
		// A clean stop (the command exited 0 or by our SIGTERM) is 0; anything else is the command's real status.
		if ee, ok := err.(*exec.ExitError); ok {
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() && ws.Signal() == syscall.SIGTERM {
				return 0
			}
		}
		return exitCode(err)
	}
}

func hold() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	go reaper(ctx)
	x, err := startXvfb(":99", "800x600x24")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("hold: pid", os.Getpid(), "Xvfb up")
	<-ctx.Done()
	wineKill()
	x.Process.Kill()
	x.Wait()
	return 0
}

func probe(pid int) int {
	f, err := launch.OpenMem(pid)
	if err != nil {
		fmt.Printf("open /proc/%d/mem: %v\n", pid, err)
		return 1
	}
	defer f.Close()
	base, err := launch.ImageBase(pid, "winprobe.exe")
	if err != nil {
		fmt.Printf("image base: %v\n", err)
		return 1
	}
	var b [2]byte
	if _, err := f.ReadAt(b[:], int64(base)); err != nil {
		fmt.Printf("read: %v\n", err)
		return 1
	}
	fmt.Printf("ok read %q at 0x%x\n", b[:], base)
	return 0
}

func fdprobe(sock string) int {
	c := &memfd.Client{Path: sock}
	defer c.Close()
	src, info, err := c.Get("game")
	if err != nil {
		fmt.Println("get:", err)
		return 1
	}
	base, _ := strconv.ParseUint(strings.TrimPrefix(info[strings.Index(info, "base=")+5:], "0x"), 16, 64)
	b := make([]byte, 2)
	if err := src.ReadAt(b, uint32(base)); err != nil {
		fmt.Println("read via passed fd:", err)
		return 1
	}
	fmt.Printf("ok read %q via passed fd (%s)\n", b, info)
	return 0
}

func self(args ...string) (string, error) {
	exe, _ := os.Executable()
	out, err := exec.Command(exe, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func status(pid int, keys ...string) string {
	b, _ := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		for _, k := range keys {
			if strings.HasPrefix(l, k+":") {
				out = append(out, strings.Join(strings.Fields(l), " "))
			}
		}
	}
	return strings.Join(out, "; ")
}

// usage sums Pss and utime+stime ticks over every process in the container.
func usage() (pssKB, ticks int, byName map[string]int) {
	byName = map[string]int{}
	ps, _ := launch.ListProcs()
	for _, p := range ps {
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/smaps_rollup", p.PID))
		if err != nil {
			continue
		}
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "Pss:") {
				f := strings.Fields(l)
				v, _ := strconv.Atoi(f[1])
				pssKB += v
				n := filepath.Base(strings.Fields(p.Comm + " ")[0])
				byName[n] += v
			}
		}
		t, _ := launch.CPUTicks(p.PID)
		ticks += int(t)
	}
	return
}

func spike(args []string) int {
	fs := flag.NewFlagSet("spike", flag.ExitOnError)
	winexe := fs.String("winexe", "/opt/probe/winprobe.exe", "windows/386 probe")
	idle := fs.Duration("idle", 20*time.Second, "idle measurement window")
	fs.Parse(args)
	pid := os.Getpid()
	yama, _ := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	sub, _ := launch.IsSubreaper()
	fmt.Printf("RESULT env: pid=%d %s ptrace_scope=%s subreaper=%v display-before=%q\n", pid, status(pid, "CapEff", "Uid"), strings.TrimSpace(string(yama)), sub, os.Getenv("DISPLAY"))
	if b, err := os.ReadFile("/proc/1/comm"); err == nil {
		fmt.Printf("RESULT pid1: /proc/1/comm=%s (we are pid %d)\n", strings.TrimSpace(string(b)), pid)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	go reaper(ctx)

	t0 := time.Now()
	x, err := startXvfb(":99", "800x600x24")
	if err != nil {
		fmt.Println("RESULT xvfb: FAIL", err)
		return 1
	}
	defer func() { x.Process.Kill(); x.Wait() }()
	fmt.Printf("RESULT xvfb: up in %s\n", time.Since(t0).Round(10*time.Millisecond))
	os.Setenv("DISPLAY", ":99")
	defer wineKill()

	t0 = time.Now()
	cmd := exec.Command("wine", *winexe)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("RESULT wine-run: launcher error", err)
	}
	fmt.Printf("RESULT wine-run: `wine winprobe.exe` (incl. first prefix creation) returned after %s\n", time.Since(t0).Round(100*time.Millisecond))

	procs, _ := launch.ListProcs()
	sl := launch.FindImage(procs, "winprobe.exe")
	if len(sl) == 0 {
		fmt.Println("RESULT find: FAIL no winprobe.exe process")
		return 1
	}
	p := sl[len(sl)-1]
	fmt.Printf("RESULT find: pid=%d ppid=%d (self=%d) descendant=%v argv=%q\n", p.PID, p.PPID, pid, launch.IsDescendant(procs, p.PID, pid), p.Argv)
	base, err := launch.ImageBase(p.PID, "winprobe.exe")
	fmt.Printf("RESULT imagebase: 0x%x err=%v\n", base, err)

	f, err := launch.OpenMem(p.PID)
	if err != nil {
		fmt.Println("RESULT self-read: FAIL", err)
	} else {
		var b [2]byte
		_, rerr := f.ReadAt(b[:], int64(base))
		fmt.Printf("RESULT self-read (subreaper, no caps needed): %q err=%v\n", b[:], rerr)
	}

	out, err := self("probe", strconv.Itoa(p.PID))
	fmt.Printf("RESULT sibling-direct-open (child of ours, not ancestor of target): %s (err=%v)\n", out, err)

	sock := "/tmp/memfd.sock"
	sctx, scancel := context.WithCancel(ctx)
	srv := &memfd.Server{Path: sock, Prov: func(string) (*os.File, string, error) {
		return f, fmt.Sprintf("pid=%d base=0x%x", p.PID, base), nil
	}}
	go srv.Serve(sctx)
	time.Sleep(500 * time.Millisecond)
	out, err = self("fdprobe", sock)
	fmt.Printf("RESULT sibling-via-memfd-fd-passing: %s (err=%v)\n", out, err)
	scancel()

	// idle cost: Xvfb + wineserver + sleeper, no game
	time.Sleep(3 * time.Second)
	pss0, t0k, _ := usage()
	time.Sleep(*idle)
	pss1, t1k, names := usage()
	hz := 100.0
	fmt.Printf("RESULT idle(Xvfb+wine+probe): PSS=%d MB (start %d MB) cpu=%.1f%% of one core over %s\n", pss1/1024, pss0/1024, float64(t1k-t0k)/hz/idle.Seconds()*100, *idle)
	var ns []string
	for n, v := range names {
		ns = append(ns, fmt.Sprintf("%s=%dMB", n, v/1024))
	}
	sort.Strings(ns)
	fmt.Println("RESULT idle-breakdown:", strings.Join(ns, " "))

	// reaping: kill the sleeper, it is our orphan child and stays a zombie until we wait
	syscall.Kill(p.PID, syscall.SIGKILL)
	time.Sleep(300 * time.Millisecond)
	zb := zombies()
	time.Sleep(3 * time.Second) // reaper ticks every 2 s
	za := zombies()
	fmt.Printf("RESULT reaping: zombies right after SIGKILL=%d, after reaper tick=%d\n", zb, za)

	// window capture
	nd := exec.Command("wine", "notepad")
	nd.Start()
	var shot string
	for i := 0; i < 60; i++ {
		time.Sleep(500 * time.Millisecond)
		if o, _ := exec.Command("xwininfo", "-root", "-tree").Output(); strings.Contains(string(o), "Untitled - Notepad") {
			shot = string(o)
			break
		}
	}
	if shot == "" {
		fmt.Println("RESULT capture: FAIL no notepad window appeared")
	} else {
		fmt.Println("RESULT window-found: Untitled - Notepad is a mapped X window")
		raw, err := exec.Command("xwd", "-root", "-silent").Output()
		distinct := colors(raw)
		fmt.Printf("RESULT capture: xwd -root = %d bytes err=%v distinct-pixel-values=%d (blank root would be 1)\n", len(raw), err, distinct)
		os.WriteFile("/tmp/shot.xwd", raw, 0o644)
	}
	nd.Process.Kill()
	nd.Wait()
	return 0
}

func zombies() int {
	n := 0
	ps, _ := launch.ListProcs()
	for _, p := range ps {
		if p.State == 'Z' {
			n++
		}
	}
	return n
}

// colors counts distinct 32-bit pixel values in an xwd dump (header+colormap skipped by taking the
// tail of the data that is a multiple of 4 bytes).
func colors(raw []byte) int {
	if len(raw) < 800*600*4 {
		return 0
	}
	px := raw[len(raw)-800*600*4:]
	seen := map[uint32]struct{}{}
	for i := 0; i+4 <= len(px); i += 4 {
		seen[uint32(px[i])|uint32(px[i+1])<<8|uint32(px[i+2])<<16|uint32(px[i+3])<<24] = struct{}{}
		if len(seen) > 1000 {
			break
		}
	}
	return len(seen)
}
