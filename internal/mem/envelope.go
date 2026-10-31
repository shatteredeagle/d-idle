package mem

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// Fixture envelope (design-gaps D10). Layout:
//
//	"D2FX" | u16 schema (LE) | u32 headerLen (LE) | header JSON | payload
//
// The payload is the pre-envelope file verbatim: a D2SN snapshot for kind "pages", a D2MI image
// for kind "mapimage" (both already compressed), and gzip-compressed JSONL for "stream" and
// "trace". Files without the magic are legacy (schema 0) and are recognised by content.
const (
	EnvMagic        = "D2FX"
	EnvSchema       = 1
	maxEnvHeaderLen = 1 << 20
)

// Fixture kinds.
const (
	KindPages    = "pages"
	KindMapImage = "mapimage"
	KindStream   = "stream"
	KindTrace    = "trace"
)

// BuildID identifies the Game.exe a fixture was recorded from.
type BuildID struct {
	Timestamp uint32 `json:"timestamp"`
	Size      uint32 `json:"size"`
}

// ExpectedBuild is the only supported build (1.14.3.71).
func ExpectedBuild() BuildID { return BuildID{ExpectTimestamp, ExpectSizeOfImage} }

// Header is the envelope metadata. Schema and Kind are authoritative; the rest is provenance.
type Header struct {
	Schema     int      `json:"schema"`
	Kind       string   `json:"kind"`
	Build      BuildID  `json:"build"`
	Runner     string   `json:"runner,omitempty"`   // wine/proton version
	Recorder   string   `json:"recorder,omitempty"` // tool and version that wrote it
	Instance   string   `json:"instance,omitempty"`
	Char       string   `json:"char,omitempty"`
	Class      string   `json:"class,omitempty"`
	Difficulty int      `json:"difficulty"`
	Seed       uint32   `json:"seed,omitempty"`
	Area       int      `json:"area,omitempty"`
	CreatedAt  string   `json:"createdAt,omitempty"` // RFC 3339
	Notes      string   `json:"notes,omitempty"`
	Verified   []string `json:"verified,omitempty"` // what a human checked, with date
}

// NewHeader returns a header for kind with the current build and time.
func NewHeader(kind string) Header {
	return Header{Schema: EnvSchema, Kind: kind, Build: ExpectedBuild(), CreatedAt: time.Now().UTC().Format(time.RFC3339)}
}

// WriteEnvelope writes magic, header and then the payload produced by body.
func WriteEnvelope(w io.Writer, h Header, body func(io.Writer) error) error {
	h.Schema = EnvSchema
	hj, err := json.Marshal(h)
	if err != nil {
		return err
	}
	var pre [10]byte
	copy(pre[:], EnvMagic)
	binary.LittleEndian.PutUint16(pre[4:], EnvSchema)
	binary.LittleEndian.PutUint32(pre[6:], uint32(len(hj)))
	if _, err := w.Write(pre[:]); err != nil {
		return err
	}
	if _, err := w.Write(hj); err != nil {
		return err
	}
	return body(w)
}

// OpenEnvelope reads the header of any fixture file and returns it with a reader positioned at the
// payload (the legacy bytes).
// For stream and trace kinds the payload is gzip JSONL (envelope) or plain JSONL (legacy); see UnwrapJSONL.
// Legacy files get Schema 0, the kind inferred from content and an empty provenance.
func OpenEnvelope(r io.Reader) (Header, io.Reader, error) {
	br := bufio.NewReader(r)
	head, err := br.Peek(4)
	if err != nil {
		return Header{}, nil, fmt.Errorf("mem: fixture header: %w", err)
	}
	switch {
	case string(head) == EnvMagic:
		var pre [10]byte
		if _, err := io.ReadFull(br, pre[:]); err != nil {
			return Header{}, nil, err
		}
		schema := int(binary.LittleEndian.Uint16(pre[4:]))
		if schema > EnvSchema {
			return Header{}, nil, fmt.Errorf("mem: fixture schema %d is newer than supported %d", schema, EnvSchema)
		}
		n := binary.LittleEndian.Uint32(pre[6:])
		if n == 0 || n > maxEnvHeaderLen {
			return Header{}, nil, fmt.Errorf("mem: bad fixture header length %d", n)
		}
		hj := make([]byte, n)
		if _, err := io.ReadFull(br, hj); err != nil {
			return Header{}, nil, err
		}
		var h Header
		if err := json.Unmarshal(hj, &h); err != nil {
			return Header{}, nil, fmt.Errorf("mem: fixture header JSON: %w", err)
		}
		h.Schema = schema
		return h, br, nil
	case string(head) == snapMagic:
		return Header{Kind: KindPages}, br, nil
	case head[0] == 0x1f && head[1] == 0x8b:
		return Header{Kind: KindMapImage}, br, nil // legacy d2mi is a bare gzip stream
	case head[0] == '{':
		return Header{Kind: KindTrace}, br, nil
	}
	return Header{}, nil, errors.New("mem: unrecognised fixture format")
}

// UnwrapJSONL returns a reader of plain JSONL for a stream or trace payload: it gunzips when the
// payload is gzip (always for envelope files, never for legacy traces).
func UnwrapJSONL(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)
	if b, err := br.Peek(2); err == nil && b[0] == 0x1f && b[1] == 0x8b {
		return gzip.NewReader(br)
	}
	return br, nil
}

// GzipJSONL is the body helper for stream and trace envelopes.
func GzipJSONL(data []byte) func(io.Writer) error {
	return func(w io.Writer) error {
		zw := gzip.NewWriter(w)
		if _, err := zw.Write(data); err != nil {
			return err
		}
		return zw.Close()
	}
}

// SaveEnvelope writes the snapshot wrapped in an envelope.
func (s *Snapshot) SaveEnvelope(path string, h Header) error {
	h.Kind = KindPages
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := WriteEnvelope(f, h, func(w io.Writer) error { _, err := s.WriteTo(w); return err }); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// LoadSnapshotHeader is LoadSnapshot that also returns the header (legacy files: Schema 0).
func LoadSnapshotHeader(path string) (Header, *Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return Header{}, nil, err
	}
	defer f.Close()
	h, pr, err := OpenEnvelope(f)
	if err != nil {
		return h, nil, err
	}
	if h.Kind != KindPages {
		return h, nil, fmt.Errorf("mem: fixture kind %q is not a page snapshot", h.Kind)
	}
	s, err := ReadSnapshot(pr)
	return h, s, err
}
