package quest

import (
	"context"
	"errors"
	"fmt"
)

// Kind is the type of a Step. NPC kinds are executed here over Dialog; field kinds are delegated to Env.
type Kind int

const (
	KGoNPC     Kind = iota // walk to NPC class
	KTalk                  // open dialog (NPC)
	KPick                  // PickPath(Labels...) on the open dialog
	KClose                 // close the dialog
	KTurnIn                // DoTurnIn (NPC, Item, Labels)
	KGoArea                // travel to area id (Area)
	KClear                 // clear every monster in Area (Den of Evil)
	KKill                  // kill the named boss (Target), in Area
	KPickup                // pick up item code (Item) from the ground / take from an object
	KUseObject             // interact with object class (Obj); Area is where it stands
	KTouchWP               // touch the town waypoint
	KWaitArea              // wait until Player.Area == Area (success detection of a hop)
	KToTown                // town portal / walk back to the act's town
)

// Step is one data step. Tag is V/U for the claims it carries; Note says why.
type Step struct {
	Kind   Kind
	NPC    int
	Labels []string
	Area   int
	Obj    int
	Item   string
	Target string
	Tag    string
	Note   string
}

// Env is everything a step needs. The task layer implements it over nav, combat, loot and town; tests use
// FakeEnv. Dialog and Has come from the same Env so one object can serve both.
type Env interface {
	Dialog
	Has
	Area() int
	GoNPC(ctx context.Context, npcClass int) error
	GoArea(ctx context.Context, area int) error
	Clear(ctx context.Context, area int) error
	Kill(ctx context.Context, target string, area int) error
	Pickup(ctx context.Context, item string) error
	UseObject(ctx context.Context, class, area int) error
	TouchWaypoint(ctx context.Context) error
	WaitArea(ctx context.Context, area int) error
	ToTown(ctx context.Context) error
}

var ErrPrereq = errors.New("quest: prerequisite not met")

// RunSteps executes steps in order and stops at the first error. status filters dialog rows by quest state.
func RunSteps(ctx context.Context, env Env, status func(string) Status, steps []Step) error {
	for i, s := range steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := runStep(ctx, env, status, s); err != nil {
			return fmt.Errorf("step %d (kind %d, npc %d, area %d): %w", i, s.Kind, s.NPC, s.Area, err)
		}
	}
	return nil
}

func runStep(ctx context.Context, env Env, status func(string) Status, s Step) error {
	switch s.Kind {
	case KGoNPC:
		return env.GoNPC(ctx, s.NPC)
	case KTalk:
		return env.Talk(ctx, s.NPC)
	case KPick:
		return PickPath(ctx, env, s.NPC, status, s.Labels...)
	case KClose:
		return env.Close(ctx)
	case KTurnIn:
		return DoTurnIn(ctx, env, env, status, TurnIn{NPC: s.NPC, Item: s.Item, Path: s.Labels})
	case KGoArea:
		return env.GoArea(ctx, s.Area)
	case KClear:
		return env.Clear(ctx, s.Area)
	case KKill:
		return env.Kill(ctx, s.Target, s.Area)
	case KPickup:
		return env.Pickup(ctx, s.Item)
	case KUseObject:
		return env.UseObject(ctx, s.Obj, s.Area)
	case KTouchWP:
		return env.TouchWaypoint(ctx)
	case KWaitArea:
		return env.WaitArea(ctx, s.Area)
	case KToTown:
		return env.ToTown(ctx)
	}
	return fmt.Errorf("quest: unknown step kind %d", s.Kind)
}

// Result of RunHandler / RunHop.
type Result struct {
	Skipped   bool // already Done on T1/T2 evidence
	Ran       bool
	After     Info // re-read after the steps (the save may lag: Unconfirmed)
	Confirmed bool // After.Status == Done on T1/T2 evidence
}

func statusFn(r Reader, diff int) func(string) Status {
	return func(q string) Status { return r.Status(diff, q) }
}
