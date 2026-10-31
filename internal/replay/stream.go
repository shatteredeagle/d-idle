// Package replay records and replays game memory over time so behaviour can be tested offline.
//
// A Stream is the "stream" fixture kind of the mem envelope: gzip JSONL, one record per line,
// either {"t":ms,"pages":{"<hex addr>":"<base64 4096 bytes>"}} (pages that changed since the
// previous record; the first record carries every page the reader touched) or
// {"t":ms,"in":"<input call>"}. Format choice: snapshot-then-delta, simple on purpose (WP-REPLAY).
package replay

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"d2lod-bot/internal/mem"
)

// Frame is the set of pages that changed at time T.
type Frame struct {
	T     time.Duration
	Pages map[uint32][]byte
}

// Call is one recorded input call in the canonical text form of Input.
type Call struct {
	T    time.Duration
	Text string
}

// Stream is a time-indexed sequence of page deltas and input calls.
type Stream struct {
	Header mem.Header
	Frames []Frame
	Calls  []Call
}

type line struct {
	T     int64             `json:"t"`
	Pages map[string]string `json:"pages,omitempty"`
	In    string            `json:"in,omitempty"`
}

// Duration is the time of the last record.
func (s *Stream) Duration() time.Duration {
	var d time.Duration
	if n := len(s.Frames); n > 0 {
		d = s.Frames[n-1].T
	}
	if n := len(s.Calls); n > 0 && s.Calls[n-1].T > d {
		d = s.Calls[n-1].T
	}
	return d
}

// Write encodes the stream as an envelope file.
func (s *Stream) Write(w io.Writer) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	type rec struct {
		t int64
		l line
	}
	var recs []rec
	for _, f := range s.Frames {
		l := line{T: f.T.Milliseconds(), Pages: map[string]string{}}
		for a, b := range f.Pages {
			l.Pages[strconv.FormatUint(uint64(a), 16)] = base64.StdEncoding.EncodeToString(b)
		}
		recs = append(recs, rec{l.T, l})
	}
	for _, c := range s.Calls {
		recs = append(recs, rec{c.T.Milliseconds(), line{T: c.T.Milliseconds(), In: c.Text}})
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].t < recs[j].t })
	for _, r := range recs {
		if err := enc.Encode(r.l); err != nil {
			return err
		}
	}
	h := s.Header
	h.Kind = mem.KindStream
	return mem.WriteEnvelope(w, h, mem.GzipJSONL(buf.Bytes()))
}

// Save writes the stream to path.
func (s *Stream) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := s.Write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// readStream decodes a stream payload (already past the envelope header).
func readStream(h mem.Header, r io.Reader) (*Stream, error) {
	jr, err := mem.UnwrapJSONL(r)
	if err != nil {
		return nil, err
	}
	st := &Stream{Header: h}
	sc := bufio.NewScanner(jr)
	sc.Buffer(make([]byte, 1<<20), 1<<28)
	n := 0
	for sc.Scan() {
		n++
		var l line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("replay: line %d: %w", n, err)
		}
		t := time.Duration(l.T) * time.Millisecond
		if l.In != "" {
			st.Calls = append(st.Calls, Call{t, l.In})
		}
		if len(l.Pages) > 0 {
			f := Frame{T: t, Pages: map[uint32][]byte{}}
			for k, v := range l.Pages {
				a, err := strconv.ParseUint(k, 16, 32)
				if err != nil || a&(mem.PageSize-1) != 0 {
					return nil, fmt.Errorf("replay: line %d: bad page address %q", n, k)
				}
				b, err := base64.StdEncoding.DecodeString(v)
				if err != nil || len(b) != mem.PageSize {
					return nil, fmt.Errorf("replay: line %d: bad page data at %q", n, k)
				}
				f.Pages[uint32(a)] = b
			}
			st.Frames = append(st.Frames, f)
		}
	}
	return st, sc.Err()
}

// FromSnapshot wraps a page snapshot as a one-frame stream at t=0.
func FromSnapshot(h mem.Header, s *mem.Snapshot) *Stream {
	f := Frame{Pages: map[uint32][]byte{}}
	s.EachPage(func(a uint32, b []byte) { f.Pages[a] = b })
	h.Kind = mem.KindStream
	return &Stream{Header: h, Frames: []Frame{f}}
}
