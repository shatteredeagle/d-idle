package town_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/town"
)

// Gate 2 (docs/townshop.md): the game said empty, the tome still read 13 scrolls, and the shop bought 7 more
// (700 gold). With a positive stat read only a proof top-up is bought.
func TestEmptySourceWithReadableTomeBuysOnlyAProof(t *testing.T) {
	geartest.Need(t)
	r := emptySourceRig(t)
	for _, u := range r.W.Items {
		if r.W.DB.Items[u.Class].Code == "tbk" {
			r.W.Stats[u.Addr] = d2.Stats{70: 13}
		}
	}
	r.T.NotePortalEmpty()
	rep, err := r.T.Chores(context.Background(), town.FullPlan())
	if err != nil {
		t.Fatal(err)
	}
	if got := rep.Bought["Scroll of Town Portal"]; got != 2 {
		t.Fatalf("bought %d scrolls, want the 2-scroll proof: %+v", got, rep.Bought)
	}
}

// A spent chore budget stops the buy loops with one greppable line and keeps what was bought.
func TestResupplyBudgetStopsTheBuyLoops(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.Env.Tune.Town.ChoreBudgetS = 8
	r.Env.Clock = &tickClock{Clock: r.Env.Clock, step: 400 * time.Millisecond}
	var buf bytes.Buffer
	r.Env.Log = slog.New(slog.NewTextHandler(&buf, nil))
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true, Force: true}); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), "town: resupply budget hit"); n != 1 {
		t.Fatalf("%d budget lines, want exactly one per Chores call:\n%s", n, buf.String())
	}
}

// tickClock makes every Now call cost step, so a buy loop runs out of budget without real sleeping.
type tickClock struct {
	env.Clock
	step time.Duration
	add  time.Duration
}

func (c *tickClock) Now() time.Time {
	c.add += c.step
	return c.Clock.Now().Add(c.add)
}
