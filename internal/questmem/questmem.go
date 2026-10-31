// Package questmem is a quest.Source that reads the live game's quest words and waypoint bits from memory
// (tier T2 of docs/progression/quests.md). Layout, evidence and dead ends: docs/questmem.md.
//
// The data lives in the embedded game server's PlayerData (the client player's copy stays empty). The server
// unit has no static pointer, so the Source finds it with a signature scan (d2.FindServerPlayer, about 50 ms),
// caches the address, re-validates it on every read and rescans when it went stale (new game, area change that
// reallocated the unit). When the game is not in a game every query answers "unknown", so a quest.Reader falls
// through to the save.
//
// Tags: V verified live (the worker i5, 2026-10-28), S secondary, U unverified.
package questmem

import (
	"sync"
	"time"

	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/saves"
)

// MaxAge is how long one read of the game's data is reused (the words change only on quest events).
const MaxAge = 200 * time.Millisecond

// Source implements quest.Source on a d2.Reader. Safe for concurrent use.
type Source struct {
	R *d2.Reader
	// Now and MaxAge are overridable for tests.
	Now    func() time.Time
	MaxAge time.Duration

	mu      sync.Mutex
	unit    uint32 // cached server-side player unit, 0 = unknown
	prog    *d2.PlayerProgress
	at      time.Time
	scans   int // number of FindServerPlayer scans so far (observability, tests)
	lastErr error
}

var _ quest.Source = (*Source)(nil)

// New returns a Source reading through r.
func New(r *d2.Reader) *Source { return &Source{R: r, Now: time.Now, MaxAge: MaxAge} }

// Evidence is EvMemory: the in-memory words are exact (V: equal to the save after Save and Exit).
func (s *Source) Evidence() quest.Evidence { return quest.EvMemory }

// Word returns the raw quest word of quest id (saves.questTable id) in difficulty diff (0..2). ok is false
// when the id is unknown, the game is not in a game, or the data could not be read.
func (s *Source) Word(diff int, id string) (uint16, bool) {
	idx, ok := QuestIndex(id)
	if !ok || diff < 0 || diff > 2 {
		return 0, false
	}
	p, err := s.Progress()
	if err != nil {
		return 0, false
	}
	return p.Quest[diff][idx], true
}

// WordIndex is Word by raw index (0..47).
func (s *Source) WordIndex(diff, idx int) (uint16, bool) {
	if idx < 0 || idx >= d2.QuestWords || diff < 0 || diff > 2 {
		return 0, false
	}
	p, err := s.Progress()
	if err != nil {
		return 0, false
	}
	return p.Quest[diff][idx], true
}

// Waypoint reports whether the waypoint of levels.txt area `area` is activated in diff. known is false when
// the area has no waypoint or the data is unavailable.
func (s *Source) Waypoint(diff, area int) (active, known bool) {
	bit, ok := WaypointBit(area)
	if !ok || diff < 0 || diff > 2 {
		return false, false
	}
	p, err := s.Progress()
	if err != nil {
		return false, false
	}
	return p.Waypoints[diff]>>uint(bit)&1 == 1, true
}

// Progress returns the current words and waypoint masks of all three difficulties (cached for MaxAge).
// Errors: d2.ErrNoServerPlayer while loading or in a menu, or the underlying read error.
func (s *Source) Progress() (*d2.PlayerProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.prog != nil && s.Now().Sub(s.at) < s.MaxAge {
		return s.prog, nil
	}
	p, err := s.read()
	s.lastErr = err
	if err != nil {
		s.prog = nil
		return nil, err
	}
	s.prog, s.at = p, s.Now()
	return p, nil
}

// Invalidate drops the cached data and unit address (call after an area change or a new game).
func (s *Source) Invalidate() {
	s.mu.Lock()
	s.unit, s.prog = 0, nil
	s.mu.Unlock()
}

// Scans is how many full scans for the server unit were needed so far.
func (s *Source) Scans() int { s.mu.Lock(); defer s.mu.Unlock(); return s.scans }

// read does one uncached read: in-game check, validate or find the server unit, then follow the pointers.
func (s *Source) read() (*d2.PlayerProgress, error) {
	if s.R.State() != d2.StateInGame {
		s.unit = 0
		return nil, d2.ErrNoServerPlayer
	}
	pl, err := s.R.PlayerIdent()
	if err != nil {
		s.unit = 0
		return nil, err
	}
	if s.unit != 0 && s.R.ServerPlayerOK(s.unit, pl) {
		if p, err := s.R.ReadProgress(s.unit); err == nil {
			return p, nil
		}
	}
	s.scans++
	if s.unit, err = s.R.FindServerPlayer(pl); err != nil {
		s.unit = 0
		return nil, err
	}
	p, err := s.R.ReadProgress(s.unit)
	if err != nil {
		s.unit = 0
	}
	return p, err
}

var (
	idxOnce  sync.Once
	questIdx map[string]int
	wpBit    map[int]int
)

// QuestIndex is the word index of a quest id. The table is internal/saves' (not copied): it is learned once
// by setting one word at a time in a scratch save and asking saves which quest lit up.
func QuestIndex(id string) (int, bool) { idxOnce.Do(buildIndex); i, ok := questIdx[id]; return i, ok }

// WaypointBit is the bit (0..38) of the waypoint of levels.txt area `area`, again learned from saves.
func WaypointBit(area int) (int, bool) { idxOnce.Do(buildIndex); b, ok := wpBit[area]; return b, ok }

func buildIndex() {
	questIdx, wpBit = map[string]int{}, map[int]int{}
	f, err := d2s.Parse(scratchSave())
	if err != nil {
		return
	}
	for i := 0; i < d2s.QuestWords; i++ {
		f.SetQuestWord(0, i, saves.QCompletedEarly)
		for _, q := range saves.ProgressOf(f).Difficulty[0].Quests {
			if q.Done {
				questIdx[q.ID] = i
			}
		}
		f.SetQuestWord(0, i, 0)
	}
	for b := 0; b < 39; b++ {
		f.SetWaypointMask(0, 1<<uint(b))
		for _, w := range saves.ProgressOf(f).Difficulty[0].Waypoints {
			if w.Active && w.Bit == b {
				wpBit[w.Level] = b
			}
		}
	}
}
