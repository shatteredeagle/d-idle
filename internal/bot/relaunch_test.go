package bot

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// Every relaunch logs "relaunch reason=.. cause=.." once and lands in the minute line by reason; a Leave right after a
// carried-out chicken is told as the chicken, a Leave without one as "leave" (7 of 13 chickens read as relaunches in the
// 2026-11-27 sandbox logs, and nothing said which). Two heroes (life lines differ).
func TestRelaunchReasonsAreLoggedAndCounted(t *testing.T) {
	for _, h := range invHeroes {
		g := newRig(t)
		var buf bytes.Buffer
		g.r.e.Log = slog.New(slog.NewTextHandler(&buf, nil))

		g.r.noteChicken("life=12/" + h.name)
		g.r.NoteLeave() // the chicken's Leave
		g.r.NoteLeave() // a second one with no chicken before it
		g.r.noteRelaunch(RelaunchNewGame, "from=x.go:1")
		g.clock.Sleep(chickenLeaveWindow + time.Second)
		g.r.noteChicken("late")
		g.clock.Sleep(chickenLeaveWindow + time.Second)
		g.r.NoteLeave() // too late to be that chicken

		out := buf.String()
		for _, want := range []string{"reason=chicken cause=\"life=12/" + h.name + "\"", "reason=leave", "reason=newgame cause=\"from=x.go:1\""} {
			if !strings.Contains(out, want) {
				t.Errorf("%s: log lacks %q:\n%s", h.name, want, out)
			}
		}
		if n := strings.Count(out, "reason=chicken"); n != 1 {
			t.Errorf("%s: %d chicken lines, want 1", h.name, n)
		}
		kv := g.r.relaunchKV()
		if len(kv) != 4 || kv[1] != 4 || kv[3] != "chicken=1,leave=2,newgame=1" {
			t.Errorf("%s: minute kv %v", h.name, kv)
		}
	}
}

// The storm line names the refusals that repeated, most frequent first, at most three; the tally restarts after progress.
func TestRefusalStormNamesTheRepeatedRefusals(t *testing.T) {
	for _, h := range invHeroes {
		m := &invMon{}
		f := h.facts(t0)
		runInv(m, f, 2, nil)
		for i := 0; i < 5; i++ {
			m.noteRefusal("countess: not ready: tower key")
		}
		for i := 0; i < 3; i++ {
			m.noteRefusal("countess: area gated")
		}
		m.noteRefusal("a")
		m.noteRefusal("b")
		f.now = f.now.Add(time.Second)
		var detail string
		for _, x := range m.tick(f) {
			if x.Name == "refusal-storm" {
				detail = x.Detail
			}
		}
		if !strings.Contains(detail, `top=["countess: not ready: tower key" x5; "countess: area gated" x3; "a" x1]`) {
			t.Errorf("%s: detail %q", h.name, detail)
		}
		if len(m.refWhy) != 0 {
			t.Errorf("%s: tally not cleared", h.name)
		}
	}
}
