package main

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/viewport"
)

type fastCap struct{ n atomic.Uint64 }

func (c *fastCap) Persistent() bool { return false }
func (c *fastCap) Open(context.Context, viewport.Spec) (viewport.Session, error) {
	return fastSess{c}, nil
}

type fastSess struct{ c *fastCap }

func (s fastSess) Close() error { return nil }
func (s fastSess) Next(context.Context) (viewport.Frame, error) {
	s.c.n.Add(1)
	return viewport.Frame{Data: make([]byte, 256<<10), Mime: "image/jpeg", At: time.Now()}, nil
}

func newTestStream(c viewport.Capturer) *viewport.Stream {
	return viewport.New(viewport.Config{Capturer: c, FPS: 50, Idle: 100 * time.Millisecond,
		Target: func() (viewport.Spec, error) { return viewport.Spec{}, nil }})
}

// A client that connects and never reads (a throttled background tab) must be dropped by the write deadline,
// must not multiply captures, and the capture must stop once it is gone.
func TestSlowClientDropped(t *testing.T) {
	cp := &fastCap{}
	st := newTestStream(cp)
	defer st.Close()
	lim := &viewerLimit{max: 4}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveStream(st, lim, "i1", w, r) }))
	defer srv.Close()

	c, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.(*net.TCPConn).SetReadBuffer(4096)
	c.Write([]byte("GET /stream/i1.mjpg HTTP/1.1\r\nHost: x\r\n\r\n"))
	// a healthy second client keeps receiving
	rr, _ := http.Get(srv.URL)
	br := bufio.NewReader(rr.Body)
	go func() {
		for {
			if _, err := br.ReadByte(); err != nil {
				return
			}
		}
	}()
	deadline := time.Now().Add(streamWriteTimeout + 4*time.Second)
	for lim.cur.Load() > 1 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if got := lim.cur.Load(); got != 1 {
		t.Fatalf("slow client not dropped: %d viewers", got)
	}
	rr.Body.Close()
	for lim.cur.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(400 * time.Millisecond)
	n := cp.n.Load()
	time.Sleep(400 * time.Millisecond)
	if cp.n.Load() != n {
		t.Fatal("capture still running with no viewers")
	}
}

func TestViewerCap(t *testing.T) {
	st := newTestStream(&fastCap{})
	defer st.Close()
	lim := &viewerLimit{max: 1}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveStream(st, lim, "i1", w, r) }))
	defer srv.Close()
	a, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Body.Close()
	b, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Body.Close()
	if b.StatusCode != 503 || b.Header.Get("Retry-After") == "" {
		t.Fatalf("second viewer: %d", b.StatusCode)
	}
}

// An instance that disappears closes its Stream while a viewer is still connected (the page keeps its <img>).
// serveStream must end the response instead of spinning on the "closed" error (it burned ~70% of a core).
func TestClosedStreamEndsResponse(t *testing.T) {
	st := newTestStream(&fastCap{})
	lim := &viewerLimit{max: 4}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveStream(st, lim, "i1", w, r) }))
	defer srv.Close()
	rr, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer rr.Body.Close()
	go func() {
		br := bufio.NewReader(rr.Body)
		for {
			if _, err := br.ReadByte(); err != nil {
				return
			}
		}
	}()
	time.Sleep(200 * time.Millisecond)
	st.Close()
	deadline := time.Now().Add(2 * time.Second)
	for lim.cur.Load() > 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if lim.cur.Load() != 0 {
		t.Fatal("serveStream still running after the stream was closed")
	}
}
