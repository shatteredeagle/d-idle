package fleet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"d2lod-bot/internal/paths"
)

// BotRunner runs the d2bot process of one slot. Run blocks until ctx ends (the slot is removed or its game
// died), supervising the process with its own restart backoff and reporting every change through report.
// It must have stopped the process when it returns.
type BotRunner interface {
	Run(ctx context.Context, a Alloc, s Spec, report func(BotStatus))
}

// Bot process states (BotStatus.State).
const (
	BotStarting   = "starting"
	BotRunning    = "running"
	BotRestarting = "restarting" // exited, waiting out the backoff
	BotFailed     = "failed"     // gave up after MaxRestarts; the slot's game keeps running
	BotStopped    = "stopped"    // ctx ended
)

// BotStatus is what a slot shows about its d2bot process (View.Bot).
type BotStatus struct {
	State     string    `json:"state"`
	Pid       int       `json:"pid,omitempty"`
	Restarts  int       `json:"restarts"`
	LastError string    `json:"lastError,omitempty"`
	Since     time.Time `json:"since"`
	Addr      string    `json:"addr,omitempty"`    // d2bot's own API, read from its "listening" log line
	GamePid   int       `json:"gamePid,omitempty"` // bot-owned game: from d2bot's "managed game attached" log line
	Log       string    `json:"log,omitempty"`
}

// BotCmd is one process to start.
type BotCmd struct {
	Path string
	Args []string
	Log  string // appended to; stdout and stderr
	// Tap, when set, also receives every output line (the launcher reads the listen address from it).
	Tap func(line string)
	Env []string // added to the inherited environment
}

// BotProc is a started process.
type BotProc interface {
	Pid() int
	Wait() error
	Signal(sig os.Signal) error
	Kill() error
}

// BotExec starts processes; OSBotExec is the real one, tests fake it.
type BotExec interface {
	Start(ctx context.Context, c BotCmd) (BotProc, error)
}

// DefaultStopGrace covers d2bot's --stop-game shutdown (runner and API waits, Save and Exit up to 30 s, game
// shutdown 10 s: about 50 s, cmd/d2bot/live.go) plus margin. A SIGKILL in the middle loses the .d2s.
const DefaultStopGrace = 70 * time.Second

// ContainerGrace is how long d2container waits for d2fleet after SIGTERM before it kills; ComposeStopGrace is the
// stop_grace_period of compose.example.yml. Chain: ComposeStopGrace > ContainerGrace > DefaultStopGrace > bot.ShutdownWorstCase.
const (
	ContainerGrace   = 80 * time.Second
	ComposeStopGrace = 90 * time.Second
)

// ProcBots is the BotRunner over d2bot --managed (docs/container.md). By default d2bot launches Game.exe itself
// as its child and subreaper (the model whose memory reads work under Yama ptrace_scope=1 without caps) and
// is passed --stop-game; the fleet's Runtime only prepares prefix, Xvfb and helper (GameOwner, Preparer).
// With Attach the fleet starts the game and d2bot attaches to it (needs CAP_SYS_PTRACE or ptrace_scope 0).
//
// UNVERIFIED live: nothing here has started a real d2bot (offline fakes only).
type ProcBots struct {
	Attach      bool    // old model: the fleet's Runtime owns the game, d2bot attaches and never stops it
	Exec        BotExec // default OSBotExec
	Binary      string  // d2bot path (default: next to this executable, else "d2bot" on PATH)
	Layout      paths.Layout
	Backoff     Backoff       // default 2 s .. 2 min, stable after 5 min
	MaxRestarts int           // consecutive restarts before BotFailed (0 = unlimited)
	StopGrace   time.Duration // SIGTERM, then wait this long before SIGKILL (default DefaultStopGrace)
	// CleanGame runs after d2bot died on its own (not on a stop) in the bot-owned model: its game may be left
	// behind, and a new d2bot that is not its ancestor could not read it (d2fleet sets Runtime.Stop).
	CleanGame func(context.Context, Alloc)
	Clock     func() time.Time
	Sleep     func(context.Context, time.Duration) error
}

// OwnsGame implements GameOwner.
func (p *ProcBots) OwnsGame() bool { return !p.Attach }

func (p *ProcBots) defaults() {
	if p.Exec == nil {
		p.Exec = OSBotExec{}
	}
	if p.Binary == "" {
		p.Binary = findBinary("d2bot")
	}
	if p.Backoff.Min == 0 {
		p.Backoff = Backoff{Min: 2 * time.Second, Max: 2 * time.Minute, StableAfter: 5 * time.Minute}
	}
	if p.StopGrace == 0 {
		p.StopGrace = DefaultStopGrace
	}
	if p.Clock == nil {
		p.Clock = time.Now
	}
	if p.Sleep == nil {
		p.Sleep = sleepCtx
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func findBinary(name string) string {
	if exe, err := os.Executable(); err == nil {
		if p := filepath.Join(filepath.Dir(exe), name); fileExists(p) {
			return p
		}
	}
	return name
}

func fileExists(p string) bool { st, err := os.Stat(p); return err == nil && !st.IsDir() }

// Args is the d2bot command line of slot a. first says the goals of the spec have not been given to this
// instance yet: d2bot appends --goals on every start, so a restart passes only --autostart and the manager's
// saved goals (instances/<id>/goals.json) carry on.
func (p *ProcBots) args(a Alloc, s Spec, cfgFile string, first bool) ([]string, error) {
	extra, err := s.BotArgs()
	if err != nil {
		return nil, err
	}
	args := []string{"--managed", "--instance", a.ID, "--char", s.Character,
		"--addr", "127.0.0.1:0", // ephemeral: no fixed port beyond the helper scheme (47012+N)
		"--data", p.Layout.Root, "--config", cfgFile}
	if !p.Attach {
		args = append(args, "--stop-game")
	}
	if len(extra) == 0 {
		return args, nil
	}
	if first {
		return append(args, extra...), nil
	}
	return append(args, "--autostart"), nil
}

// WriteConfig writes the per-instance d2bot config: the host config (Layout.Config, when it exists) with
// the slot's handicap profile set. It returns the file path.
func (p *ProcBots) WriteConfig(a Alloc, s Spec) (string, error) {
	in, err := p.Layout.Instance(a.ID)
	if err != nil {
		return "", err
	}
	top := map[string]json.RawMessage{}
	if b, err := os.ReadFile(p.Layout.Config()); err == nil {
		if err := json.Unmarshal(b, &top); err != nil {
			return "", fmt.Errorf("%s: %w", p.Layout.Config(), err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	// bot.LoadConfig reads the "bot" section when there is one, else the whole file.
	sect := top
	if raw, ok := top["bot"]; ok {
		sect = map[string]json.RawMessage{}
		if err := json.Unmarshal(raw, &sect); err != nil {
			return "", fmt.Errorf("config bot section: %w", err)
		}
	}
	if s.Profile == "" {
		delete(sect, "profile")
	} else {
		sect["profile"], _ = json.Marshal(s.Profile)
	}
	if _, ok := top["bot"]; ok {
		top["bot"], _ = json.Marshal(sect)
	}
	b, _ := json.MarshalIndent(top, "", "  ")
	path := filepath.Join(in.Dir(), "config.json")
	if err := os.MkdirAll(in.Dir(), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return "", err
	}
	return path, os.Rename(tmp, path)
}

// seeded reports whether the goals of s were already handed to the instance, and marks them handed. The mark
// is written only when d2bot reports they are stored (goalsEnqueued), not when the process starts.
func (p *ProcBots) seeded(a Alloc, s Spec) (already bool, mark func()) {
	in, err := p.Layout.Instance(a.ID)
	if err != nil {
		return false, func() {}
	}
	f := filepath.Join(in.Dir(), "goals-seeded")
	want := s.Character + "|" + s.Goals
	b, _ := os.ReadFile(f)
	// A lost, unreadable or empty goals store must be reseeded even if the marker says it was (AddSeeded is idempotent).
	return string(b) == want && goalsStored(in.Goals()), func() { _ = os.WriteFile(f, []byte(want), 0o644) }
}

// goalsStored reports whether the goals file at path exists and holds at least one goal.
func goalsStored(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var d struct {
		Goals []json.RawMessage `json:"goals"`
	}
	return json.Unmarshal(b, &d) == nil && len(d.Goals) > 0
}

// botState is the status of one runner, shared with the output tap goroutine.
type botState struct {
	mu     sync.Mutex
	st     BotStatus
	clock  func() time.Time
	report func(BotStatus)
}

// update applies fn and reports the result; state "" keeps the current one.
func (b *botState) update(state string, fn func(*BotStatus)) {
	b.mu.Lock()
	if state != "" {
		b.st.State, b.st.Since = state, b.clock()
	}
	if fn != nil {
		fn(&b.st)
	}
	cp := b.st
	b.mu.Unlock()
	b.report(cp)
}

// Run implements BotRunner.
func (p *ProcBots) Run(ctx context.Context, a Alloc, s Spec, report func(BotStatus)) {
	p.defaults()
	bs := &botState{clock: p.Clock, report: report}
	in, ierr := p.Layout.Instance(a.ID)
	if ierr != nil {
		bs.update(BotFailed, func(st *BotStatus) { st.LastError = ierr.Error() })
		return
	}
	logPath := filepath.Join(in.Logs(), "d2bot.log")
	bs.update(BotStarting, func(st *BotStatus) { st.Log = logPath })
	rotate(logPath)
	stopped := func() {
		bs.update(BotStopped, func(st *BotStatus) { st.Pid, st.Addr, st.GamePid = 0, "", 0 })
	}
	bo := p.Backoff.Min
	restarts := 0
	for {
		bs.update(BotStarting, nil)
		ran, err := p.once(ctx, a, s, logPath, bs)
		if ctx.Err() != nil {
			stopped()
			return
		}
		if exitedFatal(err) {
			// d2bot's deliberate fatal (a corrupt character, a halted bot): the same condition cmd/d2bot supervise treats as
			// final. Restarting would just repeat it MaxRestarts times.
			bs.update(BotFailed, func(st *BotStatus) {
				st.Pid, st.Addr, st.GamePid, st.LastError = 0, "", 0, "d2bot ended with a deliberate fatal (exit 3): "+err.Error()
			})
			return
		}
		if ran >= p.Backoff.StableAfter {
			bo, restarts = p.Backoff.Min, 0
		}
		restarts++
		over := p.MaxRestarts > 0 && restarts > p.MaxRestarts
		state := BotRestarting
		if over {
			state, err = BotFailed, fmt.Errorf("%d restarts: %w", restarts-1, err)
		}
		bs.update(state, func(st *BotStatus) {
			st.Pid, st.Addr, st.GamePid, st.Restarts, st.LastError = 0, "", 0, restarts, err.Error()
		})
		if over {
			return
		}
		if p.CleanGame != nil && !p.Attach {
			p.CleanGame(ctx, a)
		}
		if p.Sleep(ctx, bo) != nil {
			stopped()
			return
		}
		bo = min(bo*2, p.Backoff.Max)
	}
}

// exitFatal is cmd/d2bot's exitFatal (a deliberate fatal that must not be restarted). Kept in step by hand: fleet does
// not import cmd.
const exitFatal = 3

func exitedFatal(err error) bool {
	var ec interface{ ExitCode() int }
	return errors.As(err, &ec) && ec.ExitCode() == exitFatal
}

// once starts d2bot, waits for it and returns how long it ran and why it ended.
func (p *ProcBots) once(ctx context.Context, a Alloc, s Spec, logPath string, bs *botState) (time.Duration, error) {
	cfg, err := p.WriteConfig(a, s)
	if err != nil {
		return 0, fmt.Errorf("config: %w", err)
	}
	already, mark := p.seeded(a, s)
	args, err := p.args(a, s, cfg, !already)
	if err != nil {
		return 0, err
	}
	tap := func(line string) {
		if addr := listenAddr(line); addr != "" {
			bs.update("", func(st *BotStatus) { st.Addr = addr })
		}
		if !already && goalsEnqueued(line) {
			mark()
		}
		if pid := gamePid(line); pid > 0 {
			bs.update("", func(st *BotStatus) { st.GamePid = pid })
		}
	}
	proc, err := p.Exec.Start(ctx, BotCmd{Path: p.Binary, Args: args, Log: logPath, Tap: tap,
		Env: []string{"D2BOT_INSTANCE=" + a.ID, "D2BOT_FLEET=1"}})
	if err != nil {
		return 0, fmt.Errorf("start d2bot: %w", err)
	}
	t0 := p.Clock()
	bs.update(BotRunning, func(st *BotStatus) { st.Pid = proc.Pid() })
	exited := make(chan error, 1)
	go func() { exited <- proc.Wait() }()
	select {
	case err := <-exited:
		if err == nil {
			err = errors.New("d2bot exited")
		}
		return p.Clock().Sub(t0), err
	case <-ctx.Done():
		_ = proc.Signal(syscall.SIGTERM) // d2bot stops gracefully on SIGTERM (cmd/d2bot signal.NotifyContext)
		select {
		case <-exited:
		case <-time.After(p.StopGrace):
			_ = proc.Kill()
			<-exited
		}
		return p.Clock().Sub(t0), ctx.Err()
	}
}

// listenAddr extracts addr from d2bot's JSON log line {"msg":"listening","addr":"127.0.0.1:41233",...}.
func listenAddr(line string) string {
	if !strings.Contains(line, `"listening"`) {
		return ""
	}
	var m struct {
		Msg  string `json:"msg"`
		Addr string `json:"addr"`
	}
	if json.Unmarshal([]byte(line), &m) != nil || m.Msg != "listening" {
		return ""
	}
	return m.Addr
}

// gamePid extracts pid from d2bot's log line {"msg":"managed game attached","pid":123,...} (cmd/d2bot/game.go).
func gamePid(line string) int {
	if !strings.Contains(line, "managed game attached") {
		return 0
	}
	var m struct {
		Msg string `json:"msg"`
		Pid int    `json:"pid"`
	}
	if json.Unmarshal([]byte(line), &m) != nil || m.Msg != "managed game attached" {
		return 0
	}
	return m.Pid
}

// goalsEnqueued matches d2bot's log line {"msg":"goals enqueued",...} (cmd/d2bot/live.go), written once
// --goals are stored in the manager (which saves them to instances/<id>/goals.json).
func goalsEnqueued(line string) bool {
	if !strings.Contains(line, "goals enqueued") {
		return false
	}
	var m struct {
		Msg string `json:"msg"`
	}
	return json.Unmarshal([]byte(line), &m) == nil && m.Msg == "goals enqueued"
}

// rotate keeps one old log when the current one is over 5 MB.
func rotate(path string) {
	if st, err := os.Stat(path); err == nil && st.Size() > 5<<20 {
		_ = os.Rename(path, path+".1")
	}
}

// rotWriter appends to a log and rotates it (one old copy) when it passes max
// bytes, so a long-running d2bot cannot grow its log without bound.
type rotWriter struct {
	mu   sync.Mutex
	path string
	max  int64
	f    *os.File
	size int64
}

func newRotWriter(path string, max int64) (*rotWriter, error) {
	w := &rotWriter{path: path, max: max}
	return w, w.open()
}

func (w *rotWriter) open() error {
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	w.f, w.size = f, st.Size()
	return nil
}

func (w *rotWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size > w.max {
		w.f.Close()
		_ = os.Rename(w.path, w.path+".1")
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	n, err := w.f.Write(b)
	w.size += int64(n)
	return n, err
}

func (w *rotWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.f.Close()
}

// OSBotExec starts real processes.
type OSBotExec struct{}

type osProc struct{ cmd *exec.Cmd }

func (p osProc) Pid() int                 { return p.cmd.Process.Pid }
func (p osProc) Signal(s os.Signal) error { return p.cmd.Process.Signal(s) }
func (p osProc) Kill() error              { return p.cmd.Process.Kill() }
func (p osProc) Wait() error              { return p.cmd.Wait() }

func (OSBotExec) Start(_ context.Context, c BotCmd) (BotProc, error) {
	if err := os.MkdirAll(filepath.Dir(c.Log), 0o755); err != nil {
		return nil, err
	}
	lf, err := newRotWriter(c.Log, 5<<20)
	if err != nil {
		return nil, err
	}
	// Not CommandContext: ProcBots.Run decides when and how the process stops.
	cmd := exec.Command(c.Path, c.Args...)
	cmd.Env = append(os.Environ(), c.Env...)
	var w io.Writer = lf
	var tap *lineTap
	if c.Tap != nil {
		tap = &lineTap{w: lf, f: c.Tap}
		w = tap
	}
	cmd.Stdout, cmd.Stderr = w, w
	if err := cmd.Start(); err != nil {
		lf.Close()
		return nil, err
	}
	return &closingProc{osProc{cmd}, lf}, nil
}

type closingProc struct {
	osProc
	f io.Closer
}

func (p *closingProc) Wait() error { err := p.osProc.Wait(); p.f.Close(); return err }

// lineTap copies to w and calls f for each complete line.
type lineTap struct {
	mu  sync.Mutex
	w   io.Writer
	f   func(string)
	buf bytes.Buffer
}

func (t *lineTap) Write(b []byte) (int, error) {
	n, err := t.w.Write(b)
	t.mu.Lock()
	t.buf.Write(b)
	var lines []string
	for {
		i := bytes.IndexByte(t.buf.Bytes(), '\n')
		if i < 0 {
			break
		}
		lines = append(lines, string(t.buf.Next(i+1)))
	}
	if t.buf.Len() > 64<<10 {
		t.buf.Reset()
	}
	t.mu.Unlock()
	for _, l := range lines {
		t.f(strings.TrimRight(l, "\r\n"))
	}
	return n, err
}
