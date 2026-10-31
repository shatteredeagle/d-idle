package d2dsrc

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/d2"
)

// ErrNoProgress: this d2d cannot give quest words (stock build, no dump verb).
var ErrNoProgress = errors.New("d2dsrc: quest words unknown (no dump verb)")

// Progress returns the quest words and waypoint masks of all three difficulties from "dump quests". The
// layout is the save file's (48 little-endian words per difficulty, V in the fork source, S that the words
// are identical to the in-memory ones our questmem decoder reads), so internal/quest and internal/saves
// decoders can run on it unchanged. Stock d2d answers ErrNoProgress: its "debug quest <q> show" gives 16
// flag bits for a quest index and Act 1 record states only, which is not enough to rebuild a word table;
// callers must treat every quest as unknown then (not as "not done").
func (s *Source) Progress(ctx context.Context) (*d2.PlayerProgress, error) {
	defer s.trk.flush()
	d, err := s.dump(ctx, "quests")
	if errors.Is(err, errNoDump) {
		s.trk.hit("quests", "no dump verb")
		s.warnOnce("noquests", "d2dsrc: quest words unknown on stock d2d")
		return nil, ErrNoProgress
	}
	if err != nil {
		return nil, err
	}
	q := d.sub("quests")
	if q == nil {
		return nil, fmt.Errorf("d2dsrc: dump has no quests section")
	}
	out := &d2.PlayerProgress{}
	a, ok := q["diff"].([]any)
	if !ok || len(a) != 3 {
		return nil, fmt.Errorf("d2dsrc: quests.diff is not 3 arrays")
	}
	for dif := 0; dif < 3; dif++ {
		row := obj{"r": a[dif]}
		w, ok := row.ints("r")
		if !ok || len(w) != d2.QuestWords {
			return nil, fmt.Errorf("d2dsrc: quests.diff[%d] is not %d numbers", dif, d2.QuestWords)
		}
		for i, v := range w {
			if v < 0 || v > 0xFFFF {
				return nil, fmt.Errorf("d2dsrc: quest word out of range")
			}
			out.Quest[dif][i] = uint16(v)
		}
	}
	if wp, ok := q["waypoints"].([]any); ok && len(wp) == 3 {
		for i, v := range wp {
			if f, ok := v.(float64); ok && f >= 0 {
				out.Waypoints[i] = uint64(f)
			}
		}
	} else {
		s.trk.hit("waypoints", "dump quests has no waypoints masks")
	}
	return out, nil
}
