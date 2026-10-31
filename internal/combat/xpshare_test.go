package combat

import (
	"io"
	"log/slog"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// Tiny kills give 1-2 xp: integer division must not throw the remainder away.
func TestNoteKillsKeepsXPRemainder(t *testing.T) {
	f := &Fighter{e: &env.Env{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}}
	st := newFightState()
	st.haveExp, st.lastExp = true, 100
	st.seen[1] = seenUnit{name: "Rat"}
	st.seen[2] = seenUnit{name: "Rat"}
	dead := func(id uint32) d2.Unit { return d2.Unit{ID: id, Type: d2.TypeMonster, Mode: d2.MonModeDead} }
	snap := &d2.Snapshot{Player: &d2.Player{Exp: 103}, Monsters: []d2.Unit{dead(1), dead(2)}}
	var res ClearResult
	f.noteKills(snap, nil, st, &res)
	if res.Kills != 2 || res.XP != 2 || st.pendingXP != 1 {
		t.Fatalf("kills %d xp %d pending %d, want 2 / 2 / 1", res.Kills, res.XP, st.pendingXP)
	}
}
