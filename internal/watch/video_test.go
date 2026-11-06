package watch

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSrc is a frame source driven by the test: every capture start gets its own channel.
type fakeSrc struct {
	mu     sync.Mutex
	starts int
	live   int
	chans  []chan []byte
	err    error
}

func (f *fakeSrc) capture(ctx context.Context) (<-chan []byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	ch := make(chan []byte, 8)
	f.starts++
	f.live++
	f.chans = append(f.chans, ch)
	go func() { <-ctx.Done(); f.mu.Lock(); f.live--; f.mu.Unlock() }()
	return ch, nil
}
func (f *fakeSrc) lastChan() chan []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.chans[len(f.chans)-1]
}
func (f *fakeSrc) liveNow() int { f.mu.Lock(); defer f.mu.Unlock(); return f.live }

func mustGet(t *testing.T, u string) *http.Response {
	t.Helper()
	r, err := http.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func jpg(tag byte) []byte { return []byte{0xFF, 0xD8, tag, tag, 0xFF, 0xD9} }

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for i := 0; i < 400; i++ {
		if ok() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func videoServer(t *testing.T, token string, src *fakeSrc, o VideoOptions) (*httptest.Server, *Video, *Hub) {
	t.Helper()
	r := newRig(t, "sorceress")
	o.Capture = src.capture
	v, err := NewVideo(o)
	if err != nil {
		t.Fatal(err)
	}
	r.sim.Hub.SetVideo(v)
	m := http.NewServeMux()
	for _, p := range append(append([]string{}, ViewRoutes...), VideoRoutes...) {
		m.Handle(p, r.sim.Hub.ViewHandler())
	}
	srv := httptest.NewServer(tokenGate(token, m))
	t.Cleanup(srv.Close)
	return srv, v, r.sim.Hub
}

func TestVideoStreamFramingAuthAndOnDemand(t *testing.T) {
	src := &fakeSrc{}
	srv, v, _ := videoServer(t, "tok", src, VideoOptions{Linger: 50 * time.Millisecond})

	if resp := mustGet(t, srv.URL+"/stream"); resp.StatusCode != 401 {
		t.Fatalf("no token: %d", resp.StatusCode)
	}
	if v.Running() || src.starts != 0 {
		t.Fatal("capture started without a viewer")
	}
	resp, err := http.Get(srv.URL + "/stream?t=tok")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("stream: %v %v", resp, err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mt != "multipart/x-mixed-replace" || !strings.Contains(resp.Header.Get("Content-Type"), "boundary=") {
		t.Fatalf("content type %q", resp.Header.Get("Content-Type"))
	}
	waitFor(t, "capture", func() bool { return v.Running() && len(src.chans) == 1 })
	src.lastChan() <- jpg(1)
	br := bufio.NewReader(resp.Body)
	ctype, body := readPart(t, br)
	if ctype != "image/jpeg" || !bytes.Equal(body, jpg(1)) {
		t.Fatalf("part %q %v", ctype, body)
	}
	src.lastChan() <- jpg(2)
	if _, body = readPart(t, br); !bytes.Equal(body, jpg(2)) {
		t.Fatalf("second part %v", body)
	}

	// the last viewer leaves: the capture stops after the linger time, not at once
	resp.Body.Close()
	waitFor(t, "viewer gone", func() bool { return v.Viewers() == 0 })
	waitFor(t, "capture stopped", func() bool { return !v.Running() && src.liveNow() == 0 })
	if src.starts != 1 {
		t.Fatalf("starts %d", src.starts)
	}
}

func TestVideoLingerKeepsCaptureForReconnect(t *testing.T) {
	src := &fakeSrc{}
	srv, v, _ := videoServer(t, "", src, VideoOptions{Linger: 300 * time.Millisecond})
	r1 := mustGet(t, srv.URL+"/stream")
	waitFor(t, "start", func() bool { return v.Running() })
	r1.Body.Close()
	waitFor(t, "left", func() bool { return v.Viewers() == 0 })
	r2 := mustGet(t, srv.URL+"/stream")
	defer r2.Body.Close()
	waitFor(t, "rejoin", func() bool { return v.Viewers() == 1 })
	time.Sleep(450 * time.Millisecond)
	if !v.Running() || v.Starts() != 1 {
		t.Fatalf("running %v starts %d: a rejoin within the linger time must keep the same capture", v.Running(), v.Starts())
	}
}

func TestVideoViewerCapAndSharedCapture(t *testing.T) {
	src := &fakeSrc{}
	srv, v, _ := videoServer(t, "", src, VideoOptions{MaxViewers: 2, Linger: 20 * time.Millisecond})
	var open []*http.Response
	for i := 0; i < 2; i++ {
		r, err := http.Get(srv.URL + "/stream")
		if err != nil || r.StatusCode != 200 {
			t.Fatalf("viewer %d: %v", i, err)
		}
		open = append(open, r)
	}
	waitFor(t, "two viewers", func() bool { return v.Viewers() == 2 })
	if v.Starts() != 1 {
		t.Fatalf("two viewers, %d captures", v.Starts())
	}
	if r := mustGet(t, srv.URL+"/stream"); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("third viewer: %d", r.StatusCode)
	}
	if r := mustGet(t, srv.URL+"/shot.jpg"); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("shot over the cap: %d", r.StatusCode)
	}
	// both viewers get the same frame
	src.lastChan() <- jpg(7)
	for _, r := range open {
		br := bufio.NewReader(r.Body)
		var got bytes.Buffer
		for !bytes.Contains(got.Bytes(), jpg(7)) {
			line, err := br.ReadBytes('\n')
			if err != nil {
				t.Fatal(err)
			}
			got.Write(line)
		}
		r.Body.Close()
	}
}

func TestVideoShotAndCaptureFailure(t *testing.T) {
	src := &fakeSrc{}
	srv, v, h := videoServer(t, "", src, VideoOptions{Linger: 20 * time.Millisecond})
	go func() {
		waitFor(t, "start", func() bool { return v.Running() && len(src.chans) == 1 })
		src.lastChan() <- jpg(3)
	}()
	resp, err := http.Get(srv.URL + "/shot.jpg")
	if err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("shot: %v %v", resp, err)
	}
	if b, _ := io.ReadAll(resp.Body); !bytes.Equal(b, jpg(3)) {
		t.Fatalf("shot body %v", b)
	}
	if h.Config().FrameURL != "/stream" {
		t.Fatalf("config frame url %q", h.Config().FrameURL)
	}
	waitFor(t, "idle", func() bool { return !v.Running() })
	src.err = errors.New("no display")
	if r := mustGet(t, srv.URL+"/stream"); r.StatusCode != 503 {
		t.Fatalf("capture failure: %d", r.StatusCode)
	}
	if v.Viewers() != 0 || v.Running() {
		t.Fatal("a failed start must leave nothing behind")
	}
}

func TestVideoCaptureEndsClosesStreamAndRestarts(t *testing.T) {
	src := &fakeSrc{}
	srv, v, _ := videoServer(t, "", src, VideoOptions{})
	resp := mustGet(t, srv.URL+"/stream")
	waitFor(t, "start", func() bool { return v.Running() && len(src.chans) == 1 })
	close(src.lastChan()) // ffmpeg died
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("stream should end cleanly: %v", err)
	}
	waitFor(t, "cleared", func() bool { return !v.Running() })
	r2 := mustGet(t, srv.URL+"/stream")
	defer r2.Body.Close()
	waitFor(t, "restart", func() bool { return v.Starts() == 2 && v.Running() })
}

func TestVideoOversizeFrameDropped(t *testing.T) {
	src := &fakeSrc{}
	srv, v, _ := videoServer(t, "", src, VideoOptions{MaxFrame: 10})
	resp := mustGet(t, srv.URL+"/stream")
	defer resp.Body.Close()
	waitFor(t, "start", func() bool { return v.Running() && len(src.chans) == 1 })
	src.lastChan() <- append(jpg(1), make([]byte, 20)...)
	src.lastChan() <- jpg(2)
	br := bufio.NewReader(resp.Body)
	var got bytes.Buffer
	for !bytes.Contains(got.Bytes(), jpg(2)) {
		line, err := br.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		got.Write(line)
	}
	if bytes.Contains(got.Bytes(), []byte{0xFF, 0xD8, 1, 1}) {
		t.Fatal("oversize frame was served")
	}
}

// Two instances: each Hub has its own Video and its own capture; neither sees the other's frames.
func TestVideoTwoInstancesIndependent(t *testing.T) {
	a, b := &fakeSrc{}, &fakeSrc{}
	sa, va, _ := videoServer(t, "", a, VideoOptions{})
	sb, vb, _ := videoServer(t, "", b, VideoOptions{})
	ra := mustGet(t, sa.URL+"/stream")
	defer ra.Body.Close()
	waitFor(t, "a", func() bool { return va.Running() })
	if vb.Running() || b.starts != 0 {
		t.Fatal("instance b started by a viewer of a")
	}
	rb := mustGet(t, sb.URL+"/stream")
	defer rb.Body.Close()
	waitFor(t, "b", func() bool { return vb.Running() && len(b.chans) == 1 })
	b.lastChan() <- jpg(9)
	a.lastChan() <- jpg(8)
	for want, r := range map[byte]*http.Response{8: ra, 9: rb} {
		br := bufio.NewReader(r.Body)
		var got bytes.Buffer
		for !bytes.Contains(got.Bytes(), jpg(want)) {
			line, err := br.ReadBytes('\n')
			if err != nil {
				t.Fatal(err)
			}
			got.Write(line)
		}
		if bytes.Contains(got.Bytes(), jpg(17-want)) {
			t.Fatal("frame crossed instances")
		}
	}
}

// With video off the routes do not exist, the config offers no stream, and the viewer stays read-only.
func TestVideoOffByDefault(t *testing.T) {
	r := newRig(t, "sorceress")
	for _, p := range []string{"/stream", "/shot.jpg"} {
		if w := r.get(p, nil); w.Code != 404 {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	if r.sim.Hub.Config().FrameURL != "" {
		t.Error("frame url set without video")
	}
	w := httptest.NewRecorder()
	r.view.ServeHTTP(w, httptest.NewRequest("POST", "/stream", nil))
	if w.Code != 405 {
		t.Errorf("POST /stream: %d", w.Code)
	}
}

func TestSplitJPEG(t *testing.T) {
	// a stuffed FF00 inside the entropy data is not a marker; garbage before the first SOI is skipped
	in := append([]byte{1, 2, 3}, []byte{0xFF, 0xD8, 0xFF, 0x00, 5, 0xFF, 0xD9, 0xFF, 0xD8, 6, 0xFF, 0xD9, 0xFF, 0xD8, 7}...)
	var got [][]byte
	splitJPEG(bytes.NewReader(in), func(f []byte) bool { got = append(got, f); return true })
	if len(got) != 2 || !bytes.Equal(got[0], []byte{0xFF, 0xD8, 0xFF, 0x00, 5, 0xFF, 0xD9}) || !bytes.Equal(got[1], []byte{0xFF, 0xD8, 6, 0xFF, 0xD9}) {
		t.Fatalf("frames %v", got)
	}
}

func TestExecCaptureChoosesToolAndNamesWhatIsMissing(t *testing.T) {
	has := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, x := range names {
				if x == n {
					return "/x/" + n, nil
				}
			}
			return "", exec.ErrNotFound
		}
	}
	if _, err := ExecCapture(CaptureOptions{Display: ":105", LookPath: has()}); err == nil || !strings.Contains(err.Error(), "ffmpeg") {
		t.Fatalf("nothing installed: %v", err)
	}
	if _, err := ExecCapture(CaptureOptions{LookPath: has("ffmpeg")}); err == nil {
		t.Fatal("no display must be refused")
	}
	var name string
	var args []string
	cmdFn := func(ctx context.Context, n string, a ...string) *exec.Cmd {
		name, args = n, a
		return exec.CommandContext(ctx, "sh", "-c", "printf '\\377\\330ab\\377\\331'")
	}
	cf, err := ExecCapture(CaptureOptions{Display: ":105", FPS: 99, Quality: 7, LookPath: has("ffmpeg", "import"), Command: cmdFn})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := cf(ctx)
	if err != nil {
		t.Fatal(err)
	}
	f := <-ch
	line := strings.Join(args, " ")
	if name != "ffmpeg" || !strings.Contains(line, "-i :105") || !strings.Contains(line, "-framerate 15") || !strings.Contains(line, "-q:v 7") || len(f) != 6 {
		t.Fatalf("%s %s frame %v", name, line, f)
	}
	// the no-ffmpeg path
	cf, _ = ExecCapture(CaptureOptions{Display: ":105", FPS: 8, LookPath: has("import"), Command: cmdFn})
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	ch, _ = cf(ctx2)
	<-ch
	if name != "import" || !strings.Contains(strings.Join(args, " "), "-display :105") {
		t.Fatalf("fallback %s %v", name, args)
	}
}

// readPart reads one boundary-delimited part by its Content-Length, as a browser does.
func readPart(t *testing.T, br *bufio.Reader) (ctype string, body []byte) {
	t.Helper()
	n := -1
	line, err := br.ReadString('\n')
	if line == "\r\n" { // the CRLF that ends the previous part's data
		line, err = br.ReadString('\n')
	}
	if err != nil || !strings.HasPrefix(line, "--") {
		t.Fatalf("boundary line %q %v", line, err)
	}
	for {
		line, err = br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if line == "\r\n" {
			break
		}
		k, v, _ := strings.Cut(strings.TrimSpace(line), ": ")
		switch k {
		case "Content-Type":
			ctype = v
		case "Content-Length":
			n, _ = strconv.Atoi(v)
		}
	}
	if n < 0 {
		t.Fatal("no Content-Length")
	}
	body = make([]byte, n)
	if _, err := io.ReadFull(br, body); err != nil {
		t.Fatal(err)
	}
	return
}

// Serve mounts the video routes only with a Video, on the viewer listener, behind the token.
func TestServeMountsVideoOnlyWhenOn(t *testing.T) {
	for _, on := range []bool{false, true} {
		r := newRig(t, "sorceress")
		o := ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: "tok"}
		if on {
			o.Video, _ = NewVideo(VideoOptions{Capture: (&fakeSrc{}).capture, Linger: 10 * time.Millisecond})
		}
		ctx, cancel := context.WithCancel(context.Background())
		sv, err := r.sim.Hub.Serve(ctx, o)
		if err != nil {
			t.Fatal(err)
		}
		u := "http://" + sv.View.String() + "/shot.jpg"
		if resp := mustGet(t, u); resp.StatusCode != 401 {
			t.Errorf("on=%v no token: %d", on, resp.StatusCode)
		}
		want := 404
		if on {
			want = 503 // a capture with no frames yet
		}
		req, _ := http.NewRequestWithContext(ctx, "GET", u, nil)
		req.Header.Set("Authorization", "Bearer tok")
		ctx2, c2 := context.WithTimeout(ctx, 200*time.Millisecond)
		req = req.WithContext(ctx2)
		resp, err := http.DefaultClient.Do(req)
		c2()
		if on && err != nil {
			cancel()
			continue // the shot waits for a frame; the timeout is the proof that the route exists
		}
		if err != nil || resp.StatusCode != want {
			t.Errorf("on=%v: %v %v", on, resp, err)
		}
		cancel()
	}
}
