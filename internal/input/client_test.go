package input

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeHelper serves replies from fn on a loopback listener.
func fakeHelper(t *testing.T, fn func(line string) string) (addr string, stop func(), conns *atomic.Int32) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	n := new(atomic.Int32)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			n.Add(1)
			go func() {
				defer c.Close()
				sc := bufio.NewScanner(c)
				for sc.Scan() {
					r := fn(sc.Text())
					if r == "DROP" {
						return
					}
					c.Write([]byte(r + "\n"))
				}
			}()
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }, n
}

func TestPingInfoAndErrors(t *testing.T) {
	addr, stop, _ := fakeHelper(t, func(l string) string {
		switch {
		case l == "ping":
			return "ok pong"
		case l == "info":
			return "ok hwnd=0x10 client=800x600 visible=1"
		case strings.HasPrefix(l, "seq"):
			return "err nowindow step 1 (move): no Diablo II window"
		}
		return "err unknown x"
	})
	defer stop()
	c := NewClient(addr)
	if err := c.Ping(); err != nil {
		t.Fatal(err)
	}
	w, err := c.Info()
	if err != nil || w.Width != 800 || w.Height != 600 || !w.Visible || w.Hwnd != "0x10" {
		t.Fatalf("%+v %v", w, err)
	}
	err = c.Seq("move 1 2")
	if !errors.Is(err, ErrNoWindow) {
		t.Fatalf("want ErrNoWindow, got %v", err)
	}
	var ce *CmdError
	if !errors.As(err, &ce) || ce.Code != "nowindow" {
		t.Fatalf("%v", err)
	}
}

func TestReconnectAfterDrop(t *testing.T) {
	var drop atomic.Bool
	drop.Store(true)
	addr, stop, conns := fakeHelper(t, func(l string) string {
		if drop.CompareAndSwap(true, false) {
			return "DROP" // helper restarted: connection dies without a reply
		}
		return "ok pong"
	})
	defer stop()
	c := NewClient(addr)
	if err := c.Ping(); err != nil {
		t.Fatalf("expected transparent reconnect, got %v", err)
	}
	if conns.Load() != 2 {
		t.Fatalf("conns=%d", conns.Load())
	}
}

func TestTimeoutDropsConnection(t *testing.T) {
	addr, stop, _ := fakeHelper(t, func(l string) string {
		time.Sleep(300 * time.Millisecond)
		return "ok"
	})
	defer stop()
	c := NewClient(addr)
	c.BaseTimeout = 50 * time.Millisecond
	if _, err := c.Do("ping", 0); !errors.Is(err, ErrTimeout) {
		t.Fatalf("got %v", err)
	}
}

func TestHelperDown(t *testing.T) {
	c := NewClient("127.0.0.1:1")
	c.DialTimeout = 200 * time.Millisecond
	if err := c.Ping(); !errors.Is(err, ErrHelperDown) {
		t.Fatalf("got %v", err)
	}
}

func TestSeqWork(t *testing.T) {
	d := seqWork([]string{"move 1 2", "sleep 100", "click l 1 2 hold=200"})
	if d < 300*time.Millisecond || d > time.Second {
		t.Fatal(d)
	}
}

func TestGameClickBatchesIntoOneRoundTrip(t *testing.T) {
	var lines []string
	addr, stop, _ := fakeHelper(t, func(l string) string { lines = append(lines, l); return "ok 1" })
	defer stop()
	g := NewGame(NewClient(addr), Reader{})
	g.T = Timing{MinPress: 40 * time.Millisecond}
	if err := g.Click(Right, 10, 20, ModShift); err != nil {
		t.Fatal(err)
	}
	if err := g.Click(Left, 10, 20); err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || !strings.Contains(lines[0], "click r 10 20 hold=40 shift") || !strings.Contains(lines[1], "down l 10 20") {
		t.Fatalf("%q", lines)
	}
	g.Hold("shift")
	g.Click(Left, 1, 2)
	if !strings.Contains(lines[len(lines)-1], "mkshift") {
		t.Fatalf("held shift not forwarded: %q", lines[len(lines)-1])
	}
}

func TestTargetPrefixAndDiscover(t *testing.T) {
	var got []string
	addr, stop, _ := fakeHelper(t, func(l string) string {
		got = append(got, l)
		if l == "list" {
			return "ok hwnd=0x10 pid=5 client=800x600 visible=1 ; hwnd=0x20 pid=9 client=800x600 visible=0"
		}
		return "ok"
	})
	defer stop()
	c := NewClientFor(addr, 0x20)
	ws, err := Discover(c)
	if err != nil || len(ws) != 2 || ws[1].H != 0x20 || ws[1].Pid != 9 || ws[1].Visible || ws[0].Width != 800 {
		t.Fatalf("%v %+v", err, ws)
	}
	if err := c.Seq("move 1 2"); err != nil {
		t.Fatal(err)
	}
	if got[len(got)-1] != "@0x20 seq move 1 2" {
		t.Fatalf("%v", got)
	}
	if _, err := NewClient(addr).Info(); err != nil || got[len(got)-1] != "info" {
		t.Fatalf("untargeted: %v %v", err, got)
	}
}

func TestHealRestartsDeadHelper(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close() // nothing listens: the helper is "dead"
	c := NewClientFor(addr, 0)
	c.DialTimeout = 100 * time.Millisecond
	calls := 0
	c.Restart = func() error {
		calls++
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		go func() {
			for {
				cn, err := l.Accept()
				if err != nil {
					return
				}
				go func() {
					defer cn.Close()
					sc := bufio.NewScanner(cn)
					for sc.Scan() {
						fmt.Fprintln(cn, "ok pong")
					}
				}()
			}
		}()
		return nil
	}
	if err := c.Ping(); err != nil {
		t.Fatalf("ping after heal: %v", err)
	}
	if calls != 1 {
		t.Fatalf("restart calls = %d", calls)
	}
	// cooldown: a second dead helper is not restarted again at once
	c.Close()
	c.healed = time.Now()
	c.Addr = "127.0.0.1:1"
	if err := c.Ping(); !errors.Is(err, ErrHelperDown) || calls != 1 {
		t.Fatalf("cooldown: err=%v calls=%d", err, calls)
	}
}
