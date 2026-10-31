package saves

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const stampLayout = "20060102T150405Z"

var stampRE = regexp.MustCompile(`^\d{8}T\d{6}Z(-\d+)?$`)

// Info describes one snapshot.
type Info struct {
	ID    string // directory name, UTC stamp
	Dir   string
	Time  time.Time
	Files []string
}

// Result of Snapshot.
type Result struct {
	Info
	Unchanged bool // identical to the newest snapshot: nothing written
	Pruned    []string
}

// Snapshot copies name.d2s/.key/.map/.ma* from saveDir into backupDir/<name>/<UTC stamp>/, after the
// source verified. It refuses while the game is in a game (Options), refuses a corrupt .d2s (so a good
// snapshot is never rotated out by bad ones), copies nothing if the .d2s and .map equal the newest
// snapshot, then prunes: newest Keep plus the newest of each of the last Days UTC days.
// The copy is staged in a temp dir and renamed; the source is re-stat'ed after reading so a write
// racing the copy is detected (one retry).
func Snapshot(saveDir, backupDir, name string, o Options, now time.Time) (Result, error) {
	if err := checkName(name); err != nil {
		return Result{}, err
	}
	if err := o.gate(); err != nil {
		return Result{}, err
	}
	var files map[string][]byte
	var err error
	for try := 0; try < 2; try++ {
		if files, err = readCharFiles(saveDir, name); err == nil {
			break
		}
	}
	if err != nil {
		return Result{}, err
	}
	if _, err := Verify(files[name+".d2s"]); err != nil {
		return Result{}, fmt.Errorf("saves: source %s.d2s is corrupt, not snapshotting: %w", name, err)
	}
	root := filepath.Join(backupDir, name)
	if prev, err := List(backupDir, name); err == nil && len(prev) > 0 {
		last := prev[0]
		same := true
		for _, ext := range []string{".d2s", ".map"} {
			old, _ := os.ReadFile(filepath.Join(last.Dir, name+ext))
			if !bytes.Equal(old, files[name+ext]) {
				same = false
			}
		}
		if same {
			return Result{Info: last, Unchanged: true}, nil
		}
	}
	now = now.UTC()
	id := now.Format(stampLayout)
	for n := 2; exists(filepath.Join(root, id)); n++ {
		id = fmt.Sprintf("%s-%d", now.Format(stampLayout), n)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return Result{}, err
	}
	tmp, err := os.MkdirTemp(root, ".tmp-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(tmp)
	var names []string
	for f, b := range files {
		if err := os.WriteFile(filepath.Join(tmp, f), b, 0o644); err != nil {
			return Result{}, err
		}
		names = append(names, f)
	}
	sort.Strings(names)
	dir := filepath.Join(root, id)
	if err := os.Rename(tmp, dir); err != nil {
		return Result{}, err
	}
	res := Result{Info: Info{ID: id, Dir: dir, Time: now, Files: names}}
	res.Pruned, err = Prune(backupDir, name, o, now)
	return res, err
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// readCharFiles reads every character file and fails if any changed (size or mtime) while reading.
func readCharFiles(dir, name string) (map[string][]byte, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	type stamp struct {
		size int64
		mt   time.Time
	}
	before := map[string]stamp{}
	out := map[string][]byte{}
	for _, e := range ents {
		if e.IsDir() || !isCharFile(e.Name(), name) {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			return nil, err
		}
		before[e.Name()] = stamp{fi.Size(), fi.ModTime()}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		out[e.Name()] = b
	}
	if _, ok := out[name+".d2s"]; !ok {
		return nil, fmt.Errorf("%w: %s.d2s in %s", ErrNoFiles, name, dir)
	}
	for f, s := range before {
		fi, err := os.Stat(filepath.Join(dir, f))
		if err != nil || fi.Size() != s.size || !fi.ModTime().Equal(s.mt) {
			return nil, fmt.Errorf("saves: %s changed while copying", f)
		}
	}
	return out, nil
}

// List returns the snapshots of name, newest first.
func List(backupDir, name string) ([]Info, error) {
	if err := checkName(name); err != nil {
		return nil, err
	}
	root := filepath.Join(backupDir, name)
	ents, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Info
	for _, e := range ents {
		if !e.IsDir() || !stampRE.MatchString(e.Name()) {
			continue
		}
		t, err := time.Parse(stampLayout, strings.SplitN(e.Name(), "-", 2)[0])
		if err != nil {
			continue
		}
		in := Info{ID: e.Name(), Dir: filepath.Join(root, e.Name()), Time: t}
		sub, _ := os.ReadDir(in.Dir)
		for _, s := range sub {
			in.Files = append(in.Files, s.Name())
		}
		out = append(out, in)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

// Prune applies the retention policy and returns the removed IDs.
func Prune(backupDir, name string, o Options, now time.Time) ([]string, error) {
	all, err := List(backupDir, name)
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	seenDay := map[string]bool{}
	cutoff := now.UTC().AddDate(0, 0, -o.days())
	for i, in := range all { // newest first
		if i < o.keep() {
			keep[in.ID] = true
		}
		day := in.Time.Format("20060102")
		if !seenDay[day] {
			seenDay[day] = true
			if in.Time.After(cutoff) {
				keep[in.ID] = true
			}
		}
	}
	var gone []string
	for _, in := range all {
		if !keep[in.ID] {
			if err := os.RemoveAll(in.Dir); err != nil {
				return gone, err
			}
			gone = append(gone, in.ID)
		}
	}
	return gone, nil
}

// Restore copies snapshot id (default: newest) back into saveDir. The snapshot's .d2s must verify.
// The current files are first copied to <backupDir>/<name>/replaced-<stamp>/ (never pruned), each file is
// written to a temp name and renamed. Refused while in a game.
func Restore(saveDir, backupDir, name, id string, o Options, now time.Time) (Info, error) {
	if err := checkName(name); err != nil {
		return Info{}, err
	}
	if err := o.gate(); err != nil {
		return Info{}, err
	}
	all, err := List(backupDir, name)
	if err != nil {
		return Info{}, err
	}
	var pick *Info
	for i := range all {
		if id == "" || all[i].ID == id {
			pick = &all[i]
			break
		}
	}
	if pick == nil {
		return Info{}, fmt.Errorf("saves: no snapshot %q for %s", id, name)
	}
	src := filepath.Join(pick.Dir, name+".d2s")
	if _, err := VerifyFile(src); err != nil {
		return Info{}, fmt.Errorf("saves: snapshot %s is corrupt: %w", pick.ID, err)
	}
	if cur, err := readLoose(saveDir, name); err == nil && len(cur) > 0 {
		rep := filepath.Join(backupDir, name, "replaced-"+now.UTC().Format(stampLayout))
		if err := os.MkdirAll(rep, 0o755); err != nil {
			return Info{}, err
		}
		for f, b := range cur {
			if err := os.WriteFile(filepath.Join(rep, f), b, 0o644); err != nil {
				return Info{}, err
			}
		}
	}
	for _, f := range pick.Files {
		b, err := os.ReadFile(filepath.Join(pick.Dir, f))
		if err != nil {
			return Info{}, err
		}
		tmp := filepath.Join(saveDir, "."+f+".restore")
		if err := os.WriteFile(tmp, b, 0o644); err != nil {
			return Info{}, err
		}
		if err := os.Rename(tmp, filepath.Join(saveDir, f)); err != nil {
			return Info{}, err
		}
	}
	return *pick, nil
}

func readLoose(dir, name string) (map[string][]byte, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, e := range ents {
		if e.Type().IsRegular() && isCharFile(e.Name(), name) {
			b, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				return nil, err
			}
			out[e.Name()] = b
		}
	}
	return out, nil
}
