package sentinel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/mem"
)

// Actor performs a repair on one instance. The real implementation is LiveActor; tests use fakes.
type Actor interface {
	Do(ctx context.Context, in InstanceConf, a Action, d Decision) Result
}

// LiveActor acts through the instance's own d2input helper, the instance's own X display (dialog keys only),
// launch.Flow (OOG), launch.Life (process lifecycle) and signals. It never touches another instance.
type LiveActor struct {
	Cfg Config
	Log func(string)
	// Life runs the process lifecycle (start/stop/helper). nil = launch.NewLife (real system). The
	// instance lock is held by the caller, so Life is never asked to lock.
	Life *launch.Life
}

type typer struct {
	*input.OOG
	c *input.Client
}

func (t typer) TypeChar(r rune) error { return t.c.TypeChar(r) }

var errLocked = errors.New("instance lock held by another run")

// acquire takes the instance flock files (launch.LockPaths) for restarts, waiting up to Cfg.LockWait.
func (a *LiveActor) acquire(id string) (func(), error) {
	rel, err := launch.LockInstance(id, a.Cfg.LockWait.D())
	if errors.Is(err, launch.ErrLocked) {
		return nil, errLocked
	}
	return rel, err
}

func (a *LiveActor) logf(f string, args ...any) {
	if a.Log != nil {
		a.Log(fmt.Sprintf(f, args...))
	}
}

type env struct {
	in     InstanceConf
	client *input.Client
	oog    *input.OOG
}

func (a *LiveActor) env(in InstanceConf) *env {
	c := input.NewClientFor(in.Helper(), 0)
	return &env{in: in, client: c, oog: input.NewOOG(c)}
}

func (e *env) reader() (*d2.Reader, *mem.Proc, error) {
	p, err := mem.OpenInstance(e.in.ID)
	if err != nil {
		return nil, nil, err
	}
	return d2.New(p), p, nil
}

func (a *LiveActor) Do(ctx context.Context, in InstanceConf, act Action, d Decision) (res Result) {
	t0 := time.Now()
	defer func() { res.Took = time.Since(t0) }()
	e := a.env(in)
	defer e.client.Close()
	switch act {
	case ActEsc, ActEsc3:
		return a.esc(ctx, e, act == ActEsc3)
	case ActRespawn:
		return a.respawn(ctx, e)
	case ActDismiss:
		return a.dismiss(in)
	case ActCont:
		return signalGame(in.ID, syscall.SIGCONT)
	case ActRestartHelp:
		return a.withLock(in.ID, func() Result { return a.lifeStep(ctx, "helper", in.ID) })
	case ActRestartGame:
		return a.withLock(in.ID, func() Result { return a.restartGame(ctx, e) })
	case ActOOGStart:
		return a.withLock(in.ID, func() Result { return a.oogStart(ctx, e) })
	case ActLeaveRejoin:
		return a.withLock(in.ID, func() Result { return a.leaveRejoin(ctx, e) })
	case ActStopInstance:
		return a.withLock(in.ID, func() Result { return a.stopInstance(ctx, e) })
	case ActRestartBot:
		return a.restartBot(ctx, in)
	case ActBotReset:
		return a.botReset(in)
	case ActAlert:
		return Result{OK: true, Detail: "alert only (no automatic action left)"}
	}
	return Result{Detail: "unknown action " + string(act)}
}

func (a *LiveActor) withLock(id string, fn func() Result) Result {
	rel, err := a.acquire(id)
	if err != nil {
		return Result{Detail: err.Error()}
	}
	defer rel()
	return fn()
}

func uiClear(r *d2.Reader) (esc, panel bool, err error) {
	ui, err := r.UI()
	if err != nil {
		return false, false, err
	}
	return ui.Open(d2.UIEscMenu), len(openPanels(ui)) > 0, nil
}

// esc presses Esc until neither the Esc menu nor a panel is open. Esc with nothing open OPENS the menu,
// so it is only pressed while something is open (docs/oog.md). n3 forces up to three presses.
func (a *LiveActor) esc(ctx context.Context, e *env, n3 bool) Result {
	r, p, err := e.reader()
	if err != nil {
		return Result{Detail: "memory: " + err.Error()}
	}
	defer p.Close()
	presses := 0
	if n3 {
		// Nothing may be observable (an overlay without a flag): Esc opens the menu, the second closes it.
		if esc, panel, err := uiClear(r); err == nil && !esc && !panel {
			for k := 0; k < 2; k++ {
				if err := e.oog.Key("esc"); err != nil {
					return Result{Detail: "key: " + err.Error()}
				}
				presses++
				sleepCtx(ctx, 700*time.Millisecond)
			}
		}
	}
	for i := 0; i < 4; i++ {
		esc, panel, err := uiClear(r)
		if err != nil {
			return Result{Detail: "ui read: " + err.Error()}
		}
		if !esc && !panel {
			break
		}
		if err := e.oog.Key("esc"); err != nil {
			return Result{Detail: "key: " + err.Error()}
		}
		presses++
		sleepCtx(ctx, 700*time.Millisecond)
	}
	esc, panel, err := uiClear(r)
	if err != nil {
		return Result{OK: presses > 0, Detail: "ui read: " + err.Error()}
	}
	return Result{OK: !esc && !panel, Verified: true, Detail: fmt.Sprintf("%d Esc, escMenu=%v panel=%v", presses, esc, panel)}
}

// respawn presses Esc on the death overlay and verifies from memory: alive, life > 0.
func (a *LiveActor) respawn(ctx context.Context, e *env) Result {
	r, p, err := e.reader()
	if err != nil {
		return Result{Detail: "memory: " + err.Error()}
	}
	defer p.Close()
	// Esc on a living character OPENS the Esc menu. A life of 0 without a death mode (seen when life is
	// written to 0 by the cheat tool, VERIFIED LIVE: mode stays 5, no overlay) is not a death overlay.
	if pl, err := r.Player(); err == nil && pl.Mode != 0 && pl.Mode != 17 {
		return Result{Detail: fmt.Sprintf("refused: life %d but mode %d is not a death mode, no overlay to dismiss", pl.Life, pl.Mode)}
	}
	for press := 1; press <= 3; press++ {
		if err := e.oog.Key("esc"); err != nil {
			return Result{Detail: "key: " + err.Error()}
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && ctx.Err() == nil {
			time.Sleep(400 * time.Millisecond)
			if r.State() != d2.StateInGame {
				continue
			}
			pl, err := r.Player()
			if err != nil || pl.Life <= 0 || pl.Mode == 0 || pl.Mode == 17 {
				continue
			}
			town := pl.InTown()
			return Result{OK: true, Verified: true, Detail: fmt.Sprintf("alive after %d Esc: life %d/%d area %d town=%v", press, pl.Life, pl.MaxLife, pl.Area, town)}
		}
	}
	return Result{Detail: "still dead after 3 Esc"}
}

func (a *LiveActor) flow(e *env, p *mem.Proc) *launch.Flow {
	return &launch.Flow{Mem: p, In: typer{e.oog, e.client}, Log: func(s string) { a.logf("%s flow: %s", e.in.ID, s) }}
}

func (a *LiveActor) charSpec(in InstanceConf) launch.CharSpec {
	return launch.CharSpec{Name: in.Character, Class: in.Class}
}

func (a *LiveActor) oogStart(ctx context.Context, e *env) Result {
	if e.in.Character == "" {
		return Result{Detail: "no character configured"}
	}
	p, err := mem.OpenInstance(e.in.ID)
	if err != nil {
		return Result{Detail: "memory: " + err.Error()}
	}
	defer p.Close()
	c, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := a.flow(e, p).Start(c, a.charSpec(e.in)); err != nil {
		return Result{Detail: "flow.Start: " + err.Error()}
	}
	return a.verifyInGame(ctx, e)
}

func (a *LiveActor) verifyInGame(ctx context.Context, e *env) Result {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		if p, err := mem.OpenInstance(e.in.ID); err == nil {
			r := d2.New(p)
			if r.State() == d2.StateInGame {
				if pl, err := r.Player(); err == nil && pl.Life > 0 {
					p.Close()
					return Result{OK: true, Verified: true, Detail: fmt.Sprintf("in game as %s area %d life %d/%d", pl.Name, pl.Area, pl.Life, pl.MaxLife)}
				}
			}
			p.Close()
		}
		time.Sleep(time.Second)
	}
	return Result{Detail: "flow finished but not in game from memory"}
}

func (a *LiveActor) leaveRejoin(ctx context.Context, e *env) Result {
	p, err := mem.OpenInstance(e.in.ID)
	if err != nil {
		return Result{Detail: "memory: " + err.Error()}
	}
	fl := a.flow(e, p)
	c, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if err := fl.LeaveGame(c, 40*time.Second); err != nil {
		p.Close()
		return Result{Detail: "leave: " + err.Error()}
	}
	p.Close()
	return a.oogStart(ctx, e)
}

func (a *LiveActor) stopInstance(ctx context.Context, e *env) Result {
	if e.in.ID == "i0" {
		return Result{Detail: "refused: never stop i0"}
	}
	if p, err := mem.OpenInstance(e.in.ID); err == nil {
		if d2.New(p).State() == d2.StateInGame {
			c, cancel := context.WithTimeout(ctx, 60*time.Second)
			if err := a.flow(e, p).LeaveGame(c, 40*time.Second); err != nil {
				a.logf("%s idle stop: leave failed (%v), stopping anyway", e.in.ID, err)
			}
			cancel()
		}
		p.Close()
	}
	return a.lifeStep(ctx, "stop", e.in.ID)
}

func signalGame(id string, sig syscall.Signal) Result {
	pids := mem.FindGamesFor(id)
	if len(pids) == 0 {
		return Result{Detail: "no Game.exe"}
	}
	for _, p := range pids {
		if err := syscall.Kill(p, sig); err != nil {
			return Result{Detail: err.Error()}
		}
	}
	return Result{OK: true, Detail: fmt.Sprintf("signal %v to %v", sig, pids)}
}

// lifeStep runs one launch.Life step (stop, start, helper) for id. Cfg.GameSh is deprecated and ignored:
// the sentinel no longer shells out to game.sh (findings F1).
func (a *LiveActor) lifeStep(ctx context.Context, cmd, id string) Result {
	c, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var buf bytes.Buffer
	l := a.Life
	if l == nil {
		l = launch.NewLife(nil)
	}
	cp := *l // own output buffer per call; the shared Life stays untouched
	cp.Out = &buf
	var err error
	switch cmd {
	case "stop":
		err = cp.Stop(c, id)
	case "start":
		err = cp.Start(c, id)
	case "helper":
		err = cp.EnsureHelper(c, id)
	default:
		err = fmt.Errorf("unknown step %q", cmd)
	}
	s := strings.TrimSpace(buf.String())
	if len(s) > 300 {
		s = s[len(s)-300:]
	}
	if err != nil {
		return Result{Detail: fmt.Sprintf("launch %s %s: %v: %s", cmd, id, err, s)}
	}
	return Result{OK: true, Detail: fmt.Sprintf("launch %s %s: %s", cmd, id, strings.ReplaceAll(s, "\n", " | "))}
}

// restartGame kills the instance (SIGCONT first so a stopped process can die, SIGKILL, crash reporters),
// cleans the prefix with launch stop, starts game and helper, mutes, and logs the character back in.
func (a *LiveActor) restartGame(ctx context.Context, e *env) Result {
	id := e.in.ID
	if id == "i0" {
		return Result{Detail: "refused: never restart i0 automatically"}
	}
	signalGame(id, syscall.SIGCONT)
	signalGame(id, syscall.SIGKILL)
	for _, pr := range crashProcs(id) {
		var pid int
		fmt.Sscanf(pr, "BlizzardError.exe pid %d", &pid)
		if pid > 0 {
			syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	time.Sleep(time.Second)
	var steps []string
	for _, c := range []string{"stop", "start", "helper"} {
		r := a.lifeStep(ctx, c, id)
		steps = append(steps, c)
		if !r.OK && c != "stop" {
			r.Detail = "step " + strings.Join(steps, ">") + ": " + r.Detail
			return r
		}
	}
	if a.Cfg.MuteSh != "" {
		_ = exec.CommandContext(ctx, "bash", a.Cfg.MuteSh).Run()
	}
	// Wait for the window and memory.
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		if _, up, _, _ := pingWindow(e.in); up {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	r := a.oogStart(ctx, e)
	r.Detail = "restarted; " + r.Detail
	if a.Cfg.MuteSh != "" {
		_ = exec.CommandContext(ctx, "bash", a.Cfg.MuteSh).Run()
	}
	return r
}

func pingWindow(in InstanceConf) (alive, up bool, reply string, err error) {
	reply, err = helperCmd(in.Helper(), "info", time.Second)
	return err == nil, strings.HasPrefix(reply, "ok"), reply, err
}

// dismiss sends Return to the dialog window with an X event on the instance's OWN display (never i0's desktop).
func (a *LiveActor) dismiss(in InstanceConf) Result {
	if in.ID == "i0" {
		return Result{Detail: "refused: i0 is on the user's desktop"}
	}
	ob := &Observer{Inst: in}
	out, err := ob.xcmd(3*time.Second, "xdotool", "search", "--name", ".")
	if err != nil {
		return Result{Detail: "xdotool: " + err.Error()}
	}
	n := 0
	for _, id := range strings.Fields(string(out)) {
		nm, _ := ob.xcmd(2*time.Second, "xdotool", "getwindowname", id)
		name := strings.TrimSpace(string(nm))
		if ignoredWin[name] || strings.HasPrefix(strings.ToLower(name), "d2input") || !dialogRe.MatchString(name) {
			continue
		}
		ob.xcmd(2*time.Second, "xdotool", "key", "--window", id, "Return")
		n++
	}
	return Result{OK: n > 0, Detail: fmt.Sprintf("sent Return to %d dialog window(s) on %s", n, in.DisplayName())}
}

func (a *LiveActor) restartBot(ctx context.Context, in InstanceConf) Result {
	if len(in.BotRestartCmd) == 0 {
		return Result{Detail: "no botRestartCmd configured: alert only"}
	}
	c, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(c, in.BotRestartCmd[0], in.BotRestartCmd[1:]...).CombinedOutput()
	if err != nil {
		return Result{Detail: fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out)))}
	}
	return Result{OK: true, Detail: "ran " + strings.Join(in.BotRestartCmd, " ")}
}

func (a *LiveActor) botReset(in InstanceConf) Result {
	if in.BotResetURL == "" {
		return Result{Detail: "no botResetUrl configured: step skipped"}
	}
	c := &http.Client{Timeout: 5 * time.Second}
	resp, err := c.Post(in.BotResetURL, "application/json", strings.NewReader("{}"))
	if err != nil {
		return Result{Detail: err.Error()}
	}
	resp.Body.Close()
	return Result{OK: resp.StatusCode < 300, Detail: resp.Status}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
