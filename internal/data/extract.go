package data

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"d2lod-bot/internal/data/mpq"
)

// Stamp identifies the install a cache was extracted from.
type Stamp struct {
	Schema int              `json:"schema"`
	Files  map[string]int64 `json:"files"` // lower-case file name -> size
	// PatchSHA256 is the hex sha256 of Patch_D2.mpq: sizes alone miss a
	// same-size modded patch. Empty when the install has no patch archive.
	PatchSHA256 string `json:"patch_sha256,omitempty"`
}

// InstallStamp computes the stamp for an install directory: sizes of
// Game.exe and the data archives. (Sizes, not mtimes, so copying the install
// does not invalidate the cache.)
func InstallStamp(installDir string) (Stamp, error) {
	st := Stamp{Schema: schemaVersion, Files: map[string]int64{}}
	for _, n := range append([]string{"game.exe"}, archiveNames...) {
		p, err := findFold(installDir, n)
		if err != nil {
			continue
		}
		fi, err := os.Stat(p)
		if err != nil {
			return st, err
		}
		st.Files[n] = fi.Size()
		if n == "patch_d2.mpq" {
			if st.PatchSHA256, err = fileSHA256(p); err != nil {
				return st, err
			}
		}
	}
	if st.Files["d2data.mpq"] == 0 {
		return st, fmt.Errorf("data: d2data.mpq not found in %s", installDir)
	}
	return st, nil
}

func (s Stamp) equal(o Stamp) bool {
	if s.Schema != o.Schema || s.PatchSHA256 != o.PatchSHA256 || len(s.Files) != len(o.Files) {
		return false
	}
	for k, v := range s.Files {
		if o.Files[k] != v {
			return false
		}
	}
	return true
}

func fileSHA256(path string) (string, error) {
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

// findFold finds a file in dir by case-insensitive name (Wine installs mix
// "Patch_D2.mpq" and "patch_d2.mpq").
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

// openArchives opens the install's data archives in priority order. d2data
// and d2exp are required (this is a Lord of Destruction bot); the patch
// archive is optional.
func openArchives(installDir string) ([]*mpq.Archive, error) {
	var out []*mpq.Archive
	for _, n := range archiveNames {
		p, err := findFold(installDir, n)
		if err != nil {
			if n == "patch_d2.mpq" {
				continue
			}
			return closeAll(out), fmt.Errorf("data: %s not found in %s", n, installDir)
		}
		a, err := mpq.Open(p)
		if err != nil {
			return closeAll(out), fmt.Errorf("data: %s: %w", p, err)
		}
		out = append(out, a)
	}
	return out, nil
}

func closeAll(as []*mpq.Archive) []*mpq.Archive {
	for _, a := range as {
		a.Close()
	}
	return nil
}

// readFirst returns name from the highest-priority archive that has it.
func readFirst(as []*mpq.Archive, name string) ([]byte, error) {
	for _, a := range as {
		if a.Has(name) {
			return a.ReadFile(name)
		}
	}
	return nil, fmt.Errorf("%w: %s", mpq.ErrNotFound, name)
}

// trimTable keeps only the wanted columns (first occurrence of each,
// case-insensitive) of a raw .txt table, preserving row order and count.
func trimTable(raw []byte, cols []string) []byte {
	t := ParseTable(raw)
	var idx []int
	var hdr []string
	seen := map[string]bool{}
	for _, c := range cols {
		k := strings.ToLower(c)
		if seen[k] {
			continue
		}
		seen[k] = true
		if i, ok := t.col[k]; ok {
			idx = append(idx, i)
			hdr = append(hdr, c)
		}
	}
	var b bytes.Buffer
	b.WriteString(strings.Join(hdr, "\t"))
	b.WriteByte('\n')
	cells := make([]string, len(idx))
	for _, row := range t.rows {
		for j, i := range idx {
			if i < len(row) {
				cells[j] = row[i]
			} else {
				cells[j] = ""
			}
		}
		b.WriteString(strings.Join(cells, "\t"))
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func readSources(installDir string) (*sources, error) {
	as, err := openArchives(installDir)
	if err != nil {
		return nil, err
	}
	defer closeAll(as)
	s := &sources{tables: map[string][]byte{}, strs: map[string]string{}}
	for _, sp := range specs {
		raw, err := readFirst(as, excelDir+sp.name+".txt")
		if err != nil {
			if sp.optional {
				continue
			}
			return nil, err
		}
		s.tables[sp.name] = trimTable(raw, sp.cols)
	}
	// Lowest priority first so higher-priority tables overwrite.
	for i := len(stringTables) - 1; i >= 0; i-- {
		raw, err := readFirst(as, lngDir+stringTables[i])
		if err != nil {
			continue
		}
		m, err := ParseTBL(raw)
		if err != nil {
			return nil, fmt.Errorf("data: %s: %w", stringTables[i], err)
		}
		for k, v := range m {
			s.strs[k] = v
		}
	}
	if len(s.strs) == 0 {
		return nil, errors.New("data: no string tables found")
	}
	return s, nil
}

const (
	stampFile   = "stamp.json"
	stringsFile = "strings.tsv"
)

// Extract reads the needed tables out of the install's MPQs (Patch_D2.mpq
// over d2exp.mpq over d2data.mpq) and writes the compact cache to outDir:
// one trimmed TSV per table, strings.tsv, and stamp.json (written last, so
// an interrupted extraction is never mistaken for a valid cache).
func Extract(installDir, outDir string) error {
	stamp, err := InstallStamp(installDir)
	if err != nil {
		return err
	}
	s, err := readSources(installDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(outDir, stampFile))
	for name, b := range s.tables {
		if err := os.WriteFile(filepath.Join(outDir, name+".tsv"), b, 0o644); err != nil {
			return err
		}
	}
	var sb bytes.Buffer
	keys := make([]string, 0, len(s.strs))
	for k := range s.strs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteByte('\t')
		sb.WriteString(strconv.Quote(s.strs[k]))
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(filepath.Join(outDir, stringsFile), sb.Bytes(), 0o644); err != nil {
		return err
	}
	js, _ := json.Marshal(stamp)
	return os.WriteFile(filepath.Join(outDir, stampFile), js, 0o644)
}

// CacheCurrent reports whether cacheDir holds a complete extraction made
// from the current state of installDir.
func CacheCurrent(installDir, cacheDir string) bool {
	b, err := os.ReadFile(filepath.Join(cacheDir, stampFile))
	if err != nil {
		return false
	}
	var have Stamp
	if json.Unmarshal(b, &have) != nil {
		return false
	}
	want, err := InstallStamp(installDir)
	return err == nil && want.equal(have)
}

// Load reads a cache written by Extract.
func Load(cacheDir string) (*DB, error) {
	if _, err := os.Stat(filepath.Join(cacheDir, stampFile)); err != nil {
		return nil, fmt.Errorf("data: no valid cache in %s: %w", cacheDir, err)
	}
	s := &sources{tables: map[string][]byte{}, strs: map[string]string{}}
	for _, sp := range specs {
		b, err := os.ReadFile(filepath.Join(cacheDir, sp.name+".tsv"))
		if err != nil {
			if os.IsNotExist(err) && sp.optional {
				continue
			}
			return nil, err
		}
		s.tables[sp.name] = b
	}
	b, err := os.ReadFile(filepath.Join(cacheDir, stringsFile))
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		if u, err := strconv.Unquote(v); err == nil {
			s.strs[k] = u
		}
	}
	return buildDB(s)
}

// Open returns the DB for an install, re-extracting into cacheDir first when
// the cache is missing or was made from a different install.
func Open(installDir, cacheDir string) (*DB, error) {
	if !CacheCurrent(installDir, cacheDir) {
		if err := Extract(installDir, cacheDir); err != nil {
			return nil, err
		}
	}
	return Load(cacheDir)
}

// LoadInstall builds a DB directly from the MPQs without writing a cache.
func LoadInstall(installDir string) (*DB, error) {
	s, err := readSources(installDir)
	if err != nil {
		return nil, err
	}
	return buildDB(s)
}
