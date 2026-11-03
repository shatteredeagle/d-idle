package gclock

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestParseScale(t *testing.T) {
	for in, want := range map[string]float64{
		"": 1, "1": 1, "2": 2, "4": 4, "0.25": 0.25, "16": 16, "0.1": 1, "17": 1, "0": 1, "-2": 1, "x": 1, "NaN": 1, "Inf": 1,
	} {
		if got := ParseScale(in); got != want {
			t.Errorf("ParseScale(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDurationScaling(t *testing.T) {
	for _, k := range []float64{1, 2, 4} {
		var slept time.Duration
		s := New(k)
		s.sleep = func(d time.Duration) { slept += d }
		s.Sleep(8 * time.Second)
		if want := time.Duration(float64(8*time.Second) / k); slept != want {
			t.Errorf("k=%v: slept %v, want %v", k, slept, want)
		}
	}
	if d := New(1).D(1234567 * time.Nanosecond); d != 1234567*time.Nanosecond {
		t.Errorf("k=1 must not touch durations, got %v", d)
	}
}

func TestNowAdvancesByScale(t *testing.T) {
	real := time.Now()
	s := New(4)
	s.base = real
	s.now = func() time.Time { return real.Add(10 * time.Second) }
	if got := s.Now().Sub(real); got != 40*time.Second {
		t.Errorf("k=4 after 10 real s: game elapsed %v, want 40s", got)
	}
	if got := s.Since(real.Add(30 * time.Second)); got != 10*time.Second {
		t.Errorf("Since = %v, want 10s", got)
	}
	s1 := New(1)
	fixed := time.Unix(100, 0)
	s1.now = func() time.Time { return fixed }
	if !s1.Now().Equal(fixed) {
		t.Error("k=1 Now must be the raw clock")
	}
}

func TestWithTimeoutDeadline(t *testing.T) {
	for _, k := range []float64{1, 2, 4} {
		before := time.Now()
		ctx, cancel := New(k).WithTimeout(context.Background(), 40*time.Second)
		dl, _ := ctx.Deadline()
		cancel()
		got := dl.Sub(before)
		want := time.Duration(float64(40*time.Second) / k)
		if math.Abs(float64(got-want)) > float64(time.Second) {
			t.Errorf("k=%v: deadline in %v, want about %v", k, got, want)
		}
	}
	ctx, cancel := New(4).WithTimeoutCause(context.Background(), 40*time.Millisecond, context.Canceled)
	defer cancel()
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("scaled timeout did not fire")
	}
	select {
	case <-New(4).After(40 * time.Millisecond):
	case <-time.After(2 * time.Second):
		t.Fatal("scaled After did not fire")
	}
}

// Game-clock packages must reach time through gclock or an injected clock. A direct call is allowed only where it is
// listed here with the reason (host-clock on purpose). A new, unlisted call fails the test: classify it, then either use
// gclock or add it to the list.
var allowed = map[string]int{
	"internal/input/client.go": 3, // dial backoff, heal cooldown, helper RPC deadline: host
	"internal/bot/loop.go":     2, // recycle and restart deadlines: the game process, host
	"internal/bot/farm.go":     1, // farm duration is a wall-clock goal budget (U: could be game time)
}

var pat = regexp.MustCompile(`\btime\.(Sleep|After|NewTimer|NewTicker|Tick|AfterFunc|Since|Until)\(|context\.With(Timeout|Deadline)(Cause)?\(`)

func TestNoNewDirectTimeCalls(t *testing.T) {
	root := filepath.Join("..", "..")
	got := map[string]int{}
	for _, dir := range []string{"internal/bot", "internal/town", "internal/nav", "internal/combat", "internal/loot",
		"internal/leveler", "internal/talk", "internal/input", "internal/party", "internal/merc", "internal/boss"} {
		filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") ||
				strings.Contains(filepath.ToSlash(p), "/cmd/") {
				return nil
			}
			b, _ := os.ReadFile(p)
			rel := strings.TrimPrefix(filepath.ToSlash(p), "../../")
			for _, ln := range strings.Split(string(b), "\n") {
				if c := strings.TrimSpace(ln); !strings.HasPrefix(c, "//") && pat.MatchString(ln) {
					got[rel]++
				}
			}
			return nil
		})
	}
	for f, n := range got {
		if n > allowed[f] {
			t.Errorf("%s: %d direct time/context-timeout calls, allow-list has %d (use internal/gclock or list it in internal/gclock/gclock_test.go)", f, n, allowed[f])
		}
	}
}

func TestStartNote(t *testing.T) {
	for _, c := range []struct {
		k    float64
		want string
	}{{1, ""}, {2, "gclock: timescale 2"}, {4, "gclock: timescale 4"}, {0.5, "gclock: timescale 0.5"}} {
		if got := StartNote(c.k); got != c.want {
			t.Errorf("StartNote(%v) = %q, want %q", c.k, got, c.want)
		}
	}
}
