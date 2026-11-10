package art

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

// Extraction states reported by Progress.
const (
	StateIdle       = "idle"       // no extraction was started
	StateExtracting = "extracting" // a background or one-shot run is decoding
	StateReady      = "ready"      // the last run finished; Missing counts icons the install has no art for
)

// Progress is the state of the pre-extraction (the page shows "preparing art Done/Total").
type Progress struct {
	State   string `json:"state"`
	Done    int    `json:"done"`
	Total   int    `json:"total"`
	Missing int    `json:"missing,omitempty"` // keys the install has no art for (the page keeps its drawn cell)
}

// Keys is every key the UI can request: the panel frame, the eight slot silhouettes and one icon per name in
// invFiles (the tables' icon names, db.InvFiles). Names outside the key grammar are skipped. Sorted, no duplicates.
func Keys(invFiles []string) []string {
	seen := map[string]bool{"panel-inv": true}
	for n := range slotFiles {
		seen["slot-"+n] = true
	}
	for _, n := range invFiles {
		if k := "item-" + n; ValidKey(k) {
			seen[k] = true
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// cached reports whether the PNG of key is already on disk or in memory, without reading it.
func (s *Store) cached(key string) bool {
	s.mu.Lock()
	_, ok := s.mem[key]
	s.mu.Unlock()
	if ok {
		return true
	}
	if s.dir == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(s.dir, key+".png"))
	return err == nil && st.Size() > 0
}

// Extract decodes every key into the disk cache that is not there yet; keys already cached cost one stat, so a
// second run is idempotent and nearly free. It does not keep the PNGs in memory (the page reads them from disk on
// demand). report, when set, gets (done, total) after every key. Cancelling ctx stops it between keys. Keys the
// install has no art for count as missing, any other error stops the run.
func (s *Store) Extract(ctx context.Context, keys []string, report func(done, total int)) (Progress, error) {
	p := Progress{State: StateExtracting, Total: len(keys)}
	s.setProgress(p)
	for i, k := range keys {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		if !s.cached(k) {
			if err := s.warm(k); err != nil {
				if !errors.Is(err, ErrNotFound) {
					return p, err
				}
				p.Missing++
			}
		}
		p.Done = i + 1
		s.setProgress(p)
		if report != nil {
			report(p.Done, p.Total)
		}
	}
	p.State = StateReady
	s.setProgress(p)
	return p, nil
}

// warm renders key into the disk cache only (no memory copy).
func (s *Store) warm(key string) error {
	b, err := s.encode(key)
	if err != nil {
		return err
	}
	if s.dir != "" {
		writeAtomic(filepath.Join(s.dir, key+".png"), b)
	}
	return nil
}

// StartExtract runs Extract in the background; Close cancels it and waits. A second call while one runs is a no-op.
func (s *Store) StartExtract(keys []string, done func(Progress, error)) {
	if !s.running.CompareAndSwap(false, true) {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer s.running.Store(false)
		p, err := s.Extract(ctx, keys, nil)
		if done != nil {
			done(p, err)
		}
	}()
}

// Progress returns the extraction state.
func (s *Store) Progress() Progress {
	s.pmu.Lock()
	defer s.pmu.Unlock()
	if s.prog.State == "" {
		return Progress{State: StateIdle}
	}
	return s.prog
}

func (s *Store) setProgress(p Progress) {
	s.pmu.Lock()
	s.prog = p
	s.pmu.Unlock()
}
