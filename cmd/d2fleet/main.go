// Command d2fleet runs and drives the fleet supervisor of one bot host: instance slots that are added and
// removed at runtime (docs/fleet.md). `serve` runs internal/server: the admin API (internal/fleet.Handler) is at
// /api/v1/fleet-admin/, and /api/v1/fleet lists the slots for the dashboard fleet strip. The client commands are
// thin HTTP calls to a running `d2fleet serve`.
//
//	d2fleet serve [-addr 127.0.0.1:8431] [-data DIR] [-token-file F]   run the supervisor (SIGTERM drains every slot)
//	(serve starts a d2bot --managed per slot unless -no-bots; docs/container.md "Bot launcher")
//	d2fleet add CHARACTER [-class sorceress] [-role killer] [-profile P] [-goals G] [-build B] [-difficulty D] [-hc]
//	d2fleet remove ID [-force]    list    status    restart ID [reason]
//
// Client flags: -url (default $D2FLEET_URL or http://127.0.0.1:8431/api/v1/fleet-admin), token from $D2FLEET_TOKEN.
// The serve process must be able to read game memory (ptrace rights, or the subreaper design of docs/launch.md)
// for the OOG login and leave steps; starting, stopping and admission need nothing special.
package main

import (
	"bytes"
	"context"
	"d2lod-bot/internal/buildinfo"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/event"
	"d2lod-bot/internal/fleet"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/party"
	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/server"
	"d2lod-bot/internal/setup"
	"d2lod-bot/internal/setup/setuptest"
	"d2lod-bot/internal/viewport"
	"d2lod-bot/web"
)

func main() {
	buildinfo.Exit("d2fleet")
	setup.MaybeProbe() // the wizard's subreaper probe re-executes this binary
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, "usage: d2fleet serve|add|remove|list|status|restart ...")
		return 2
	}
	cmd, rest := args[0], args[1:]
	if cmd == "serve" {
		return serve(rest, out, errw)
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(errw)
	base := fs.String("url", envOr("D2FLEET_URL", "http://127.0.0.1:8431/api/v1/fleet-admin"), "d2fleet serve admin URL")
	class := fs.String("class", "", "class (used when the character is created)")
	role := fs.String("role", "", "party role")
	profile := fs.String("profile", "", "handicap profile (optimal, relaxed, ... see bot.Profiles)")
	goalsF := fs.String("goals", "", "goal presets: campaign:T,levelto:N,farm:AREA")
	build := fs.String("build", "", "build key")
	diff := fs.String("difficulty", "", "normal|nightmare|hell")
	hc := fs.Bool("hc", false, "hardcore")
	force := fs.Bool("force", false, "remove even when leased")
	// flags may follow the positional argument: split them off
	var pos []string
	var fl []string
	for i := 0; i < len(rest); i++ {
		if strings.HasPrefix(rest[i], "-") {
			fl = append(fl, rest[i])
			if !strings.Contains(rest[i], "=") && rest[i] != "-hc" && rest[i] != "-force" && i+1 < len(rest) {
				i++
				fl = append(fl, rest[i])
			}
		} else {
			pos = append(pos, rest[i])
		}
	}
	if err := fs.Parse(fl); err != nil {
		return 2
	}
	c := &client{base: strings.TrimRight(*base, "/"), token: os.Getenv("D2FLEET_TOKEN"), out: out, errw: errw}
	switch cmd {
	case "add":
		if len(pos) != 1 {
			fmt.Fprintln(errw, "usage: d2fleet add CHARACTER [flags]")
			return 2
		}
		return c.do("POST", "/slots", fleet.Spec{Character: pos[0], Class: *class, Role: roleOf(*role), Profile: *profile, Goals: *goalsF, Build: *build, Difficulty: *diff, Hardcore: *hc})
	case "remove":
		if len(pos) != 1 {
			fmt.Fprintln(errw, "usage: d2fleet remove ID [-force]")
			return 2
		}
		p := "/slots/" + pos[0]
		if *force {
			p += "?force=1"
		}
		return c.do("DELETE", p, nil)
	case "list":
		return c.do("GET", "/slots", nil)
	case "status":
		return c.do("GET", "/status", nil)
	case "restart":
		if len(pos) < 1 {
			fmt.Fprintln(errw, "usage: d2fleet restart ID [reason]")
			return 2
		}
		return c.do("POST", "/slots/"+pos[0]+"/restart", map[string]string{"reason": strings.Join(pos[1:], " ")})
	}
	fmt.Fprintln(errw, "unknown command", cmd)
	return 2
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func roleOf(s string) party.Role { return party.Role(s) }

type client struct {
	base, token string
	out, errw   io.Writer
}

func (c *client) do(method, path string, body any) int {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, rd)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	cl := &http.Client{Timeout: 3 * time.Minute} // remove drains through OOG
	res, err := cl.Do(req)
	if err != nil {
		fmt.Fprintln(c.errw, "d2fleet:", err)
		return 1
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if len(b) > 0 {
		fmt.Fprintf(c.out, "%s", b)
	}
	if res.StatusCode >= 300 {
		fmt.Fprintf(c.errw, "d2fleet: HTTP %d\n", res.StatusCode)
		return 1
	}
	return 0
}

func serve(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(errw)
	addr := fs.String("addr", "127.0.0.1:8431", "listen address (loopback by default)")
	data := fs.String("data", "", "data dir (default paths.FromEnv)")
	tokenFile := fs.String("token-file", "", "file holding the bearer token (empty = none; loopback only)")
	withSetup := fs.Bool("setup", envOr("D2FLEET_SETUP", "") == "1", "serve the first-run wizard at /setup.html; slots are restored only once it is done (env D2FLEET_SETUP=1)")
	display := fs.String("setup-display", envOr("DISPLAY", ""), "X display the wizard's installer windows appear on")
	noBots := fs.Bool("no-bots", false, "start only the game and helper of a slot, no d2bot process")
	stopGrace := fs.Duration("stop-grace", fleet.DefaultStopGrace, "how long d2bot may take to save and exit on SIGTERM before it is killed (a container needs stop_grace_period above this)")
	soak := fs.String("soak-dir", envOr("D2BOT_SOAK_DIR", ""), "directory of soak runs (~/d2lod-live/soak): show the newest run's alerts and status, read-only (env D2BOT_SOAK_DIR; default off)")
	loadFactor := fs.Float64("max-load-factor", envFloat("D2FLEET_MAX_LOAD_FACTOR"), "admission: refuse a slot when host load1 + 1 > cores*FACTOR (0 = 1.25, negative = off). In a container load1 is the HOST's, so a busy host needs a negative value (env D2FLEET_MAX_LOAD_FACTOR)")
	fake := fs.Bool("fake", false, "TEST ONLY: fake wizard host effects and a runtime that starts no game (tools/container-checklist.sh)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	l, err := paths.FromEnv()
	if *data != "" {
		l, err = paths.Layout{Root: *data}, nil
	}
	if err != nil {
		fmt.Fprintln(errw, "d2fleet:", err)
		return 1
	}
	token := ""
	if *tokenFile != "" {
		b, err := os.ReadFile(*tokenFile)
		if err != nil {
			fmt.Fprintln(errw, "d2fleet:", err)
			return 1
		}
		token = strings.TrimSpace(string(b))
	}
	life := launch.NewLife(out)
	if !dirExists(filepath.Join(life.Base.Root, "proton", "pfx")) {
		// No main prefix (a container): the wizard's base prefix (built later in this process on a fresh volume)
		// is the clone source of every slot prefix.
		life.Base.PrefixDir = setup.BasePrefix(l.Root)
		_ = os.MkdirAll(filepath.Join(life.Base.Root, "logs"), 0o755) // Xvfb and helper logs of the shared base config
	}
	bus := event.NewBus("host", nil)
	rt := fleet.LifeRuntime{Life: life, Flow: func(id string) (*launch.Flow, io.Closer, error) {
		src, err := mem.OpenInstance(id)
		if err != nil {
			return nil, nil, err
		}
		cli := input.NewClient(input.AddrFor(id))
		return &launch.Flow{Mem: src, In: flowInput{input.NewOOG(cli), cli}}, src, nil
	}}
	var runtime fleet.Runtime = rt
	if life.Base.PrefixDir != "" {
		runtime = seedRuntime{LifeRuntime: rt, base: life.Base.PrefixDir}
	}
	// CleanGame ends only the orphan Game.exe: the Xvfb and helper the fleet prepared stay up.
	if *fake {
		runtime = fakeRuntime{}
	}
	var bots fleet.BotRunner
	if !*fake && !*noBots {
		bots = &fleet.ProcBots{Layout: l, MaxRestarts: 10,
			StopGrace: *stopGrace,
			CleanGame: func(ctx context.Context, a fleet.Alloc) { _ = rt.StopGame(ctx, a) }}
	}
	f, err := fleet.New(fleet.Config{
		Runtime: runtime, Bots: bots, Bus: bus, DataDir: l.Root, Limits: fleet.Limits{MaxLoadFactor: *loadFactor},
		Prefix:      l.GameRoot, // the Wine-side root launch uses (legacy ~/Games/d2lod kept in place)
		StateFile:   filepath.Join(l.Root, "fleet.json"),
		MaxRestarts: 10,
	})
	if err != nil {
		fmt.Fprintln(errw, "d2fleet:", err)
		return 1
	}
	sub := bus.Subscribe("log", 256)
	go func() {
		for e := range sub.C() {
			b, _ := json.Marshal(e)
			fmt.Fprintf(out, "%s\n", b)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	vh := newViewHub(viewport.XWDCapturer{})
	defer vh.Close()
	cfg := server.Config{
		Addr: *addr, Token: token, Version: "d2fleet", Assets: web.FS,
		FleetAdmin: fleet.Handler(f, ""), // the server's guard authenticates
		Slots:      func() []server.Slot { return slotsOf(f) },
		Remotes:    func() []server.Remote { return remotesOf(f, vh) },
		SoakDir:    *soak,
	}
	restore := func() {
		for _, e := range f.Restore(ctx) {
			fmt.Fprintln(errw, "d2fleet:", e)
		}
	}
	if *withSetup {
		// The wizard's character step starts the game from this process: it must own the game's process tree.
		wc := setup.WebConfig{DataDir: l.Root, Display: *display, BaseCtx: ctx}
		if *fake {
			wc.Deps = setuptest.Deps(l.Root)
		} else if err := launch.BecomeSubreaper(); err != nil {
			fmt.Fprintln(errw, "d2fleet: subreaper:", err)
			return 1
		} else {
			// Adopted Wine children that exit (the wizard's character game) would stay zombies otherwise.
			go (&launch.ZombieReaper{}).Run(ctx, 0)
		}
		wz, err := setup.NewWeb(wc)
		if err != nil {
			fmt.Fprintln(errw, "d2fleet:", err)
			return 1
		}
		cfg.Setup = wz.Handler()
		// Bots stay paused until the wizard has a character and has finished.
		go func() {
			if err := afterWizard(ctx, wz, f); err != nil && ctx.Err() == nil {
				fmt.Fprintln(errw, "d2fleet:", err)
			}
		}()
	} else {
		restore()
	}
	srv, err := server.New(cfg)
	if err != nil {
		fmt.Fprintln(errw, "d2fleet:", err)
		return 1
	}
	ln, err := srv.Listen()
	if err != nil {
		fmt.Fprintln(errw, "d2fleet:", err)
		return 1
	}
	fmt.Fprintf(out, "d2fleet: admin API on %s/api/v1/fleet-admin/, data %s\n", ln.Addr(), l.Root)
	if err := srv.Serve(ctx, ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(errw, "d2fleet:", err)
		return 1
	}
	dctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if err := f.Close(dctx); err != nil {
		fmt.Fprintln(errw, "d2fleet: drain:", err)
		return 1
	}
	return 0
}

// afterWizard waits for the wizard, restores the saved slots and makes sure the wizard's character has a
// slot (its class and goals come from the wizard inputs). Nothing starts before the wizard is done.
func afterWizard(ctx context.Context, wz *setup.Web, f *fleet.Fleet) error {
	if err := wz.WaitReady(ctx); err != nil {
		return err
	}
	var errs []error
	errs = append(errs, f.Restore(ctx)...)
	in := wz.W.Inputs()
	if in.CharName != "" {
		have := false
		for _, v := range f.List() {
			have = have || v.Spec.Character == in.CharName
		}
		if !have {
			_, err := f.Add(ctx, fleet.Spec{Character: in.CharName, Class: in.CharClass, Profile: in.Profile, Goals: in.Goals})
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// seedRuntime copies the wizard's character from the base prefix into a slot's own save dir (slot prefixes are
// clones that exclude saves) once, before the slot's bot first looks for it. Used when slots clone the wizard base.
type seedRuntime struct {
	fleet.LifeRuntime
	base string
}

func (s seedRuntime) Prepare(ctx context.Context, a fleet.Alloc, sp fleet.Spec) error {
	if err := s.LifeRuntime.Prepare(ctx, a, sp); err != nil {
		return err
	}
	rel := filepath.Join("drive_c", "users", "steamuser", "Saved Games", "Diablo II")
	return seedCharacter(filepath.Join(s.base, "pfx", rel), filepath.Join(a.Prefix, "proton", "pfx", rel), sp.Character)
}

// seedCharacter copies <name>.* from src to dst unless dst already has the character.
func seedCharacter(src, dst, name string) error {
	if name == "" {
		return nil
	}
	if _, err := os.Stat(filepath.Join(dst, name+".d2s")); err == nil {
		return nil
	}
	files, _ := filepath.Glob(filepath.Join(src, name+".*"))
	if len(files) == 0 {
		return nil
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, filepath.Base(f)), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// fakeRuntime starts no game (serve -fake).
type fakeRuntime struct{}

func (fakeRuntime) Start(context.Context, fleet.Alloc, fleet.Spec) error   { return nil }
func (fakeRuntime) Prepare(context.Context, fleet.Alloc, fleet.Spec) error { return nil }
func (fakeRuntime) Login(context.Context, fleet.Alloc, fleet.Spec) error   { return nil }
func (fakeRuntime) Alive(context.Context, fleet.Alloc) (bool, error)       { return true, nil }
func (fakeRuntime) LeaveGame(context.Context, fleet.Alloc) error           { return nil }
func (fakeRuntime) Stop(context.Context, fleet.Alloc) error                { return nil }

type flowInput struct {
	*input.OOG
	c *input.Client
}

func (t flowInput) TypeChar(r rune) error { return t.c.TypeChar(r) }

func slotsOf(f *fleet.Fleet) []server.Slot {
	var out []server.Slot
	for _, v := range f.List() {
		sl := server.Slot{ID: v.ID, Title: v.Spec.Character, State: string(v.State), Error: v.LastError}
		if v.Bot != nil && sl.Error == "" && v.Bot.State != fleet.BotRunning && v.Bot.LastError != "" {
			sl.Error = "d2bot " + v.Bot.State + ": " + v.Bot.LastError
		}
		out = append(out, sl)
	}
	return out
}

func envFloat(k string) float64 {
	v, _ := strconv.ParseFloat(os.Getenv(k), 64)
	return v
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
