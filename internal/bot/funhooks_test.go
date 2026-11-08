package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// The waypoint hook hears each Normal waypoint once, with the area name; nightmare and repeats are silent. Two classes.
func TestWaypointHookFiresOncePerNormalWaypoint(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := newRig(t)
		type hit struct {
			id   int
			name string
		}
		var hits []hit
		g.r.SetWaypointHook(func(id int, name string) { hits = append(hits, hit{id, name}) })
		s := &d2.Snapshot{Player: &d2.Player{Class: class, Area: 3, Difficulty: 0}}
		g.r.markTouched(s, 3, 4)
		g.r.markTouched(s, 3) // again: nothing new
		nm := &d2.Snapshot{Player: &d2.Player{Class: class, Area: 3, Difficulty: 1}}
		g.r.markTouched(nm, 5)
		if len(hits) != 2 || hits[0].id != 3 || hits[1].id != 4 || hits[0].name == "" {
			t.Errorf("%v: %+v", class, hits)
		}
	}
}

// SetLadderFirsts makes the ladder's store accept the Codex point; without it the catalogue stays the shipped one.
func TestSetLadderFirstsAddsTheCodexPoint(t *testing.T) {
	for _, withFirsts := range []bool{false, true} {
		g := newRig(t)
		ladder.Install(nil)
		t.Cleanup(func() { ladder.Install(nil) })
		g.e.Tune.Ladder.Enabled = 1
		if withFirsts {
			tbl, err := decide.ShippedFirsts()
			if err != nil {
				t.Fatal(err)
			}
			g.r.SetLadderFirsts(tbl)
		}
		var has bool
		_, err := g.r.EnableLadderWith("", "human", func(_ *decide.Store, cat *decide.Catalogue, _ decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
			_, e := cat.Point(decide.FirstsPoint)
			has = e == nil
			return &ladder.RecordSink{}, nil, nil
		})
		if err != nil || has != withFirsts {
			t.Errorf("withFirsts=%v: has=%v err=%v", withFirsts, has, err)
		}
		ladder.Install(nil)
	}
}
