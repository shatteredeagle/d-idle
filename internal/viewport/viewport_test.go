package viewport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/input"
)

type fakeCap struct {
	persistent bool
	delay      time.Duration
	opens      atomic.Int32
	nexts      atomic.Int32
	cur, peak  atomic.Int32
	fail       atomic.Bool
}

func (f *fakeCap) Persistent() bool { return f.persistent }
func (f *fakeCap) Open(ctx context.Context, s Spec) (Session, error) {
	f.opens.Add(1)
	return &fakeSess{f: f}, nil
}

type fakeSess struct{ f *fakeCap }

func (s *fakeSess) Close() error { return nil }
func (s *fakeSess) Next(ctx context.Context) (Frame, error) {
	f := s.f
	c := f.cur.Add(1)
	defer f.cur.Add(-1)
	for {
		p := f.peak.Load()
		if c <= p || f.peak.CompareAndSwap(p, c) {
			break
		}
	}
	time.Sleep(f.delay)
	n := f.nexts.Add(1)
	if f.fail.Load() {
		return Frame{}, errors.New("boom")
	}
	return Frame{Data: []byte{0xFF, 0xD8, byte(n), 0xFF, 0xD9}, Mime: "image/jpeg", At: time.Now()}, nil
}

func target() (Spec, error) { return Spec{Display: ":9", Window: "0x1"}, nil }

func TestLazyNoCaptureWithoutViewers(t *testing.T) {
	fc := &fakeCap{}
	s := New(Config{Capturer: fc, Target: target, FPS: 50, Idle: 100 * time.Millisecond})
	defer s.Close()
	time.Sleep(200 * time.Millisecond)
	if fc.nexts.Load() != 0 {
		t.Fatal("captured with nobody watching")
	}
	if _, err := s.Frame(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond) // idle expires
	n := fc.nexts.Load()
	time.Sleep(300 * time.Millisecond)
	if fc.nexts.Load() != n || s.Status().Running {
		t.Fatalf("capture continued after idle: %d -> %d running=%v", n, fc.nexts.Load(), s.Status().Running)
	}
}

func TestFPSCapAndSharedFrames(t *testing.T) {
	fc := &fakeCap{}
	s := New(Config{Capturer: fc, Target: target, FPS: 10})
	defer s.Close()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ { // many viewers, one capture
		wg.Add(1)
		go func() {
			defer wg.Done()
			done := s.Watch()
			defer done()
			var seq uint64
			end := time.Now().Add(1 * time.Second)
			for time.Now().Before(end) {
				ctx, c := context.WithTimeout(context.Background(), time.Second)
				f, err := s.Next(ctx, seq)
				c()
				if err == nil {
					seq = f.Seq
				}
			}
		}()
	}
	wg.Wait()
	if n := fc.nexts.Load(); n < 7 || n > 13 {
		t.Fatalf("captures in 1 s at 10 fps cap with 8 viewers: %d", n)
	}
	if fc.peak.Load() != 1 {
		t.Fatalf("more than one capture in flight: %d", fc.peak.Load())
	}
}

func TestSlowCaptureNeverStacks(t *testing.T) {
	fc := &fakeCap{delay: 120 * time.Millisecond}
	s := New(Config{Capturer: fc, Target: target, FPS: 50})
	defer s.Close()
	done := s.Watch()
	time.Sleep(600 * time.Millisecond)
	done()
	if fc.peak.Load() != 1 {
		t.Fatalf("peak in flight %d", fc.peak.Load())
	}
	if n := fc.nexts.Load(); n > 6 {
		t.Fatalf("slow capturer ran back to back too often: %d", n)
	}
}

func TestGlobalConcurrencyCap(t *testing.T) {
	fc := &fakeCap{delay: 60 * time.Millisecond}
	lim := NewLimiter(2)
	var ss []*Stream
	for i := 0; i < 6; i++ {
		s := New(Config{Capturer: fc, Target: target, FPS: 100, Limiter: lim})
		ss = append(ss, s)
		defer s.Close()
		defer s.Watch()()
	}
	time.Sleep(500 * time.Millisecond)
	if p := fc.peak.Load(); p != 2 {
		t.Fatalf("peak concurrent captures %d, want 2", p)
	}
	for _, s := range ss {
		if _, ok := s.Latest(); !ok {
			t.Fatal("a stream starved")
		}
	}
}

func TestPersistentHoldsSlotAndClosesWhenIdle(t *testing.T) {
	fc := &fakeCap{persistent: true, delay: 10 * time.Millisecond}
	lim := NewLimiter(1)
	s := New(Config{Capturer: fc, Target: target, FPS: 50, Limiter: lim, Idle: 50 * time.Millisecond})
	defer s.Close()
	if _, err := s.Frame(context.Background(), time.Second); err != nil {
		t.Fatal(err)
	}
	if lim.InUse() != 1 || fc.opens.Load() != 1 {
		t.Fatalf("slot %d opens %d", lim.InUse(), fc.opens.Load())
	}
	time.Sleep(300 * time.Millisecond)
	if lim.InUse() != 0 || s.Status().Running {
		t.Fatal("session kept open with nobody watching")
	}
	if _, err := s.Frame(context.Background(), time.Millisecond); err != nil || fc.opens.Load() != 2 {
		t.Fatalf("reopen: %v opens %d", err, fc.opens.Load())
	}
}

func TestErrorsReachWaitersAndRecover(t *testing.T) {
	fc := &fakeCap{}
	fc.fail.Store(true)
	var seen atomic.Int32
	s := New(Config{Capturer: fc, Target: target, FPS: 20, OnError: func(error) { seen.Add(1) }})
	defer s.Close()
	if _, err := s.Frame(context.Background(), time.Second); err == nil || err.Error() != "boom" {
		t.Fatalf("want boom, got %v", err)
	}
	if seen.Load() == 0 {
		t.Fatal("OnError not called")
	}
	fc.fail.Store(false)
	ctx, c := context.WithTimeout(context.Background(), 3*time.Second)
	defer c()
	for {
		if _, err := s.Frame(ctx, time.Second); err == nil {
			break
		} else if ctx.Err() != nil {
			t.Fatal("no recovery")
		}
	}
}

func TestNoTarget(t *testing.T) {
	s := New(Config{Capturer: &fakeCap{}, Target: func() (Spec, error) { return Spec{}, ErrNoTarget }, FPS: 20})
	defer s.Close()
	if _, err := s.Frame(context.Background(), time.Second); !errors.Is(err, ErrNoTarget) {
		t.Fatalf("got %v", err)
	}
}

func TestHandler(t *testing.T) {
	fc := &fakeCap{}
	s := New(Config{Capturer: fc, Target: target, FPS: 20})
	defer s.Close()
	h := Handler(HandlerConfig{Stream: s, Scene: func() (Scene, bool) {
		return Scene{Player: [2]float64{100, 100}, Shapes: []WorldShape{
			{Kind: KindPoint, Pts: [][2]float64{{100, 100}}},
			{Kind: KindCircle, Pts: [][2]float64{{100, 100}}, Radius: 10},
			{Kind: KindPath, Pts: [][2]float64{{900, 100}, {901, 100}}},
		}}, true
	}})
	srv := httptest.NewServer(h)
	defer srv.Close()

	r, err := http.Get(srv.URL + "/frame.jpg")
	if err != nil || r.StatusCode != 200 || r.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("frame: %v %v", err, r)
	}
	b, _ := io.ReadAll(r.Body)
	if !bytes.HasPrefix(b, []byte{0xFF, 0xD8}) {
		t.Fatal("not a jpeg")
	}
	if r, _ := http.Post(srv.URL+"/frame.jpg", "", nil); r.StatusCode != 405 {
		t.Fatalf("POST %d", r.StatusCode)
	}
	if r, _ := http.Get(srv.URL + "/nope"); r.StatusCode != 404 {
		t.Fatalf("404 expected, got %d", r.StatusCode)
	}

	var o Overlay
	r, _ = http.Get(srv.URL + "/overlay")
	json.NewDecoder(r.Body).Decode(&o)
	if o.W != 800 || len(o.Shapes) != 3 {
		t.Fatalf("overlay %+v", o)
	}
	if p := o.Shapes[0].Pts[0]; p != [2]float64{input.Calibrated.PlayerX, input.Calibrated.PlayerY} || !o.Shapes[0].Visible {
		t.Fatalf("player point %v", p)
	}
	if e := o.Shapes[1]; e.Kind != "ellipse" || e.RX < 226 || e.RX > 227 || e.RY < 113 || e.RY > 114 {
		t.Fatalf("ellipse %+v", e)
	}
	if o.Shapes[2].Visible {
		t.Fatal("far path must not be visible")
	}

	// stream: two parts, then stop capturing after the client leaves
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/stream.mjpg", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 256)
	for bytes.Count(buf, []byte("--frame")) < 3 {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			t.Fatal(err)
		}
	}
	cancel()
	resp.Body.Close()
	time.Sleep(3500 * time.Millisecond)
	n := fc.nexts.Load()
	time.Sleep(300 * time.Millisecond)
	if fc.nexts.Load() != n {
		t.Fatal("capturing after the last viewer left")
	}
}

func TestProjectVP(t *testing.T) {
	o := Project(input.Calibrated, Scene{VP: &[2]int{16*(100-100) - 400, 8*200 - 284}, Shapes: []WorldShape{{Kind: KindPoint, Pts: [][2]float64{{100, 100}}}}})
	if p := o.Shapes[0].Pts[0]; p != [2]float64{400, 284} {
		t.Fatalf("%v", p)
	}
}

func TestJPEGSplitter(t *testing.T) {
	j1 := []byte{0xFF, 0xD8, 1, 2, 0xFF, 0x00, 3, 0xFF, 0xD9}
	j2 := []byte{0xFF, 0xD8, 9, 0xFF, 0xD9}
	data := append(append([]byte{7, 7}, j1...), j2...)
	sp := &jpegSplitter{r: &oneByte{b: data}}
	a, err := sp.Next()
	if err != nil || !bytes.Equal(a, j1) {
		t.Fatalf("%v %v", a, err)
	}
	b, err := sp.Next()
	if err != nil || !bytes.Equal(b, j2) {
		t.Fatalf("%v %v", b, err)
	}
	if _, err := sp.Next(); err == nil {
		t.Fatal("want end")
	}
}

type oneByte struct{ b []byte }

func (o *oneByte) Read(p []byte) (int, error) {
	if len(o.b) == 0 {
		return 0, io.EOF
	}
	p[0] = o.b[0]
	o.b = o.b[1:]
	return 1, nil
}

func TestFFmpegArgs(t *testing.T) {
	a := FFmpegCapturer{}.args(Spec{Display: ":101", Window: "0x400003", FPS: 10, MaxW: 640, Quality: 65})
	want := map[string]bool{"-window_id": false, "0x400003": false, ":101": false, "10": false, "-vf": false}
	for _, s := range a {
		if _, ok := want[s]; ok {
			want[s] = true
		}
	}
	for k, v := range want {
		if !v {
			t.Errorf("missing %s in %v", k, a)
		}
	}
}
