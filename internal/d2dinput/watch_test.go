package d2dinput_test

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dinput"
)

// The watcher runs on the stepping goroutine, once per crossed interval, after the step that crossed it; a Sleep inside
// it steps the game but does not re-enter it; stop unregisters it.
func TestPacerWatchRunsOnTheSteppingGoroutine(t *testing.T) {
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	srv, err := fake.New(path, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	defer c.Close()
	p := &d2dinput.Pacer{C: c, Chunk: 5}

	var at []time.Duration
	nested := 0
	stop := p.Watch(400*time.Millisecond, func() { // 10 ticks
		_, _, v := p.Stats()
		at = append(at, v)
		if len(at) == 1 {
			nested++
			p.Sleep(2 * time.Second) // 50 ticks inside the watcher: steps, but must not re-enter
		}
	})
	p.Sleep(1 * time.Second) // 25 ticks in chunks of 5: crossings at 400 (fires, sleeps 2 s inside), then 800..
	if len(at) < 2 || at[0] != 400*time.Millisecond {
		t.Fatalf("watch calls at %v", at)
	}
	// the nested Sleep crossed 2400..3000 without re-entering; the outer Sleep's remaining chunks cross 3200 at the latest
	for i := 1; i < len(at); i++ {
		if at[i] <= 2000*time.Millisecond {
			t.Fatalf("re-entered during the nested sleep: %v", at)
		}
	}
	if nested != 1 {
		t.Fatalf("nested %d", nested)
	}
	n := len(at)
	stop()
	p.Sleep(2 * time.Second)
	if len(at) != n {
		t.Fatalf("watcher ran after stop: %v", at)
	}
	_, ticks, _ := p.Stats()
	if ticks != 25+50+50 {
		t.Fatalf("ticks %d", ticks)
	}
}
