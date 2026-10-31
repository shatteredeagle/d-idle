package replay

import (
	"bytes"
	"context"
	"sync"
	"time"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

// Recorder captures a live run into a Stream. Wrap the source the readers use with it; each Tick
// re-reads every page the readers have touched so far and stores the ones that changed. Only
// touched pages are recorded (also keeps unrelated process memory out of fixtures).
type Recorder struct {
	Header mem.Header
	// Now is the clock (default time.Now); times are relative to the first Tick or call.
	Now func() time.Time

	rec *mem.Recording

	mu    sync.Mutex
	start time.Time
	last  map[uint32][]byte
	st    Stream
}

// NewRecorder wraps src. The returned Recorder is itself a mem.Source.
func NewRecorder(src mem.Source, h mem.Header) *Recorder {
	return &Recorder{Header: h, Now: time.Now, rec: mem.NewRecording(src), last: map[uint32][]byte{}}
}

func (r *Recorder) ReadAt(p []byte, addr uint32) error { return r.rec.ReadAt(p, addr) }

func (r *Recorder) elapsed() time.Duration {
	now := r.Now()
	if r.start.IsZero() {
		r.start = now
	}
	return now.Sub(r.start)
}

// Tick stores the pages that changed since the previous Tick (all touched pages on the first).
// It returns the number of changed pages. Pages whose whole-page read fails are skipped.
func (r *Recorder) Tick() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := r.elapsed()
	f := Frame{T: t, Pages: map[uint32][]byte{}}
	for _, a := range r.rec.Pages() {
		buf := make([]byte, mem.PageSize)
		if err := r.rec.Src.ReadAt(buf, a); err != nil {
			continue
		}
		if old, ok := r.last[a]; ok && bytes.Equal(old, buf) {
			continue
		}
		r.last[a] = buf
		f.Pages[a] = buf
	}
	if len(f.Pages) > 0 {
		r.st.Frames = append(r.st.Frames, f)
	}
	return len(f.Pages)
}

// WrapInput returns an Input that forwards to c and records every call into the stream.
func (r *Recorder) WrapInput(c input.Controller) *Input {
	in := &Input{Inner: c}
	in.Now = func() time.Duration { r.mu.Lock(); defer r.mu.Unlock(); return r.elapsed() }
	in.OnCall = func(c Call) { r.mu.Lock(); r.st.Calls = append(r.st.Calls, c); r.mu.Unlock() }
	return in
}

// Run calls work (the code under observation, which reads through r) then Tick every interval
// until ctx is done, with a final Tick.
func (r *Recorder) Run(ctx context.Context, every time.Duration, work func()) {
	tk := time.NewTicker(every)
	defer tk.Stop()
	for {
		if work != nil {
			work()
		}
		r.Tick()
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
		}
	}
}

// Stream returns the recorded stream (a copy of the slices) with the header filled in.
func (r *Recorder) Stream() *Stream {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.Header
	h.Kind = mem.KindStream
	return &Stream{Header: h, Frames: append([]Frame(nil), r.st.Frames...), Calls: append([]Call(nil), r.st.Calls...)}
}

// Save writes the recording to path as an envelope file.
func (r *Recorder) Save(path string) error { return r.Stream().Save(path) }
