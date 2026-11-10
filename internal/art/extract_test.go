package art

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestKeys(t *testing.T) {
	k := Keys([]string{"invtst", "invtst", "BAD NAME", "invexp"})
	has := map[string]bool{}
	for _, x := range k {
		if has[x] {
			t.Errorf("duplicate %s", x)
		}
		has[x] = true
		if !ValidKey(x) {
			t.Errorf("invalid key %s", x)
		}
	}
	for _, want := range []string{"panel-inv", "slot-helm", "slot-boots", "item-invtst", "item-invexp"} {
		if !has[want] {
			t.Errorf("missing %s", want)
		}
	}
	if len(k) != 1+len(slotFiles)+2 {
		t.Errorf("%d keys", len(k))
	}
}

// Extract fills the disk cache, counts art the install lacks, and a second run decodes nothing.
func TestExtractIdempotent(t *testing.T) {
	cache := t.TempDir()
	s, err := Open(synthInstall(t), cache)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if s.Progress().State != StateIdle {
		t.Errorf("before: %+v", s.Progress())
	}
	keys := Keys([]string{"invtst", "invexp", "invnone"})
	var last int
	p, err := s.Extract(context.Background(), keys, func(d, tot int) { last = d })
	if err != nil {
		t.Fatal(err)
	}
	// synthetic install: slot-helm, slot-gloves, panel-inv, item-invtst, item-invexp exist; the other slots and invnone do not
	wantMissing := len(slotFiles) - 2 + 1
	if p.State != StateReady || p.Total != len(keys) || p.Done != len(keys) || p.Missing != wantMissing || last != len(keys) {
		t.Fatalf("%+v last %d want missing %d", p, last, wantMissing)
	}
	if len(s.mem) != 0 {
		t.Errorf("Extract kept %d PNGs in memory", len(s.mem))
	}
	files, _ := filepath.Glob(filepath.Join(s.dir, "*.png"))
	if len(files) != len(keys)-wantMissing {
		t.Fatalf("%d files", len(files))
	}
	// idempotent: remove the archives' ability to decode by corrupting nothing; instead check mtimes stay.
	st0, _ := os.Stat(files[0])
	time.Sleep(10 * time.Millisecond)
	if _, err := s.Extract(context.Background(), keys, nil); err != nil {
		t.Fatal(err)
	}
	if st1, _ := os.Stat(files[0]); !st1.ModTime().Equal(st0.ModTime()) {
		t.Error("second run rewrote a cached file")
	}
	// the page path serves an extracted key from disk
	if b, err := s.PNG("item-invtst"); err != nil || len(b) == 0 {
		t.Errorf("PNG after extract: %v", err)
	}
}

func TestExtractCancelAndBackground(t *testing.T) {
	s, err := Open(synthInstall(t), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Extract(ctx, Keys(nil), nil); err == nil {
		t.Error("cancelled run returned nil")
	}
	done := make(chan Progress, 1)
	s.StartExtract(Keys([]string{"invtst"}), func(p Progress, err error) {
		if err != nil {
			t.Error(err)
		}
		done <- p
	})
	select {
	case p := <-done:
		if p.State != StateReady || p.Done != p.Total {
			t.Errorf("%+v", p)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("background extraction did not finish")
	}
	s.Close() // must not hang
}
