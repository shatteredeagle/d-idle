package replay

import (
	"sync"
	"time"

	"d2lod-bot/internal/mem"
)

// Source is a mem.Source over a Stream at a cursor: reads see the pages as of the last applied
// frame. Pages never recorded fail with mem.ErrUnmapped, like the live process.
type Source struct {
	mu   sync.Mutex
	st   *Stream
	n    int // frames applied
	snap *mem.Snapshot
}

// NewSource returns a Source with the first frame applied (cursor 1).
func NewSource(st *Stream) *Source {
	s := &Source{st: st, snap: mem.NewSnapshot()}
	s.SeekFrame(1)
	return s
}

// Frames is the number of frames in the stream.
func (s *Source) Frames() int { return len(s.st.Frames) }

// Cursor is the number of frames applied.
func (s *Source) Cursor() int { s.mu.Lock(); defer s.mu.Unlock(); return s.n }

// Time is the timestamp of the last applied frame (0 before any).
func (s *Source) Time() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n == 0 {
		return 0
	}
	return s.st.Frames[s.n-1].T
}

// Step applies the next frame; false at the end.
func (s *Source) Step() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.n >= len(s.st.Frames) {
		return false
	}
	s.apply(s.st.Frames[s.n])
	s.n++
	return true
}

// SeekFrame sets the cursor to n applied frames (clamped). Seeking backwards rebuilds from the start.
func (s *Source) SeekFrame(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n > len(s.st.Frames) {
		n = len(s.st.Frames)
	}
	if n < s.n {
		s.snap, s.n = mem.NewSnapshot(), 0
	}
	for s.n < n {
		s.apply(s.st.Frames[s.n])
		s.n++
	}
}

// Seek applies every frame with T <= t (at least none; go back with a smaller t).
func (s *Source) Seek(t time.Duration) {
	n := 0
	for n < len(s.st.Frames) && s.st.Frames[n].T <= t {
		n++
	}
	s.SeekFrame(n)
}

func (s *Source) apply(f Frame) {
	for a, b := range f.Pages {
		s.snap.Put(a, b)
	}
}

func (s *Source) ReadAt(p []byte, addr uint32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snap.ReadAt(p, addr)
}
