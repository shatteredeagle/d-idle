// d2dreplay records and plays back runs of the lockstep d2d fork (docs/d2d-replay.md).
//
//	d2dreplay record -upstream D2D.sock -listen TAP.sock -out RUN.replay.gz -d2d-bin BIN -seed N -class N -name X [-d2d-args "..."] [-every 500]
//	d2dreplay record -launch -run DIR -data MPQDIR -d2d-bin BIN -seed N -class N -name X -out FILE   (starts its own d2d under the lab lock)
//	d2dreplay play FILE -d2d-bin BIN -data MPQDIR [-run DIR] [-until TICK]
//	d2dreplay info FILE
//	d2dreplay render FILE -out DIR -d2d-bin BIN -data MPQDIR [-stride 500ms] [-fps 10] [-video]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"d2lod-bot/internal/d2dreplay"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var err error
	switch os.Args[1] {
	case "record":
		err = record(ctx, os.Args[2:])
	case "play":
		err = play(ctx, os.Args[2:], false)
	case "render":
		err = play(ctx, os.Args[2:], true)
	case "info":
		err = info(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		if errors.Is(err, errDiverged) {
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "d2dreplay:", err)
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: d2dreplay record|play|info|render ... (see docs/d2d-replay.md)")
	os.Exit(2)
}

var errDiverged = errors.New("diverged")

func logf(f string, a ...any) { fmt.Fprintf(os.Stderr, time.Now().Format("15:04:05 ")+f+"\n", a...) }

// d2dFlags are the launch flags shared by record -launch, play and render.
type d2dFlags struct {
	bin, data, run string
	nice           int
	lockWait       time.Duration
	noLock         bool
	settle, ready  time.Duration
}

func (d *d2dFlags) add(fs *flag.FlagSet) {
	fs.StringVar(&d.bin, "d2d-bin", os.Getenv("D2D_BIN"), "d2d lab binary (env D2D_BIN); hashed into the file / checked against it")
	fs.StringVar(&d.data, "data", os.Getenv("D2D_DATA"), "d2d --data MPQ dir (env D2D_DATA)")
	fs.StringVar(&d.run, "run", "", "private run dir for socket, saves and d2d log (default: a temp dir, removed afterwards)")
	fs.IntVar(&d.nice, "nice", 10, "nice level for d2d")
	fs.DurationVar(&d.lockWait, "lock-wait", 3000*time.Second, "how long to queue for a free d2d slot (pool of 4, docs/process/worker-compute.md)")
	fs.BoolVar(&d.noLock, "no-lock", false, "do not take a d2d slot (only when the caller already holds one, e.g. under tools/d2dslot.sh run)")
	fs.DurationVar(&d.settle, "settle", 2*time.Second, "extra wait after the game is in-game before the first verb (level load runs on real time)")
	fs.DurationVar(&d.ready, "ready", 30*time.Second, "max wait for screen=ingame")
}

// start takes the lab lock and launches d2d; the returned stop releases both.
func (d *d2dFlags) start(ctx context.Context, h d2dreplay.Header) (*d2dreplay.Proc, func(), error) {
	if d.bin == "" {
		return nil, nil, errors.New("-d2d-bin is required")
	}
	// Validate the recipe and header args before queueing for a slot (a bad file must not wait 50 min to fail).
	if _, err := (&d2dreplay.Launch{Sock: "x", Build: h.D2D, Recipe: h.Start}).Args(); err != nil {
		return nil, nil, err
	}
	release := func() {}
	if !d.noLock {
		slot, r, err := d2dreplay.TakeAny(ctx, d2dreplay.SlotLocks(), d.lockWait, logf)
		if err != nil {
			return nil, nil, err
		}
		logf("got d2d slot %d", slot)
		release = r
	}
	run := d.run
	cleanup := func() {}
	if run == "" {
		tmp, err := os.MkdirTemp("", "d2drp")
		if err != nil {
			release()
			return nil, nil, err
		}
		run, cleanup = tmp, func() { os.RemoveAll(tmp) }
	} else if err := os.MkdirAll(run, 0o700); err != nil {
		release()
		return nil, nil, err
	}
	logFile, err := os.Create(filepath.Join(run, "d2d.log"))
	if err != nil {
		release()
		cleanup()
		return nil, nil, err
	}
	l := &d2dreplay.Launch{Bin: d.bin, Data: d.data, Sock: filepath.Join(run, "d2d.sock"), Home: filepath.Join(run, "home"),
		Nice: d.nice, Build: h.D2D, Recipe: h.Start, Log: logFile}
	args, _ := l.Args()
	logf("starting d2d %s", strings.Join(args, " "))
	p, err := l.Start(ctx)
	if err != nil {
		logFile.Close()
		release()
		cleanup()
		return nil, nil, err
	}
	st, err := d2dreplay.WaitReady(ctx, p.Conn, d.ready)
	if err != nil {
		finish(p, logFile, release, cleanup)
		return nil, nil, err
	}
	logf("d2d up: %s", st)
	time.Sleep(d.settle)
	return p, func() { finish(p, logFile, release, cleanup) }, nil
}

// finish stops d2d first; the slot is released only when d2d is really gone (otherwise the kernel drops the lock
// when the process exits, and the run dir is left in place).
func finish(p *d2dreplay.Proc, log *os.File, release, cleanup func()) {
	gone := p.Stop()
	log.Close()
	if !gone {
		logf("WARNING: d2d did not exit; keeping its slot and run dir (kill it by hand)")
		return
	}
	release()
	cleanup()
}

// renderDir validates -out for render: no parent traversal, no characters that break ffmpeg's frame pattern or the
// screenshot verb, an absolute path (so ffmpeg never reads it as an option), created 0700, not a symlink.
func renderDir(out string) (string, error) {
	if out == "" {
		return "", errors.New("render: -out DIR is required")
	}
	if strings.Contains(out, "..") || strings.ContainsAny(out, "%\n\r\t\x00 ") || strings.ContainsFunc(out, unicode.IsControl) {
		return "", fmt.Errorf("render: -out %q has '..', whitespace, '%%' or a control character", out)
	}
	abs, err := filepath.Abs(out)
	if err != nil {
		return "", err
	}
	if st, err := os.Lstat(abs); err == nil {
		if !st.IsDir() {
			return "", fmt.Errorf("render: %s exists and is not a directory (symlinks are refused)", abs)
		}
	} else if err := os.MkdirAll(abs, 0o700); err != nil {
		return "", err
	}
	return abs, nil
}

func firstToken(v string) string {
	if f := strings.Fields(v); len(f) > 0 {
		return f[0]
	}
	return "?"
}

// pfx is the first 12 characters of a hash (shorter input is returned whole).
func pfx(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func record(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	var d d2dFlags
	d.add(fs)
	upstream := fs.String("upstream", "", "d2d's control socket (the bot or script dials -listen instead); not with -launch")
	listen := fs.String("listen", "", "the tap's socket for the client (required)")
	out := fs.String("out", "", "replay file to write (required, .replay.gz)")
	every := fs.Int("every", 250, "checkpoint interval in ticks (250 = 10 virtual s, the detloop journal default)")
	seed := fs.Int64("seed", 0, "map seed of the run (the --seed the d2d was started with)")
	class := fs.Int("class", -1, "class index of the fresh character (d2d UI order, 1 = Necromancer, 4 = Sorceress)")
	name := fs.String("name", "", "character name")
	d2dArgs := fs.String("d2d-args", "--toggle autoloot=off,autoparty=off --lockstep", "non-path d2d arguments of the run, space separated")
	launch := fs.Bool("launch", false, "start the d2d here (under the lab lock) from -seed/-class/-name instead of attaching to -upstream")
	idle := fs.Duration("idle", 30*time.Second, "stop after no client was connected for this long (0: only on quit or signal)")
	maxDur := fs.Duration("max", 60*time.Minute, "hard stop")
	note := fs.String("note", "", "free text stored in the file")
	bot := fs.String("bot", "", "bot version text stored in the file")
	fs.Parse(args)
	if *listen == "" || *out == "" || (*upstream == "" && !*launch) {
		return errors.New("record: -listen, -out and -upstream (or -launch) are required")
	}
	if _, err := d2dreplay.SafeArgs(strings.Fields(*d2dArgs)); err != nil {
		return err
	}
	h := d2dreplay.Header{Every: *every, D2D: d2dreplay.Build{Args: strings.Fields(*d2dArgs)}, Meta: d2dreplay.Meta{Note: *note, Bot: *bot, Tool: "d2dreplay record"}}
	if d.bin != "" {
		sum, err := d2dreplay.FileSHA256(d.bin)
		if err != nil {
			return err
		}
		h.D2D.SHA256 = sum
	}
	if *name != "" || *launch {
		if *class < 0 || *name == "" {
			return errors.New("record: -class and -name describe the fresh character (needed with -launch)")
		}
		h.Start = &d2dreplay.Recipe{Seed: *seed, Class: *class, Name: *name, Screen: "ingame"}
		h.Meta.Fresh = true
	}
	ctx, cancel := context.WithTimeout(ctx, *maxDur)
	defer cancel()
	up := *upstream
	if *launch {
		p, stop, err := d.start(ctx, h)
		if err != nil {
			return err
		}
		defer stop()
		p.Conn.Close() // the tap owns the only connection from here on
		up = sockOf(p)
	}
	w, err := d2dreplay.Create(*out, h)
	if err != nil {
		return err
	}
	t := &d2dreplay.Tap{Upstream: up, Listen: *listen, W: w, Every: *every, Idle: *idle, Logf: logf}
	logf("recording %s: client socket %s, upstream %s, checkpoint every %d ticks", *out, *listen, up, *every)
	err = t.Serve(ctx)
	st, _ := os.Stat(*out)
	size := int64(0)
	if st != nil {
		size = st.Size()
	}
	s := t.Stats()
	perHour := float64(0)
	if s.Ticks > 0 {
		perHour = float64(size) / (float64(s.Ticks) * d2dreplay.TickMS / 3600000)
	}
	logf("recorded %s: %s, %d bytes (%.0f bytes per virtual hour)", *out, s, size, perHour)
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// sockOf finds the --devctl path in a launched d2d's arguments.
func sockOf(p *d2dreplay.Proc) string {
	a := p.Cmd.Args
	for i := range a {
		if a[i] == "--devctl" && i+1 < len(a) {
			return a[i+1]
		}
	}
	return ""
}

func play(ctx context.Context, args []string, render bool) error {
	name := "play"
	if render {
		name = "render"
	}
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	var d d2dFlags
	d.add(fs)
	until := fs.Int("until", 0, "stop after the step that reaches this tick")
	gapMin := fs.Duration("gap-min", time.Second, "reproduce recorded wall gaps at least this long (0: never)")
	out := fs.String("out", "", "render: directory for frames (f000001.png ...) and the video")
	stride := fs.Duration("stride", 500*time.Millisecond, "render: virtual time between frames")
	fps := fs.Int("fps", 10, "render: video frame rate (speed factor = stride x fps)")
	video := fs.Bool("video", true, "render: make replay.mp4 with ffmpeg when it is on PATH (never installed)")
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("%s: one replay file", name)
	}
	rp, err := d2dreplay.Read(fs.Arg(0))
	if err != nil {
		return err
	}
	if rp.Truncated {
		logf("note: the file has no trailer (recorder was stopped early), playing the readable prefix")
	}
	var absOut string
	if render {
		if absOut, err = renderDir(*out); err != nil {
			return err
		}
	}
	if d.bin != "" && rp.Header.D2D.SHA256 != "" {
		sum, err := d2dreplay.FileSHA256(d.bin)
		if err != nil {
			return err
		}
		if sum != rp.Header.D2D.SHA256 {
			logf("WARNING: d2d build differs: file %s.., binary %s.. (a divergence may be the build)", pfx(rp.Header.D2D.SHA256), pfx(sum))
		} else {
			logf("d2d build matches the file (%s..)", pfx(sum))
		}
	}
	p, stop, err := d.start(ctx, rp.Header)
	if err != nil {
		return err
	}
	defer stop()
	pl := &d2dreplay.Player{Conn: p.Conn, GapMin: *gapMin, Until: *until, Logf: logf}
	frames := 0
	if render {
		abs := absOut
		strideTicks := max(1, int(*stride/(d2dreplay.TickMS*time.Millisecond)))
		next := 0
		pl.AfterStep = func(tick int) error {
			for tick >= next {
				frames++
				path := filepath.Join(abs, fmt.Sprintf("f%06d.png", frames))
				_, status, err := p.Conn.Do("screenshot " + path)
				if err != nil {
					return err
				}
				if d2dreplay.IsErr(status) {
					return fmt.Errorf("screenshot: %s", status)
				}
				next += strideTicks
			}
			return nil
		}
	}
	t0 := time.Now()
	res, err := pl.Run(ctx, rp)
	wall := time.Since(t0)
	if err != nil {
		return err
	}
	fmt.Println(res.String())
	logf("played %d ticks in %s (%.0fx real time)", res.Ticks, wall.Round(time.Millisecond), float64(res.Ticks)*d2dreplay.TickMS/float64(wall.Milliseconds()+1))
	if render {
		logf("%d frames in %s", frames, absOut)
		if *video && frames > 0 {
			if ff, err := exec.LookPath("ffmpeg"); err == nil {
				mp4 := filepath.Join(absOut, "replay.mp4")
				cmd := exec.CommandContext(ctx, ff, "-nostdin", "-y", "-loglevel", "error", "-framerate", fmt.Sprint(*fps), "-i", filepath.Join(absOut, "f%06d.png"),
					"-pix_fmt", "yuv420p", "-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2", mp4)
				cmd.Stderr = os.Stderr
				if err := cmd.Run(); err != nil {
					logf("ffmpeg failed: %v", err)
				} else {
					logf("video %s (%d fps, %.1fx real time)", mp4, *fps, stride.Seconds()*float64(*fps))
				}
			} else {
				logf("ffmpeg not on PATH: frames only, no video")
			}
		}
	}
	if res.Diverged != nil {
		return errDiverged
	}
	return nil
}

func info(args []string) error {
	if len(args) != 1 {
		return errors.New("info: one replay file")
	}
	rp, err := d2dreplay.Read(args[0])
	if err != nil {
		return err
	}
	h := rp.Header
	st, _ := os.Stat(args[0])
	fmt.Printf("file      %s (%d bytes)%s\n", args[0], st.Size(), map[bool]string{true: " TRUNCATED (no trailer)", false: ""}[rp.Truncated])
	fmt.Printf("format    replay v%d, hash %s, checkpoint every %d ticks, created %s\n", h.Replay, h.Hash, h.Every, h.Created)
	fmt.Printf("d2d       sha256 %s args %s\n", h.D2D.SHA256, strings.Join(h.D2D.Args, " "))
	if h.Start != nil {
		fmt.Printf("start     seed %d class %d name %s screen %s hardcore=%v classic=%v\n", h.Start.Seed, h.Start.Class, h.Start.Name, h.Start.Screen, h.Start.Hardcore, h.Start.Classic)
	}
	if h.Save != nil {
		fmt.Printf("save      %s sha256 %s seed %d\n", h.Save.Name, h.Save.SHA256, h.Save.Seed)
	}
	if h.Meta != (d2dreplay.Meta{}) {
		fmt.Printf("meta      tool=%q bot=%q area=%q class=%q note=%q\n", h.Meta.Tool, h.Meta.Bot, h.Meta.Area, h.Meta.Class, h.Meta.Note)
	}
	verbs, restarts := 0, 0
	byVerb := map[string]int{}
	for _, e := range rp.Entries {
		switch e.Kind() {
		case "verb":
			verbs++
			byVerb[firstToken(e.Verb)]++
		case "restart":
			restarts++
		}
	}
	ticks := rp.Ticks()
	cps := rp.Checkpoints()
	fmt.Printf("body      %d verbs, %d ticks (virtual %s), %d checkpoints, %d restarts\n", verbs, ticks, (time.Duration(ticks) * d2dreplay.TickMS * time.Millisecond).Round(time.Second), len(cps), restarts)
	var kinds []string
	for k, n := range byVerb {
		kinds = append(kinds, fmt.Sprintf("%s=%d", k, n))
	}
	fmt.Printf("verbs     %s\n", strings.Join(kinds, " "))
	if rp.End != nil {
		fmt.Printf("trailer   ticks %d verbs %d wall %s\n", rp.End.Ticks, rp.End.Verbs, (time.Duration(rp.End.WallMS) * time.Millisecond).Round(time.Second))
	}
	if ticks > 0 {
		fmt.Printf("size      %.0f bytes per virtual hour\n", float64(st.Size())/(float64(ticks)*d2dreplay.TickMS/3600000))
	}
	for i, c := range cps {
		if i < 10 || i == len(cps)-1 {
			fmt.Printf("  cp %3d tick %7d (virtual %7s) clvl %d area %d life %s %s..\n", i, c.C, (time.Duration(c.C) * d2dreplay.TickMS * time.Millisecond).Round(time.Second), c.Lv, c.Area, c.Life, pfx(c.H))
		} else if i == 10 {
			fmt.Printf("  ... %d more\n", len(cps)-11)
		}
	}
	return nil
}
