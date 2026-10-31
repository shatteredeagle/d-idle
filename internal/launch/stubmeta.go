package launch

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

// StubMeta is what the Blizzard downloader stub (Downloader_Diablo2_*.exe) carries inside itself: a
// BitTorrent metainfo naming the installer files the stub fetches. VERIFIED by reading both stubs.
// The piece hashes let us check a finished download without running anything, and let the app tell the
// user precisely what is missing. Nothing here is Blizzard content; it is parsed from the user's stub.
type StubMeta struct {
	Name           string // directory the stub creates on the Desktop, e.g. D2-1.14b-Installer-enUS
	DirectDownload string // CDN base the stub can also use; returned 403 when probed (see docs/launch.md)
	PieceLength    int64
	Files          []StubFile
	pieces         []byte
}

// StubFile is one file of the torrent, path elements joined with the OS separator.
type StubFile struct {
	Path   string
	Length int64
}

// ReadStubMeta finds the embedded bencoded torrent in the stub and parses it.
func ReadStubMeta(stubPath string) (*StubMeta, error) {
	b, err := os.ReadFile(stubPath)
	if err != nil {
		return nil, err
	}
	i := bytes.Index(b, []byte("d8:announce"))
	if i < 0 {
		return nil, errors.New("no embedded torrent metainfo found")
	}
	v, _, err := bdecode(b[i:])
	if err != nil {
		return nil, fmt.Errorf("torrent: %w", err)
	}
	top, _ := v.(map[string]any)
	info, _ := top["info"].(map[string]any)
	if top == nil || info == nil {
		return nil, errors.New("torrent has no info dictionary")
	}
	m := &StubMeta{}
	m.Name, _ = info["name"].(string)
	m.DirectDownload, _ = top["direct download"].(string)
	pl, _ := info["piece length"].(int64)
	m.PieceLength = pl
	ps, _ := info["pieces"].(string)
	m.pieces = []byte(ps)
	files, _ := info["files"].([]any)
	for _, f := range files {
		fm, _ := f.(map[string]any)
		l, _ := fm["length"].(int64)
		var parts []string
		for _, p := range fm["path"].([]any) {
			parts = append(parts, p.(string))
		}
		m.Files = append(m.Files, StubFile{filepath.Join(parts...), l})
	}
	if m.Name == "" || m.PieceLength <= 0 || len(m.Files) == 0 || len(m.pieces)%20 != 0 {
		return nil, errors.New("torrent metainfo is incomplete")
	}
	return m, nil
}

// TotalSize is the sum of all file lengths.
func (m *StubMeta) TotalSize() int64 {
	var n int64
	for _, f := range m.Files {
		n += f.Length
	}
	return n
}

// VerifyDir checks the files under dir (the Desktop/<Name> folder inside the prefix). With pieces=false
// only existence and sizes are checked (instant); with pieces=true every SHA-1 piece hash is checked
// too (reads ~2 GB for the base game). Returns the first problem.
func (m *StubMeta) VerifyDir(dir string, pieces bool) error {
	for _, f := range m.Files {
		st, err := os.Stat(filepath.Join(dir, f.Path))
		if err != nil {
			return fmt.Errorf("missing %s", f.Path)
		}
		if st.Size() != f.Length {
			return fmt.Errorf("%s: size %d, want %d", f.Path, st.Size(), f.Length)
		}
	}
	if !pieces {
		return nil
	}
	var readers []io.Reader
	var closers []io.Closer
	defer func() {
		for _, c := range closers {
			c.Close()
		}
	}()
	for _, f := range m.Files {
		fh, err := os.Open(filepath.Join(dir, f.Path))
		if err != nil {
			return err
		}
		closers = append(closers, fh)
		readers = append(readers, fh)
	}
	stream := io.MultiReader(readers...)
	buf := make([]byte, m.PieceLength)
	for i := 0; i*20 < len(m.pieces); i++ {
		n, err := io.ReadFull(stream, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("piece %d: %w", i, err)
		}
		sum := sha1.Sum(buf[:n])
		if !bytes.Equal(sum[:], m.pieces[i*20:i*20+20]) {
			return fmt.Errorf("piece %d hash mismatch", i)
		}
	}
	return nil
}

// bdecode parses one bencoded value and returns the rest. Strings are Go strings (binary safe),
// integers int64, lists []any, dictionaries map[string]any.
func bdecode(b []byte) (any, []byte, error) {
	if len(b) == 0 {
		return nil, nil, io.ErrUnexpectedEOF
	}
	switch c := b[0]; {
	case c == 'i':
		e := bytes.IndexByte(b, 'e')
		if e < 0 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		n, err := strconv.ParseInt(string(b[1:e]), 10, 64)
		return n, b[e+1:], err
	case c == 'l' || c == 'd':
		rest := b[1:]
		var list []any
		for {
			if len(rest) == 0 {
				return nil, nil, io.ErrUnexpectedEOF
			}
			if rest[0] == 'e' {
				rest = rest[1:]
				break
			}
			v, r, err := bdecode(rest)
			if err != nil {
				return nil, nil, err
			}
			list, rest = append(list, v), r
		}
		if c == 'l' {
			return list, rest, nil
		}
		d := map[string]any{}
		for i := 0; i+1 < len(list); i += 2 {
			k, ok := list[i].(string)
			if !ok {
				return nil, nil, errors.New("non-string dictionary key")
			}
			d[k] = list[i+1]
		}
		return d, rest, nil
	case c >= '0' && c <= '9':
		col := bytes.IndexByte(b, ':')
		if col < 0 {
			return nil, nil, io.ErrUnexpectedEOF
		}
		n, err := strconv.Atoi(string(b[:col]))
		if err != nil || n < 0 || col+1+n > len(b) {
			return nil, nil, errors.New("bad string length")
		}
		return string(b[col+1 : col+1+n]), b[col+1+n:], nil
	}
	return nil, nil, fmt.Errorf("bad bencode byte %q", b[0])
}
