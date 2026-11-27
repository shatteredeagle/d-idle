package faultproxy

import (
	"bufio"
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

func start(t *testing.T, s Spec) (*Proxy, *fakeGame, net.Conn, *bufio.Reader, *int64) {
	t.Helper()
	dir := t.TempDir()
	g := newGame(t, dir)
	var slept int64
	p := &Proxy{Spec: s, Listen: filepath.Join(dir, "p.sock"), Upstream: filepath.Join(dir, "d2d.sock"),
		Sleep: func(d time.Duration) { atomic.AddInt64(&slept, int64(d/time.Millisecond)) }}
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Stop() })
	c, err := net.Dial("unix", p.Listen)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return p, g, c, bufio.NewReader(c), &slept
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

func TestDelayReplyEvery(t *testing.T) {
	p, _, c, rd, slept := start(t, Spec{DelayReplyEvery: 2, DelayReplyMs: 900})
	for i := 0; i < 4; i++ {
		send(t, c, rd, "state")
	}
	if *slept != 1800 {
		t.Fatalf("slept %d ms, want 2 x 900", *slept)
	}
	if ev := p.Events(); len(ev) != 2 || ev[0].Kind != "delay-reply" || ev[0].Count != 2 {
		t.Fatalf("%+v", ev)
	}
}

func TestDropReplyForwardsButSwallows(t *testing.T) {
	p, g, c, rd, _ := start(t, Spec{DropReplyEvery: 2})
	send(t, c, rd, "state")
	fmt.Fprintln(c, "state") // the second: dropped
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
	for _, s := range []Spec{{DelayReplyEvery: 2}, {DelayInputMs: 5}, {CancelEveryMs: 5}, {DropReplyEvery: -1}} {
		if s.Validate() == nil {
			t.Errorf("%+v accepted", s)
		}
	}
	if !(Spec{}).Empty() || (Spec{DropReplyEvery: 3}).Empty() {
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
	p := &Proxy{Spec: Spec{DelayReplyEvery: 2, DelayReplyMs: 3_600_000}, Listen: filepath.Join(dir, "p.sock"), Upstream: filepath.Join(dir, "d2d.sock")}
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Stop() })
	a, err := net.Dial("unix", p.Listen)
	if err != nil {
		t.Fatal(err)
	}
	send(t, a, bufio.NewReader(a), "state")
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
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("the redialled bot got no reply while the dead connection's reply was held")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
