package watch

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// pumpCapture sends a frame every gap until ctx ends.
func pumpCapture(frame []byte, gap time.Duration) CaptureFunc {
	return func(ctx context.Context) (<-chan []byte, error) {
		ch := make(chan []byte, 1)
		go func() {
			defer close(ch)
			for {
				select {
				case <-ctx.Done():
					return
				case ch <- frame:
				}
				if gap > 0 {
					time.Sleep(gap)
				}
			}
		}()
		return ch, nil
	}
}

// The per-part write deadline is renewed at every part: a stream lives far longer than one deadline (a global
// http.Server WriteTimeout would have cut it).
func TestVideoStreamOutlivesTheWriteDeadline(t *testing.T) {
	v, _ := NewVideo(VideoOptions{Capture: pumpCapture(make([]byte, 1000), 20*time.Millisecond), WriteTimeout: 100 * time.Millisecond})
	srv := httptest.NewServer(http.HandlerFunc(v.handleStream))
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	rd := bufio.NewReader(resp.Body)
	end, parts := time.Now().Add(time.Second), 0
	for time.Now().Before(end) {
		readPart(t, rd)
		parts++
	}
	if parts < 20 {
		t.Fatalf("only %d parts in 1 s (10 deadlines)", parts)
	}
}

// A viewer that stops reading without closing (a sleeping phone) is dropped after the write deadline and frees its
// slot; the capture stops after the linger.
func TestVideoDropsAViewerThatStoppedReading(t *testing.T) {
	v, _ := NewVideo(VideoOptions{Capture: pumpCapture(make([]byte, 256<<10), 0), MaxFrame: 1 << 20,
		WriteTimeout: 200 * time.Millisecond, Linger: 50 * time.Millisecond})
	srv := httptest.NewServer(http.HandlerFunc(v.handleStream))
	defer srv.Close()
	c, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if tc, ok := c.(*net.TCPConn); ok {
		tc.SetReadBuffer(4 << 10)
	}
	c.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\n\r\n"))
	waitFor(t, "joined", func() bool { return v.Viewers() == 1 })
	// never read again: the server's writes block, the deadline fires, the handler leaves
	deadline := time.Now().Add(10 * time.Second)
	for v.Viewers() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("a viewer that stopped reading still holds its slot")
		}
		time.Sleep(20 * time.Millisecond)
	}
	waitFor(t, "capture stopped", func() bool { return !v.Running() })
}

// The stream reader module is served next to the panels (the panel imports ../../mjpeg.js), with a JS type.
func TestMJPEGModuleServed(t *testing.T) {
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sv, err := r.sim.Hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	resp := mustGet(t, "http://"+sv.View.String()+"/watch/mjpeg.js?t=tok")
	defer resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(resp.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("%d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}
