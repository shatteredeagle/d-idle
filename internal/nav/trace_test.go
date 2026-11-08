package nav

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/input"
)

func TestWalkTraceOffAndOn(t *testing.T) {
	for _, on := range []bool{false, true} {
		var buf bytes.Buffer
		n := &Navigator{log: slog.New(slog.NewTextHandler(&buf, nil)), Trace: on}
		tr := &walkTrace{n: n}
		t0 := time.Unix(100, 0)
		tr.poll(input.Point{X: 0, Y: 0}, t0, 300*time.Millisecond, false)
		tr.click(input.Point{X: 0, Y: 0}, input.Point{X: 10, Y: 0}, t0, "first", 50*time.Millisecond, 0, 3)
		tr.poll(input.Point{X: 0, Y: 0}, t0.Add(time.Second), 300*time.Millisecond, true)
		tr.poll(input.Point{X: 9, Y: 0}, t0.Add(2*time.Second), 300*time.Millisecond, false)
		tr.end(input.Point{X: 9, Y: 0}, t0.Add(2*time.Second), nil)
		out := buf.String()
		if !on {
			if out != "" {
				t.Fatalf("trace off must log nothing, got %q", out)
			}
			continue
		}
		for _, want := range []string{"nav.trace click", "nav.trace stall", "nav.trace resume", "nav.trace leg", "pauseMs=1000"} {
			if !strings.Contains(out, want) {
				t.Errorf("missing %q in\n%s", want, out)
			}
		}
	}
}
