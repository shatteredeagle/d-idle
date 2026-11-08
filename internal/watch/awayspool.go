package watch

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// The away log is memory only. The policy preview (docs/policy-preview.md) reads fight life lows and pickups from it,
// so those two kinds are also appended to a small JSON-lines spool and read back at start: the preview then has data
// right after a restart. Passive: nothing reads the spool except AttachSpool.

const (
	spoolKeep     = 24 * time.Hour // the preview looks back at most 1440 minutes
	spoolMaxBytes = 1 << 20
)

func spooled(kind string) bool { return kind == AwayClose || kind == AwayDrop }

// AttachSpool loads the spooled events younger than spoolKeep onto the log's time base, rewrites the file with just
// those, and appends later close and drop events. An unreadable file is started afresh; a write error only stops the
// spooling (the in-memory log is unaffected).
func (l *AwayLog) AttachSpool(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.wall == nil {
		l.wall = time.Now
	}
	nowWall, nowT := l.wall().Unix(), 0.0
	if l.now != nil {
		nowT = l.now()
	}
	var keep []AwayEvent
	if f, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<16), 1<<20)
		for sc.Scan() {
			var e AwayEvent
			if json.Unmarshal(sc.Bytes(), &e) != nil || !spooled(e.Kind) || e.Wall == 0 {
				continue
			}
			age := nowWall - e.Wall
			if age < 0 || time.Duration(age)*time.Second > spoolKeep {
				continue
			}
			e.T = nowT - float64(age)
			keep = append(keep, e)
		}
		f.Close()
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	for _, e := range keep {
		_ = enc.Encode(e)
	}
	l.spool = out
	l.evs = append(keep, l.evs...)
	return nil
}

// spoolLocked appends one event; l.mu is held.
func (l *AwayLog) spoolLocked(e AwayEvent) {
	if l.spool == nil || !spooled(e.Kind) {
		return
	}
	e.Wall = l.wall().Unix()
	b, _ := json.Marshal(e)
	if st, err := l.spool.Stat(); err != nil || st.Size() > spoolMaxBytes {
		l.spool.Close()
		l.spool = nil
		return
	}
	if _, err := l.spool.Write(append(b, '\n')); err != nil {
		l.spool.Close()
		l.spool = nil
	}
}
