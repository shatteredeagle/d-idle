package ladder_test

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// The route question of the t0 drill: the situation lasts, TooHard is asked every minute (the route guard re-plans),
// nobody answers. Without the policy a thought opens about every minute; with it three open, then the key goes quiet
// with one journal line. Both builds.
func TestTooHardDoesNotReopenEveryMinute(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		count := func(policy bool) (thoughts, quiet int, rows []string) {
			l, clk, sink := newLadder(t, nil)
			if policy {
				l.Store().SetRepeatPolicy(decide.DefaultRepeatCooldown, decide.DefaultQuietAfter)
			}
			h := hero(class, 0)
			for i := 0; i < 240; i++ { // four hours
				if got, err := l.TooHard(h, 8, "none", "OK", "skip"); err != nil || got != "skip" {
					t.Fatalf("%v: TooHard = %q, %v", class, got, err)
				}
				clk.Advance(time.Minute)
			}
			for _, r := range sink.Lines() {
				switch {
				case strings.HasPrefix(r, "thought "):
					thoughts++
				case strings.HasPrefix(r, "quiet "):
					quiet++
				}
			}
			return thoughts, quiet, sink.Lines()
		}
		if n, _, _ := count(false); n < 100 {
			t.Errorf("%v: without the policy %d thoughts in 4 h; the defect is gone from the reproduction", class, n)
		}
		n, q, rows := count(true)
		if n != decide.DefaultQuietAfter || q != 1 {
			t.Errorf("%v: with the policy %d thoughts and %d quiet lines in 4 h, want %d and 1\n%s", class, n, q, decide.DefaultQuietAfter, strings.Join(rows, "\n"))
		}
	}
}

// A scripted player who answers once is not asked again within the cooldown, and the answer acts (Stats.ByAnswer).
func TestTooHardAnswerIsReused(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		src := &countSrc{r: ladder.Reply{Answer: "go", Delay: time.Second}}
		l, clk, _ := newLadder(t, src)
		l.Store().SetRepeatPolicy(decide.DefaultRepeatCooldown, decide.DefaultQuietAfter)
		h := hero(class, 0)
		var got []string
		for i := 0; i < 10; i++ {
			a, err := l.TooHard(h, 8, "none", "OK", "skip")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, a)
			clk.Advance(30 * time.Second)
			l.Pump()
		}
		if src.n != 1 {
			t.Errorf("%v: %d questions put, want 1", class, src.n)
		}
		if got[0] != "skip" || got[len(got)-1] != "go" {
			t.Errorf("%v: answers %v, want skip first then the player's go", class, got)
		}
		if st := l.Stats(); st.ByAnswer == 0 {
			t.Errorf("%v: ByAnswer 0", class)
		}
	}
}
