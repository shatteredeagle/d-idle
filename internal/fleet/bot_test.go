package fleet

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/paths"
)

type fakeProc struct {
	pid    int
	exit   chan error
	mu     sync.Mutex
	sigs   []os.Signal
	killed bool
	once   sync.Once
}

func (p *fakeProc) Pid() int    { return p.pid }
func (p *fakeProc) Wait() error { return <-p.exit }
func (p *fakeProc) end(err error) {
	p.once.Do(func() { p.exit <- err })
}
func (p *fakeProc) Signal(s os.Signal) error {
	p.mu.Lock()
	p.sigs = append(p.sigs, s)
	p.mu.Unlock()
	if s == syscall.SIGTERM {
		p.end(nil)
	}
	return nil
}
func (p *fakeProc) Kill() error {
	p.mu.Lock()
	p.killed = true
	p.mu.Unlock()
	p.end(errors.New("killed"))
	return nil
}

type fakeExec struct {
	mu      sync.Mutex
	cmds    []BotCmd
	procs   []*fakeProc
	ignore  bool // ignore SIGTERM (needs Kill)
	failN   int  // fail the first N starts
	started chan struct{}
	onStart func(BotCmd) // bot-owned model: d2bot brings its game up
	noGoals bool         // d2bot never reaches "goals enqueued" (its game launch failed)
	exitMs  int          // exit this long after SIGTERM (a graceful shutdown), 0 = at once
}

func newExec() *fakeExec { return &fakeExec{started: make(chan struct{}, 100)} }

func (e *fakeExec) Start(_ context.Context, c BotCmd) (BotProc, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.failN > 0 {
		e.failN--
		return nil, errors.New("no such file")
	}
	p := &fakeProc{pid: 100 + len(e.procs), exit: make(chan error, 1)}
	e.cmds = append(e.cmds, c)
	e.procs = append(e.procs, p)
	e.started <- struct{}{}
	if e.onStart != nil {
		e.onStart(c)
	}
	if c.Tap != nil {
		c.Tap(`{"level":"INFO","msg":"managed game attached","pid":4242}`)
	}
	if c.Tap != nil {
		c.Tap(`{"level":"INFO","msg":"listening","addr":"127.0.0.1:41233","inst":"i1"}`)
	}
	if c.Tap != nil && !e.noGoals && slices.Contains(c.Args, "--goals") {
		c.Tap(`{"level":"INFO","msg":"goals enqueued"}`)
		writeGoals(c.Args)
	}
	return &sigProc{fakeProc: p, ignore: e.ignore, delay: time.Duration(e.exitMs) * time.Millisecond}, nil
}

// goalsFile is where the fake d2bot "stores" goals: <data>/instances/<id>/goals.json, as d2bot does.
func goalsFile(args []string) string {
	var data, id string
	for i := 0; i+1 < len(args); i++ {
		switch args[i] {
		case "--data":
			data = args[i+1]
		case "--instance":
			id = args[i+1]
		}
	}
	return filepath.Join(data, "instances", id, "goals.json")
}

func writeGoals(args []string) {
	f := goalsFile(args)
	os.MkdirAll(filepath.Dir(f), 0o755)
	os.WriteFile(f, []byte(`{"goals":[{"id":1}]}`), 0o644)
}

type sigProc struct {
	*fakeProc
	ignore bool
	delay  time.Duration
}

func (p *sigProc) Signal(s os.Signal) error {
	if p.ignore {
		return nil
	}
	if p.delay > 0 && s == syscall.SIGTERM {
		p.mu.Lock()
		p.sigs = append(p.sigs, s)
		p.mu.Unlock()
		time.AfterFunc(p.delay, func() { p.end(nil) })
		return nil
	}
	return p.fakeProc.Signal(s)
}

func (e *fakeExec) proc(i int) *fakeProc { e.mu.Lock(); defer e.mu.Unlock(); return e.procs[i] }
func (e *fakeExec) cmd(i int) BotCmd     { e.mu.Lock(); defer e.mu.Unlock(); return e.cmds[i] }
func (e *fakeExec) n() int               { e.mu.Lock(); defer e.mu.Unlock(); return len(e.procs) }
func (e *fakeExec) waitStart(t *testing.T) {
	t.Helper()
	select {
	case <-e.started:
	case <-time.After(5 * time.Second):
		t.Fatal("d2bot not started")
	}
}

func newBots(t *testing.T, ex *fakeExec) *ProcBots {
	t.Helper()
	return &ProcBots{Attach: true, Exec: ex, Binary: "/bin/d2bot", Layout: paths.Layout{Root: t.TempDir()}, Sleep: fastSleep,
		Backoff: Backoff{Min: time.Millisecond, Max: 4 * time.Millisecond, StableAfter: time.Hour}, StopGrace: 20 * time.Millisecond}
}

func waitBot(t *testing.T, f *Fleet, id, state string) BotStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		v, err := f.Get(id)
		if err == nil && v.Bot != nil && v.Bot.State == state {
			return *v.Bot
		}
		if time.Now().After(deadline) {
			t.Fatalf("bot never reached %s: %+v", state, v.Bot)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestBotArgsFirstAndRestart(t *testing.T) {
	ex := newExec()
	pb := newBots(t, ex)
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.Bots = pb })
	v, err := f.Add(context.Background(), Spec{Character: "Alpha", Goals: "levelto:9,farm:Blood_Moor", Profile: "relaxed"})
	if err != nil {
		t.Fatal(err)
	}
	ex.waitStart(t)
	c := ex.cmd(0)
	if c.Path != "/bin/d2bot" {
		t.Fatal(c.Path)
	}
	for _, want := range [][]string{{"--managed"}, {"--instance", v.ID}, {"--char", "Alpha"}, {"--addr", "127.0.0.1:0"}, {"--goals", "reach:9;farm:Blood Moor"}, {"--autostart"}} {
		if i := slices.Index(c.Args, want[0]); i < 0 || (len(want) == 2 && c.Args[i+1] != want[1]) {
			t.Fatalf("args %v lack %v", c.Args, want)
		}
	}
	if slices.Contains(c.Args, "--stop-game") {
		t.Fatal("the fleet owns the game")
	}
	b := waitBot(t, f, v.ID, BotRunning)
	for b.Addr == "" {
		b = waitBot(t, f, v.ID, BotRunning)
	}
	if b.Pid != 100 || b.Addr != "127.0.0.1:41233" || !strings.HasSuffix(b.Log, filepath.Join("instances", v.ID, "logs", "d2bot.log")) {
		t.Fatalf("%+v", b)
	}
	// the d2bot exits: restart without --goals (they are in goals.json now)
	ex.proc(0).end(errors.New("exit status 1"))
	ex.waitStart(t)
	c = ex.cmd(1)
	if slices.Contains(c.Args, "--goals") || !slices.Contains(c.Args, "--autostart") {
		t.Fatalf("restart args %v", c.Args)
	}
	b = waitBot(t, f, v.ID, BotRunning)
	if b.Restarts != 1 || !strings.Contains(b.LastError, "exit status 1") {
		t.Fatalf("%+v", b)
	}
}

func TestBotConfigProfile(t *testing.T) {
	root := t.TempDir()
	base := `{"bot":{"profile":"casual","autostart":true},"other":1}`
	if err := os.WriteFile(filepath.Join(root, "config.json"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	pb := &ProcBots{Layout: paths.Layout{Root: root}}
	al := AllocFor(2, nil)
	path, err := pb.WriteConfig(al, Spec{Character: "Beta", Profile: "sloppy"})
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "instances", "i2", "config.json") {
		t.Fatal(path)
	}
	c, err := bot.LoadConfig(path)
	if err != nil || c.Profile != "sloppy" {
		t.Fatalf("%v profile %q", err, c.Profile)
	}
	var top map[string]json.RawMessage
	b, _ := os.ReadFile(path)
	if json.Unmarshal(b, &top) != nil || string(top["other"]) != "1" {
		t.Fatalf("other key lost: %s", b)
	}
	// optimal removes the key; the host config file is untouched
	path, _ = pb.WriteConfig(al, Spec{Character: "Beta"})
	c, err = bot.LoadConfig(path)
	if err != nil || c.Profile != "" {
		t.Fatalf("%v profile %q", err, c.Profile)
	}
	if got, _ := os.ReadFile(filepath.Join(root, "config.json")); string(got) != base {
		t.Fatal("host config modified")
	}
	// no host config at all
	pb2 := &ProcBots{Layout: paths.Layout{Root: t.TempDir()}}
	path, err = pb2.WriteConfig(al, Spec{Character: "Beta", Profile: "rookie"})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := bot.LoadConfig(path); err != nil || c.Profile != "rookie" {
		t.Fatalf("%v %q", err, c.Profile)
	}
}

func TestBotRestartBackoffAndGiveUp(t *testing.T) {
	ex := newExec()
	ex.failN = 100
	pb := newBots(t, ex)
	pb.MaxRestarts = 3
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.Bots = pb })
	v, _ := f.Add(context.Background(), Spec{Character: "Gamma"})
	b := waitBot(t, f, v.ID, BotFailed)
	if b.Restarts != 4 || !strings.Contains(b.LastError, "no such file") {
		t.Fatalf("%+v", b)
	}
	// the game keeps running
	if got, _ := f.Get(v.ID); got.State != Running {
		t.Fatal(got.State)
	}
}

func TestBotStoppedOnRemoveBeforeLeave(t *testing.T) {
	ex := newExec()
	ex.ignore = true // needs the kill after StopGrace
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = newBots(t, ex) })
	v, _ := f.Add(context.Background(), Spec{Character: "Delta"})
	ex.waitStart(t)
	waitBot(t, f, v.ID, BotRunning)
	if err := f.Remove(context.Background(), v.ID, RemoveOpts{}); err != nil {
		t.Fatal(err)
	}
	p := ex.proc(0)
	p.mu.Lock()
	killed := p.killed
	p.mu.Unlock()
	if !killed {
		t.Fatal("stuck d2bot was not killed")
	}
}

func TestBotFollowsGameRestart(t *testing.T) {
	ex := newExec()
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = newBots(t, ex) })
	v, _ := f.Add(context.Background(), Spec{Character: "Eps"})
	ex.waitStart(t)
	waitBot(t, f, v.ID, BotRunning)
	rt.kill(v.ID)
	ex.waitStart(t) // the game came back, so did the bot
	if ex.n() != 2 {
		t.Fatal(ex.n())
	}
	p := ex.proc(0)
	p.mu.Lock()
	n := len(p.sigs)
	p.mu.Unlock()
	if n == 0 {
		t.Fatal("old d2bot was not told to stop when the game died")
	}
}

func TestListenAddr(t *testing.T) {
	if got := listenAddr(`{"msg":"listening","addr":"127.0.0.1:5"}`); got != "127.0.0.1:5" {
		t.Fatal(got)
	}
	for _, l := range []string{`{"msg":"x","addr":"a"}`, "plain listening", ""} {
		if listenAddr(l) != "" {
			t.Fatal(l)
		}
	}
}

func TestLineTap(t *testing.T) {
	var got []string
	var sink strings.Builder
	tp := &lineTap{w: &sink, f: func(s string) { got = append(got, s) }}
	tp.Write([]byte("a\nb"))
	tp.Write([]byte("c\n"))
	if len(got) != 2 || got[0] != "a" || got[1] != "bc" || sink.String() != "a\nbc\n" {
		t.Fatalf("%q %q", got, sink.String())
	}
}

func TestBotGoalsRetriedUntilEnqueued(t *testing.T) {
	ex := newExec()
	ex.noGoals = true // first start: the game never came up, d2bot died before enqueueing
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = newBots(t, ex) })
	v, _ := f.Add(context.Background(), Spec{Character: "Alpha", Goals: "levelto:9"})
	ex.waitStart(t)
	waitBot(t, f, v.ID, BotRunning)
	ex.proc(0).end(errors.New("exit status 1"))
	ex.waitStart(t)
	if !slices.Contains(ex.cmd(1).Args, "--goals") {
		t.Fatalf("goals lost after a failed first start: %v", ex.cmd(1).Args)
	}
	ex.noGoals = false
	ex.proc(1).end(errors.New("exit status 1"))
	ex.waitStart(t)
	if !slices.Contains(ex.cmd(2).Args, "--goals") {
		t.Fatalf("still not enqueued: %v", ex.cmd(2).Args)
	}
	ex.proc(2).end(errors.New("exit status 1"))
	ex.waitStart(t)
	if slices.Contains(ex.cmd(3).Args, "--goals") {
		t.Fatalf("goals passed again after they were enqueued: %v", ex.cmd(3).Args)
	}
}

func TestBotGoalsReseededWhenStoreLost(t *testing.T) {
	ex := newExec()
	rt := newRT()
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = newBots(t, ex) })
	v, _ := f.Add(context.Background(), Spec{Character: "Alpha", Goals: "levelto:9"})
	ex.waitStart(t)
	waitBot(t, f, v.ID, BotRunning)
	for i, lose := range []string{"missing", "empty"} {
		gf := goalsFile(ex.cmd(i).Args)
		if lose == "missing" {
			os.Remove(gf)
		} else {
			os.WriteFile(gf, []byte(`{"goals":[]}`), 0o644)
		}
		ex.proc(i).end(errors.New("exit status 1"))
		ex.waitStart(t)
		if !slices.Contains(ex.cmd(i+1).Args, "--goals") {
			t.Fatalf("store %s: goals not passed again: %v", lose, ex.cmd(i+1).Args)
		}
	}
}

func TestBotEnvCarriesInstance(t *testing.T) {
	ex := newExec()
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.Bots = newBots(t, ex) })
	v, _ := f.Add(context.Background(), Spec{Character: "Alpha"})
	ex.waitStart(t)
	if !slices.Contains(ex.cmd(0).Env, "D2BOT_INSTANCE="+v.ID) {
		t.Fatalf("env %v", ex.cmd(0).Env)
	}
}

func TestBotGracefulStopNotKilled(t *testing.T) {
	ex := newExec()
	ex.exitMs = 60 // slower than the old 10 s budget scaled down: needs the full grace
	rt := newRT()
	pb := newBots(t, ex)
	pb.StopGrace = 2 * time.Second
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = pb })
	v, _ := f.Add(context.Background(), Spec{Character: "Delta"})
	ex.waitStart(t)
	waitBot(t, f, v.ID, BotRunning)
	if err := f.Remove(context.Background(), v.ID, RemoveOpts{}); err != nil {
		t.Fatal(err)
	}
	p := ex.proc(0)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.killed || len(p.sigs) == 0 {
		t.Fatalf("killed=%v sigs=%v", p.killed, p.sigs)
	}
}

func TestDefaultStopGraceCoversShutdown(t *testing.T) {
	p := &ProcBots{}
	p.defaults()
	if p.StopGrace < 60*time.Second {
		t.Fatal(p.StopGrace)
	}
}

func TestRotWriterRotatesBySize(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d2bot.log")
	w, err := newRotWriter(p, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	line := []byte(strings.Repeat("x", 60) + "\n")
	for i := 0; i < 4; i++ {
		if _, err := w.Write(line); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(p + ".1"); err != nil {
		t.Fatalf("no rotated log: %v", err)
	}
	if st, _ := os.Stat(p); st == nil || st.Size() > 200 {
		t.Fatal("current log not bounded")
	}
}

type fatalExit struct{}

func (fatalExit) Error() string { return "exit status 3" }
func (fatalExit) ExitCode() int { return 3 }

func TestBotExitFatalIsFinal(t *testing.T) {
	ex := newExec()
	pb := newBots(t, ex)
	f, _ := mk(t, newRT(), bigProbe(), func(c *Config) { c.Bots = pb })
	v, _ := f.Add(context.Background(), Spec{Character: "Gamma"})
	<-ex.started
	ex.mu.Lock()
	p := ex.procs[0]
	ex.mu.Unlock()
	p.end(fatalExit{})
	b := waitBot(t, f, v.ID, BotFailed)
	if b.Restarts != 0 || !strings.Contains(b.LastError, "exit 3") {
		t.Fatalf("%+v", b)
	}
	time.Sleep(50 * time.Millisecond)
	ex.mu.Lock()
	n := len(ex.procs)
	ex.mu.Unlock()
	if n != 1 {
		t.Fatalf("restarted after a deliberate fatal: %d starts", n)
	}
}
