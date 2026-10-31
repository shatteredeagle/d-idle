package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Log is an append-only JSONL file with size and age rotation. Zero MaxBytes / MaxAge disable that trigger.
// Rotated files are <Path>.<UTC yyyymmddThhmmss.mmm>; the newest Keep (default 5) are retained.
// Age is measured from the first write through this Log, or from the file's mtime when an existing file is
// adopted (a conservative approximation: the true creation time is not portable). Safe for concurrent use
// within one process.
type Log struct {
	Path     string
	MaxBytes int64
	MaxAge   time.Duration
	Keep     int
	Sync     bool // fsync after every append (default off: logs may lose the tail on power loss)
	Now      func() time.Time

	mu      sync.Mutex
	started time.Time
	size    int64
	open    bool
}

func (l *Log) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

// Append marshals v as one line.
func (l *Log) Append(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return l.AppendLine(b)
}

// AppendLine writes b (which must not contain a newline) as one line.
func (l *Log) AppendLine(b []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	if !l.open {
		l.started, l.size = l.now(), 0
		if st, err := os.Stat(l.Path); err == nil {
			l.started, l.size = st.ModTime(), st.Size()
		}
		l.open = true
	}
	now := l.now()
	if l.size > 0 && ((l.MaxBytes > 0 && l.size+int64(len(b))+1 > l.MaxBytes) || (l.MaxAge > 0 && now.Sub(l.started) >= l.MaxAge)) {
		if err := l.rotate(now); err != nil {
			return err
		}
	}
	fh, err := os.OpenFile(l.Path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	n, err := fh.Write(append(b[:len(b):len(b)], '\n'))
	l.size += int64(n)
	if err == nil && l.Sync {
		err = fh.Sync()
	}
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	return err
}

func (l *Log) rotate(now time.Time) error {
	dst := l.Path + "." + now.UTC().Format("20060102T150405.000")
	for n := 1; ; n++ {
		if _, err := os.Lstat(dst); err != nil {
			break
		}
		dst = l.Path + "." + now.UTC().Format("20060102T150405.000") + "-" + strconv.Itoa(n)
	}
	if err := os.Rename(l.Path, dst); err != nil {
		return err
	}
	l.started, l.size = now, 0
	keep := l.Keep
	if keep <= 0 {
		keep = 5
	}
	old, _ := filepath.Glob(l.Path + ".*")
	var rotated []string
	for _, p := range old {
		if strings.HasPrefix(strings.TrimPrefix(p, l.Path+"."), "20") {
			rotated = append(rotated, p)
		}
	}
	sort.Strings(rotated) // timestamp names sort chronologically
	for len(rotated) > keep {
		os.Remove(rotated[0])
		rotated = rotated[1:]
	}
	return nil
}
