package d2denv

import (
	"context"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/questmem"
)

// QuestSource is a quest.Source over d2dsrc.Source.Progress ("dump quests"): the same words and waypoint masks
// questmem decodes from process memory, so quest.Reader and the task layer use it unchanged. Evidence is EvMemory:
// d2d's live words are exact like the in-memory ones (S: the dump has the save layout). On a stock d2d (no dump verb)
// every answer is unknown, never "not done".
type QuestSource struct {
	Src    *d2dsrc.Source
	MaxAge time.Duration // reuse of one dump (0 = 200 ms, questmem's value)
	Now    func() time.Time

	mu   sync.Mutex
	prog *d2.PlayerProgress
	at   time.Time
}

var _ quest.Source = (*QuestSource)(nil)

func (s *QuestSource) Evidence() quest.Evidence { return quest.EvMemory }

func (s *QuestSource) progress() (*d2.PlayerProgress, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now, age := time.Now, s.MaxAge
	if s.Now != nil {
		now = s.Now
	}
	if age == 0 {
		age = questmem.MaxAge
	}
	if s.prog != nil && now().Sub(s.at) < age {
		return s.prog, true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := s.Src.Progress(ctx)
	if err != nil {
		s.prog = nil
		return nil, false
	}
	s.prog, s.at = p, now()
	return p, true
}

func (s *QuestSource) Word(diff int, id string) (uint16, bool) {
	idx, ok := questmem.QuestIndex(id)
	if !ok || diff < 0 || diff > 2 {
		return 0, false
	}
	p, ok := s.progress()
	if !ok {
		return 0, false
	}
	return p.Quest[diff][idx], true
}

func (s *QuestSource) Waypoint(diff, area int) (active, known bool) {
	bit, ok := questmem.WaypointBit(area)
	if !ok || diff < 0 || diff > 2 {
		return false, false
	}
	p, ok := s.progress()
	if !ok {
		return false, false
	}
	return p.Waypoints[diff]>>uint(bit)&1 == 1, true
}

// Progress is the raw words, for the task layer's change log.
func (s *QuestSource) Progress() (*d2.PlayerProgress, error) {
	if p, ok := s.progress(); ok {
		return p, nil
	}
	return nil, d2dsrc.ErrNoProgress
}
