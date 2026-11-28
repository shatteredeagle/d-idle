package questboard

import (
	"context"
	"errors"
	"sync"
)

// Fake is a scripted game for the executor and for the S4 API tests: an Env and Actions over a tiny world that
// survives an Exec restart (the world is the evidence, the Exec holds none). Not for production.
type Fake struct {
	mu sync.Mutex

	Class   string
	Level   int
	DiffIdx int
	// Quest words by quest id and the tier they read at; PendingBit makes the objective show as the game's
	// pending bit on T2 (otherwise only the T3 objective shows).
	DoneQuest  map[string]Tier
	PendingBit bool
	// ClearedQuest marks quests whose objective is reached (Fact.Cleared, proxy tier).
	ClearedQuest map[string]bool
	// Steps done per chunk key: the handler progress that the game itself holds (items, position, kills).
	Progress  map[string]int
	ChunkDone map[string]Tier
	// Denied marks chunks whose exact evidence says "not done" (ChunkFact.Denied).
	Denied map[string]bool
	// Scripts: Script is consumed before the default (a step that progresses).
	Script []StepResult
	// HandIn script: errors first, then silent no-ops (the dialogue ran, nothing registered), then success at
	// HandInTier (default T2; T3 leaves only the proxy).
	HandInErrs   int
	HandInSilent int
	// HandInLateErr: that many hand-ins register the reward and still return an error.
	HandInLateErr int
	HandInTier    Tier

	Calls []string // "step quest/chunk", "handin quest/chunk", "stop quest/chunk"
	board *Board
}

// NewFake returns a world for class at clvl in Normal.
func NewFake(b *Board, class string, clvl int) *Fake {
	return &Fake{board: b, Class: class, Level: clvl, DoneQuest: map[string]Tier{}, Progress: map[string]int{}, ChunkDone: map[string]Tier{}, HandInTier: TierMemory}
}

func (f *Fake) Diff() int { return f.DiffIdx }

func (f *Fake) Facts() Facts { return fakeFacts{f} }

type fakeFacts struct{ f *Fake }

func (ff fakeFacts) Quest(diff int, id string) Fact {
	f := ff.f
	f.mu.Lock()
	defer f.mu.Unlock()
	if diff != f.DiffIdx {
		return Fact{}
	}
	if t, ok := f.DoneQuest[id]; ok {
		return Fact{Known: true, Done: true, Tier: t}
	}
	fact := Fact{Known: true, Tier: TierMemory, Cleared: f.ClearedQuest[id]}
	if f.PendingBit {
		if q, err := f.board.Quest(id); err == nil && len(q.Chunks) == 1 && f.Progress[ChunkKey(id, q.Chunks[0].ID)] >= len(q.Chunks[0].Steps) {
			fact.Pending = true
		}
	}
	return fact
}

func (f *Fake) Chunk(q Quest, c Chunk) ChunkFact {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := ChunkKey(q.ID, c.ID)
	n := f.Progress[k]
	cf := ChunkFact{Steps: make([]bool, len(c.Steps)), Objective: n >= len(c.Steps)}
	for i := range cf.Steps {
		cf.Steps[i] = i < n
	}
	if t, ok := f.ChunkDone[k]; ok {
		cf.Done, cf.Tier = true, t
	}
	cf.Denied = f.Denied[k]
	return cf
}

func (f *Fake) Ready(q Quest, c Chunk) Word {
	r, err := f.board.LegacyReadiness(f.Class, f.Level)
	if err != nil {
		return ReadyUnknown
	}
	return r[ChunkKey(q.ID, c.ID)]
}

func (f *Fake) Step(_ context.Context, q Quest, c Chunk) StepResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := ChunkKey(q.ID, c.ID)
	f.Calls = append(f.Calls, "step "+k)
	if len(f.Script) > 0 {
		r := f.Script[0]
		f.Script = f.Script[1:]
		if r.Outcome != StepRan {
			return r
		}
	}
	f.Progress[k]++
	if f.Progress[k] >= len(c.Steps) && c.Ends != "handin" {
		f.ChunkDone[k] = TierMemory
		if len(q.Chunks) == 1 {
			f.DoneQuest[q.ID] = TierMemory
		}
	}
	return StepResult{Outcome: StepRan}
}

func (f *Fake) HandIn(_ context.Context, q Quest, c Chunk) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := ChunkKey(q.ID, c.ID)
	f.Calls = append(f.Calls, "handin "+k)
	switch {
	case f.HandInLateErr > 0: // the reward registers and the dialogue still reports an error
		f.HandInLateErr--
		if len(q.Chunks) == 1 {
			f.DoneQuest[q.ID] = f.HandInTier
		}
		f.ChunkDone[k] = f.HandInTier
		return errors.New("dialog closed early")
	case f.HandInErrs > 0:
		f.HandInErrs--
		return errors.New("dialog row not found")
	case f.HandInSilent > 0:
		f.HandInSilent--
		return nil
	}
	if len(q.Chunks) == 1 {
		f.DoneQuest[q.ID] = f.HandInTier
	}
	f.ChunkDone[k] = f.HandInTier
	return nil
}

func (f *Fake) Stop(q Quest, c Chunk) {
	f.mu.Lock()
	f.Calls = append(f.Calls, "stop "+ChunkKey(q.ID, c.ID))
	f.mu.Unlock()
}

// Count returns how many calls start with prefix.
func (f *Fake) Count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.Calls {
		if len(c) >= len(prefix) && c[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}
