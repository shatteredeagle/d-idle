package drill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// RetreatSpec is one retreat variant: the setup is written before the run starts (life, belt, scroll, monster
// adjacent); from the decision on the bot alone acts and the drill only reads.
type RetreatSpec struct {
	Variant  string // "drink", "portal" or "leave"
	LifePct  int    // life as a percent of the maximum, set by cheat before the start
	Belt     bool   // the belt holds healing potions
	Scroll   bool   // exactly one town portal scroll in the pack
	Adjacent int    // melee monsters wanted within the crowd reach
	// MaxTicks bounds the ticks from the decision to its observable effect (U: unit is the d2d step, the
	// bound is a first guess to be tuned on the first live run).
	MaxTicks int
}

// RetreatInfo is what the lab read around the executed action.
type RetreatInfo struct {
	Err                         string // setup or read failure (Blocked)
	Adjacent                    int
	Life, MaxLife               int    // at the decision
	LifeFloor, LifeEnd          int    // lowest life read after the decision; at the end
	Decision                    string // Fighter.Safety
	Acted                       bool
	ActErr                      string
	TicksToAct                  int // ticks from the life write to the decision being acted on
	TicksToEffect               int // ticks from the act to the effect; -1 none within MaxTicks
	BeltBefore, BeltAfter       int // healing potions in the belt
	ScrollsBefore, ScrollsAfter int
	PortalOpen                  bool
	LeftGame                    bool // snapshot state no longer in game after the act
	Dead                        bool
}

func (i RetreatInfo) String() string {
	return fmt.Sprintf("adjacent %d life %d/%d decision %q acted %v (err %q) ticks to act %d, to effect %d; life floor %d end %d; belt %d->%d scrolls %d->%d portal %v left %v dead %v",
		i.Adjacent, i.Life, i.MaxLife, i.Decision, i.Acted, i.ActErr, i.TicksToAct, i.TicksToEffect, i.LifeFloor, i.LifeEnd, i.BeltBefore, i.BeltAfter, i.ScrollsBefore, i.ScrollsAfter, i.PortalOpen, i.LeftGame, i.Dead)
}

// RetreatLab is the instance as the retreat drills use it.
type RetreatLab interface {
	FieldLab
	// Retreat does the setup writes, then lets Fighter.Safety decide and Fighter.Act carry the decision out, and reads
	// the effect for at most s.MaxTicks ticks. No cheat is sent after the Act.
	Retreat(ctx context.Context, r *Run, s RetreatSpec) RetreatInfo
}

// RT returns the retreat action group (docs/retreatdrill.md): the action executed, not only decided.
func RT(l RetreatLab) []Drill {
	lim := 10 * time.Minute
	mk := func(id, title string, s RetreatSpec, want string, effect func(RetreatInfo) bool) Drill {
		return Drill{ID: id, Primitive: "P29", Title: title, Limit: lim, Do: func(ctx context.Context, r *Run) Outcome {
			return retreatVerdict(s, want, effect, l.Retreat(ctx, r, s))
		}}
	}
	return []Drill{
		mk("RT.drink", "retreat executed: 1 adjacent, 20 percent life, belt with healing potions: the bot drinks (belt -1, life up)",
			RetreatSpec{Variant: "drink", LifePct: 20, Belt: true, Adjacent: 1, MaxTicks: 50}, "drink",
			func(i RetreatInfo) bool { return i.BeltAfter == i.BeltBefore-1 && i.LifeEnd > i.Life }),
		mk("RT.portal", "retreat executed: 1 adjacent, 34 percent life, empty belt, one scroll: the bot opens a town portal (scroll spent, portal listed)",
			RetreatSpec{Variant: "portal", LifePct: 34, Scroll: true, Adjacent: 1, MaxTicks: 50}, "town-portal",
			func(i RetreatInfo) bool { return i.ScrollsAfter == i.ScrollsBefore-1 && i.PortalOpen }),
		mk("RT.leave", "retreat executed: 1 adjacent, 10 percent life, empty belt, no scroll: the bot chickens (the game is left)",
			RetreatSpec{Variant: "leave", LifePct: 10, Adjacent: 1, MaxTicks: 200}, "chicken",
			func(i RetreatInfo) bool { return i.LeftGame }),
	}
}

func retreatVerdict(s RetreatSpec, want string, effect func(RetreatInfo) bool, i RetreatInfo) Outcome {
	o := Outcome{
		Intended: fmt.Sprintf("%s: Safety decides %q, Act carries it out, effect within %d ticks, life floor above 0", s.Variant, want, s.MaxTicks),
		Observed: i.String(),
		Sig:      fmt.Sprintf("%s|%v", i.Decision, effect(i)),
	}
	switch {
	case i.Err != "":
		o.Verdict, o.Reason = Blocked, i.Err
	case i.Adjacent < s.Adjacent:
		o.Verdict, o.Reason = Blocked, fmt.Sprintf("setup not reached: %d adjacent, want %d", i.Adjacent, s.Adjacent)
	case !strings.HasPrefix(i.Decision, want):
		o.Verdict, o.Reason = Fail, fmt.Sprintf("decision %q, want %q", i.Decision, want)
	case i.ActErr != "" || !i.Acted:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("decided but not executed (acted %v): %s", i.Acted, i.ActErr)
	case i.Dead || s.Variant != "leave" && i.LifeFloor <= 0:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("life floor %d: died before the retreat took effect", i.LifeFloor)
	case i.TicksToEffect < 0 || i.TicksToEffect > s.MaxTicks || !effect(i):
		o.Verdict, o.Reason = Fail, fmt.Sprintf("no effect within %d ticks (effect at %d)", s.MaxTicks, i.TicksToEffect)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("%s executed: %d ticks to act, %d to effect, life floor %d", want, i.TicksToAct, i.TicksToEffect, i.LifeFloor)
	}
	return o
}
