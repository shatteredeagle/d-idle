package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// testFrame is a busy 800x600 picture (gradients and noise-like stripes), opaque like d2d's frames.
func testFrame(w, h int) []byte {
	pix := make([]byte, 4*w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := 4 * (y*w + x)
			pix[i], pix[i+1], pix[i+2], pix[i+3] = uint8(x), uint8(y), uint8((x*7+y*13)^(x*y)), 0xff
		}
	}
	return pix
}

func TestRingRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "frames.ring")
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, _, err := readRing(f, nil); err == nil {
		t.Fatal("empty file must not parse")
	}
	a, b := testFrame(16, 8), make([]byte, 4*16*8)
	for i := range b {
		b[i] = 0x55
	}
	for i, pix := range [][]byte{a, b, a} {
		if err := writeRing(f, 16, 8, pix, uint64(100+i), uint64(40*i)); err != nil {
			t.Fatal(err)
		}
		h, got, err := readRing(f, nil)
		if err != nil || h.Seq != uint64(i+1) || h.Frame != uint64(100+i) || h.W != 16 || h.H != 8 || !bytes.Equal(got, pix) {
			t.Fatalf("frame %d: %+v %v equal=%v", i, h, err, bytes.Equal(got, pix))
		}
	}
	// a frame of another size restarts the sequence
	if err := writeRing(f, 8, 4, testFrame(8, 4), 1, 0); err != nil {
		t.Fatal(err)
	}
	if h, _, err := readRing(f, nil); err != nil || h.Seq != 1 || h.W != 8 {
		t.Fatalf("%+v %v", h, err)
	}
}

func TestRingRejectsBadHeaders(t *testing.T) {
	good := make([]byte, ringHdr)
	copy(good, ringMagic)
	good[4], good[8], good[12], good[16], good[20] = 1, 8, 4, 32, 1
	if _, err := parseRingHeader(good); err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(b []byte){
		"magic":   func(b []byte) { b[0] = 'X' },
		"version": func(b []byte) { b[4] = 2 },
		"format":  func(b []byte) { b[20] = 2 },
		"stride":  func(b []byte) { b[16] = 4 },
		"huge":    func(b []byte) { b[10] = 1 },
	} {
		b := bytes.Clone(good)
		edit(b)
		if _, err := parseRingHeader(b); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestFeedVerbOnlyForTheRing(t *testing.T) {
	const p = "/dev/shm/d2dview/frames.ring"
	for line, want := range map[string]bool{
		"framefeed " + p + " 1 3000":    true,
		"framefeed off":                 true,
		"framefeed " + p + " 0 3000":    false,
		"framefeed " + p + " 1 99":      false,
		"framefeed /etc/passwd 1 3000":  false,
		"framefeed " + p + " 1":         false,
		"framefeed " + p + " x 3000":    false,
		"screenshot " + p:               false,
		"framefeed " + p + " 1 3000 xx": false,
	} {
		if got := feedVerb(line, p); got != want {
			t.Errorf("feedVerb(%q) = %v", line, got)
		}
	}
}

func TestFeedVerbThroughTheViewerSocket(t *testing.T) {
	// Without FeedPath the verb is refused at the tap; with it, it reaches d2d (the fake d2d has no such verb, which is
	// exactly what an unpatched d2d answers: the sampler then falls back to screenshots).
	mux, _, _, view := rig(t)
	fr, err := NewFrames(view, mux, "/tmp/shots", 5)
	if err != nil {
		t.Fatal(err)
	}
	fr.Feed = "/tmp/shots/frames.ring"
	err = fr.feedLease(context.Background())
	if err == nil || fr.feedFail {
		t.Fatalf("refused at the tap expected, got %v fail=%v", err, fr.feedFail)
	}
	mux.FeedPath = fr.Feed
	if err := fr.feedLease(context.Background()); err == nil || !fr.feedFail {
		t.Fatalf("unknown verb from d2d expected, got %v fail=%v", err, fr.feedFail)
	}
}

func TestGrabRingEncodesNewFramesOnly(t *testing.T) {
	mux, _, _, view := rig(t)
	dir := t.TempDir()
	fr, err := NewFrames(view, mux, dir, 25)
	if err != nil {
		t.Fatal(err)
	}
	fr.Feed = filepath.Join(dir, "frames.ring")
	if _, fresh, err := fr.grabRing(); fresh || err != nil {
		t.Fatalf("no ring yet: fresh=%v err=%v", fresh, err)
	}
	w, err := os.OpenFile(fr.Feed, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	writeRing(w, 64, 48, testFrame(64, 48), 1, 40)
	mux.ticks.Store(7)
	if _, fresh, err := fr.grabRing(); !fresh || err != nil {
		t.Fatalf("fresh=%v err=%v", fresh, err)
	}
	c := fr.Current()
	if c.Type != "image/jpeg" || c.Ticks != 7 {
		t.Fatalf("%+v", c)
	}
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(c.Data)); err != nil || cfg.Width != 64 || cfg.Height != 48 {
		t.Fatalf("%+v %v", cfg, err)
	}
	if _, fresh, _ := fr.grabRing(); fresh {
		t.Fatal("the same ring frame twice")
	}
	fr.Half = true
	writeRing(w, 64, 48, testFrame(64, 48), 2, 80)
	fr.grabRing()
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(fr.Current().Data)); err != nil || cfg.Width != 32 || cfg.Height != 24 {
		t.Fatalf("half: %+v %v", cfg, err)
	}
	if st, ok := fr.Stats.roll(time.Now().Add(time.Minute), fr.knobs()); !ok || st.Frames != 2 || st.Source != "framefeed" || st.Width != 32 || !st.Half {
		t.Fatalf("%+v", st)
	}
}

func TestHalveAveragesBlocks(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{100, 0, 0, 255})
	src.Set(1, 0, color.RGBA{200, 0, 0, 255})
	src.Set(0, 1, color.RGBA{0, 0, 0, 255})
	src.Set(1, 1, color.RGBA{100, 0, 0, 255})
	d := halve(src)
	if d.Rect.Dx() != 2 || d.Rect.Dy() != 1 || d.Pix[0] != 100 || d.Pix[3] != 255 {
		t.Fatalf("%v %v", d.Rect, d.Pix)
	}
}

func TestRGBAOfKeepsOpaqueNRGBA(t *testing.T) {
	n := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for i := range n.Pix {
		n.Pix[i] = 0xff
	}
	if r := rgbaOf(n); &r.Pix[0] != &n.Pix[0] {
		t.Fatal("an opaque NRGBA should be reused without a copy")
	}
	n.Pix[3] = 0x80
	if r := rgbaOf(n); &r.Pix[0] == &n.Pix[0] {
		t.Fatal("a translucent NRGBA must be converted")
	}
}

func TestFrameStatsWindow(t *testing.T) {
	s := newFrameStats(10 * time.Second)
	t0 := s.start
	for i := 0; i < 50; i++ {
		s.add("screenshot", 40*time.Millisecond, 10*time.Millisecond, 1000, 800, 600)
	}
	s.parts.Add(100)
	if _, ok := s.roll(t0.Add(5*time.Second), FrameStats{}); ok {
		t.Fatal("window closed early")
	}
	st, ok := s.roll(t0.Add(10*time.Second), FrameStats{Quality: 72})
	if !ok || st.FPS != 5 || st.PartsPerSec != 10 || st.GrabMSAvg != 40 || st.EncodeMSAvg != 10 || st.BytesAvg != 1000 || st.Quality != 72 {
		t.Fatalf("%+v", st)
	}
	if last, ok := s.lastWindow(); !ok || last.Frames != 50 {
		t.Fatalf("%+v", last)
	}
}

// benchFrame is a real game frame when D2DVIEW_BENCH_PNG names one (never committed), else the synthetic picture.
func benchFrame(b *testing.B) (*image.RGBA, []byte) {
	if p := os.Getenv("D2DVIEW_BENCH_PNG"); p != "" {
		raw, err := os.ReadFile(p)
		if err != nil {
			b.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			b.Fatal(err)
		}
		return rgbaOf(img), raw
	}
	img := &image.RGBA{Pix: testFrame(800, 600), Stride: 4 * 800, Rect: image.Rect(0, 0, 800, 600)}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return img, buf.Bytes()
}

// BenchmarkFramePath measures the per-frame CPU of each path on this machine: the old -jpeg path (PNG decode, generic
// JPEG encode), the new one (PNG decode, RGBA fast path), and the ring path (raw RGBA, full and half size).
func BenchmarkFramePath(b *testing.B) {
	img, raw := benchFrame(b)
	var buf bytes.Buffer
	report := func(b *testing.B, n int) { b.ReportMetric(float64(n), "bytes/frame") }
	b.Run("png-decode-generic-jpeg", func(b *testing.B) {
		n := 0
		for i := 0; i < b.N; i++ {
			m, _ := png.Decode(bytes.NewReader(raw))
			buf.Reset()
			jpeg.Encode(&buf, m, &jpeg.Options{Quality: 72})
			n = buf.Len()
		}
		report(b, n)
	})
	b.Run("png-decode-rgba-jpeg", func(b *testing.B) {
		n := 0
		for i := 0; i < b.N; i++ {
			m, _ := png.Decode(bytes.NewReader(raw))
			d, _ := encodeJPEG(&buf, rgbaOf(m), false, 72)
			n = len(d)
		}
		report(b, n)
	})
	for _, q := range []int{72, 50} {
		for _, half := range []bool{false, true} {
			name := "ring-jpeg-q" + strconv.Itoa(q)
			if half {
				name += "-half"
			}
			b.Run(name, func(b *testing.B) {
				n := 0
				for i := 0; i < b.N; i++ {
					d, _ := encodeJPEG(&buf, img, half, q)
					n = len(d)
				}
				report(b, n)
			})
		}
	}
	b.Run("png-encode-go-bestspeed", func(b *testing.B) {
		e := png.Encoder{CompressionLevel: png.BestSpeed}
		for i := 0; i < b.N; i++ {
			buf.Reset()
			e.Encode(&buf, img)
		}
		report(b, buf.Len())
	})
}
