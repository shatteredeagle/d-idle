package watch

import (
	"bufio"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeTap stands in for cmd/d2dview: GET /frame with the same status codes (200, 401, 404, 503).
type fakeTap struct {
	mu     sync.Mutex
	frame  []byte
	typ    string
	status int // 0: serve the frame
	token  string
	hits   atomic.Int64
	srv    *httptest.Server
}

func newFakeTap(t *testing.T, token string) *fakeTap {
	t.Helper()
	f := &fakeTap{token: token}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/frame" {
			http.NotFound(w, r)
			return
		}
		f.hits.Add(1)
		f.mu.Lock()
		frame, typ, status := f.frame, f.typ, f.status
		f.mu.Unlock()
		if f.token != "" && r.Header.Get("Authorization") != "Bearer "+f.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if status != 0 {
			http.Error(w, "x", status)
			return
		}
		if frame == nil {
			http.Error(w, "no frame yet", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", typ)
		w.Write(frame)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeTap) set(frame []byte, typ string, status int) {
	f.mu.Lock()
	f.frame, f.typ, f.status = frame, typ, status
	f.mu.Unlock()
}

func pngOf(t testing.TB, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func jpegOf(t testing.TB, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	img.Set(1, 1, c)
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func recv(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	select {
	case f, ok := <-ch:
		if !ok {
			t.Fatal("channel closed")
		}
		return f
	case <-time.After(3 * time.Second):
		t.Fatal("no frame")
	}
	return nil
}

func TestTapCapturePNGBecomesJPEGAndJPEGPassesThrough(t *testing.T) {
	tap := newFakeTap(t, "")
	tap.set(pngOf(t, 80, 60, color.RGBA{200, 10, 10, 255}), "image/png", 0)
	c, err := TapCapture(TapOptions{URL: tap.srv.URL, FPS: 6})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	ch, err := c(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f := recv(t, ch)
	img, err := jpeg.Decode(bytes.NewReader(f))
	if err != nil || img.Bounds().Dx() != 80 || img.Bounds().Dy() != 60 {
		t.Fatalf("not a JPEG of the screenshot: %v", err)
	}
	// a tap started with -jpeg: passed through byte for byte; a changed frame is sent, an unchanged one is not
	j := jpegOf(t, color.RGBA{9, 9, 9, 255})
	tap.set(j, "image/jpeg", 0)
	if got := recv(t, ch); !bytes.Equal(got, j) {
		t.Fatal("jpeg not passed through")
	}
	select {
	case <-ch:
		t.Fatal("unchanged frame sent again")
	case <-time.After(400 * time.Millisecond):
	}
}

func TestTapCaptureStartErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		token, opt string
		status     int
		want       string
	}{
		"wrong token": {token: "right", opt: "wrong", want: "token"},
		"no frames":   {status: 404, want: "-fps 0"},
		"server err":  {status: 500, want: "500"},
	} {
		t.Run(name, func(t *testing.T) {
			tap := newFakeTap(t, tc.token)
			tap.set(jpegOf(t, color.Black), "image/jpeg", tc.status)
			c, _ := TapCapture(TapOptions{URL: tap.srv.URL, Token: tc.opt})
			before := tap.hits.Load()
			if _, err := c(t.Context()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err %v, want %q", err, tc.want)
			}
			if tap.hits.Load()-before != 1 {
				t.Fatalf("retried a failure that will not heal: %d requests", tap.hits.Load()-before)
			}
		})
	}
	// tap down: an error naming no host, not a hang
	tap := newFakeTap(t, "")
	url := tap.srv.URL
	tap.srv.Close()
	c, _ := TapCapture(TapOptions{URL: url})
	if _, err := c(t.Context()); err == nil || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("tap down: %v", err)
	}
	// a frame that is neither
	tap = newFakeTap(t, "")
	tap.set([]byte("not an image"), "image/png", 0)
	c, _ = TapCapture(TapOptions{URL: tap.srv.URL})
	if _, err := c(t.Context()); err == nil {
		t.Fatal("garbage accepted")
	}
	for _, o := range []TapOptions{{URL: ""}, {URL: "ftp://x"}, {URL: "http://x", FPS: 7}, {URL: "http://x", FPS: -1}} {
		if _, err := TapCapture(o); err == nil {
			t.Errorf("accepted %+v", o)
		}
	}
}

func TestTapCaptureWaitsForFirstFrameOnly(t *testing.T) {
	tap := newFakeTap(t, "tok") // no frame yet: 503
	c, _ := TapCapture(TapOptions{URL: tap.srv.URL, Token: "tok", Start: 2 * time.Second})
	go func() { time.Sleep(300 * time.Millisecond); tap.set(jpegOf(t, color.Black), "image/jpeg", 0) }()
	ch, err := c(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	recv(t, ch)
	// never gets a frame: gives up after Start
	tap2 := newFakeTap(t, "")
	c, _ = TapCapture(TapOptions{URL: tap2.srv.URL, Start: 300 * time.Millisecond})
	if _, err := c(t.Context()); err == nil {
		t.Fatal("no error without a frame")
	}
}

func TestTapCaptureStopsOnContextAndOnTapLoss(t *testing.T) {
	tap := newFakeTap(t, "")
	tap.set(jpegOf(t, color.Black), "image/jpeg", 0)
	c, _ := TapCapture(TapOptions{URL: tap.srv.URL, FPS: 6})

	cctx, stop := context.WithCancel(t.Context())
	ch, _ := c(cctx)
	recv(t, ch)
	stop()
	waitClosed(t, ch)
	n := tap.hits.Load()
	time.Sleep(400 * time.Millisecond)
	if tap.hits.Load() != n {
		t.Fatal("still polling after the context ended")
	}

	// the tap goes away mid-stream: the channel closes after three misses, without a tight loop
	ch, _ = c(t.Context())
	recv(t, ch)
	tap.set(nil, "", 503)
	start := tap.hits.Load()
	waitClosed(t, ch)
	if miss := tap.hits.Load() - start; miss > 6 {
		t.Fatalf("%d requests while the tap was down", miss)
	}
}

func TestTapVideoOnDemandViewerCapAndReconnect(t *testing.T) {
	tap := newFakeTap(t, "tok")
	tap.set(pngOf(t, 64, 48, color.RGBA{1, 2, 3, 255}), "image/png", 0)
	c, _ := TapCapture(TapOptions{URL: tap.srv.URL, Token: "tok", FPS: 6})
	srv, v, _ := tapVideoServer(t, c, VideoOptions{Linger: 80 * time.Millisecond})

	time.Sleep(250 * time.Millisecond)
	if tap.hits.Load() != 0 || v.Running() {
		t.Fatal("the tap was polled with nobody watching")
	}
	var resps []*http.Response
	for i := 0; i < 3; i++ {
		r := mustGet(t, srv.URL+"/stream")
		if r.StatusCode != 200 {
			t.Fatalf("viewer %d: %d", i, r.StatusCode)
		}
		resps = append(resps, r)
	}
	if r := mustGet(t, srv.URL+"/stream"); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("fourth viewer: %d", r.StatusCode)
	}
	if v.Starts() != 1 {
		t.Fatalf("starts %d: the capture is shared", v.Starts())
	}
	ctype, body := readPart(t, bufio.NewReader(resps[0].Body))
	if _, err := jpeg.Decode(bytes.NewReader(body)); ctype != "image/jpeg" || err != nil {
		t.Fatalf("part %q %v", ctype, err)
	}
	// the tap dies: the viewers' streams end (the page reconnects), the next viewer starts a fresh capture
	tap.set(nil, "", 503)
	waitFor(t, "stream ended", func() bool { return !v.Running() })
	for _, r := range resps {
		r.Body.Close()
	}
	tap.set(pngOf(t, 64, 48, color.RGBA{9, 9, 9, 255}), "image/png", 0)
	r := mustGet(t, srv.URL+"/stream")
	defer r.Body.Close()
	waitFor(t, "restart", func() bool { return v.Starts() == 2 && v.Running() })
	readPart(t, bufio.NewReader(r.Body))
	// last viewer gone: polling stops
	r.Body.Close()
	waitFor(t, "stopped", func() bool { return !v.Running() })
	n := tap.hits.Load()
	time.Sleep(400 * time.Millisecond)
	if tap.hits.Load() != n {
		t.Fatal("tap still polled with no viewer")
	}
}

func TestTapVideoFrameCapAndAuthFailure(t *testing.T) {
	tap := newFakeTap(t, "tok")
	rnd := image.NewRGBA(image.Rect(0, 0, 300, 300))
	rand.New(rand.NewSource(1)).Read(rnd.Pix)
	var pb bytes.Buffer
	png.Encode(&pb, rnd)
	tap.set(pb.Bytes(), "image/png", 0)
	c, _ := TapCapture(TapOptions{URL: tap.srv.URL, Token: "tok", Quality: 2})
	srv, v, _ := tapVideoServer(t, c, VideoOptions{MaxFrame: 20 << 10})
	r := mustGet(t, srv.URL+"/shot.jpg")
	if r.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("an oversize frame was served: %d", r.StatusCode)
	}
	waitFor(t, "idle", func() bool { return v.Viewers() == 0 })

	bad, _ := TapCapture(TapOptions{URL: tap.srv.URL, Token: "nope"})
	srv2, _, _ := tapVideoServer(t, bad, VideoOptions{})
	if r := mustGet(t, srv2.URL+"/stream"); r.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("bad tap token: %d", r.StatusCode)
	}
}

func TestTapVideoTwoInstancesIndependent(t *testing.T) {
	a, b := newFakeTap(t, ""), newFakeTap(t, "")
	a.set(pngOf(t, 32, 32, color.RGBA{255, 0, 0, 255}), "image/png", 0)
	b.set(pngOf(t, 32, 32, color.RGBA{0, 0, 255, 255}), "image/png", 0)
	ca, _ := TapCapture(TapOptions{URL: a.srv.URL})
	cb, _ := TapCapture(TapOptions{URL: b.srv.URL})
	sa, _, _ := tapVideoServer(t, ca, VideoOptions{})
	_, vb, _ := tapVideoServer(t, cb, VideoOptions{})
	ra := mustGet(t, sa.URL+"/stream")
	defer ra.Body.Close()
	_, body := readPart(t, bufio.NewReader(ra.Body))
	img, _ := jpeg.Decode(bytes.NewReader(body))
	if r, _, bl, _ := img.At(5, 5).RGBA(); r>>8 < 200 || bl>>8 > 60 {
		t.Fatal("instance a does not show its own tap")
	}
	if b.hits.Load() != 0 || vb.Running() {
		t.Fatal("instance b woke up for a viewer of a")
	}
}

// TestTapFrameCost reports the offline numbers for the task result: bytes per frame and CPU of one stream on a
// synthetic 800x600 picture (a gradient with noise; a real d2d screenshot is U until testerrt measures it live).
func TestTapFrameCost(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	rng := rand.New(rand.NewSource(2))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			n := uint8(rng.Intn(24))
			img.Set(x, y, color.RGBA{uint8(x/4) + n, uint8(y/3) + n, uint8((x + y) / 8), 255})
		}
	}
	var pb bytes.Buffer
	png.Encode(&pb, img)
	o := TapOptions{Quality: 5}
	const n = 20
	t0 := time.Now()
	var out []byte
	for i := 0; i < n; i++ {
		var err error
		if out, err = o.toJPEG(pb.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	per := time.Since(t0) / n
	t.Logf("800x600: PNG %d B -> JPEG %d B (q%d); convert %v per frame, so one stream at %d fps is about %.0f%% of one core",
		pb.Len(), len(out), o.jpegQuality(), per, DefaultTapFPS, 100*per.Seconds()*DefaultTapFPS)
	if len(out) > 512<<10 {
		t.Fatalf("a frame exceeds the 512 KiB cap: %d", len(out))
	}
}

func tapVideoServer(t *testing.T, c CaptureFunc, o VideoOptions) (*httptest.Server, *Video, *Hub) {
	t.Helper()
	r := newRig(t, "sorceress")
	o.Capture = c
	v, err := NewVideo(o)
	if err != nil {
		t.Fatal(err)
	}
	r.sim.Hub.SetVideo(v)
	m := http.NewServeMux()
	for _, p := range append(append([]string{}, ViewRoutes...), VideoRoutes...) {
		m.Handle(p, r.sim.Hub.ViewHandler())
	}
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	return srv, v, r.sim.Hub
}

func waitClosed(t *testing.T, ch <-chan []byte) {
	t.Helper()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-time.After(3 * time.Second):
			t.Fatal("channel not closed")
		}
	}
}
