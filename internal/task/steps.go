package task

import (
	"context"
	"strconv"
	"time"
)

// Primitive step constructors (tasklayer.md section 4). Each takes a done predicate ("the group's work this step
// serves is complete") so the same primitive serves every handler; a step is needed only while !done.

// Pred is a pure test on the world.
type Pred func(w World) bool

func needIf(done Pred, also Pred) func(World) (bool, Evidence) {
	return func(w World) (bool, Evidence) {
		if done(w) || (also != nil && !also(w)) {
			return false, evOf(w)
		}
		return true, evOf(w)
	}
}

// evOf is the evidence tier of the world as a whole: memory when a live quest reader fed it, else proxies (a save-only
// reader lags the game, so it cannot confirm what a step just did).
func evOf(w World) Evidence {
	if len(w.Quests) > 0 && !w.SaveOnly {
		return EvMemory
	}
	return EvProxy
}

// StepTravel walks to area a. Going to the town area uses ToTown (portal or walk), anywhere else nav.ToArea.
func StepTravel(id, title string, a Area, done Pred) Step {
	return Step{ID: id, Title: title, Phase: "travel", Budget: 6 * time.Minute, Est: 90 * time.Second, Ready: fieldReady(a),
		Need: needIf(done, func(w World) bool { return w.Area != a }),
		Do: func(ctx context.Context, x Actions) error {
			if a == RogueEncampment {
				return x.ToTown(ctx)
			}
			return x.Travel(ctx, a)
		}}
}

// StepWaypoint activates the waypoint of area a (optional: it only shortens later trips).
func StepWaypoint(id string, a Area, done Pred) Step {
	return Step{ID: id, Title: "Activate waypoint", Phase: "travel", Optional: true, Budget: 4 * time.Minute, Est: 60 * time.Second, Ready: fieldReady(a),
		Need: func(w World) (bool, Evidence) { return !w.WPs[a] && !done(w), EvMemory },
		Do:   func(ctx context.Context, x Actions) error { return x.ActivateWaypoint(ctx, a) }}
}

// StepClear clears the whole area.
func StepClear(id, title string, a Area, done Pred, budget time.Duration) Step {
	return Step{ID: id, Title: title, Phase: "fight", Budget: budget, Est: budget / 2, Ready: fieldReady(a),
		Need: needIf(done, func(w World) bool { return w.Area == a }),
		Do:   func(ctx context.Context, x Actions) error { return x.Clear(ctx, a) }}
}

// StepKill kills the named unique in area a. The done predicate must be true once it is dead.
func StepKill(id, boss string, a Area, done Pred, budget time.Duration) Step {
	return Step{ID: id, Title: "Kill " + boss, Phase: "fight", Budget: budget, Est: budget / 2, Ready: fieldReady(a),
		Need: needIf(done, func(w World) bool { return w.Area == a }),
		Do:   func(ctx context.Context, x Actions) error { return x.KillBoss(ctx, boss) }}
}

// StepPickUp collects a quest item that dropped.
func StepPickUp(id, item string, done Pred, ready Pred) Step {
	return Step{ID: id, Title: "Pick up " + item, Phase: "fight", Budget: 90 * time.Second, Est: 20 * time.Second,
		Need: needIf(func(w World) bool { return done(w) || w.Items[item] > 0 }, ready),
		Do:   func(ctx context.Context, x Actions) error { return x.PickUp(ctx, item) }}
}

// StepInteract clicks an object (tree, cage, stones).
func StepInteract(id, title, object string, done, ready Pred) Step {
	return Step{ID: id, Title: title, Phase: "fight", Budget: 3 * time.Minute, Est: 60 * time.Second,
		Need: needIf(done, ready), Do: func(ctx context.Context, x Actions) error { return x.Interact(ctx, object) }}
}

// StepTalk talks to an NPC in town and picks an option. When town is false the NPC is in the field.
// A town talk with the character in the field returns a Prereq fault that inserts a trip home.
func StepTalk(id, title, npc, pick string, done, ready Pred) Step {
	s := Step{ID: id, Title: title, Phase: "turn-in", Budget: 3 * time.Minute, Est: 45 * time.Second,
		Need: needIf(done, ready),
		Do:   func(ctx context.Context, x Actions) error { return x.Talk(ctx, npc, pick) }}
	s.Ready = func(w World) *Fault {
		if !w.InTown {
			return &Fault{Kind: KindPrereq, Why: "NPC is in town", Insert: []Step{StepGoHome(id + "/home")}}
		}
		return nil
	}
	return s
}

// StepGoHome returns to town when the character is not there.
func StepGoHome(id string) Step {
	return Step{ID: id, Title: "Return to town", Phase: "travel", Budget: 3 * time.Minute, Est: 30 * time.Second,
		Need: func(w World) (bool, Evidence) { return !w.InTown, EvMemory },
		Do:   func(ctx context.Context, x Actions) error { return x.ToTown(ctx) }}
}

// StepChores runs the town loop once.
func StepChores(id string) Step {
	return Step{ID: id, Title: "Buy potions and supplies", Phase: "chore", Budget: 6 * time.Minute, Est: 2 * time.Minute,
		Need: func(w World) (bool, Evidence) { return w.Potions < minPotions, EvMemory },
		Ready: func(w World) *Fault {
			if !w.InTown {
				return &Fault{Kind: KindPrereq, Why: "chores run in town", Insert: []Step{StepGoHome(id + "/home")}}
			}
			return nil
		},
		Do: func(ctx context.Context, x Actions) error { return x.Chores(ctx) }}
}

// StepLevel runs a slice of the bot's reach_level loop until the character has level n.
func StepLevel(n int) Step {
	return Step{ID: "level." + strconv.Itoa(n), Title: "Level up to " + strconv.Itoa(n), Phase: "level", Repeat: true, Budget: 30 * time.Minute, Est: levelEst,
		Need: func(w World) (bool, Evidence) { return w.Level < n, EvMemory },
		Do:   func(ctx context.Context, x Actions) error { return x.LevelTo(ctx, n) }}
}

const (
	minPotions = 4
	levelEst   = 20 * time.Minute // U: rough, only for ETA
)
