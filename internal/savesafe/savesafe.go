// Package savesafe keeps rolling copies of the character file of a running bot and puts the newest good one back
// when the live file is found missing, empty or corrupt at startup. It exists because d2d writes the .d2s in place:
// a full disk at the moment of a save leaves a zero-length file and the hero is gone (incident 2026-11-28,
// docs/design/fork-save-atomic.md). Always on, no knob. Runbook: docs/ops/save-backups.md.
//
// Layout: <state>/savebackups/<name>.<unix>.d2s, with the side files of the same moment next to it
// (<name>.<unix>.key, .map, .ma0.. and <name>.<unix>.shared-<file> for a shared stash). A set counts only when its
// .d2s exists, and the .d2s is written last.
package savesafe

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/cheat/d2s"
)

const (
	Dir        = "savebackups"
	Every      = 10 * time.Minute
	KeepNewest = 12
	KeepDays   = 7
	MinFree    = 200 << 20 // below this nothing is written
	WarnFree   = 2 << 30   // below this `ops status --why` warns
	// MinSize is the fixed part of a version 96 character file: the stat section starts at byte 765 (V: internal/cheat/d2s).
	MinSize = 765

	repairFile   = "last-repair.json"
	repairShown  = 7 * 24 * time.Hour
	maxSideBytes = 16 << 20
	offSize      = 8
	offChecksum  = 12
)

var (
	ErrDiskLow = errors.New("savesafe: less than 200 MB free, not writing")
	ErrInvalid = errors.New("savesafe: not a valid character file")
)

// FreeFunc reports the bytes a normal user can still write under path.
type FreeFunc func(path string) (uint64, error)

// Valid checks what a truncated or half-written file fails: length, signature, the size field and the checksum
// (V: the checksum algorithm on three real saves, internal/cheat/d2s). The sections are not parsed, so a save from any
// class or patch level of the 1.10+ format passes.
func Valid(b []byte) error {
	switch {
	case len(b) == 0:
		return fmt.Errorf("%w: zero length", ErrInvalid)
	case len(b) < MinSize:
		return fmt.Errorf("%w: %d bytes, the header alone is %d", ErrInvalid, len(b), MinSize)
	case binary.LittleEndian.Uint32(b) != d2s.Signature:
		return fmt.Errorf("%w: bad signature", ErrInvalid)
	case int(binary.LittleEndian.Uint32(b[offSize:])) != len(b):
		return fmt.Errorf("%w: size field %d, file %d", ErrInvalid, binary.LittleEndian.Uint32(b[offSize:]), len(b))
	case binary.LittleEndian.Uint32(b[offChecksum:]) != d2s.Checksum(b):
		return fmt.Errorf("%w: checksum mismatch", ErrInvalid)
	}
	return nil
}

// Backup is one stored set.
type Backup struct {
	Name string
	Unix int64
	Path string // the .d2s
	Size int64
	Err  error // nil: the .d2s is valid
}

func checkName(name string) error {
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\.`) {
		return fmt.Errorf("savesafe: bad character name %q", name)
	}
	return nil
}

// List returns the backups of name, newest first, each one validated.
func List(state, name string) ([]Backup, error) {
	if err := checkName(name); err != nil {
		return nil, err
	}
	dir := filepath.Join(state, Dir)
	ents, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Backup
	for _, e := range ents {
		mid, ok := strings.CutPrefix(e.Name(), name+".")
		if !ok {
			continue
		}
		if mid, ok = strings.CutSuffix(mid, ".d2s"); !ok {
			continue
		}
		u, err := strconv.ParseInt(mid, 10, 64)
		if err != nil || u < 0 {
			continue
		}
		b := Backup{Name: name, Unix: u, Path: filepath.Join(dir, e.Name())}
		if data, err := os.ReadFile(b.Path); err != nil {
			b.Err = err
		} else {
			b.Size, b.Err = int64(len(data)), Valid(data)
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Unix > out[j].Unix })
	return out, nil
}

// sideFiles lists the files that travel with name.d2s in saveDir, as backup suffix -> source base name.
// The shared stash names are U (no d2d shared stash seen yet): any *.d2x or *.sss in the folder is taken.
func sideFiles(saveDir, name string) map[string]string {
	out := map[string]string{}
	ents, _ := os.ReadDir(saveDir)
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() {
			continue
		}
		if ext, ok := strings.CutPrefix(n, name+"."); ok {
			if ext == "key" || ext == "map" || (len(ext) == 3 && ext[:2] == "ma" && ext[2] >= '0' && ext[2] <= '9') {
				out[ext] = n
			}
			continue
		}
		if x := strings.ToLower(filepath.Ext(n)); x == ".d2x" || x == ".sss" {
			out["shared-"+n] = n
		}
	}
	return out
}

// writeAtomic writes b to dir/base through a temp file in the same folder, synced before the rename.
func writeAtomic(dir, base string, b []byte) error {
	f, err := os.CreateTemp(dir, ".tmp-"+base+"-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, filepath.Join(dir, base))
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

func low(free FreeFunc, path string) bool {
	if free == nil || path == "" {
		return false
	}
	n, err := free(path)
	return err == nil && n < MinFree
}

// Low reports whether any of the paths has less than MinFree left. A path that cannot be probed counts as fine.
func Low(free FreeFunc, paths ...string) bool {
	for _, p := range paths {
		if low(free, p) {
			return true
		}
	}
	return false
}

// Result of Save.
type Result struct {
	Backup
	Unchanged bool     // the live file equals the newest backup: nothing written
	Pruned    []string // base names removed by the rotation
}

// Save copies name.d2s and its side files from saveDir into the backup folder when the live file is valid and differs
// from the newest backup, then rotates. A live file that fails Valid (which includes one d2d is in the middle of
// writing) is never copied, so a bad file cannot push the good ones out.
func Save(saveDir, state, name string, free FreeFunc, now time.Time) (Result, error) {
	if err := checkName(name); err != nil {
		return Result{}, err
	}
	live, err := os.ReadFile(filepath.Join(saveDir, name+".d2s"))
	if err != nil {
		return Result{}, err
	}
	if err := Valid(live); err != nil {
		return Result{}, fmt.Errorf("live %s.d2s: %w", name, err)
	}
	prev, err := List(state, name)
	if err != nil {
		return Result{}, err
	}
	for _, p := range prev {
		if p.Err != nil {
			continue
		}
		if old, err := os.ReadFile(p.Path); err == nil && bytes.Equal(old, live) {
			return Result{Backup: p, Unchanged: true}, nil
		}
		break
	}
	dir := filepath.Join(state, Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, err
	}
	if low(free, dir) {
		return Result{}, ErrDiskLow
	}
	u := now.Unix()
	if len(prev) > 0 && u <= prev[0].Unix {
		u = prev[0].Unix + 1
	}
	stem := name + "." + strconv.FormatInt(u, 10) + "."
	for suffix, src := range sideFiles(saveDir, name) {
		if st, err := os.Stat(filepath.Join(saveDir, src)); err != nil || st.Size() == 0 || st.Size() > maxSideBytes {
			continue
		}
		if b, err := os.ReadFile(filepath.Join(saveDir, src)); err == nil {
			if err := writeAtomic(dir, stem+suffix, b); err != nil {
				return Result{}, err
			}
		}
	}
	if err := writeAtomic(dir, stem+"d2s", live); err != nil {
		return Result{}, err
	}
	res := Result{Backup: Backup{Name: name, Unix: u, Path: filepath.Join(dir, stem+"d2s"), Size: int64(len(live))}}
	res.Pruned, err = prune(state, name, now)
	return res, err
}

// prune keeps the newest KeepNewest sets plus the newest set of each of the last KeepDays UTC days. Invalid sets do
// not count toward either quota and are removed, unless no set is valid (then nothing is touched).
func prune(state, name string, now time.Time) ([]string, error) {
	all, err := List(state, name)
	if err != nil {
		return nil, err
	}
	keep := map[int64]bool{}
	days := map[string]bool{}
	n := 0
	cutoff := now.UTC().AddDate(0, 0, -KeepDays)
	for _, b := range all {
		if b.Err != nil {
			continue
		}
		if n < KeepNewest {
			keep[b.Unix] = true
			n++
		}
		t := time.Unix(b.Unix, 0).UTC()
		if d := t.Format("2006-01-02"); t.After(cutoff) && !days[d] {
			days[d], keep[b.Unix] = true, true
		}
	}
	if len(keep) == 0 {
		return nil, nil // nothing valid: leave everything for a human
	}
	dir := filepath.Join(state, Dir)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var gone []string
	for _, e := range ents {
		rest, ok := strings.CutPrefix(e.Name(), name+".")
		if !ok {
			continue
		}
		stamp, _, ok := strings.Cut(rest, ".")
		if !ok {
			continue
		}
		u, err := strconv.ParseInt(stamp, 10, 64)
		if err != nil || keep[u] {
			continue
		}
		if os.Remove(filepath.Join(dir, e.Name())) == nil {
			gone = append(gone, e.Name())
		}
	}
	return gone, nil
}

// Repaired is the record of one startup repair; it is kept in the backup folder and shown as a notice for a week.
type Repaired struct {
	Name   string    `json:"name"`
	At     time.Time `json:"at"`
	File   string    `json:"file"`   // the live file that was replaced
	From   string    `json:"from"`   // the backup it came from
	Broken string    `json:"broken"` // where the bad file was kept ("" when it was missing)
	Why    string    `json:"why"`
}

func (r Repaired) String() string {
	return fmt.Sprintf("save repair: restored %s from %s (%s)", r.File, r.From, r.Why)
}

// Repair looks at the live name.d2s. Valid: nothing happens (ok false). Missing, empty or invalid with a valid
// backup: the bad file is renamed to name.d2s.broken-<unix>, the newest valid backup is written in its place (temp
// and rename) and side files that are missing or empty come back from the same set. Without a valid backup the live
// file is left alone and the error says so.
func Repair(saveDir, state, name string, free FreeFunc, now time.Time) (Repaired, bool, error) {
	if err := checkName(name); err != nil {
		return Repaired{}, false, err
	}
	live := filepath.Join(saveDir, name+".d2s")
	b, rerr := os.ReadFile(live)
	why := ""
	switch {
	case errors.Is(rerr, os.ErrNotExist):
		why = "live file missing"
	case rerr != nil:
		return Repaired{}, false, rerr
	default:
		if err := Valid(b); err == nil {
			return Repaired{}, false, nil
		} else {
			why = strings.TrimPrefix(err.Error(), ErrInvalid.Error()+": ")
		}
	}
	all, err := List(state, name)
	if err != nil {
		return Repaired{}, false, err
	}
	var from *Backup
	for i := range all {
		if all[i].Err == nil {
			from = &all[i]
			break
		}
	}
	if from == nil {
		if rerr != nil {
			return Repaired{}, false, nil // no save and no backup: a new hero, nothing to repair
		}
		return Repaired{}, false, fmt.Errorf("savesafe: %s is broken (%s) and there is no valid backup under %s", live, why, filepath.Join(state, Dir))
	}
	good, err := os.ReadFile(from.Path)
	if err != nil {
		return Repaired{}, false, err
	}
	if low(free, saveDir) {
		return Repaired{}, false, fmt.Errorf("%w: %s stays broken (%s) until space is freed; backup %s is ready", ErrDiskLow, live, why, from.Path)
	}
	rep := Repaired{Name: name, At: now.UTC(), File: live, From: from.Path, Why: why}
	if rerr == nil {
		rep.Broken = live + ".broken-" + strconv.FormatInt(now.Unix(), 10)
		if err := os.Rename(live, rep.Broken); err != nil {
			return Repaired{}, false, err
		}
	}
	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		return Repaired{}, false, err
	}
	if err := writeAtomic(saveDir, name+".d2s", good); err != nil {
		return Repaired{}, false, fmt.Errorf("savesafe: restore of %s failed (the broken file is %s, the backup %s): %w", live, rep.Broken, from.Path, err)
	}
	stem := strings.TrimSuffix(from.Path, "d2s")
	sides, _ := filepath.Glob(stem + "*")
	for _, s := range sides {
		suffix := strings.TrimPrefix(s, stem)
		if suffix == "d2s" {
			continue
		}
		dst := name + "." + suffix
		if sh, ok := strings.CutPrefix(suffix, "shared-"); ok {
			dst = sh
		}
		if dst != filepath.Base(dst) {
			continue
		}
		if st, err := os.Stat(filepath.Join(saveDir, dst)); err == nil && st.Size() > 0 {
			continue
		}
		if sb, err := os.ReadFile(s); err == nil {
			_ = writeAtomic(saveDir, dst, sb)
		}
	}
	if j, err := json.Marshal(rep); err == nil {
		_ = writeAtomic(filepath.Join(state, Dir), repairFile, j)
	}
	return rep, true, nil
}

// LastRepair reads the record of the last repair.
func LastRepair(state string) (Repaired, bool) {
	var r Repaired
	b, err := os.ReadFile(filepath.Join(state, Dir, repairFile))
	if err != nil || json.Unmarshal(b, &r) != nil || r.Name == "" {
		return r, false
	}
	return r, true
}

// Notices are the standing lines for `ops status --why` and the viewer: a repair of the last week and low disk space.
func Notices(state string, free FreeFunc, now time.Time) []string {
	var out []string
	if r, ok := LastRepair(state); ok && now.Sub(r.At) < repairShown {
		out = append(out, r.String()+" at "+r.At.Format(time.RFC3339)+"; progress since that backup is lost")
	}
	if free != nil && state != "" {
		if n, err := free(state); err == nil && n < WarnFree {
			msg := fmt.Sprintf("disk space: %d MB free under %s (warning below %d MB)", n>>20, state, WarnFree>>20)
			if n < MinFree {
				msg += fmt.Sprintf("; below %d MB: no save backups and no planned save points until space is freed", MinFree>>20)
			}
			out = append(out, msg)
		}
	}
	return out
}

// Report is what `d2bot ops save check` prints. Read only.
type Report struct {
	Live    string
	LiveLen int64
	LiveErr error // nil: valid
	Backups []Backup
	Notices []string
}

// OK is true when the live file is valid.
func (r Report) OK() bool { return r.LiveErr == nil }

func Check(saveDir, state, name string, free FreeFunc, now time.Time) (Report, error) {
	if err := checkName(name); err != nil {
		return Report{}, err
	}
	r := Report{Live: filepath.Join(saveDir, name+".d2s")}
	if b, err := os.ReadFile(r.Live); err != nil {
		r.LiveErr = err
	} else {
		r.LiveLen, r.LiveErr = int64(len(b)), Valid(b)
	}
	var err error
	r.Backups, err = List(state, name)
	r.Notices = Notices(state, free, now)
	return r, err
}

// Guard runs the rolling backup inside the bot: Observe is called on every poll.
type Guard struct {
	State   string
	SaveDir func() string
	Name    func() string
	Now     func() time.Time
	Free    FreeFunc
	Logf    func(msg string, kv ...any)

	mu      sync.Mutex
	last    time.Time
	inTown  bool
	lastErr string
}

// Observe backs up when the hero has just arrived in town and every Every otherwise. Errors are logged once per
// distinct text (a full disk logs one line, not one a poll) and never returned: the bot keeps playing.
func (g *Guard) Observe(inTown bool) {
	now := g.Now()
	g.mu.Lock()
	arrived := inTown && !g.inTown
	g.inTown = inTown
	due := arrived || g.last.IsZero() || now.Sub(g.last) >= Every
	if due {
		g.last = now
	}
	g.mu.Unlock()
	if !due {
		return
	}
	dir, name := g.SaveDir(), g.Name()
	if dir == "" || name == "" || g.State == "" {
		return
	}
	res, err := Save(dir, g.State, name, g.Free, now)
	msg := ""
	if err != nil && !errors.Is(err, os.ErrNotExist) { // no file yet: a new hero before its first save
		msg = err.Error()
	}
	g.mu.Lock()
	changed := msg != g.lastErr
	g.lastErr = msg
	g.mu.Unlock()
	if g.Logf == nil {
		return
	}
	switch {
	case msg != "" && changed:
		g.Logf("save backup: skipped", "err", msg, "char", name)
	case err == nil && !res.Unchanged:
		g.Logf("save backup: written", "file", res.Path, "bytes", res.Size, "pruned", len(res.Pruned))
	}
}

// DiskLow reports whether the state folder or the save folder has less than MinFree left.
func (g *Guard) DiskLow() bool {
	return Low(g.Free, g.State, g.SaveDir())
}
