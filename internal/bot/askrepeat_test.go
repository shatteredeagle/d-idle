package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/ladder"
)

// The route guard re-plans every minute and asks t0.area.toohard each time; nobody answers (TASK-137). With the
// shipped ladder knobs the tower question opens ladder.quietAfter times, one per cooldown, and then the key is quiet.
func TestTowerQuestionDoesNotReopenEveryMinute(t *testing.T) {
	ladder.Install(nil)
	defer ladder.Install(nil)
	g := campaignRig(t, &fakeQuests{}, 14)
	g.e.Tune.Ladder.Enabled = 1
	l, err := g.r.EnableLadder("", "askrepeat", nil)
	if err != nil || l == nil {
		t.Fatalf("EnableLadder: %v %v", l, err)
	}
	g.r.poll()
	s := g.r.latest()
	want := g.r.e.Tune.Ladder.QuietAfter
	for i := 0; i < 240; i++ { // four hours of one-minute re-plans
		g.r.towerGuardWhy(s)
		g.clock.Sleep(time.Minute)
		l.Pump()
	}
	st := l.Stats()
	if st.Opened != want || st.Quieted != 1 {
		t.Fatalf("opened %d, quieted %d in 4 h; want %d and 1 (stats %+v)", st.Opened, st.Quieted, want, st)
	}
}
