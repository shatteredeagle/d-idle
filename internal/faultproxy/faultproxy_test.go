package faultproxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeGame answers "state" with a payload line and ok, "err x" for anything else, and counts what it saw.
type fakeGame struct {
	ln   net.Listener
	mu   sync.Mutex
	seen []string
}

func newGame(t *testing.T, dir string) *fakeGame {
	t.Helper()
	ln, err := net.Listen("unix", filepath.Join(dir, "d2d.sock"))
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGame{ln: ln}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				rd := bufio.NewReader(c)
				for {
					l, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					l = strings.TrimSpace(l)
					g.mu.Lock()
					g.seen = append(g.seen, l)
					g.mu.Unlock()
					if l == "state" {
						fmt.Fprint(c, "life 100\nok\n")
					} else {
						fmt.Fprint(c, "ok "+l+"\n")
					}
				}
			}()
		}
	}()
	return g
}

func (g *fakeGame) count(line string) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := 0
	for _, s := range g.seen {
		if s == line {
			n++
		}
	}
	return n
}

// fakeClock is the proxy's Now; tests move it with add.
type fakeClock struct{ off atomic.Int64 }

var epoch = time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)

func (c *fakeClock) now() time.Time        { return epoch.Add(time.Duration(c.off.Load())) }
func (c *fakeClock) add(d time.Duration)   { c.off.Add(int64(d)) }
func (c *fakeClock) proxy(p *Proxy) *Proxy { p.Now = c.now; return p }

func start(t *testing.T, s Spec) (*Proxy, *fakeGame, net.Conn, *bufio.Reader, *int64) {
	t.Helper()
	p, g, c, rd, slept, _ := startClock(t, s)
	return p, g, c, rd, slept
}

// startClock starts the proxy on a fake clock; Sleep and After count the ms they were asked for and return at once.
func startClock(t *testing.T, s Spec) (*Proxy, *fakeGame, net.Conn, *bufio.Reader, *int64, *fakeClock) {
	t.Helper()
	dir := t.TempDir()
	g := newGame(t, dir)
	var slept int64
	clk := &fakeClock{}
	p := clk.proxy(&Proxy{Spec: s, Listen: filepath.Join(dir, "p.sock"), Upstream: filepath.Join(dir, "d2d.sock"),
		Sleep: func(d time.Duration) { atomic.AddInt64(&slept, int64(d/time.Millisecond)) },
		After: func(d time.Duration) <-chan time.Time {
			atomic.AddInt64(&slept, int64(d/time.Millisecond))
			ch := make(chan time.Time, 1)
			ch <- time.Time{}
			return ch
		}})
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Stop() })
	c, err := net.Dial("unix", p.Listen)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return p, g, c, bufio.NewReader(c), &slept, clk
}

func send(t *testing.T, c net.Conn, rd *bufio.Reader, line string) []string {
	t.Helper()
	fmt.Fprintln(c, line)
	var out []string
	for {
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		l, err := rd.ReadString('\n')
		if err != nil {
			t.Fatalf("%s: %v (got %v)", line, err, out)
		}
		l = strings.TrimSpace(l)
		out = append(out, l)
		if l == "ok" || strings.HasPrefix(l, "ok ") || strings.HasPrefix(l, "err") {
			return out
		}
	}
}

func TestPassThrough(t *testing.T) {
	_, g, c, rd, _ := start(t, Spec{})
	if r := send(t, c, rd, "state"); len(r) != 2 || r[0] != "life 100" {
		t.Fatalf("%v", r)
	}
	if g.count("state") != 1 {
		t.Fatal("not forwarded")
	}
}

// The rate is in time, however many commands the bot sends in between (the soak stall: one held reply every 40 commands).
func TestDelayReplyEveryIsTimeNotCommands(t *testing.T) {
	p, _, c, rd, slept, clk := startClock(t, Spec{DelayReplyEveryMs: 60_000, DelayReplyMs: 25_000})
	for i := 0; i < 200; i++ { // a chatty bot within the first minute: nothing held
		send(t, c, rd, "state")
	}
	clk.add(60 * time.Second)
	for i := 0; i < 200; i++ {
		send(t, c, rd, "state")
	}
	if *slept != 25_000 {
		t.Fatalf("held %d ms in all, want one hold of 25000", *slept)
	}
	if ev := p.Events(); len(ev) != 1 || ev[0].Kind != "delay-reply" || ev[0].Count != 201 {
		t.Fatalf("%+v", ev)
	}
}

func TestDropReplyForwardsButSwallows(t *testing.T) {
	p, g, c, rd, _, clk := startClock(t, Spec{DropReplyEveryMs: 1000})
	send(t, c, rd, "state")
	clk.add(time.Second)
	fmt.Fprintln(c, "state") // the second, a second later: dropped
	_ = c.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	if l, err := rd.ReadString('\n'); err == nil {
		t.Fatalf("a dropped reply arrived: %q", l)
	}
	if g.count("state") != 2 {
		t.Fatal("the dropped command did not reach the game")
	}
	if ev := p.Events(); len(ev) != 1 || ev[0].Kind != "drop-reply" {
		t.Fatalf("%+v", ev)
	}
}

func TestDelayInputOnlyForListedVerbs(t *testing.T) {
	_, _, c, rd, slept := start(t, Spec{DelayInputMs: 300, InputVerbs: []string{"key", "click"}})
	send(t, c, rd, "state")
	send(t, c, rd, "key Escape")
	send(t, c, rd, "click 1 2")
	if *slept != 600 {
		t.Fatalf("slept %d, want 600", *slept)
	}
}

func TestCancelStorm(t *testing.T) {
	p, g, c, rd, _ := start(t, Spec{CancelEveryMs: 10, CancelLine: "cancel"})
	deadline := time.Now().Add(2 * time.Second)
	for g.count("cancel") < 3 && time.Now().Before(deadline) {
		send(t, c, rd, "state") // the bot keeps working; the replies stay intact between cancels
	}
	if g.count("cancel") < 3 {
		t.Fatalf("only %d cancels", g.count("cancel"))
	}
	if ev := p.Stop(); len(ev) < 3 || ev[0].Kind != "cancel" {
		t.Fatalf("%+v", ev)
	}
}

func TestValidate(t *testing.T) {
	for _, s := range []Spec{{DelayReplyEveryMs: 2}, {DelayInputMs: 5}, {CancelEveryMs: 5}, {DropReplyEveryMs: -1},
		{DelayReplyEveryMs: 30_000, DelayReplyMs: 20_000}} {
		if s.Validate() == nil {
			t.Errorf("%+v accepted", s)
		}
	}
	if !(Spec{}).Empty() || (Spec{DropReplyEveryMs: 3}).Empty() {
		t.Error("Empty")
	}
}

// singleGame is d2d's control socket as far as the proxy can tell: one client at a time, a second one gets "err busy" and
// is hung up on (V docs/d2dctl.md).
func singleGame(t *testing.T, path string) {
	t.Helper()
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var busy atomic.Bool
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			if !busy.CompareAndSwap(false, true) {
				fmt.Fprint(c, "err busy\n")
				c.Close()
				continue
			}
			go func() {
				defer busy.Store(false)
				defer c.Close()
				rd := bufio.NewReader(c)
				for {
					l, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					fmt.Fprint(c, "ok "+strings.TrimSpace(l)+"\n")
				}
			}()
		}
	}()
}

// The soak's stall (docs/probes/feature-soak-2026-11-26.md 3.4): a held reply outlives the bot's command timeout, the bot
// hangs up and redials, and the proxy must give the game's one control slot back at once instead of keeping it (and its
// lock) for the rest of the hold. Before the fix every redial got "err busy" or waited out the whole hold.
func TestHeldReplyReleasedWhenTheBotHangsUp(t *testing.T) {
	dir := t.TempDir()
	singleGame(t, filepath.Join(dir, "d2d.sock"))
	clk := &fakeClock{}
	p := clk.proxy(&Proxy{Spec: Spec{DelayReplyEveryMs: 7_200_000, DelayReplyMs: 3_600_000}, Listen: filepath.Join(dir, "p.sock"), Upstream: filepath.Join(dir, "d2d.sock")})
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Stop() })
	a, err := net.Dial("unix", p.Listen)
	if err != nil {
		t.Fatal(err)
	}
	send(t, a, bufio.NewReader(a), "state")
	clk.add(2 * time.Hour)
	fmt.Fprintln(a, "state") // the second command: its reply is held for an hour
	time.Sleep(50 * time.Millisecond)
	a.Close() // the bot's command timeout

	deadline := time.Now().Add(10 * time.Second) // a ratio: the hold is an hour
	for {
		b, err := net.Dial("unix", p.Listen)
		if err == nil {
			fmt.Fprintln(b, "state")
			_ = b.SetReadDeadline(time.Now().Add(time.Second))
			l, rerr := bufio.NewReader(b).ReadString('\n')
			b.Close()
			if rerr == nil && strings.TrimSpace(l) == "ok state" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("the redialled bot got no reply while the dead connection's reply was held")
		}
		time.Sleep(20 * time.Millisecond)
	}
	stopped := make(chan []Event)
	go func() { stopped <- p.Stop() }()
	select {
	case ev := <-stopped:
		if len(ev) != 1 || ev[0].Kind != "delay-reply" {
			t.Fatalf("%+v", ev)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop waited for the held reply")
	}
}

func TestSpecRejectsUnknownKeys(t *testing.T) {
	var s Spec
	if err := json.Unmarshal([]byte(`{"delay_reply_every": 40, "delay_reply_ms": 20000}`), &s); err == nil {
		t.Fatal("the old command-counted key was accepted")
	}
	if err := json.Unmarshal([]byte(`{"drop_reply_every_ms": 60000}`), &s); err != nil || s.DropReplyEveryMs != 60000 {
		t.Fatalf("%+v %v", s, err)
	}
}
