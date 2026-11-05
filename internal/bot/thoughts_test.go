package bot

import (
	"testing"

	"d2lod-bot/internal/ladder"
)

func TestEnableLadderFollowsTheKnob(t *testing.T) {
	g := newRig(t)
	ladder.Install(nil)
	if l, err := g.r.EnableLadder("", "t0", nil); l != nil || err != nil || ladder.Active() != nil {
		t.Fatalf("knob off must install nothing: %v %v", l, err)
	}
	g.e.Tune.Ladder.Enabled = 1
	l, err := g.r.EnableLadder("", "t0", nil)
	if err != nil || l == nil || ladder.Active() != l {
		t.Fatalf("knob on: %v %v", l, err)
	}
	ladder.Install(nil)
}
