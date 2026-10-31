// Package store persists versioned JSON documents (File) and append-only JSONL logs (Log) for the bot's
// state files (design-gaps D11, docs/api.md persistence rules).
//
// On disk a File is {"version": N, "data": <T>}. Save writes <path>.tmp, fsyncs it, copies the previous file
// to <path>.bak, renames the tmp over the file and fsyncs the directory. Load tries the file, then the .bak; a
// file that does not parse is moved to <path>.corrupt-<unix> (newest 3 kept) and reported in Info.Warning.
// A file newer than the code (version > File.Version) is never touched: Load returns ErrFutureVersion.
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrFutureVersion: the file was written by a newer build; refuse to read or overwrite it.
	ErrFutureVersion = errors.New("store: file version is newer than this build")
	// ErrNoMigration: the file is older than this build and no registered chain reaches the current version.
	ErrNoMigration = errors.New("store: no migration path")
)

// MigrateFunc rewrites the data member of a document from one version to another.
type MigrateFunc func(json.RawMessage) (json.RawMessage, error)

type step struct {
	to int
	fn MigrateFunc
}

// Registry holds migration steps. A step is registered for one from-version; chains are followed greedily
// until the target version. The zero value is ready to use.
type Registry struct {
	mu    sync.Mutex
	steps map[int]step
}

// Register adds the migration from version `from` to version `to` (to > from). It panics on a duplicate
// from or a non-increasing step: both are programming errors found at init.
func (r *Registry) Register(from, to int, fn MigrateFunc) {
	if to <= from || fn == nil {
		panic(fmt.Sprintf("store: bad migration %d -> %d", from, to))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.steps == nil {
		r.steps = map[int]step{}
	}
	if _, dup := r.steps[from]; dup {
		panic(fmt.Sprintf("store: duplicate migration from %d", from))
	}
	r.steps[from] = step{to, fn}
}

// Plan lists the versions visited going from `from` to `to` (inclusive), or ErrNoMigration.
func (r *Registry) Plan(from, to int) ([]int, error) {
	if r == nil {
		r = &Registry{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	path := []int{from}
	for v := from; v < to; {
		s, ok := r.steps[v]
		if !ok || s.to > to {
			return nil, fmt.Errorf("%w: %d -> %d (stuck at %d)", ErrNoMigration, from, to, v)
		}
		v = s.to
		path = append(path, v)
	}
	return path, nil
}

// Migrate runs the chain on raw.
func (r *Registry) Migrate(raw json.RawMessage, from, to int) (json.RawMessage, error) {
	path, err := r.Plan(from, to)
	if err != nil {
		return nil, err
	}
	for _, v := range path[:len(path)-1] {
		raw, err = r.steps[v].fn(raw)
		if err != nil {
			return nil, fmt.Errorf("store: migration from %d: %w", v, err)
		}
	}
	return raw, nil
}

// File is one versioned JSON document. Version is the current schema version (>= 1).
type File[T any] struct {
	Path       string
	Version    int
	Migrations *Registry // may be nil when Version is 1
	Perm       os.FileMode
	// Keep is how many .corrupt-* files are retained (default 3).
	Keep int
	// Now is the clock for quarantine names (default time.Now).
	Now func() time.Time
}

// Info describes what Load did besides returning the value.
type Info struct {
	Exists      bool   // a usable file (or .bak) was read
	FromBackup  bool   // the main file was unusable and .bak was used
	FromVersion int    // version found on disk
	Migrated    bool   // the value was migrated in memory; Save to persist it
	Quarantined string // path the corrupt file was moved to
	Warning     string // human-readable note for a loadWarning field
}

type envelope struct {
	Version int             `json:"version"`
	Data    json.RawMessage `json:"data"`
}

func (f *File[T]) version() int {
	if f.Version < 1 {
		return 1
	}
	return f.Version
}

func (f *File[T]) now() time.Time {
	if f.Now != nil {
		return f.Now()
	}
	return time.Now()
}

// Load reads the document. A missing file (and no .bak) yields the zero T and Info.Exists == false.
// Migration happens in memory and the original is untouched until Save.
func (f *File[T]) Load() (T, Info, error) {
	var zero T
	var info Info
	v, err := f.read(f.Path)
	if err == nil {
		info.Exists = true
		return f.finish(v, info)
	}
	if errors.Is(err, ErrFutureVersion) || errors.Is(err, ErrNoMigration) {
		return zero, info, err
	}
	if !errors.Is(err, fs.ErrNotExist) {
		// unparsable main file: quarantine, then try the backup
		info.Quarantined = f.quarantine()
		info.Warning = fmt.Sprintf("%s unusable (%v)", filepath.Base(f.Path), err)
	}
	b, berr := f.read(f.Path + ".bak")
	if berr == nil {
		info.Exists, info.FromBackup = true, true
		if info.Warning == "" {
			info.Warning = filepath.Base(f.Path) + " missing, recovered from .bak"
		} else {
			info.Warning += ", recovered from .bak"
		}
		return f.finish(b, info)
	}
	if errors.Is(berr, ErrFutureVersion) || errors.Is(berr, ErrNoMigration) {
		return zero, info, berr
	}
	if !errors.Is(berr, fs.ErrNotExist) {
		f.quarantineFile(f.Path + ".bak")
		info.Warning += "; .bak unusable too"
	}
	return zero, info, nil
}

type loaded struct {
	version int
	data    json.RawMessage
}

func (f *File[T]) read(path string) (loaded, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return loaded{}, err
	}
	var e envelope
	if err := json.Unmarshal(b, &e); err != nil {
		return loaded{}, fmt.Errorf("parse: %w", err)
	}
	if e.Version < 1 || len(bytes.TrimSpace(e.Data)) == 0 {
		return loaded{}, errors.New("parse: missing version or data")
	}
	if e.Version > f.version() {
		return loaded{}, fmt.Errorf("%w: file %d, build %d", ErrFutureVersion, e.Version, f.version())
	}
	return loaded{e.Version, e.Data}, nil
}

func (f *File[T]) finish(l loaded, info Info) (T, Info, error) {
	var v T
	info.FromVersion = l.version
	data := l.data
	if l.version < f.version() {
		var err error
		if data, err = f.Migrations.Migrate(data, l.version, f.version()); err != nil {
			return v, info, err
		}
		info.Migrated = true
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return v, info, fmt.Errorf("store: decode %s: %w", filepath.Base(f.Path), err)
	}
	return v, info, nil
}

// Save writes v atomically (see package doc). If the file on disk is from a newer build it refuses.
func (f *File[T]) Save(v T) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(envelope{f.version(), data}, "", "  ")
	if err != nil {
		return err
	}
	if _, err := f.read(f.Path); errors.Is(err, ErrFutureVersion) {
		return err
	} else if err == nil {
		// keep the previous good file as .bak; a hard copy so the main path always exists
		if raw, rerr := os.ReadFile(f.Path); rerr == nil {
			if err := writeAtomic(f.Path+".bak", raw, f.perm()); err != nil {
				return err
			}
		}
	}
	return writeAtomic(f.Path, append(b, '\n'), f.perm())
}

func (f *File[T]) perm() os.FileMode {
	if f.Perm == 0 {
		return 0o644
	}
	return f.Perm
}

func writeAtomic(path string, b []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	fh, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := fh.Write(b); err != nil {
		fh.Close()
		os.Remove(tmp)
		return err
	}
	if err := fh.Sync(); err != nil {
		fh.Close()
		os.Remove(tmp)
		return err
	}
	if err := fh.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func (f *File[T]) quarantine() string { return f.quarantineFile(f.Path) }

// quarantineFile moves path to <path>.corrupt-<unix>[-n] and prunes to the newest Keep.
func (f *File[T]) quarantineFile(path string) string {
	base := path + ".corrupt-" + strconv.FormatInt(f.now().Unix(), 10)
	dst := base
	for n := 1; ; n++ {
		if _, err := os.Lstat(dst); errors.Is(err, fs.ErrNotExist) {
			break
		}
		dst = base + "-" + strconv.Itoa(n)
	}
	if err := os.Rename(path, dst); err != nil {
		return ""
	}
	keep := f.Keep
	if keep <= 0 {
		keep = 3
	}
	matches, _ := filepath.Glob(f.Path + "*.corrupt-*")
	type item struct {
		p     string
		stamp int64
		n     int
	}
	var items []item
	for _, m := range matches {
		_, suffix, _ := strings.Cut(m, ".corrupt-")
		st, nn, _ := strings.Cut(suffix, "-")
		stamp, err := strconv.ParseInt(st, 10, 64)
		if err != nil {
			continue
		}
		n, _ := strconv.Atoi(nn)
		items = append(items, item{m, stamp, n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].stamp != items[j].stamp {
			return items[i].stamp > items[j].stamp
		}
		return items[i].n > items[j].n
	})
	for _, it := range items[min(keep, len(items)):] {
		os.Remove(it.p)
	}
	return dst
}
