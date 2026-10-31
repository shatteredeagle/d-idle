// Package quest is the quest layer: it reads quest state (Done/Started/Needed) from any source, models the
// NPC dialog as an interface with per-NPC row tables, and expresses act hops and Act 1 quest handlers as
// step data run over small interfaces. Everything here is offline and tested with fakes.
//
// Tags: V verified live, S secondary source, U unverified (docs/quest.md lists the live checks).
package quest

import "d2lod-bot/internal/saves"

// Status is what the bot needs to know about one quest in one difficulty.
type Status int

const (
	Needed  Status = iota // not started, or nothing known
	Started               // in progress (bits 2-4 of the word)
	Done                  // reward granted or completed in a previous game
)

func (s Status) String() string { return [...]string{"needed", "started", "done"}[s] }

// Evidence says how sure a Reader is (docs/progression/quests.md section 1). A step may skip work only on
// EvSave or EvMemory, never on EvProxy.
type Evidence int

const (
	EvNone Evidence = iota
	EvProxy
	EvSave
	EvMemory
)

// Interpret is the ONE place that turns a quest word into a status. Done = bit 0 or bit 15 (V: 0x0001 and
// 0x8001 both read "completed in a previous game"); Started = bits 2, 3 or 4 (V: 0x000c started, 0x0010
// alone active; bit 4 alone as started is U). Pending = bit 1 (reward waiting, U).
func Interpret(w uint16) (st Status, pending bool) {
	pending = w&saves.QRewardPending != 0
	switch {
	case w&(saves.QRewardGranted|saves.QCompletedEarly) != 0:
		return Done, pending
	case w&(saves.QStarted|saves.QLeftTown|saves.QEnteredArea) != 0:
		return Started, pending
	}
	return Needed, pending
}

// Source gives the raw quest word of one quest (saves.questTable id) in one difficulty (0 normal, 1 nightmare,
// 2 hell). ok is false when the source does not know it.
type Source interface {
	Word(diff int, quest string) (w uint16, ok bool)
	Evidence() Evidence
}

// SaveSource reads a decoded .d2s (T1; stale until the game writes the file).
type SaveSource struct{ P saves.Progress }

func (s SaveSource) Word(diff int, quest string) (uint16, bool) {
	if diff < 0 || diff > 2 {
		return 0, false
	}
	for _, q := range s.P.Difficulty[diff].Quests {
		if q.ID == quest {
			return q.Word, true
		}
	}
	return 0, false
}
func (SaveSource) Evidence() Evidence { return EvSave }

// MapSource holds words known from elsewhere (in-memory quest pointers once located, T2; or a test). A
// memory-based source replaces SaveSource by implementing Source and being listed first in a Reader.
type MapSource struct {
	Words map[int]map[string]uint16
	Ev    Evidence
}

func (m *MapSource) Set(diff int, quest string, w uint16) {
	if m.Words == nil {
		m.Words = map[int]map[string]uint16{}
	}
	if m.Words[diff] == nil {
		m.Words[diff] = map[string]uint16{}
	}
	m.Words[diff][quest] = w
}

func (m *MapSource) Word(diff int, quest string) (uint16, bool) {
	w, ok := m.Words[diff][quest]
	return w, ok
}

func (m *MapSource) Evidence() Evidence {
	if m.Ev == EvNone {
		return EvMemory
	}
	return m.Ev
}

// Info is one answered quest query.
type Info struct {
	Quest    string
	Raw      uint16
	Status   Status
	Pending  bool
	Evidence Evidence // EvNone when no source knew the quest
}

// Reader asks its sources in order (freshest first); the first one that knows the quest wins.
type Reader struct{ Sources []Source }

// NewReader builds a Reader with memory-like sources first and the save last.
func NewReader(src ...Source) Reader { return Reader{Sources: src} }

func (r Reader) Quest(diff int, id string) Info {
	for _, s := range r.Sources {
		if w, ok := s.Word(diff, id); ok {
			st, pend := Interpret(w)
			return Info{Quest: id, Raw: w, Status: st, Pending: pend, Evidence: s.Evidence()}
		}
	}
	return Info{Quest: id}
}

func (r Reader) Status(diff int, id string) Status { return r.Quest(diff, id).Status }

// Skippable reports whether work for the quest may be skipped: Done on T1/T2 evidence only.
func (r Reader) Skippable(diff int, id string) bool {
	q := r.Quest(diff, id)
	return q.Status == Done && q.Evidence >= EvSave
}

// ActOpen reports whether the "able to go to the next act" word of act (1-4) is Done in diff.
func (r Reader) ActOpen(diff, act int) bool {
	return act >= 1 && act <= 4 && r.Skippable(diff, actDone[act])
}

var actDone = [5]string{"", "act1_done", "act2_done", "act3_done", "act4_done"}

// Cond is a prerequisite: quest has at least status Want.
type Cond struct {
	Quest string
	Want  Status
}

// Met reports whether every condition holds.
func (r Reader) Met(diff int, conds []Cond) (bool, Cond) {
	for _, c := range conds {
		if r.Status(diff, c.Quest) < c.Want {
			return false, c
		}
	}
	return true, Cond{}
}
