package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
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
	WallMS int    // wall time the screenshot verb took
	At     time.Time
}

// Frames samples screenshots through the viewer socket, only while someone watches the stream (Touch), and never
// faster than a quarter of the wall time: the next sample waits 4x the last one's duration (so the d2d spends at
// most about 20 percent of its time on pictures, whatever the fps asked for).
type Frames struct {
	c    *d2dctl.Client
	mux  *Mux
	dir  string
	fps  float64
	JPEG bool // re-encode as JPEG q72 (about 25 ms CPU per 800x600 frame here) instead of passing d2d's PNG through
	mu   sync.RWMutex
	cur  Frame
	seen int64 // wall unix nanos of the last Touch
	last int64 // mux ticks at the previous frame
	Dur  time.Duration
	Got  int64
}

func NewFrames(viewSock string, mux *Mux, dir string, fps float64) (*Frames, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Frames{c: d2dctl.New(viewSock, d2dctl.Options{Timeout: 5 * time.Second}), mux: mux, dir: dir, fps: fps}, nil
}

func (f *Frames) Touch() { f.mu.Lock(); f.seen = time.Now().UnixNano(); f.mu.Unlock() }

func (f *Frames) watching() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return time.Since(time.Unix(0, f.seen)) < 5*time.Second
}

func (f *Frames) Current() Frame { f.mu.RLock(); defer f.mu.RUnlock(); return f.cur }

func (f *Frames) Run(ctx context.Context) {
	base := time.Duration(float64(time.Second) / f.fps)
	next := time.Now()
	for ctx.Err() == nil {
		if d := time.Until(next); d > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
			}
		}
		if !f.watching() {
			next = time.Now().Add(200 * time.Millisecond)
			continue
		}
		t0 := time.Now()
		err := f.grab(ctx)
		took := time.Since(t0)
		gap := max(base, 4*took)
		if err != nil {
			gap = time.Second
		}
		next = t0.Add(gap)
	}
}

func (f *Frames) grab(ctx context.Context) error {
	path := filepath.Join(f.dir, "shot.png")
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	t0 := time.Now()
	if _, err := f.c.Screenshot(cctx, path); err != nil {
		return err
	}
	took := time.Since(t0)
	ticks := f.mux.ticks.Load()
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data, typ := b, "image/png"
	if len(b) < 8 || string(b[1:4]) != "PNG" {
		return fmt.Errorf("screenshot: not a PNG (%d bytes)", len(b))
	}
	if f.JPEG {
		img, _, err := image.Decode(bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("screenshot decode: %w", err)
		}
		var out bytes.Buffer
		if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 72}); err != nil {
			return err
		}
		data, typ = out.Bytes(), "image/jpeg"
	}
	f.mu.Lock()
	f.Got++
	f.Dur = took
	f.cur = Frame{Seq: f.Got, Data: data, Type: typ, Ticks: ticks - f.last, WallMS: int(took / time.Millisecond), At: time.Now()}
	f.last = ticks
	f.mu.Unlock()
	return nil
}
