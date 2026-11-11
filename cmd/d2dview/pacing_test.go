package main

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDivisorKeepsEvenGameSpacing(t *testing.T) {
	f := &Frames{fps: 25, Backoff: 4, srcPeriod: time.Second / 60}
	// 25 fps on 60 Hz draws: every 3rd draw (20 fps, 50 game ms each), never an alternating 2/3 pattern
	if k := f.divisor(); k != 3 {
		t.Fatalf("divisor %d", k)
	}
	f.every = 3
	// encode got cheap enough for every 2nd draw: applied only when two renewals agree
	f.fps, f.costEMA = 30, 7*time.Millisecond
	if k := f.divisor(); k != 3 {
		t.Fatalf("changed on the first renewal: %d", k)
	}
	if k := f.divisor(); k != 2 {
		t.Fatalf("not changed on the second: %d", k)
	}
	// the encode budget wins over the fps target: 12 ms x 4 = 48 ms -> every 3rd draw
	f.every, f.pending, f.costEMA = 2, 0, 12*time.Millisecond
	f.divisor()
	if k := f.divisor(); k != 3 {
		t.Fatalf("budget ignored: %d", k)
	}
	// a 25 Hz source (old builds, lockstep render) at 25 fps: every frame
	g := &Frames{fps: 25, Backoff: 4, srcPeriod: 40 * time.Millisecond, costEMA: 9 * time.Millisecond}
	if k := g.divisor(); k != 1 {
		t.Fatalf("25 Hz source: %d", k)
	}
}

func TestRingMeasuresTheSourcePeriod(t *testing.T) {
	mux, _, _, view := rig(t)
	dir := t.TempDir()
	fr, err := NewFrames(view, mux, dir, 25)
	if err != nil {
		t.Fatal(err)
	}
	fr.Feed, fr.every = filepath.Join(dir, "frames.ring"), 3
	w, _ := os.OpenFile(fr.Feed, os.O_RDWR|os.O_CREATE, 0o600)
	defer w.Close()
	pix := testFrame(32, 24)
	for i := 0; i < 40; i++ {
		writeRing(w, 32, 24, pix, uint64(3*i), uint64(i*3*20)) // 3 draws of 20 ms between publishes
		if _, fresh, err := fr.grabRing(); !fresh || err != nil {
			t.Fatalf("frame %d: fresh=%v err=%v", i, fresh, err)
		}
		if _, fresh, _ := fr.grabRing(); fresh {
			t.Fatal("a header-only poll must not re-send")
		}
	}
	if p := fr.srcPeriod; p < 19*time.Millisecond || p > 21*time.Millisecond {
		t.Fatalf("source period %v, want 20 ms", p)
	}
	if c := fr.Current(); c.GameMS != 39*60 {
		t.Fatalf("game ms %d", c.GameMS)
	}
}

func TestStreamSendsEachFrameWhenPublished(t *testing.T) {
	mux, _, _, view := rig(t)
	fr, err := NewFrames(view, mux, t.TempDir(), 25)
	if err != nil {
		t.Fatal(err)
	}
	s := &server{frames: fr, mux: mux}
	srv := httptest.NewServer(http.HandlerFunc(s.stream))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	rd := bufio.NewReader(resp.Body)
	for i := 0; i < 5; i++ {
		t0 := time.Now()
		fr.publish([]byte("jpegdata"), "image/jpeg", 0, int64(100*i))
		var hdr strings.Builder
		for {
			l, err := rd.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if l == "\r\n" && hdr.Len() > 0 {
				break
			}
			hdr.WriteString(l)
		}
		rd.Discard(len("jpegdata") + 2)
		if d := time.Since(t0); d > 15*time.Millisecond {
			t.Errorf("part %d took %v after publish (the old loop slept up to 20 ms)", i, d)
		}
		if want := "X-Frame-Game-Ms: " + itoa64(int64(100*i)); !strings.Contains(hdr.String(), want) {
			t.Fatalf("part %d header %q lacks %q", i, hdr.String(), want)
		}
	}
}

func TestEventsWithoutFrames(t *testing.T) {
	// regression: /events counted stream parts on a nil Frames (a tap started with -fps 0)
	mux, _, _, view := rig(t)
	feed := NewFeed(view, mux, 50, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go feed.Run(ctx)
	s := &server{feed: feed, mux: mux}
	srv := httptest.NewServer(http.HandlerFunc(s.events))
	defer srv.Close()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if l, err := bufio.NewReader(resp.Body).ReadString('\n'); err != nil || !strings.HasPrefix(l, "data: ") {
		t.Fatalf("%q %v", l, err)
	}
}

func itoa64(n int64) string { return strconv.FormatInt(n, 10) }

func TestDegradedFlag(t *testing.T) {
	s := newFrameStats(time.Second)
	t0 := s.start
	for i := 0; i < 5; i++ {
		s.add("screenshot", 50*time.Millisecond, 0, 1000, 800, 600)
	}
	st, _ := s.roll(t0.Add(time.Second), FrameStats{FeedWanted: true})
	if !st.Degraded || !strings.HasPrefix(st.String(), "DEGRADED") {
		t.Fatalf("%+v", st)
	}
	for i := 0; i < 20; i++ {
		s.add("framefeed", time.Millisecond, 9*time.Millisecond, 1000, 800, 600)
	}
	if st, _ := s.roll(t0.Add(2*time.Second), FrameStats{FeedWanted: true}); st.Degraded {
		t.Fatalf("%+v", st)
	}
}

// okD2D answers every line with "ok 800 600": enough of a d2d for the framefeed lease.
func okD2D(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "up.sock")
	ln, err := net.Listen("unix", p)
	if err != nil {
		t.Fatal(err)
	}
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
					if _, err := rd.ReadString('\n'); err != nil {
						return
					}
					c.Write([]byte("ok 800 600\n"))
				}
			}()
		}
	}()
	return p
}

// TestPacingEndToEnd drives the whole tap path: a 60 Hz ring writer (as real-time d2d), the sampler with the
// sandbox flags (-fps 25, back-off 4, q72, 800x600) and one /stream.mjpg client stamping arrivals. Long (8 s); the
// numbers go to docs/viewer-fps.md section 7. Run: go test -run PacingEndToEnd -v ./cmd/d2dview
func TestPacingEndToEnd(t *testing.T) {
	if testing.Short() || raceOn {
		t.Skip("8 s pacing run; timing is meaningless under -race")
	}
	up := okD2D(t)
	mux := NewMux(up)
	dir := t.TempDir()
	mux.FeedPath = filepath.Join(dir, "frames.ring")
	view := filepath.Join(dir, "view.sock")
	ln, err := mux.Serve(view, false, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	fr, err := NewFrames(view, mux, dir, 25)
	if err != nil {
		t.Fatal(err)
	}
	fr.Feed = mux.FeedPath
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// the d2d side: draws at 60 Hz, publishes every `every`-th draw as framefeed would (every read from the lease line
	// is not visible here, so the writer follows the sampler's own choice)
	go func() {
		w, _ := os.OpenFile(fr.Feed, os.O_RDWR|os.O_CREATE, 0o600)
		defer w.Close()
		pix := testFrame(800, 600)
		tk := time.NewTicker(time.Second / 60)
		defer tk.Stop()
		for draw := uint64(1); ctx.Err() == nil; draw++ {
			<-tk.C
			every := uint64(max(1, fr.Stats.everyHint.Load()))
			if draw%every == 0 {
				pix[int(draw)%len(pix)]++ // a different picture each frame
				writeRing(w, 800, 600, pix, draw, draw*1000/60)
			}
		}
	}()
	go fr.Run(ctx)
	s := &server{frames: fr, mux: mux}
	srv := httptest.NewServer(http.HandlerFunc(s.stream))
	defer srv.Close()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	rd := bufio.NewReader(resp.Body)
	var at []time.Time
	var game []int64
	end := time.Now().Add(8 * time.Second)
	for time.Now().Before(end) {
		gms, n := int64(-1), 0
		for {
			l, err := rd.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if l == "\r\n" && n > 0 {
				break
			}
			if v, ok := strings.CutPrefix(l, "X-Frame-Game-Ms: "); ok {
				gms, _ = strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			}
			if v, ok := strings.CutPrefix(l, "Content-Length: "); ok {
				n, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
		rd.Discard(n + 2)
		at, game = append(at, time.Now()), append(game, gms)
	}
	at, game = at[len(at)/4:], game[len(game)/4:] // skip the start (divisor settling)
	var wall, gg []float64
	for i := 1; i < len(at); i++ {
		wall = append(wall, ms(at[i].Sub(at[i-1])))
		gg = append(gg, float64(game[i]-game[i-1]))
	}
	every := fr.Stats.everyHint.Load() // the sampler's own fields belong to its goroutine
	t.Logf("every=%d fps=%.1f", every, float64(len(wall))/at[len(at)-1].Sub(at[0]).Seconds())
	t.Logf("arrival gap ms: %s", summary(wall))
	t.Logf("game-time gap ms: %s", summary(gg))
	if every < 2 {
		t.Fatalf("divisor %d on a 60 Hz source at -fps 25", every)
	}
	for _, g := range gg {
		if int(g) != int(gg[0]) && int(g) != 2*int(gg[0]) { // a skipped frame doubles a gap, nothing in between
			t.Fatalf("uneven game-time spacing: %v", gg)
		}
	}
}

func summary(g []float64) string {
	s := append([]float64(nil), g...)
	sort.Float64s(s)
	n := len(s)
	q := func(p float64) float64 { return s[min(n-1, int(p*float64(n)))] }
	return fmt.Sprintf("n=%d p50=%.1f p90=%.1f p99=%.1f max=%.1f", n, q(.5), q(.9), q(.99), s[n-1])
}
