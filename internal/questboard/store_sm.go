package questboard

import "fmt"

// Plan file, state machine part (card bsm3). Runs and Latches are stored next to Accepted. While the knob
// quests.boardSM is 0 nothing here is called and the file only holds Accepted, byte for byte as before. When it is
// on, every write goes through UpdateSM, which keeps Accepted equal to the live and finished runs so an older build
// (or the knob back at 0) still finds its records.

// maxStoppedRuns bounds the stopped runs kept in the file.
const maxStoppedRuns = 20

// DoneFn says whether a chunk reads done now and at which tier; the v1 import uses it to turn a record of a finished
// chunk into a finished run with a latch. A nil DoneFn reads nothing as done.
type DoneFn func(quest, chunk string, diff int) (Tier, bool)

// storedStatus is the status a run keeps on disk: running and held are recomputed at load (3.7).
func storedStatus(r Run) Run {
	if r.Status == RunRunning || r.Status == RunHeld {
		r.Status, r.Code = RunQueued, "waits_for"
	}
	return r
}

func (p Plan) liveOrFinished(quest, chunk, diff string) bool {
	for _, r := range p.Runs {
		if r.Quest == quest && r.Chunk == chunk && r.Diff == diff && r.Status != RunStopped {
			return true
		}
	}
	return false
}

// storedPark is the parked run of an accepted chunk, nil when there is none. The legacy board reads it so a park the
// machine stored is honoured with quests.boardSM at 0.
func (p Plan) storedPark(quest, chunk, diff string) *Run {
	for _, a := range p.Accepted {
		if a.Quest != quest || a.Chunk != chunk || a.Diff != diff {
			continue
		}
		for i, r := range p.Runs {
			if r.Quest == quest && r.Chunk == chunk && r.Diff == diff && r.Status == RunParked {
				return &p.Runs[i]
			}
		}
	}
	return nil
}

// anyStoredPark reports a parked run of any accepted chunk in this difficulty.
func (p Plan) anyStoredPark(diff string) bool {
	for _, a := range p.Accepted {
		if a.Diff == diff && p.storedPark(a.Quest, a.Chunk, diff) != nil {
			return true
		}
	}
	return false
}

// reconcile makes the runs agree with Accepted (the older code may have changed it since the runs were written) and
// runs the D4 repair: a record without a live or finished run is imported (a record whose chunk reads done becomes a
// finished run with a latch at the tier read, any other a queued run), a live run without a record was withdrawn and is
// stopped. It reports whether it changed anything.
func (s *Store) reconcile(p *Plan, done DoneFn) bool {
	changed := false
	m := SM{Runs: append([]Run(nil), p.Runs...), Latches: append([]Latch(nil), p.Latches...)}
	for _, r := range p.Accepted {
		if p.liveOrFinished(r.Quest, r.Chunk, r.Diff) {
			continue
		}
		run := Run{Quest: r.Quest, Chunk: r.Chunk, Diff: r.Diff, Status: RunQueued, Code: "waits_for", By: "player", Mode: r.Mode, At: r.At}
		if r.Auto {
			run.By = "auto"
		}
		if di, err := DiffIndex(r.Diff); err == nil && done != nil {
			if tier, ok := done(r.Quest, r.Chunk, di); ok {
				run.Status, run.Code = RunFinished, ""
				if m.latch(r.Quest, r.Chunk, r.Diff) == nil {
					m.Latches = append(m.Latches, Latch{Quest: r.Quest, Chunk: r.Chunk, Diff: r.Diff, Tier: tier, At: s.now().UTC(), Source: "import"})
				}
			}
		}
		m.Runs = append(m.Runs, run)
		p.Runs = append(p.Runs, run)
		changed = true
	}
	for i := range m.Runs {
		r := &m.Runs[i]
		if !r.Status.Live() {
			continue
		}
		have := false
		for _, a := range p.Accepted {
			have = have || a.Quest == r.Quest && a.Chunk == r.Chunk && a.Diff == r.Diff
		}
		if !have {
			r.Status, r.Code = RunStopped, ""
			changed = true
		}
	}
	for _, d := range Difficulties {
		if added, parked := m.Repair(s.b, d, s.now().UTC(), nil); len(added)+len(parked) > 0 {
			changed = true
		}
	}
	p.Runs, p.Latches = m.Runs, m.Latches
	return changed
}

// writeSM stores m in p: running and held runs as queued, and Accepted rebuilt (dual write) from the live and
// finished runs, keeping the old record (time, mode, auto flag) where there is one.
func writeSM(p *Plan, m SM) {
	p.Runs, p.Latches = nil, m.Latches
	stopped := 0
	for i := len(m.Runs) - 1; i >= 0; i-- { // stopped runs are history with no reader: keep the newest few
		if m.Runs[i].Status == RunStopped {
			if stopped++; stopped > maxStoppedRuns {
				m.Runs = append(m.Runs[:i], m.Runs[i+1:]...)
			}
		}
	}
	for _, r := range m.Runs {
		p.Runs = append(p.Runs, storedStatus(r))
	}
	old := p.Accepted
	p.Accepted = nil
	for _, r := range p.Runs {
		if r.Status == RunStopped {
			continue
		}
		rec := Record{Quest: r.Quest, Chunk: r.Chunk, Diff: r.Diff, At: r.At, Mode: r.Mode, Auto: r.By == "auto"}
		for _, o := range old {
			if o.Quest == r.Quest && o.Chunk == r.Chunk && o.Diff == r.Diff {
				rec = o
				break
			}
		}
		p.Accepted = append(p.Accepted, rec)
	}
}

// LoadSM returns the run machine of the hero, importing and repairing first (and saving when that changed anything).
func (s *Store) LoadSM(done DoneFn) (SM, error) {
	var out SM
	err := s.UpdateSM(done, func(m *SM) error { out = *m; return nil })
	return out, err
}

// UpdateSM loads the plan, reconciles the runs with Accepted, selects the head, applies f and saves with the dual
// write. f may change the machine; an error from f saves nothing. A reconcile that changed the file saves it even when
// f changed nothing.
func (s *Store) UpdateSM(done DoneFn, f func(*SM) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, _, err := s.load()
	if err != nil {
		return err
	}
	dirty := s.reconcile(&p, done)
	m := SM{Runs: p.Runs, Latches: p.Latches}
	for _, d := range Difficulties {
		m.Promote(s.b, d)
	}
	before := fmt.Sprint(m.Runs, m.Latches)
	if err := f(&m); err != nil {
		return err
	}
	if !dirty && before == fmt.Sprint(m.Runs, m.Latches) {
		return nil
	}
	writeSM(&p, m)
	return s.file.Save(p)
}

// acceptedKeys lists the records of Accepted as "difficulty/quest/chunk" (tests compare the dual write with it).
func (p Plan) acceptedKeys() []string {
	var out []string
	for _, r := range p.Accepted {
		out = append(out, r.Diff+"/"+ChunkKey(r.Quest, r.Chunk))
	}
	return out
}
