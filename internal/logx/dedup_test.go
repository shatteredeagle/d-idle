package logx

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func newTest(flush time.Duration) (*slog.Logger, *Dedup, *bytes.Buffer, *time.Time) {
	var buf bytes.Buffer
	d := NewDedup(slog.NewTextHandler(&buf, &slog.HandlerOptions{ReplaceAttr: func(g []string, a slog.Attr) slog.Attr {
		if len(g) == 0 && a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}}), flush)
	now := time.Unix(1000, 0)
	d.st.now = func() time.Time { return now }
	return slog.New(d), d, &buf, &now
}

func TestFoldsIdenticalRuns(t *testing.T) {
	l, d, buf, now := newTest(time.Hour)
	for range 500 {
		l.Info("nav exit", "to", 2)
		*now = now.Add(10 * time.Millisecond)
	}
	l.Info("other")
	out := buf.String()
	if strings.Count(out, "msg=\"nav exit\"") != 1 {
		t.Fatalf("want one nav exit line:\n%s", out)
	}
	if !strings.Contains(out, "times=499") || !strings.Contains(out, "line=\"nav exit\"") {
		t.Fatalf("counter line missing:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimSpace(out), "msg=other") {
		t.Fatalf("counter must precede the next line:\n%s", out)
	}
	d.Flush()
}

func TestDifferentAttrsNotFolded(t *testing.T) {
	l, _, buf, _ := newTest(time.Hour)
	l.Info("x", "a", 1)
	l.Info("x", "a", 2)
	l.Info("x", "a", 2)
	l.Warn("x", "a", 2)
	if n := strings.Count(buf.String(), "msg=x"); n != 3 {
		t.Fatalf("want 3 lines (one folded), got %d:\n%s", n, buf.String())
	}
}

func TestPeriodicFlushAndFinalFlush(t *testing.T) {
	l, d, buf, now := newTest(time.Minute)
	for range 10 {
		l.Info("retry")
		*now = now.Add(20 * time.Second)
	}
	if !strings.Contains(buf.String(), "last line repeated") {
		t.Fatalf("no periodic counter:\n%s", buf.String())
	}
	before := strings.Count(buf.String(), "repeated")
	d.Flush()
	if strings.Count(buf.String(), "repeated") < before {
		t.Fatal("flush lost lines")
	}
}

func TestExemptAndWithAttrs(t *testing.T) {
	l, _, buf, _ := newTest(time.Hour)
	l = l.With("inst", "i1")
	for range 3 {
		l.Info("combat fight area=1 kills=3")
		l.Warn("death diff=hell area=1")
		l.Info("minute", "kills", 1)
	}
	if n := strings.Count(buf.String(), "\n"); n != 9 {
		t.Fatalf("exempt lines must all pass, got %d lines:\n%s", n, buf.String())
	}
	l2, _, b2, _ := newTest(time.Hour)
	a, b := l2.With("inst", "1"), l2.With("inst", "2")
	a.Info("m")
	b.Info("m")
	if strings.Count(b2.String(), "msg=m") != 2 {
		t.Fatal("different handler attrs must not fold")
	}
}
