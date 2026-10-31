package metrics

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"d2lod-bot/internal/event"
)

func TestExposition(t *testing.T) {
	r := NewRegistry()
	r.Counter("d2_kills_total", "Kills.", "boss", "no").Add(3)
	r.Counter("d2_kills_total", "Kills.", "boss", "yes").Inc()
	r.Counter("d2_kills_total", "", "boss", "yes").Add(-5) // ignored
	g := r.Gauge("d2_life", "Life.")
	g.Set(10)
	g.Add(-2.5)
	r.GaugeFunc("d2_fn", "Fn.", func() float64 { return 7 })
	h := r.Histogram("d2_step_seconds", "Step.", []float64{0.1, 1}, "op", `a"b`)
	h.Observe(0.05)
	h.Observe(0.5)
	h.Observe(3)
	r.AddCollector(func(c *Collect) { c.Gauge("d2_col", "Col.", 1, "x", "1"); c.Gauge("d2_col", "Col.", 2, "x", "2") })

	var b bytes.Buffer
	if _, err := r.WriteTo(&b); err != nil {
		t.Fatal(err)
	}
	want := `# HELP d2_col Col.
# TYPE d2_col gauge
d2_col{x="1"} 1
d2_col{x="2"} 2
# HELP d2_fn Fn.
# TYPE d2_fn gauge
d2_fn 7
# HELP d2_kills_total Kills.
# TYPE d2_kills_total counter
d2_kills_total{boss="no"} 3
d2_kills_total{boss="yes"} 1
# HELP d2_life Life.
# TYPE d2_life gauge
d2_life 7.5
# HELP d2_step_seconds Step.
# TYPE d2_step_seconds histogram
d2_step_seconds_bucket{op="a\"b",le="0.1"} 1
d2_step_seconds_bucket{op="a\"b",le="1"} 2
d2_step_seconds_bucket{op="a\"b",le="+Inf"} 3
d2_step_seconds_sum{op="a\"b"} 3.55
d2_step_seconds_count{op="a\"b"} 3
`
	if b.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestHandler(t *testing.T) {
	r := NewRegistry()
	r.RegisterRuntime()
	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/anything")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	b.ReadFrom(resp.Body)
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain; version=0.0.4") || !strings.Contains(b.String(), "go_goroutines ") {
		t.Fatalf("bad response %v\n%s", resp.Header, b.String())
	}
	if !strings.Contains(b.String(), "process_open_fds ") {
		t.Fatal("no process stats")
	}
	resp2, _ := http.Post(srv.URL, "text/plain", nil)
	if resp2.StatusCode != 405 {
		t.Fatalf("POST = %d", resp2.StatusCode)
	}
}

func TestTypeMismatchPanics(t *testing.T) {
	r := NewRegistry()
	r.Counter("x", "")
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	r.Gauge("x", "")
}

func fakeProc(t *testing.T, root string, pid int, comm, state string) {
	t.Helper()
	d := filepath.Join(root, itoa(pid))
	os.MkdirAll(filepath.Join(d, "fd"), 0o755)
	for _, n := range []string{"0", "1", "2"} {
		os.WriteFile(filepath.Join(d, "fd", n), nil, 0o644)
	}
	// fields 3..22: state ppid pgrp session tty tpgid flags minflt cminflt majflt cmajflt utime stime cutime cstime prio nice threads itrealvalue starttime
	stat := itoa(pid) + " (" + comm + ") " + state + " 1 1 1 0 -1 0 0 0 0 0 150 50 0 0 20 0 9 0 777 1 1\n"
	os.WriteFile(filepath.Join(d, "stat"), []byte(stat), 0o644)
	os.WriteFile(filepath.Join(d, "status"), []byte("Name:\tx\nVmRSS:\t   2048 kB\n"), 0o644)
	os.WriteFile(filepath.Join(d, "smaps_rollup"), []byte("Rss: 2048 kB\nPss:   1024 kB\n"), 0o644)
	os.WriteFile(filepath.Join(d, "comm"), []byte(comm+"\n"), 0o644)
	os.WriteFile(filepath.Join(d, "cmdline"), []byte("Z:\\game\\"+comm+"\x00-w\x00"), 0o644)
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestReadProcAt(t *testing.T) {
	root := t.TempDir()
	fakeProc(t, root, 42, "Game (x).exe", "S")
	p, err := ReadProcAt(root, 42)
	if err != nil {
		t.Fatal(err)
	}
	if p.Comm != "Game (x).exe" || p.State != "S" || p.RSSKB != 2048 || p.PSSKB != 1024 || p.FDs != 3 ||
		p.Threads != 9 || p.CPUTicks() != 200 || p.Start != 777 {
		t.Fatalf("%+v", p)
	}
	if _, err := ReadProcAt(root, 43); err != ErrNoProc {
		t.Fatalf("err = %v", err)
	}
	fakeProc(t, root, 50, "Xvfb", "Z")
	got := FindPIDs(root, func(c, cmd string) bool { return strings.Contains(cmd, "Game") })
	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("FindPIDs = %v", got)
	}
}

func TestReadSelf(t *testing.T) {
	p, err := Self()
	if err != nil {
		t.Skip("no /proc:", err)
	}
	if p.RSSKB <= 0 || p.Threads <= 0 || p.State == "" {
		t.Fatalf("%+v", p)
	}
}

func TestLogger(t *testing.T) {
	var b bytes.Buffer
	l := WithGame(NewLogger(&b, Ident{Inst: "i4", Run: "abc"}, slog.LevelInfo), 3)
	LogEvent(l, event.Event{Seq: 9, Kind: event.KindDeath, Source: event.SrcObserver})
	LogEvent(l, event.Event{Seq: 10, Kind: event.KindKill}) // debug: filtered
	s := b.String()
	for _, w := range []string{`"inst":"i4"`, `"run":"abc"`, `"game":3`, `"msg":"death"`, `"seq":9`, `"level":"WARN"`} {
		if !strings.Contains(s, w) {
			t.Errorf("missing %s in %s", w, s)
		}
	}
	if strings.Contains(s, `"seq":10`) {
		t.Error("debug line not filtered")
	}
	if len(NewRunID()) != 8 {
		t.Error("run id")
	}
}
