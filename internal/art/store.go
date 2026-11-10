package art

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"d2lod-bot/internal/data/mpq"
)

var (
	// ErrBadKey is returned for a key outside the grammar; the HTTP layer answers 404.
	ErrBadKey = errors.New("art: bad key")
	// ErrNotFound means the key is valid but the install has no such art.
	ErrNotFound = errors.New("art: not found")
)

// Archives in priority order, highest first (the game's own lookup order).
var archiveNames = []string{"patch_d2.mpq", "d2exp.mpq", "d2data.mpq"}

const (
	itemDir  = `data\global\items\`
	panelDir = `data\global\ui\PANEL\`
	palPath  = `data\global\palette\ACT1\pal.dat`
)

var itemKey = regexp.MustCompile(`^item-([a-z0-9_]{1,16})$`)

// slotFiles maps slot-<name> keys to a panel silhouette file and frame. Frame choices are checked in
// TestRealArt against the install (docs/paperdoll-art-build.md).
var slotFiles = map[string]struct {
	file  string
	frame int
}{
	"armor":  {"inv_armor.dc6", 0},
	"weapon": {"inv_weapons.dc6", 0},
	"helm":   {"inv_helm_glove.dc6", 0},
	"gloves": {"inv_helm_glove.dc6", 1},
	"ring":   {"inv_ring_amulet.dc6", 0},
	"amulet": {"inv_ring_amulet.dc6", 1},
	"belt":   {"inv_belt.dc6", 0},
	"boots":  {"inv_boots.dc6", 0},
}

// Store decodes art from an install on demand, with a memory and a disk cache.
type Store struct {
	archives []*mpq.Archive
	pal      *Palette
	dir      string // disk cache, "" when unavailable

	mu  sync.Mutex
	mem map[string][]byte
}

// Open opens the install's archives (patch_d2.mpq optional, d2data.mpq and d2exp.mpq required) and
// prepares the cache under cacheDir. A cacheDir that cannot be created only disables the disk cache.
func Open(installDir, cacheDir string) (*Store, error) {
	s := &Store{mem: map[string][]byte{}}
	var sizes []string
	for _, n := range archiveNames {
		p, err := findFold(installDir, n)
		if err != nil {
			if n == "patch_d2.mpq" {
				continue
			}
			s.Close()
			return nil, fmt.Errorf("art: %s not found in %s", n, installDir)
		}
		a, err := mpq.Open(p)
		if err != nil {
			s.Close()
			return nil, fmt.Errorf("art: %s: %w", p, err)
		}
		s.archives = append(s.archives, a)
		st, err := os.Stat(p)
		if err != nil {
			s.Close()
			return nil, err
		}
		stamp := fmt.Sprintf("%s:%d", n, st.Size())
		if n == "patch_d2.mpq" {
			h, err := fileSHA(p)
			if err != nil {
				s.Close()
				return nil, err
			}
			stamp += ":" + h
		}
		sizes = append(sizes, stamp)
	}
	raw, err := s.read(palPath)
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("art: palette: %w", err)
	}
	if s.pal, err = ParsePalette(raw); err != nil {
		s.Close()
		return nil, err
	}
	if cacheDir != "" {
		sum := sha256.Sum256([]byte(strings.Join(sizes, "|")))
		dir := filepath.Join(cacheDir, hex.EncodeToString(sum[:6]))
		if os.MkdirAll(dir, 0o755) == nil {
			s.dir = dir
		}
	}
	return s, nil
}

// Close releases the archives.
func (s *Store) Close() {
	for _, a := range s.archives {
		a.Close()
	}
	s.archives = nil
}

func (s *Store) read(name string) ([]byte, error) {
	for _, a := range s.archives {
		if a.Has(name) {
			return a.ReadFile(name)
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
}

func (s *Store) frames(name string) ([]Frame, error) {
	raw, err := s.read(name)
	if err != nil {
		return nil, err
	}
	return DecodeDC6(raw)
}

// ValidKey reports whether key is inside the grammar: item-<invfile>, slot-<name> or panel-inv.
// Nothing else ever reaches an archive lookup.
func ValidKey(key string) bool {
	if key == "panel-inv" {
		return true
	}
	if itemKey.MatchString(key) {
		return true
	}
	if name, ok := strings.CutPrefix(key, "slot-"); ok {
		_, ok = slotFiles[name]
		return ok
	}
	return false
}

// PNG returns the PNG for key, from memory, the disk cache or a fresh decode.
func (s *Store) PNG(key string) ([]byte, error) {
	if !ValidKey(key) {
		return nil, ErrBadKey
	}
	s.mu.Lock()
	b, ok := s.mem[key]
	s.mu.Unlock()
	if ok {
		return b, nil
	}
	if s.dir != "" {
		if b, err := os.ReadFile(filepath.Join(s.dir, key+".png")); err == nil && len(b) > 0 {
			s.remember(key, b)
			return b, nil
		}
	}
	img, err := s.render(key)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	b = buf.Bytes()
	if s.dir != "" {
		writeAtomic(filepath.Join(s.dir, key+".png"), b)
	}
	s.remember(key, b)
	return b, nil
}

func (s *Store) remember(key string, b []byte) {
	s.mu.Lock()
	s.mem[key] = b
	s.mu.Unlock()
}

func (s *Store) render(key string) (image.Image, error) {
	switch {
	case key == "panel-inv":
		return s.panelInv()
	case strings.HasPrefix(key, "slot-"):
		sf := slotFiles[strings.TrimPrefix(key, "slot-")]
		fr, err := s.frames(panelDir + sf.file)
		if err != nil {
			return nil, err
		}
		if sf.frame >= len(fr) {
			return nil, fmt.Errorf("%w: %s frame %d", ErrNotFound, sf.file, sf.frame)
		}
		return fr[sf.frame].Image(s.pal), nil
	}
	m := itemKey.FindStringSubmatch(key)
	fr, err := s.frames(itemDir + m[1] + ".dc6")
	if err != nil {
		return nil, err
	}
	return fr[0].Image(s.pal), nil
}

func writeAtomic(path string, b []byte) {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) != nil {
		return
	}
	if os.Rename(tmp, path) != nil {
		os.Remove(tmp)
	}
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func findFold(dir, name string) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if strings.EqualFold(e.Name(), name) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", os.ErrNotExist
}
