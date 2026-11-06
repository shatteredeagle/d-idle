package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/watch"
)

// The decision ladder and the player screen on the real bot (docs/t0-slice.md "How to run it"). Both are off by default:
// without -ladder nothing in this file runs, no directory is made and no port is opened.

type ladderOptions struct {
	on      bool
	profile string // "" = a person answers through the watch screen; NAME or holdout:SEED = a scripted player
	watch   bool
	store   string // "" = a store per D2BOT_RUN when a run is named, else the shared one; "shared" = the shared one; else a named store
	// -watch: the read-only viewer and the SEPARATE answer listener (same semantics as d2dview -watch).
	viewAddr, answerAddr, origins string
	// The watch screen's OWN access token (viewer GETs and answer/orders POSTs), never the bot API token.
	watchToken, watchTokenFile string
	// -watch-video: the game window as MJPEG on the viewer (off by default; capture only while someone watches).
	video        bool
	videoFPS     int
	videoQuality int
	instance     string // set by the parser: the instance whose private display is captured
}

func ladderFlags(fs *flag.FlagSet, l *ladderOptions) {
	fs.BoolVar(&l.on, "ladder", envBool("D2BOT_LADDER"), "turn the decision ladder on (knob ladder.enabled 1): the T0/T1 points ask a store whose untrained answer is the legacy one; state under <instance state>/ladder (env D2BOT_LADDER; default off)")
	fs.StringVar(&l.profile, "ladder-profile", envOr("D2BOT_LADDER_PROFILE", ""), "with --ladder: a scripted player for drills (cautious, bold, inconsistent, holdout:SEED); default none, a person answers on the watch screen")
	fs.StringVar(&l.store, "ladder-store", envOr("D2BOT_LADDER_STORE", ""), "with --ladder: the store under <instance state>/ladder/runs/NAME; default the run named by D2BOT_RUN (so drills are independent), or the shared store when no run is named; \"shared\" forces the shared store (learning carries on across sessions)")
	fs.BoolVar(&l.watch, "watch", envBool("D2BOT_WATCH"), "with --ladder: serve the player screen (/watch) from this process (env D2BOT_WATCH; default off)")
	fs.StringVar(&l.viewAddr, "watch-addr", envOr("D2BOT_WATCH_ADDR", "127.0.0.1:8891"), "with --watch: listen address of the read-only viewer; non-loopback requires --watch-token or --watch-token-file")
	fs.StringVar(&l.answerAddr, "watch-answer-addr", envOr("D2BOT_WATCH_ANSWER_ADDR", "127.0.0.1:8892"), "with --watch: listen address of the SEPARATE answer route (answer, confirm, revoke, orders); non-loopback requires --watch-token or --watch-token-file")
	fs.StringVar(&l.origins, "watch-origin", envOr("D2BOT_WATCH_ORIGIN", ""), "with --watch: comma list of page origins the answer route accepts (default: the viewer's port on localhost)")
	fs.StringVar(&l.watchToken, "watch-token", envOr("D2BOT_WATCH_TOKEN", ""), "with --watch: access token of the watch screen, separate from the bot API token --token (env D2BOT_WATCH_TOKEN; it shows in ps, prefer --watch-token-file)")
	fs.StringVar(&l.watchTokenFile, "watch-token-file", envOr("D2BOT_WATCH_TOKEN_FILE", ""), "with --watch: file holding the watch token; created with a random token (mode 0600) when missing, and the viewer URL is printed once (env D2BOT_WATCH_TOKEN_FILE)")
	fs.BoolVar(&l.video, "watch-video", envBool("D2BOT_WATCH_VIDEO"), "with --watch: live video of this instance's private game display on the viewer (/stream, /shot.jpg), captured only while someone watches (env D2BOT_WATCH_VIDEO; default off)")
	fs.IntVar(&l.videoFPS, "watch-video-fps", envInt("D2BOT_WATCH_VIDEO_FPS", watch.DefaultVideoFPS), "with --watch-video: frames per second of the shared capture (1 to 15)")
	fs.IntVar(&l.videoQuality, "watch-video-quality", envInt("D2BOT_WATCH_VIDEO_QUALITY", 5), "with --watch-video: JPEG quality as ffmpeg -q:v, 2 (best, big) to 31 (small)")
}

func (l ladderOptions) validate(sim bool) error {
	if !l.on {
		switch {
		case l.watch:
			return errors.New("--watch needs --ladder: the screen shows the ladder's thoughts")
		case l.profile != "":
			return errors.New("--ladder-profile needs --ladder")
		case l.watchToken != "" || l.watchTokenFile != "":
			return errors.New("--watch-token and --watch-token-file need --ladder and --watch")
		case l.video:
			return errors.New("--watch-video needs --ladder and --watch")
		}
		return nil
	}
	if sim {
		return errors.New("--ladder needs the real bot runner: use --attach or --managed, not --sim")
	}
	if l.watch {
		if l.watchToken != "" && l.watchTokenFile != "" {
			return errors.New("--watch-token and --watch-token-file are alternatives: give one")
		}
		if l.watchToken == "" && l.watchTokenFile == "" && !(watch.Loopback(l.viewAddr) && watch.Loopback(l.answerAddr)) {
			return errors.New("--watch on a non-loopback address needs --watch-token or --watch-token-file (the bot API --token is not used for the screen)")
		}
	} else if l.watchToken != "" || l.watchTokenFile != "" {
		return errors.New("--watch-token and --watch-token-file need --watch")
	}
	if l.profile == "" && !l.watch {
		return errors.New("--ladder needs someone to answer: add --watch (a person) or --ladder-profile (a scripted player)")
	}
	if l.video {
		if !l.watch {
			return errors.New("--watch-video needs --watch")
		}
		if l.videoFPS < 1 || l.videoFPS > watch.MaxVideoFPS {
			return fmt.Errorf("--watch-video-fps must be 1 to %d", watch.MaxVideoFPS)
		}
		if l.videoQuality < 2 || l.videoQuality > 31 {
			return errors.New("--watch-video-quality must be 2 to 31")
		}
		if _, err := l.newVideo(); err != nil { // refuse at start, not when the first viewer comes
			return err
		}
	}
	if l.profile != "" {
		if _, _, err := l.source(); err != nil {
			return err
		}
	}
	return nil
}

// source parses -ladder-profile into the decide profile name and seed.
func (l ladderOptions) source() (name string, seed int64, err error) {
	name = l.profile
	if rest, ok := strings.CutPrefix(name, "holdout:"); ok {
		if seed, err = strconv.ParseInt(rest, 10, 64); err != nil {
			return "", 0, fmt.Errorf("--ladder-profile holdout:SEED needs a whole number, got %q", rest)
		}
		name = "holdout"
	}
	if _, err := decide.NewProfile(name, seed); err != nil {
		return "", 0, fmt.Errorf("--ladder-profile: %w", err)
	}
	return name, seed, nil
}

// storeProfile is the store's profile (file name) for this run: the scripted player or "human".
func (l ladderOptions) storeProfile() string {
	if l.profile == "" {
		return "human"
	}
	return strings.ReplaceAll(l.profile, ":", "-")
}

// ladderRig is what startLadder built; nil fields when -watch is off.
type ladderRig struct {
	Ladder *ladder.Ladder
	Hub    *watch.Hub
	Served watch.Served
}

// enableFunc is bot.(*Runner).EnableLadderWith (a seam for tests).
type enableFunc func(stateDir, profile string, wire bot.LadderWire) (*ladder.Ladder, error)

// startLadder installs the ladder on the runner (call after the goal manager exists and before the runner starts),
// attaches the watch Hub as the thought sink and answer source when -watch is on, and feeds the Hub from the goal
// manager's events and the runner's status. here names the area the hero stands in, for "fight here"; guard says why the
// hero's own guards close a game area now (the "too hard" mark of the orders map; nil: none).
func startLadder(ctx context.Context, l ladderOptions, apiToken, stateDir string, enable enableFunc, m *goals.Manager, status func() goals.Status, here func() string, guard func(areaID int) string) (*ladderRig, error) {
	if !l.on {
		return nil, nil
	}
	watchTok := ""
	if l.watch {
		var err error
		if watchTok, err = l.resolveWatchToken(apiToken); err != nil {
			return nil, err
		}
	}
	rig := &ladderRig{}
	var orders *watch.GoalsOrders
	wire := func(st *decide.Store, cat *decide.Catalogue, clk decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
		var src ladder.AnswerSource
		if l.profile != "" {
			name, seed, err := l.source()
			if err != nil {
				return nil, nil, err
			}
			if src, err = ladder.NewProfileSource(name, seed); err != nil {
				return nil, nil, err
			}
		}
		if !l.watch {
			return nil, src, nil
		}
		orders = &watch.GoalsOrders{Q: m, Here: here, Rules: func() []string { return liveRuleKeys(st) }}
		hub, err := watch.New(watch.Config{Store: st, Cat: cat, Clock: clk, Areas: watch.OrderableAreas(), Orders: orders, AreaGuard: guard, WaitForSink: true})
		if err != nil {
			return nil, nil, err
		}
		orders.Hub = hub
		rig.Hub = hub
		if src == nil {
			src = hub // a person answers: the Hub never answers by itself
		}
		return hub, src, nil
	}
	lad, err := enable(stateDir, l.storeProfile(), wire)
	if err != nil {
		return nil, err
	}
	if lad == nil {
		return nil, errors.New("--ladder: the ladder.enabled knob is off")
	}
	rig.Ladder = lad
	if rig.Hub == nil {
		return rig, nil
	}
	vid, err := l.newVideo()
	if err != nil {
		ladder.Install(nil)
		return nil, err
	}
	served, err := rig.Hub.Serve(ctx, watch.ServeOptions{ViewAddr: l.viewAddr, AnswerAddr: l.answerAddr, Token: watchTok, Origins: l.origins, Video: vid})
	if err != nil {
		ladder.Install(nil)
		return nil, err
	}
	rig.Served = served
	go rig.Hub.RunTicker(ctx, time.Second)
	go feedHub(ctx, rig.Hub, orders, m, status)
	return rig, nil
}

// watchTokenOut receives the one-time viewer URL when a token file is created (the only place a token is ever printed).
var watchTokenOut io.Writer = os.Stderr

// minWatchToken is the shortest token accepted from a flag, env or file; generated ones are 32 characters.
const minWatchToken = 16

// resolveWatchToken returns the access token of the watch screen: the flag/env value, the content of the token file
// (created 0600 with a random token on first start), or "" (loopback only, as before). It never equals the bot API token.
func (l ladderOptions) resolveWatchToken(apiToken string) (string, error) {
	tok := l.watchToken
	fresh := false
	if l.watchTokenFile != "" {
		var err error
		if tok, fresh, err = loadOrCreateToken(l.watchTokenFile); err != nil {
			return "", err
		}
		if tok == "" {
			return "", fmt.Errorf("watch token file %s is empty: delete it to have a new token made", l.watchTokenFile)
		}
	}
	if tok == "" {
		return "", nil
	}
	if len(tok) < minWatchToken {
		return "", fmt.Errorf("the watch token must be at least %d characters", minWatchToken)
	}
	if apiToken != "" && tok == apiToken {
		return "", errors.New("the watch token must differ from the bot API token: the screen is shared with testers, the API is not")
	}
	if fresh {
		base := "http://" + l.viewAddr
		if o, _, _ := strings.Cut(l.origins, ","); o != "" {
			base = strings.TrimRight(o, "/") // the page origin is the viewer's public URL
		}
		fmt.Fprintf(watchTokenOut, "watch: token created in %s; open %s/watch?t=%s\n", l.watchTokenFile, base, tok)
	}
	return tok, nil
}

// loadOrCreateToken reads path, or creates it with 24 random bytes (base64url, 32 characters) and mode 0600.
func loadOrCreateToken(path string) (tok string, created bool, err error) {
	b, err := os.ReadFile(path)
	if err == nil {
		if st, e := os.Stat(path); e == nil && st.Mode().Perm()&0o077 != 0 {
			if e := os.Chmod(path, 0o600); e != nil {
				return "", false, fmt.Errorf("watch token file %s is readable by others and cannot be fixed: %w", path, e)
			}
		}
		return strings.TrimSpace(string(b)), false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", false, err
	}
	tok = base64.RawURLEncoding.EncodeToString(raw)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", false, err
	}
	if _, err := f.WriteString(tok + "\n"); err != nil {
		f.Close()
		return "", false, err
	}
	return tok, true, f.Close()
}

func liveRuleKeys(st *decide.Store) []string {
	var keys []string
	for _, r := range st.Rules() {
		keys = append(keys, r.Key)
	}
	return keys
}

// feedHub forwards the goal manager's events (orders, deaths, kills, area changes) and a status poll to the Hub: the
// uptime source (docs/research/watch-screen-design.md, 3.5).
func feedHub(ctx context.Context, hub *watch.Hub, o *watch.GoalsOrders, m *goals.Manager, status func() goals.Status) {
	ch, cancel := m.Subscribe()
	defer cancel()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			o.OnGoalEvent(ev)
		case <-tick.C:
			if status != nil {
				o.OnStatus(status())
			}
		}
	}
}

// ladderEnv switches the knob when the flag asks for it; the knob stays what the config file says otherwise.
func ladderEnv(l ladderOptions, t *env.Tuning) {
	if l.on {
		t.Ladder.Enabled = 1
	}
}

// ladderPost is ladderEnv as a Tuner hook: a saved tuning file or a viewer edit rebuilds the whole tuning and must not
// turn the flag off. Nil without the flag, so the config file and tuning.json stay authoritative.
func ladderPost(l ladderOptions) func(*env.Tuning) {
	if !l.on {
		return nil
	}
	return func(t *env.Tuning) { ladderEnv(l, t) }
}

// hereFunc names the area of the runner's last snapshot ("" when unknown).
func hereFunc(r *bot.Runner) func() string {
	return func() string {
		s := r.LastSnapshot()
		if s == nil || s.Player == nil {
			return ""
		}
		if c := r.Catalog(); c != nil {
			if n := c.AreaName(s.Player.Area); n != "" {
				return n
			}
		}
		return status.AreaName(s.Player.Area)
	}
}

// ladderDir is the decision store's directory, made only when -ladder is on. The shared store is <state>/ladder and
// carries learning across sessions; a named run (--ladder-store, else D2BOT_RUN) gets <state>/ladder/runs/NAME, so
// two drills on one instance do not see each other's thoughts and rules (t0drill Result 2, defect 7).
func ladderDir(stateDir string, l ladderOptions) (string, error) {
	shared := filepath.Join(stateDir, "ladder")
	name := l.store
	if !l.on {
		return shared, nil
	}
	if name == "" {
		name = os.Getenv("D2BOT_RUN")
	}
	if name == "" || name == "shared" {
		return shared, nil
	}
	if strings.ContainsAny(name, `/\:`) || strings.Contains(name, "..") || name[0] == '.' {
		return "", fmt.Errorf("--ladder-store / D2BOT_RUN %q is not a plain name", name)
	}
	return filepath.Join(shared, "runs", name), nil
}

// logLadder prints the one startup line.
func logLadder(log *slog.Logger, l ladderOptions, rig *ladderRig, dir string) {
	if rig == nil {
		return
	}
	args := []any{"profile", l.storeProfile(), "dir", dir}
	if rig.Hub != nil {
		args = append(args, "view", rig.Served.View.String(), "answer", rig.Served.Answer.String())
	}
	log.Info("ladder on", args...)
}

// newVideo builds the game video for -watch-video: the capture of the instance's own private display (100+N, the rule of
// launch.XvfbDisplay), never the desktop. nil when the flag is off.
func (l ladderOptions) newVideo() (*watch.Video, error) {
	if !l.video {
		return nil, nil
	}
	in := launch.Instance{ID: l.instance}
	if !launch.ValidID(l.instance) || in.Num() <= 0 {
		return nil, fmt.Errorf("--watch-video: instance %q has no private display (only i1..i9; i0 runs on the desktop)", l.instance)
	}
	capture, err := watch.ExecCapture(watch.CaptureOptions{Display: launch.XvfbDisplay{Num: 100 + in.Num()}.Name(), FPS: l.videoFPS, Quality: l.videoQuality})
	if err != nil {
		return nil, err
	}
	return watch.NewVideo(watch.VideoOptions{Capture: capture})
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return n
	}
	return def
}
