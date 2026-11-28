package vault

import (
	"errors"
	"sync"
	"testing"
)

// Card vault2: a reserved record is either sold by its order or released; nothing else moves it. Two hero stores (the
// store is per hero; nothing in it is build-specific).
func TestReserveBlocksEveryOtherTransition(t *testing.T) {
	for _, hero := range []string{"sorc", "pala"} {
		t.Run(hero, func(t *testing.T) {
			dir := t.TempDir()
			s := open(t, dir, hero, 50)
			a, _ := s.Add(mk(1, "cap", 4, 10))
			b, _ := s.Add(mk(2, "rin", 4, 10))
			if bad, err := s.Reserve([]string{a.ID, b.ID}, "s-1", s.Version()); err != nil {
				t.Fatalf("reserve: %s %v", bad, err)
			}
			for name, op := range map[string]func() error{
				"keep":    func() error { return s.Keep(a.ID) },
				"pin":     func() error { return s.Pin(a.ID) },
				"discard": func() error { return s.Discard(a.ID, "x") },
				"sellq":   func() error { return s.MarkSell(a.ID) },
				"revq":    func() error { return s.RevealWant(a.ID) },
				"stage":   func() error { return s.SetStage(a.ID, StageRevBegin, 0) },
				"other":   func() error { return s.SellNow(a.ID, 5, "s-2") },
				"reveal":  func() error { return s.Reveal(a.ID, mk(9, "cap", 4, 10)) },
			} {
				if err := op(); !errors.Is(err, ErrHeld) || !errors.Is(err, ErrState) {
					t.Fatalf("%s on a reserved record: %v", name, err)
				}
			}
			if bad, err := s.Reserve([]string{a.ID}, "s-2", 0); !errors.Is(err, ErrHeld) || bad != a.ID {
				t.Fatalf("second order reserved it: %q %v", bad, err)
			}
			// the hold survives a restart (it is in the log, and in the snapshot)
			must(t, s.Close())
			s = open(t, dir, hero, 50)
			if h := s.Held()["s-1"]; len(h) != 2 {
				t.Fatalf("held after reopen: %v", s.Held())
			}
			if done, failed, err := s.SellNowBatch([]SoldLine{{ID: a.ID, Gold: 7}}, "s-1"); done != 1 || failed != 0 || err != nil {
				t.Fatalf("own order could not sell: %d %d %v", done, failed, err)
			}
			if n, err := s.Release("s-1"); n != 1 || err != nil {
				t.Fatalf("release: %d %v", n, err)
			}
			must(t, s.Keep(b.ID))
			if r, _ := s.Get(a.ID); r.State != Sold || r.Hold != "" {
				t.Fatalf("sold record: %+v", r)
			}
		})
	}
}

func TestReserveIsAllOrNoneAndVersionChecked(t *testing.T) {
	s := open(t, t.TempDir(), "sorc", 50)
	a, _ := s.Add(mk(1, "cap", 4, 10))
	b, _ := s.Add(mk(2, "cap", 4, 10))
	ver := s.Version()
	must(t, s.Pin(b.ID))
	if _, err := s.Reserve([]string{a.ID, b.ID}, "s-1", ver); !errors.Is(err, ErrStale) {
		t.Fatalf("stale version accepted: %v", err)
	}
	if bad, err := s.Reserve([]string{a.ID, b.ID}, "s-1", s.Version()); !errors.Is(err, ErrPinned) || bad != b.ID {
		t.Fatalf("pinned record reserved: %q %v", bad, err)
	}
	if len(s.Held()) != 0 {
		t.Fatalf("a refused reservation held something: %v", s.Held())
	}
	if _, err := s.Reserve([]string{a.ID, "v-nope"}, "s-1", 0); !errors.Is(err, ErrNotFound) || len(s.Held()) != 0 {
		t.Fatalf("missing id: %v %v", err, s.Held())
	}
	// a short write in the middle of the batch: nothing stays held
	must(t, s.Unpin(b.ID))
	n := 0
	s.hook = func(line []byte) ([]byte, error) {
		if n++; n == 2 {
			return nil, errors.New("disk full")
		}
		return line, nil
	}
	if _, err := s.Reserve([]string{a.ID, b.ID}, "s-1", 0); err == nil || len(s.Held()) != 0 {
		t.Fatalf("partial reservation kept: %v %v", err, s.Held())
	}
	s.hook = nil
	// the trim never takes a reserved record
	if _, err := s.Reserve([]string{a.ID}, "s-1", 0); err != nil {
		t.Fatal(err)
	}
	if c := s.trimCandidate(); c == nil || c.ID == a.ID {
		t.Fatalf("trim candidate %+v", c)
	}
}

// The race the card closes: a sale reserves while Keep, Pin and Discard are called for the same record. Exactly one side
// wins, whatever the interleaving (run with -race on the worker host).
func TestReserveRaceOneWinner(t *testing.T) {
	for i := 0; i < 50; i++ {
		s := open(t, t.TempDir(), "sorc", 50)
		a, _ := s.Add(mk(1, "cap", 4, 10))
		var wg sync.WaitGroup
		var sold, other error
		ops := []func() error{func() error { return s.Keep(a.ID) }, func() error { return s.Pin(a.ID) }, func() error { return s.Discard(a.ID, "x") }}
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, sold = s.Reserve([]string{a.ID}, "s-1", 0); sold == nil {
				_, _, sold = s.SellNowBatch([]SoldLine{{ID: a.ID, Gold: 3}}, "s-1")
			}
		}()
		go func() { defer wg.Done(); other = ops[i%3]() }()
		wg.Wait()
		r, _ := s.Get(a.ID)
		if (sold == nil) == (other == nil) {
			t.Fatalf("round %d: sale err %v, other err %v, state %s", i, sold, other, r.State)
		}
		if sold == nil && r.State != Sold {
			t.Fatalf("round %d: reserved and credited, yet %s", i, r.State)
		}
		s.Close()
	}
}
