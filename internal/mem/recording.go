package mem

import (
	"sort"
	"sync"
)

// Recording wraps a Source and remembers every successfully read range (address and the bytes
// returned), so a code path run once against the live game can be replayed offline from a
// Snapshot. Failed reads are not recorded.
type Recording struct {
	Src Source

	mu    sync.Mutex
	pages map[uint32]*recPage
}

type recPage struct {
	data [PageSize]byte
	have [PageSize]bool
}

func NewRecording(src Source) *Recording {
	return &Recording{Src: src, pages: map[uint32]*recPage{}}
}

func (r *Recording) ReadAt(p []byte, addr uint32) error {
	if err := r.Src.ReadAt(p, addr); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range p {
		a := addr + uint32(i)
		base := a &^ (PageSize - 1)
		pg := r.pages[base]
		if pg == nil {
			pg = &recPage{}
			r.pages[base] = pg
		}
		pg.data[a-base] = p[i]
		pg.have[a-base] = true
	}
	return nil
}

// Pages returns the sorted base addresses of touched pages.
func (r *Recording) Pages() []uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]uint32, 0, len(r.pages))
	for a := range r.pages {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Snapshot builds a Snapshot of the touched pages. Each page is re-read whole from the backing
// source (so nearby unread bytes are kept), then overlaid with the exact bytes the code saw, so
// the replay returns what the reader originally observed. If the whole-page read fails (page
// partly unmapped), unseen bytes are zero.
func (r *Recording) Snapshot() *Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := NewSnapshot()
	for base, pg := range r.pages {
		var buf [PageSize]byte
		if err := r.Src.ReadAt(buf[:], base); err != nil {
			buf = [PageSize]byte{}
		}
		for i := range buf {
			if pg.have[i] {
				buf[i] = pg.data[i]
			}
		}
		s.pages[base] = &buf
	}
	return s
}
