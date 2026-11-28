package logx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"
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

// maxTailLine caps one line of Tail; the rest of a longer line is dropped, the read goes on.
const maxTailLine = 4 << 20

// eachLine calls fn for every line of r with NUL bytes and the line end removed. A crash can leave a hole of NUL bytes
// of any size with no newline in it: the bytes are skipped as they are read, so such a run costs no memory and never
// ends the read (a Scanner stops with "token too long" and loses every newer line).
func eachLine(r io.Reader, fn func(string)) error {
	br := bufio.NewReaderSize(r, 64<<10)
	var line []byte
	for {
		chunk, err := br.ReadSlice('\n')
		for len(chunk) > 0 && len(line) < maxTailLine {
			i := bytes.IndexByte(chunk, 0)
			if i < 0 {
				i = len(chunk)
			}
			line = append(line, chunk[:min(i, maxTailLine-len(line))]...)
			chunk = bytes.TrimLeft(chunk[i:], "\x00")
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == nil || len(line) > 0 {
			fn(strings.TrimRight(string(line), "\r\n"))
		}
		line = line[:0]
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// Tail returns the last n lines (0 = all) of the log at path and its rotations (oldest first) whose text matches re
// (nil = all). Read only.
func Tail(path string, n int, re *regexp.Regexp) ([]string, error) {
	files := append(rotations(path), path)
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
		err = eachLine(fh, func(l string) {
			if l == "" || (re != nil && !re.MatchString(l)) {
				return
			}
			out = append(out, l)
			if n > 0 && len(out) > 4*n+1024 { // bound memory on a long file
				out = append(out[:0], out[len(out)-n:]...)
			}
		})
		fh.Close()
		if err != nil {
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

// rotations lists path.N (N a number) oldest first. Only that shape counts: a sibling such as bot.log.bak or
// bot.log.tmp is not a rotation, and a glob character in path is a plain character.
func rotations(path string) []string {
	ents, _ := os.ReadDir(filepath.Dir(path))
	pfx := filepath.Base(path) + "."
	var m []string
	for _, e := range ents {
		n, ok := strings.CutPrefix(e.Name(), pfx)
		if !ok || n == "" || strings.Trim(n, "0123456789") != "" || !e.Type().IsRegular() {
			continue
		}
		m = append(m, filepath.Join(filepath.Dir(path), e.Name()))
	}
	sort.Slice(m, func(i, j int) bool { return rotNum(m[i]) > rotNum(m[j]) }) // .5 before .1
	return m
}

// Printable makes a log line safe for a terminal: control characters (escape sequences that move the cursor, clear
// the screen or set the title) are shown as \xNN instead of being sent. Tab stays.
func Printable(s string) string {
	clean := true
	for _, r := range s {
		if (r < 0x20 && r != '\t') || (r >= 0x7f && r < 0xa0) || r == utf8.RuneError {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		r, n := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == utf8.RuneError && n == 1:
			fmt.Fprintf(&b, `\x%02x`, s[i])
		case (r < 0x20 && r != '\t') || (r >= 0x7f && r < 0xa0):
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteString(s[i : i+n])
		}
		i += n
	}
	return b.String()
}

func rotNum(p string) int {
	var n int
	fmt.Sscanf(p[strings.LastIndex(p, ".")+1:], "%d", &n)
	return n
}
