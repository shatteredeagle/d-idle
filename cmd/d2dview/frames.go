package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// Frame is one sampled picture of the d2d window.
type Frame struct {
	Seq    int64
	Data   []byte
	Type   string // image/png (as d2d wrote it, no CPU here) or image/jpeg
	Ticks  int64  // bot ticks stepped since the previous sampled frame (the N of "1 frame per N virtual ticks")
	WallMS int    // wall time the grab took (screenshot verb, or the ring read)
	GameMS int64  // game time of the frame (ring source; -1 for screenshots)
	At     time.Time
}

const (
	// srcGuess is d2d's draw period until the ring has measured it: real-time d2d draws at 60 Hz (V, ring publish gaps
	// 16.9 ms p50, sd 0.5, on the sandbox 2026-11-11; the game logic itself runs at 25).
	srcGuess = time.Second / 60
	// feedPoll is how often the ring header is checked; d2d's publish divisor sets the cadence, this only bounds the delay.
	feedPoll = 3 * time.Millisecond
)

// Frames samples game pictures, only while someone watches the stream (Touch). Two sources:
//   - screenshot (any d2d): the verb makes d2d encode a PNG on its game thread and write a file, so the next sample
//     waits Backoff times the verb's wall time (default 4: d2d spends at most about 20 percent of its time on pictures);
//   - framefeed (the fork's ring, docs/viewer-fps.md): d2d copies drawn frames into a shared file, we read and encode
//     them here, and the back-off applies to our own read and encode time instead. A d2d without the verb falls back
//     to screenshots; a ring that stops moving (lockstep without rendering) is bridged by single screenshots.
type Frames struct {
	c       *d2dctl.Client
	mux     *Mux
	dir     string
	fps     float64
	JPEG    bool    // screenshot source: re-encode d2d's PNG as JPEG; the ring source always sends JPEG
	Quality int     // JPEG quality
	Half    bool    // halve width and height before the JPEG encode
	Backoff float64 // next sample waits at least Backoff x the last sample's cost (0: no back-off)
	Feed    string  // ring path for the framefeed verb ("": screenshots only)
	// HideCursor asks the fork's `cursor off` verb for frames without the game's pointer sprite (the bot's pointer
	// rests mid-screen; docs/viewer-fps.md section 8). A d2d without the verb is left alone, silently.
	HideCursor bool
	Stats      *frameStats

	mu   sync.RWMutex
	cur  Frame
	seen int64 // wall unix nanos of the last Touch
	last int64 // mux ticks at the previous frame
	Dur  time.Duration
	Got  int64

	// sampler goroutine only
	enc       bytes.Buffer
	ring      *os.File
	ringBuf   []byte
	ringSeq   uint64
	ringAt    time.Time // last time the ring moved
	feedOn    time.Time // last framefeed lease renewal (zero: off)
	feedFail  bool      // this d2d has no framefeed verb
	cursorAt  time.Time // last `cursor off` sent (it is sent again now and then: a restarted d2d draws the pointer again)
	cursorOld bool      // this d2d has no cursor verb
	lastStats time.Time
	every     int           // publish divisor sent to d2d (one frame in every)
	pending   int           // a divisor proposed by the last renewal, applied when the next one agrees (hysteresis)
	srcPeriod time.Duration // d2d's draw period, measured from the ring's game ms
	ringMS    uint64        // game ms of the last ring frame
	costEMA   time.Duration // read+encode cost, smoothed
	notify    chan struct{} // closed at every publish (stream clients wait on it)
}

func NewFrames(viewSock string, mux *Mux, dir string, fps float64) (*Frames, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Frames{c: d2dctl.New(viewSock, d2dctl.Options{Timeout: 5 * time.Second}), mux: mux, dir: dir, fps: fps,
		Quality: 72, Backoff: 4, Stats: newFrameStats(10 * time.Second), srcPeriod: srcGuess, notify: make(chan struct{})}, nil
}

// Next returns the newest frame and a channel closed when a newer one is published.
func (f *Frames) Next() (Frame, <-chan struct{}) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.cur, f.notify
}

func (f *Frames) Touch() { f.mu.Lock(); f.seen = time.Now().UnixNano(); f.mu.Unlock() }

func (f *Frames) watching() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return time.Since(time.Unix(0, f.seen)) < 5*time.Second
}

func (f *Frames) Current() Frame { f.mu.RLock(); defer f.mu.RUnlock(); return f.cur }

func (f *Frames) knobs() FrameStats {
	return FrameStats{Quality: f.Quality, Half: f.Half, Backoff: f.Backoff, TargetFPS: f.fps, Every: f.every,
		SourceHz: float64(time.Second) / float64(f.srcPeriod), FeedWanted: f.Feed != ""}
}

func (f *Frames) Run(ctx context.Context) {
	base := time.Duration(float64(time.Second) / f.fps)
	next := time.Now()
	defer f.feedOff()
	for ctx.Err() == nil {
		if d := time.Until(next); d > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
			}
		}
		if !f.watching() {
			f.feedOff()
			next = time.Now().Add(200 * time.Millisecond)
			continue
		}
		t0 := time.Now()
		cost, err := f.sample(ctx)
		gap := max(base, time.Duration(f.Backoff*float64(cost)))
		if f.feeding() {
			// The ring paces itself (d2d publishes every f.every-th drawn frame, chosen so the encode fits the
			// back-off): poll the header often so a frame is sent at the cadence it was drawn, not at our timer's.
			gap = feedPoll
		}
		if err != nil {
			gap = time.Second
		}
		next = t0.Add(gap)
		if st, ok := f.Stats.roll(time.Now(), f.knobs()); ok && (st.Frames > 0 || st.Degraded) && time.Since(f.lastStats) >= 30*time.Second {
			f.lastStats = time.Now()
			log.Printf("d2dview: %s", st)
		}
	}
}

func (f *Frames) feeding() bool { return f.Feed != "" && !f.feedFail && !f.feedOn.IsZero() }

// sample produces at most one frame and returns what it cost the shared box (the back-off input).
func (f *Frames) sample(ctx context.Context) (time.Duration, error) {
	f.hideCursor(ctx)
	if f.Feed == "" || f.feedFail {
		return f.grab(ctx)
	}
	if err := f.feedLease(ctx); err != nil {
		if f.feedFail {
			return f.grab(ctx)
		}
		return 0, err
	}
	cost, fresh, err := f.grabRing()
	if err == nil && !fresh && time.Since(f.ringAt) > time.Second {
		// Nothing drawn for a second (lockstep without render, or a restarted d2d made a new file): reopen the ring
		// next time and take one screenshot, which draws and answers.
		if f.ring != nil {
			f.ring.Close()
			f.ring = nil
		}
		return f.grab(ctx)
	}
	return cost, err
}

// cursorEvery is how often `cursor off` is sent again: the switch lives in the d2d process, so a restarted d2d shows
// its pointer for at most this long.
const cursorEvery = 10 * time.Second

// hideCursor keeps the game's pointer out of the frames (HideCursor). The verb only skips a draw: the pointer, the
// item in hand and every click work as before, so nothing here can change the game. Errors never stop a frame.
func (f *Frames) hideCursor(ctx context.Context) {
	if !f.HideCursor || f.cursorOld || (!f.cursorAt.IsZero() && time.Since(f.cursorAt) < cursorEvery) {
		return
	}
	f.cursorAt = time.Now()
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := f.c.Do(cctx, "cursor off"); err != nil && strings.Contains(err.Error(), "unknown verb") {
		f.cursorOld = true // an older d2d: it draws its pointer, and we do not ask again
	}
}

// feedLease starts or renews the ring: d2d stops publishing by itself when the lease runs out (viewer gone).
func (f *Frames) feedLease(ctx context.Context) error {
	if !f.feedOn.IsZero() && time.Since(f.feedOn) < time.Second {
		return nil
	}
	every := f.divisor()
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if _, err := f.c.Do(cctx, fmt.Sprintf("framefeed %s %d 3000", f.Feed, every)); err != nil {
		if strings.Contains(err.Error(), "unknown verb") {
			f.feedFail = true
			log.Printf("d2dview: this d2d has no framefeed verb, using screenshots")
		}
		return err
	}
	if f.feedOn.IsZero() {
		f.ringAt = time.Now()
	}
	f.feedOn, f.every = time.Now(), every
	f.Stats.everyHint.Store(int64(every))
	return nil
}

// divisor picks d2d's publish divisor: the smallest whole number of draws whose period fits both the -fps target and
// the back-off budget of our own encode. A whole number keeps the frames evenly spaced in game time (60 Hz draws at
// 25 fps would alternate 33 and 50 ms of game time, which reads as judder). A change needs two renewals in a row.
func (f *Frames) divisor() int {
	target := max(time.Duration(float64(time.Second)/f.fps), time.Duration(f.Backoff*float64(f.costEMA)))
	k := max(1, int(math.Ceil(0.95*float64(target)/float64(f.srcPeriod))))
	if f.every == 0 {
		return k
	}
	if k != f.every && k == f.pending {
		f.pending = 0
		return k
	}
	f.pending = k
	return f.every
}

func (f *Frames) feedOff() {
	if f.feedOn.IsZero() {
		return
	}
	f.feedOn = time.Time{}
	if f.ring != nil {
		f.ring.Close()
		f.ring = nil
	}
	f.every, f.pending = 0, 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	f.c.Do(ctx, "framefeed off")
}

// grabRing takes the newest ring frame if it is new; fresh is false when the d2d has drawn nothing since the last one.
func (f *Frames) grabRing() (cost time.Duration, fresh bool, err error) {
	t0 := time.Now()
	if f.ring == nil {
		if f.ring, err = os.Open(f.Feed); err != nil {
			f.ring = nil
			return 0, false, nil // d2d creates it at its next drawn frame
		}
	}
	var hb [ringHdr]byte // header first: most polls find no new frame and must stay cheap
	if _, err := f.ring.ReadAt(hb[:], 0); err != nil {
		return time.Since(t0), false, nil
	}
	if h, err := parseRingHeader(hb[:]); err != nil || h.Seq == 0 || h.Seq == f.ringSeq {
		return time.Since(t0), false, nil
	}
	h, pix, err := readRing(f.ring, f.ringBuf)
	f.ringBuf = pix
	if err != nil || h.Seq == 0 || h.Seq == f.ringSeq {
		return time.Since(t0), false, err
	}
	if h.Seq > f.ringSeq && h.GameMS > f.ringMS && f.every > 0 && f.ringSeq > 0 {
		p := time.Duration(h.GameMS-f.ringMS) * time.Millisecond / time.Duration((h.Seq-f.ringSeq)*uint64(f.every))
		if p >= 5*time.Millisecond && p <= 100*time.Millisecond {
			f.srcPeriod = (f.srcPeriod*7 + p) / 8
		}
	}
	f.ringSeq, f.ringMS, f.ringAt = h.Seq, h.GameMS, time.Now()
	read := time.Since(t0)
	img := &image.RGBA{Pix: pix, Stride: h.Stride, Rect: image.Rect(0, 0, h.W, h.H)}
	data, err := encodeJPEG(&f.enc, img, f.Half, f.Quality)
	if err != nil {
		return time.Since(t0), false, err
	}
	f.publish(data, "image/jpeg", read, int64(h.GameMS))
	if c := time.Since(t0); f.costEMA == 0 {
		f.costEMA = c
	} else {
		f.costEMA = (f.costEMA*7 + c) / 8
	}
	f.Stats.add("framefeed", read, time.Since(t0)-read, len(data), h.W>>b2i(f.Half), h.H>>b2i(f.Half))
	return time.Since(t0), true, nil
}

// grab takes one screenshot. Its cost for the back-off is the verb's wall time, what d2d spent on the picture; our
// own decode and encode are not counted (as before this card).
func (f *Frames) grab(ctx context.Context) (time.Duration, error) {
	path := filepath.Join(f.dir, "shot.png")
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	t0 := time.Now()
	if _, err := f.c.Screenshot(cctx, path); err != nil {
		return 0, err
	}
	took := time.Since(t0)
	b, err := os.ReadFile(path)
	if err != nil {
		return took, err
	}
	if len(b) < 8 || string(b[1:4]) != "PNG" {
		return took, fmt.Errorf("screenshot: not a PNG (%d bytes)", len(b))
	}
	data, typ, w, h := b, "image/png", 0, 0
	t1 := time.Now()
	if f.JPEG || f.Half {
		img, _, err := image.Decode(bytes.NewReader(b))
		if err != nil {
			return took, fmt.Errorf("screenshot decode: %w", err)
		}
		if data, err = encodeJPEG(&f.enc, rgbaOf(img), f.Half, f.Quality); err != nil {
			return took, err
		}
		typ = "image/jpeg"
		w, h = img.Bounds().Dx()>>b2i(f.Half), img.Bounds().Dy()>>b2i(f.Half)
	}
	f.publish(data, typ, took, -1)
	f.Stats.add("screenshot", took, time.Since(t1), len(data), w, h)
	return took, nil
}

func (f *Frames) publish(data []byte, typ string, took time.Duration, gameMS int64) {
	ticks := f.mux.ticks.Load()
	f.mu.Lock()
	f.Got++
	f.Dur = took
	f.cur = Frame{Seq: f.Got, Data: data, Type: typ, Ticks: ticks - f.last, WallMS: int(took / time.Millisecond), GameMS: gameMS, At: time.Now()}
	f.last = ticks
	close(f.notify)
	f.notify = make(chan struct{})
	f.mu.Unlock()
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}
