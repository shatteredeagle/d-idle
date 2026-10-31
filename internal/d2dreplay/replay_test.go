package d2dreplay

import (
	"bufio"
	"context"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl/fake"
)

func init() { CheckpointVerbs = []string{"monsters", "ground"} } // the fake has no dump verb (same as detloop's test)

func header() Header {
	return Header{Every: 100, D2D: Build{SHA256: strings.Repeat("cd", 32), Args: []string{"--lockstep", "--no-save"}},
		Start: &Recipe{Seed: 3, Class: 1, Name: "Det", Screen: "ingame"}, Meta: Meta{Tool: "test"}}
}

func TestFormatRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.replay.gz")
	w, err := Create(p, header())
	if err != nil {
		t.Fatal(err)
	}
	w.Add(Entry{T: 0, Verb: "click 400 500", W: 12})
	w.Add(Entry{T: 0, Verb: "step 50"})
	w.Add(Entry{C: 50, H: strings.Repeat("ab", 32), Lv: 1, Area: 1, Life: "45/45"})
	w.Add(Entry{R: true, T: 50})
	if err := w.Close(&Entry{Ticks: 50, Verbs: 2, WallMS: 99}); err != nil {
		t.Fatal(err)
	}
	rp, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if rp.Header.Replay != Version || rp.Header.Hash != HashV1 || rp.Header.Start.Seed != 3 || rp.Header.Created == "" {
		t.Fatalf("header %+v", rp.Header)
	}
	kinds := []string{}
	for _, e := range rp.Entries {
		kinds = append(kinds, e.Kind())
	}
	if strings.Join(kinds, ",") != "verb,verb,checkpoint,restart" {
		t.Fatalf("kinds %v", kinds)
	}
	if rp.End == nil || rp.End.Ticks != 50 || rp.Truncated || rp.Ticks() != 50 || len(rp.Checkpoints()) != 1 {
		t.Fatalf("end %+v truncated %v ticks %d", rp.End, rp.Truncated, rp.Ticks())
	}
	// no paths, no hosts in the bytes
	gz, _ := os.Open(p)
	raw, _ := ReadFrom(gz)
	if raw.Header.D2D.Args[0] != "--lockstep" {
		t.Fatal("args lost")
	}
}

func TestTruncatedFileStillReads(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.replay.gz")
	w, _ := Create(p, header())
	w.Add(Entry{T: 0, Verb: "step 10"})
	w.Flush()
	// the recorder dies here: no trailer, gzip stream unterminated
	w.f.Close()
	rp, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if !rp.Truncated || rp.End != nil || len(rp.Entries) != 1 {
		t.Fatalf("truncated %v end %v n %d", rp.Truncated, rp.End, len(rp.Entries))
	}
}

func TestVersionRefused(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.replay.gz")
	h := header()
	h.Replay = 99
	w, _ := Create(p, h)
	w.Close(nil)
	if _, err := Read(p); err == nil || !strings.Contains(err.Error(), "unsupported replay version") {
		t.Fatalf("err %v", err)
	}
}

func TestMutating(t *testing.T) {
	yes := []string{"step 50", "click 1 2", "key Escape", "cmd move 1 2", "debug goto 2", "debug stat 12 8", "debug quest 1 done", "save", "quit", "lockstep on", "pad a"}
	no := []string{"state", "monsters", "clock", "ping", "dump items", "screenshot /tmp/x.png", "debug players", "debug level", "debug grid 3", "debug quest 1 show", ""}
	for _, l := range yes {
		if !Mutating(l) {
			t.Errorf("%q should be mutating", l)
		}
	}
	for _, l := range no {
		if Mutating(l) {
			t.Errorf("%q should be a read", l)
		}
	}
}

// client is a raw downstream client for the tap in tests.
type client struct {
	c  net.Conn
	rd *bufio.Reader
}

func dialTap(t *testing.T, path string) *client {
	t.Helper()
	var c net.Conn
	var err error
	for i := 0; i < 40; i++ {
		c, err = net.Dial("unix", path)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	return &client{c, bufio.NewReader(c)}
}

func (c *client) do(t *testing.T, line string) (lines []string, status string) {
	t.Helper()
	io.WriteString(c.c, line+"\n")
	for {
		l, err := c.rd.ReadString('\n')
		if err != nil {
			t.Fatalf("%q: %v", line, err)
		}
		l = strings.TrimRight(l, "\n")
		if l == "ok" || strings.HasPrefix(l, "ok ") || strings.HasPrefix(l, "err") {
			return lines, l
		}
		lines = append(lines, l)
	}
}

func startFake(t *testing.T, fix fake.Fixtures, opts fake.Options) *fake.Server {
	t.Helper()
	sock, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	s, err := fake.New(sock, fix, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

// record runs a tap in front of a fake and drives it with lines; returns the replay path.
func record(t *testing.T, up *fake.Server, every int, lines ...string) (string, *Tap, []string) {
	t.Helper()
	dir, _ := os.MkdirTemp("", "d2drp")
	t.Cleanup(func() { os.RemoveAll(dir) })
	out := filepath.Join(dir, "r.replay.gz")
	w, err := Create(out, header())
	if err != nil {
		t.Fatal(err)
	}
	tap := &Tap{Upstream: up.Path(), Listen: filepath.Join(dir, "tap.sock"), W: w, Every: every, Timeout: 5 * time.Second, Redial: 3 * time.Second,
		Logf: t.Logf}
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { done <- tap.Serve(ctx) }()
	c := dialTap(t, tap.Listen)
	var replies []string
	for _, l := range lines {
		pl, st := c.do(t, l)
		replies = append(replies, strings.Join(append(pl, st), "|"))
	}
	c.c.Close()
	if strings.HasSuffix(lines[len(lines)-1], "quit") {
		// the fake is gone after quit; give the tap's idle stop a moment, then cancel like a signal would
		time.Sleep(150 * time.Millisecond)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("tap did not stop")
	}
	return out, tap, replies
}

func TestTapRecordsMutatingVerbsAndCheckpoints(t *testing.T) {
	up := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	out, tap, replies := record(t, up, 100, "state", "click 400 500", "step 60", "monsters", "step 60", "debug goto 2", "cmd move 10 10", "debug players", "quit")
	// replies are d2d's own
	if !strings.HasPrefix(replies[0], "screen=ingame") || replies[1] != "ok" || !strings.Contains(replies[2], "tick=60|ok") || replies[5] != "ok 30.5 20.5" {
		t.Fatalf("replies %q", replies)
	}
	// nothing injected into the client's stream: the fake saw our reads only at the checkpoint
	seen := up.Lines()
	for _, l := range seen {
		if strings.HasPrefix(l, "step") && strings.Contains(l, "render") {
			t.Fatalf("rewritten line %q", l)
		}
	}
	rp, err := Read(out)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range rp.Entries {
		switch e.Kind() {
		case "verb":
			got = append(got, e.Verb+"@"+itoa(e.T))
		case "checkpoint":
			got = append(got, "cp@"+itoa(e.C))
		}
	}
	want := "click 400 500@0 step 60@0 step 60@60 cp@120 debug goto 2@120 cmd move 10 10@120 quit@120"
	if strings.Join(got, " ") != want {
		t.Fatalf("entries:\n got %s\nwant %s", strings.Join(got, " "), want)
	}
	cp := rp.Checkpoints()[0]
	if len(cp.H) != 64 || cp.Lv != 1 || cp.Life != "45/45" {
		t.Fatalf("checkpoint %+v", cp)
	}
	if rp.End == nil || rp.End.Ticks != 120 || rp.End.Verbs != 6 || tap.Stats().Checkpoints != 1 {
		t.Fatalf("end %+v stats %+v", rp.End, tap.Stats())
	}
	// no socket path, no home, no host inside
	for _, e := range rp.Entries {
		if strings.Contains(e.Verb, "/") {
			t.Fatalf("path in verb %q", e.Verb)
		}
	}
}

func TestPlayerMatchesSameFixtures(t *testing.T) {
	up := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	out, _, _ := record(t, up, 50, "click 400 500", "step 60", "step 60", "state", "step 30", "quit")
	rp, _ := Read(out)
	if n := len(rp.Checkpoints()); n != 3 { // at 60, 120, 150
		t.Fatalf("checkpoints %d", n)
	}
	// playback against a fresh fake with the same fixtures
	up2 := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	c, err := Dial(up2.Path(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var steps []int
	res, err := (&Player{Conn: c, AfterStep: func(tick int) error { steps = append(steps, tick); return nil }}).Run(context.Background(), rp)
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Checkpoints != 3 || res.Ticks != 150 || res.Verbs != 4 || len(steps) != 3 {
		t.Fatalf("result %+v steps %v", res, steps)
	}
	if up2.Quit() {
		t.Fatal("player must not send quit")
	}
	// the player sent exactly the recorded verbs plus the checkpoint reads
	var sent []string
	for _, l := range up2.Lines() {
		if Mutating(l) {
			sent = append(sent, l)
		}
	}
	if strings.Join(sent, ",") != "click 400 500,step 60,step 60,step 30" {
		t.Fatalf("sent %v", sent)
	}
}

func TestPlayerReportsFirstDivergence(t *testing.T) {
	up := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	out, _, _ := record(t, up, 50, "click 400 500", "step 60", "step 60", "quit")
	rp, _ := Read(out)
	fix := fake.Sample()
	fix.Monsters = append(fix.Monsters, "zombie1\t1.0\t1.0\t0\t0\t5/5\tNU\t-\t-\tlvl1\tZombie\t#9\t-")
	up2 := startFake(t, fix, fake.Options{NoQuirk: true})
	c, _ := Dial(up2.Path(), 5*time.Second)
	defer c.Close()
	res, err := (&Player{Conn: c}).Run(context.Background(), rp)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK || res.Diverged == nil || res.Diverged.Tick != 60 || res.Diverged.Index != 0 || res.Checkpoints != 0 {
		t.Fatalf("result %+v", res)
	}
	if !strings.HasPrefix(res.String(), "DIVERGED at tick 60") {
		t.Fatal(res.String())
	}
	// a stop before the divergence is "OK so far"
	c2, _ := Dial(up2.Path(), 5*time.Second)
	_ = c2.Close()
	c3 := startFake(t, fix, fake.Options{NoQuirk: true})
	cc, _ := Dial(c3.Path(), 5*time.Second)
	defer cc.Close()
	res, _ = (&Player{Conn: cc, Until: 30}).Run(context.Background(), rp)
	if res.Stopped == "" || res.Diverged != nil {
		t.Fatalf("until: %+v", res)
	}
}

func TestPlayerRefusesUnknownHash(t *testing.T) {
	rp := &Replay{Header: Header{Hash: "v9"}}
	if _, err := (&Player{Conn: nil}).Run(context.Background(), rp); err == nil || !strings.Contains(err.Error(), "v9") {
		t.Fatal(err)
	}
}

func TestTapUpstreamRestartMarker(t *testing.T) {
	// the fake drops the connection on the 4th command (the tap's own ping is the 1st) without a reply; the tap answers err, redials and marks a restart
	up := startFake(t, fake.Sample(), fake.Options{NoQuirk: true, DropAfter: 4})
	out, tap, replies := record(t, up, 1000, "click 400 500", "step 10", "step 10", "step 10", "quit")
	if !strings.HasPrefix(replies[2], "err replay: upstream") || !strings.Contains(replies[3], "ok") {
		t.Fatalf("replies %q", replies)
	}
	rp, _ := Read(out)
	var kinds []string
	for _, e := range rp.Entries {
		kinds = append(kinds, e.Kind())
	}
	if strings.Join(kinds, ",") != "verb,verb,restart,verb,verb" || tap.Stats().Restarts != 1 {
		t.Fatalf("kinds %v stats %+v", kinds, tap.Stats())
	}
	// the dropped step got no reply from d2d, so it is neither recorded nor counted
	if rp.End.Ticks != 20 {
		t.Fatalf("ticks %d", rp.End.Ticks)
	}
	// playback stops at the marker
	up2 := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	c, _ := Dial(up2.Path(), 5*time.Second)
	defer c.Close()
	res, err := (&Player{Conn: c}).Run(context.Background(), rp)
	if err != nil || !strings.Contains(res.Stopped, "restart marker at tick 10") {
		t.Fatalf("%v %+v", err, res)
	}
}

func TestLaunchArgs(t *testing.T) {
	l := &Launch{Data: "/d", Sock: "/s/d2d.sock", Build: Build{Args: []string{"--lockstep", "--toggle", "autoloot=off"}},
		Recipe: &Recipe{Seed: 5, Class: 4, Name: "Viewy", Hardcore: true}}
	a, err := l.Args()
	if err != nil {
		t.Fatal(err)
	}
	want := "--data /d --headless --devctl /s/d2d.sock --no-video --seed 5 --start-screen ingame --start-class 4 --start-name Viewy --start-hardcore --lockstep --toggle autoloot=off"
	if strings.Join(a, " ") != want {
		t.Fatalf("args %q", strings.Join(a, " "))
	}
	if _, err := (&Launch{}).Args(); err == nil {
		t.Fatal("no recipe must fail")
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// tapRig starts a tap in front of a fake and returns it with its context cancel and result channel.
func tapRig(t *testing.T, tap func(*Tap)) (*Tap, context.CancelFunc, chan error, string) {
	t.Helper()
	up := startFake(t, fake.Sample(), fake.Options{NoQuirk: true})
	dir, _ := os.MkdirTemp("", "d2drp")
	t.Cleanup(func() { os.RemoveAll(dir) })
	out := filepath.Join(dir, "r.replay.gz")
	w, err := Create(out, header())
	if err != nil {
		t.Fatal(err)
	}
	tp := &Tap{Upstream: up.Path(), Listen: filepath.Join(dir, "tap.sock"), W: w, Every: 100, Timeout: 5 * time.Second, Logf: t.Logf}
	if tap != nil {
		tap(tp)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- tp.Serve(ctx) }()
	return tp, cancel, done, out
}

func TestTapStopsWithClientAttachedAndWritesTrailer(t *testing.T) {
	tp, cancel, done, out := tapRig(t, nil)
	c := dialTap(t, tp.Listen)
	c.do(t, "step 10")
	cancel() // a signal or -max while the client stays connected
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Serve still blocked with a client attached")
	}
	rp, err := Read(out)
	if err != nil {
		t.Fatal(err)
	}
	if rp.End == nil || rp.Truncated || rp.End.Ticks != 10 {
		t.Fatalf("trailer %+v truncated %v", rp.End, rp.Truncated)
	}
}

func TestTapSecondClientGetsBusy(t *testing.T) {
	tp, cancel, done, _ := tapRig(t, nil)
	c := dialTap(t, tp.Listen)
	c.do(t, "state")
	c2 := dialTap(t, tp.Listen)
	c2.c.SetReadDeadline(time.Now().Add(2 * time.Second))
	l, err := c2.rd.ReadString('\n')
	if err != nil || strings.TrimSpace(l) != "err busy" {
		t.Fatalf("second client: %q %v", l, err)
	}
	if _, st := c.do(t, "state"); !strings.HasPrefix(st, "ok") { // the first client is unharmed
		t.Fatalf("first client: %q", st)
	}
	cancel()
	<-done
}

func TestTapIdleStopsAfterClientLeaves(t *testing.T) {
	tp, _, done, out := tapRig(t, func(tp *Tap) { tp.Idle = time.Second })
	c := dialTap(t, tp.Listen)
	c.do(t, "step 5")
	c.c.Close()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("idle did not stop the tap")
	}
	if rp, _ := Read(out); rp == nil || rp.End == nil {
		t.Fatal("no trailer after idle stop")
	}
}

func TestTapRefusesUnsafeSockets(t *testing.T) {
	tp, cancel, done, _ := tapRig(t, nil)
	cancel()
	<-done
	// a listen path that is a regular file is not removed
	f := filepath.Join(t.TempDir(), "precious")
	os.WriteFile(f, []byte("x"), 0o600)
	w, _ := Create(filepath.Join(t.TempDir(), "o.gz"), header())
	bad := &Tap{Upstream: tp.Upstream, Listen: f, W: w}
	if err := bad.Serve(context.Background()); err == nil {
		t.Fatal("listen on a regular file must fail")
	}
	if b, err := os.ReadFile(f); err != nil || string(b) != "x" {
		t.Fatal("regular file was removed")
	}
	same := &Tap{Upstream: "/tmp/x.sock", Listen: "/tmp/x.sock", W: w}
	if err := same.Serve(context.Background()); err == nil {
		t.Fatal("listen == upstream must fail")
	}
}
