package main

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
