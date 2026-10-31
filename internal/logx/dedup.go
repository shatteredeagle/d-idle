// Package logx holds logging helpers shared by the commands.
package logx

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// DefaultFlush is how long a run of identical lines may stay silent before its counter line is written.
const DefaultFlush = time.Minute

// Exempt reports lines that must never be folded: the ones cmd/d2soak parses one by one (fight and death telemetry)
// and the once-a-minute summary.
func Exempt(r slog.Record) bool {
	return r.Message == "minute" || strings.Contains(r.Message, "fight area=") || strings.Contains(r.Message, "death diff=")
}

// Dedup wraps a handler and folds runs of identical consecutive records (same level, message and attributes) into the
// first line plus one "last line repeated" counter line. Records with different content, or a flush interval without
// output, end the run. Records of WithAttrs/WithGroup children share the state of the parent.
type Dedup struct {
	slog.Handler
	st    *dedupState
	attrs string // rendered pre-attached attrs of this handler, part of the key
}

type dedupState struct {
	mu     sync.Mutex
	flush  time.Duration
	now    func() time.Time
	exempt func(slog.Record) bool
	key    string
	rec    slog.Record
	h      slog.Handler
	n      int // suppressed repeats of key
	first  time.Time
	lastW  time.Time // last time anything about this run was written
}

// NewDedup wraps h. flush <= 0 means DefaultFlush.
func NewDedup(h slog.Handler, flush time.Duration) *Dedup {
	if flush <= 0 {
		flush = DefaultFlush
	}
	return &Dedup{Handler: h, st: &dedupState{flush: flush, now: time.Now, exempt: Exempt}}
}

func (d *Dedup) WithAttrs(a []slog.Attr) slog.Handler {
	var b strings.Builder
	b.WriteString(d.attrs)
	for _, x := range a {
		b.WriteString(x.String())
		b.WriteByte(' ')
	}
	return &Dedup{Handler: d.Handler.WithAttrs(a), st: d.st, attrs: b.String()}
}

func (d *Dedup) WithGroup(name string) slog.Handler {
	return &Dedup{Handler: d.Handler.WithGroup(name), st: d.st, attrs: d.attrs + name + "."}
}

func recordKey(r slog.Record, pre string) string {
	var b strings.Builder
	b.WriteString(pre)
	b.WriteString(r.Level.String())
	b.WriteByte(' ')
	b.WriteString(r.Message)
	r.Attrs(func(a slog.Attr) bool {
		b.WriteByte(' ')
		b.WriteString(a.String())
		return true
	})
	return b.String()
}

func (d *Dedup) Handle(ctx context.Context, r slog.Record) error {
	s := d.st
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.exempt(r) {
		s.flushLocked(ctx)
		s.h, s.key = nil, "" // a line after an exempt one is shown again, not folded into the run before it
		return d.Handler.Handle(ctx, r)
	}
	now := s.now()
	k := recordKey(r, d.attrs)
	if s.h != nil && k == s.key {
		if s.n == 0 {
			s.first = now
		}
		s.n++
		if now.Sub(s.lastW) >= s.flush {
			s.flushLocked(ctx)
			s.lastW = now
		}
		return nil
	}
	s.flushLocked(ctx)
	s.key, s.rec, s.h, s.n, s.lastW = k, r, d.Handler, 0, now
	return d.Handler.Handle(ctx, r)
}

// Flush writes the pending counter line, if any (call it before the process exits).
func (d *Dedup) Flush() {
	d.st.mu.Lock()
	defer d.st.mu.Unlock()
	d.st.flushLocked(context.Background())
}

func (s *dedupState) flushLocked(ctx context.Context) {
	if s.n == 0 || s.h == nil {
		return
	}
	now := s.now()
	r := slog.NewRecord(now, s.rec.Level, "last line repeated", s.rec.PC)
	r.AddAttrs(slog.String("line", s.rec.Message), slog.Int("times", s.n), slog.Duration("in", now.Sub(s.first).Round(time.Second)))
	_ = s.h.Handle(ctx, r)
	s.n = 0
	s.first = now
}
