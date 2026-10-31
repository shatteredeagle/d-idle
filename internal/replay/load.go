package replay

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/mem"
)

// Fixture is any fixture file after adapter decoding. Exactly one of Stream, Image, Trace is set;
// a legacy page snapshot appears as a one-frame Stream, so Source works on every memory fixture.
type Fixture struct {
	Header mem.Header
	Stream *Stream
	Image  *d2map.Image
	Trace  []json.RawMessage // JSONL records of a death/behaviour trace, one per line
}

// Mem returns the fixture as a mem.Source at its last state, or nil for traces.
func (f *Fixture) Mem() mem.Source {
	switch {
	case f.Stream != nil:
		s := NewSource(f.Stream)
		s.SeekFrame(len(f.Stream.Frames))
		return s
	case f.Image != nil:
		return f.Image
	}
	return nil
}

// Load reads any fixture: envelope files of every kind and the three legacy formats (D2SN, d2mi,
// JSONL traces, reported as schema 0).
func Load(path string) (*Fixture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h, pr, err := mem.OpenEnvelope(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	fx := &Fixture{Header: h}
	switch h.Kind {
	case mem.KindPages:
		s, err := mem.ReadSnapshot(pr)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		fx.Stream = FromSnapshot(h, s)
		fx.Stream.Header.Kind = mem.KindPages
	case mem.KindMapImage:
		img, err := d2map.LoadImage(pr)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		fx.Image = img
	case mem.KindStream:
		st, err := readStream(h, pr)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		fx.Stream = st
	case mem.KindTrace:
		jr, err := mem.UnwrapJSONL(pr)
		if err != nil {
			return nil, err
		}
		fx.Trace, err = readLines(jr)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("%s: unknown fixture kind %q", path, h.Kind)
	}
	return fx, nil
}

func readLines(r io.Reader) ([]json.RawMessage, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	var out []json.RawMessage
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		if !json.Valid(sc.Bytes()) {
			return nil, fmt.Errorf("invalid JSON line %d", len(out)+1)
		}
		out = append(out, append(json.RawMessage(nil), sc.Bytes()...))
	}
	return out, sc.Err()
}
