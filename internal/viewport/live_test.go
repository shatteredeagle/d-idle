package viewport

import (
	"bytes"
	"context"
	"image/jpeg"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// Needs an X display with a dummy client window (xclock, xeyes, ...); never run against a game:
//
//	D2VP_DISPLAY=:199 D2VP_WINDOW=<id> go test -run TestLiveX11 -v -count=1 ./internal/viewport
//
// D2VP_SECS (default 8) is the length of each CPU measurement.
func TestLiveX11(t *testing.T) {
	disp, win := os.Getenv("D2VP_DISPLAY"), os.Getenv("D2VP_WINDOW")
	if disp == "" || win == "" {
		t.Skip("D2VP_DISPLAY / D2VP_WINDOW not set")
	}
	secs := 8
	if v := os.Getenv("D2VP_SECS"); v != "" {
		secs, _ = strconv.Atoi(v)
	}
	caps := map[string]Capturer{"import": ImportCapturer{}, "ffmpeg": FFmpegCapturer{}}
	for name, c := range caps {
		t.Run(name+"/limits", func(t *testing.T) {
			s := New(Config{Capturer: c, FPS: 5, MaxW: 100, MaxH: 100,
				Target: func() (Spec, error) { return Spec{Display: disp, Window: win}, nil }})
			defer s.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			f, err := s.Frame(ctx, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(f.Data))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Width > 100 || cfg.Height > 100 {
				t.Fatalf("size limit ignored: %dx%d", cfg.Width, cfg.Height)
			}
			t.Logf("%s frame %dx%d, %d bytes", name, cfg.Width, cfg.Height, len(f.Data))
		})
		for _, fps := range []float64{4, 10} {
			t.Run(name+"/cpu", func(t *testing.T) {
				s := New(Config{Capturer: c, FPS: fps, Quality: 65,
					Target: func() (Spec, error) { return Spec{Display: disp, Window: win}, nil }})
				done := s.Watch()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				if _, err := s.Frame(ctx, time.Second); err != nil { // warm-up: start the process
					t.Fatal(err)
				}
				cancel()
				c0, w0, seq0 := cpu(), time.Now(), s.Status().Seq
				time.Sleep(time.Duration(secs) * time.Second)
				seq1, w1 := s.Status().Seq, time.Now()
				done()
				s.Close()
				time.Sleep(200 * time.Millisecond) // reap children so their rusage counts
				c1 := cpu()
				n := float64(seq1 - seq0)
				t.Logf("BENCH %s fps=%g: %.1f frames/s, CPU %.1f%% of a core, %.1f ms CPU per frame",
					name, fps, n/w1.Sub(w0).Seconds(), (c1-c0)/w1.Sub(w0).Seconds()*100, (c1-c0)/n*1000)
			})
		}
	}
}

// cpu is process plus reaped-children CPU seconds.
func cpu() float64 {
	var a, b syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &a)
	syscall.Getrusage(syscall.RUSAGE_CHILDREN, &b)
	tv := func(t syscall.Timeval) float64 { return float64(t.Sec) + float64(t.Usec)/1e6 }
	return tv(a.Utime) + tv(a.Stime) + tv(b.Utime) + tv(b.Stime)
}
