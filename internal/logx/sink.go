package logx

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Rotating is a size-capped log file: when a write would pass Max bytes the file is renamed to name.1 (name.1 to
// name.2 and so on, the oldest of Keep dropped) and a fresh file is opened. Safe for concurrent use.
type Rotating struct {
	mu   sync.Mutex
	path string
	max  int64
	keep int
	f    *os.File
	size int64
}

// OpenRotating opens (appending) the file at path, creating its directory. max<=0 or keep<=0 select 20 MiB and 5.
func OpenRotating(path string, max int64, keep int) (*Rotating, error) {
	if max <= 0 {
		max = 20 << 20
	}
	if keep <= 0 {
		keep = 5
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	r := &Rotating{path: path, max: max, keep: keep}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Rotating) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.f, r.size = f, st.Size()
	return nil
}

func (r *Rotating) rotate() error {
	r.f.Close()
	for i := r.keep - 1; i >= 1; i-- {
		os.Rename(fmt.Sprintf("%s.%d", r.path, i), fmt.Sprintf("%s.%d", r.path, i+1))
	}
	if err := os.Rename(r.path, r.path+".1"); err != nil {
		return err
	}
	return r.open()
}

func (r *Rotating) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.size > 0 && r.size+int64(len(p)) > r.max {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

// Name is the path of the live file.
func (r *Rotating) Name() string { return r.path }

// Close closes the file.
func (r *Rotating) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.f.Close()
}

// Tee writes every record to all writers and never lets one fail the others: a closed or full stdout pipe must not
// silence the file (that is the failure that left the sandboxes without a bot log for two days), and a full disk
// must not stop stdout. The error of a write is reported as success.
func Tee(ws ...io.Writer) io.Writer { return tee(ws) }

type tee []io.Writer

func (t tee) Write(p []byte) (int, error) {
	for _, w := range t {
		w.Write(p) //nolint:errcheck // best effort by design, see Tee
	}
	return len(p), nil
}

// Tail returns the last n lines (0 = all) of the log at path and its rotations (oldest first) whose text matches re
// (nil = all). Read only.
func Tail(path string, n int, re *regexp.Regexp) ([]string, error) {
	files := []string{path}
	if m, _ := filepath.Glob(path + ".*"); len(m) > 0 {
		sort.Slice(m, func(i, j int) bool { return rotNum(m[i]) > rotNum(m[j]) }) // .5 before .1: oldest first
		files = append(m, path)
	}
	var out []string
	found := false
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		found = true
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 64<<10), 4<<20)
		for sc.Scan() {
			l := strings.TrimRight(strings.ReplaceAll(sc.Text(), "\x00", ""), "\r")
			if l == "" || (re != nil && !re.MatchString(l)) {
				continue
			}
			out = append(out, l)
			if n > 0 && len(out) > 4*n+1024 { // bound memory on a long file
				out = append(out[:0], out[len(out)-n:]...)
			}
		}
		fh.Close()
		if err := sc.Err(); err != nil {
			return out, err
		}
	}
	if !found {
		return nil, fmt.Errorf("no log at %s", path)
	}
	if n > 0 && len(out) > n {
		out = out[len(out)-n:]
	}
	return out, nil
}

func rotNum(p string) int {
	var n int
	fmt.Sscanf(p[strings.LastIndex(p, ".")+1:], "%d", &n)
	return n
}
