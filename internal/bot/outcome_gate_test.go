package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

// The travel gate refuses as a typed outcome: no failure count, one refusal line, the sentinel still reachable, the
// planner's skip and the standing line still set; knob off: the legacy error and the legacy counting.
func TestGateRefusalIsTypedNoStrike(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		g := newRig(t)
		g.w.set(func(w *world) { w.state = d2.StateInGame; w.area = 2; w.class = class })
		g.r.poll() // the wake facts of a refusal read the latest snapshot: without one the first look has no facts and the second logs a change
		buf := &lockedBuf{}
		g.r.e.Log = slog.New(slog.NewTextHandler(buf, nil))
		mk := func() error {
			return g.r.refuse(fmt.Errorf("%w: Cold Plains (3): route guard (belt 0)", errAreaGated))
		}
		var ref *Refused
		if err := mk(); !errors.As(err, &ref) || !errors.Is(err, errAreaGated) || ref.Need != "route_closed" {
			t.Fatalf("%v: not a typed gate refusal: %v", class, err)
		}
		for i := 0; i < 20; i++ {
			if g.r.do(context.Background(), control{}, StateTraveling, "x", 0, "walk", func(context.Context) error { return mk() }) {
				t.Fatal("a refusal is not success")
			}
		}
		out := buf.String()
		if g.r.loop.failures != 0 || strings.Count(out, "step refused") != 1 || strings.Contains(out, "action failed") {
			t.Fatalf("%v: failures %d, log:\n%s", class, g.r.loop.failures, out)
		}
		g.r.e.Tune.Bot.TypedOutcome = 0
		if err := mk(); errors.As(err, &ref) || !errors.Is(err, errAreaGated) {
			t.Fatalf("knob off must return the legacy error: %v", err)
		}
		g.r.do(context.Background(), control{}, StateTraveling, "x", 0, "walk", func(context.Context) error { return mk() })
		if g.r.loop.failures != 0 { // legacy gate path never counted either; its line is the old gatedWait one
			t.Fatalf("legacy gate counted %d", g.r.loop.failures)
		}
		if !strings.Contains(buf.String(), "area gate refused the walk") {
			t.Fatal("legacy gate line missing")
		}
	}
}

// The walk home with no portal scroll is not cancelled by a portal or crowd verdict (cancel storm 2026-11-26); with a
// scroll, in the field, or with the knob off the verdict still cancels. Two classes.
func TestWalkHomeNoScrollIsNotCancelledByPortalVerdict(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		for _, a := range []SafetyAction{SafetyTownPortal, SafetyCrowdAbandon} {
			for _, c := range []struct {
				name   string
				st     State
				scroll bool
				knob   int
				fires  bool
			}{
				{"no scroll, walk home", StateResupplying, false, 1, false},
				{"scroll", StateResupplying, true, 1, true},
				{"no scroll, clearing", StateClearing, false, 1, true},
				{"knob off", StateResupplying, false, 0, true},
			} {
				g := newRig(t)
				g.r.e.Tune.Bot.TypedOutcome = c.knob
				g.r.e.Tune.Bot.ReactionMs = 0
				g.w.set(func(w *world) {
					w.state = d2.StateInGame
					w.area = 2
					w.life, w.maxL = 10, 100
					w.class = class
					if c.scroll {
						w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
					}
				})
				g.fight.safety = func(*d2.Snapshot) SafetyAction { return a }
				ctx, cancel := context.WithCancelCause(context.Background())
				fired := g.r.guardOnce(ctx, cancel, c.st)
				cancel(nil)
				if fired != c.fires {
					t.Errorf("%v %v %s: fired %v want %v", class, a, c.name, fired, c.fires)
				}
			}
		}
	}
}

func isRefusedErr(err error) bool {
	var ref *Refused
	return errors.As(err, &ref)
}
