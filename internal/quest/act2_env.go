package quest

import (
	"context"
	"errors"
	"fmt"
)

// Act2Env is Env plus the primitives only Act 2 needs. The task layer implements them over nav, town and loot; the
// fake in act2_fake.go implements them for tests. Interface requests for the owners (INTEGRATE2, town) are listed in
// docs/act2.md section "Requests".
type Act2Env interface {
	Env
	Cube
	// UseItem right-clicks the pack item with the code (reads the Book of Skill) and returns once it is consumed.
	UseItem(ctx context.Context, code string) error
	// PlaceItem puts the pack item into the object of the class in the current area (the Staff into the orifice):
	// pick the item to the cursor and click the object. U in the real client.
	PlaceItem(ctx context.Context, code string, objClass int) error
	// OpenChest walks to the nearest object of the class and clicks it; the contents drop on the ground.
	OpenChest(ctx context.Context, objClass int) error
	// HasObject reports whether an object of the class exists in the current area. It must answer from presets (the
	// static map), not from the nearby unit table, or a tomb scan cannot tell tombs apart from a distance.
	HasObject(ctx context.Context, objClass int) bool
	// HireMerc hires the act's mercenary with the given aura (town.Hirer.Hire with MercWant{Act: 2, Diff: Normal}).
	HireMerc(ctx context.Context, act int, aura string, maxGold int) error
}

var (
	// ErrNoTomb: no Tal Rasha's Tomb level held the orifice object.
	ErrNoTomb = errors.New("quest: no tomb with the Horadric orifice found")
	// ErrNotInPack: a step that works on a pack item found it missing.
	ErrNotInPack = errors.New("quest: item not in the pack")
)

// Act2Runner runs Act 2 handlers over an Act2Env. Tomb is filled by KFindTomb (the boss script for Duriel needs it).
type Act2Runner struct {
	Env  Act2Env
	R    Reader
	Diff int
	Tomb int
}

// TombAreas are the seven candidate Tal Rasha's Tomb levels (V levels.txt 66..72).
func TombAreas() []int {
	var a []int
	for t := AreaTalTomb1; t <= AreaTalTomb7; t++ {
		a = append(a, t)
	}
	return a
}

// Run runs one Act 2 handler: skip on T1/T2 Done, or (fetch legs only) on the item proxy; check Pre; run the steps;
// re-read. As with RunHandler, a quest still not Done afterwards is not an error.
func (a *Act2Runner) Run(ctx context.Context, h Handler) (Result, error) {
	if a.R.Skippable(a.Diff, h.Quest) {
		return Result{Skipped: true, After: a.R.Quest(a.Diff, h.Quest), Confirmed: true}, nil
	}
	if p := Act2Proxy[h.Quest]; p != nil && p(a.Env) {
		return Result{Skipped: true, After: a.R.Quest(a.Diff, h.Quest)}, nil
	}
	if ok, c := a.R.Met(a.Diff, h.Pre); !ok {
		return Result{}, fmt.Errorf("%w: %s needs %s %s", ErrPrereq, h.Name, c.Quest, c.Want)
	}
	status := statusFn(a.R, a.Diff)
	for i, s := range h.Steps {
		if err := ctx.Err(); err != nil {
			return Result{Ran: true}, err
		}
		if err := a.step(ctx, status, s); err != nil {
			return Result{Ran: true}, fmt.Errorf("%s: step %d (kind %d, npc %d, area %d): %w", h.Name, i, s.Kind, s.NPC, s.Area, err)
		}
	}
	after := a.R.Quest(a.Diff, h.Quest)
	return Result{Ran: true, After: after, Confirmed: after.Status == Done && after.Evidence >= EvSave}, nil
}

func (a *Act2Runner) step(ctx context.Context, status func(string) Status, s Step) error {
	e := a.Env
	switch s.Kind {
	case KUseItem:
		if !e.Has(s.Item) {
			return fmt.Errorf("%w: %s", ErrNotInPack, s.Item)
		}
		return e.UseItem(ctx, s.Item)
	case KTransmute:
		if s.Item != "" && e.Has(s.Item) {
			return nil // assembled earlier; the inputs are gone
		}
		for _, c := range s.Labels {
			if !e.Has(c) {
				return fmt.Errorf("%w: %s", ErrCubeMissing, c)
			}
		}
		if !e.Has(ItemCube) {
			return fmt.Errorf("%w: %s", ErrCubeMissing, ItemCube)
		}
		out, err := e.Transmute(ctx, s.Labels)
		if err != nil {
			return err
		}
		if s.Item != "" && !contains(out, s.Item) {
			return fmt.Errorf("%w: wanted %s, got %v", ErrCubeNoResult, s.Item, out)
		}
		return nil
	case KPlaceItem:
		if !e.Has(s.Item) {
			return fmt.Errorf("%w: %s", ErrNotInPack, s.Item)
		}
		return e.PlaceItem(ctx, s.Item, s.Obj)
	case KOpenChest:
		return e.OpenChest(ctx, s.Obj)
	case KFindTomb:
		t, err := FindTomb(ctx, e, s.Area)
		if err != nil {
			return err
		}
		a.Tomb = t
		return nil
	case KHire:
		aura := ""
		if len(s.Labels) > 0 {
			aura = s.Labels[0]
		}
		return e.HireMerc(ctx, 2, aura, s.Obj)
	}
	return RunSteps(ctx, e, status, []Step{s})
}

// TombHinter is an optional Act2Env extension: the level number of the real Tal Rasha's Tomb as the game stores it
// (ActMisc +0x94 dwStaffTombLevel, V in D2BS sources, U on 1.14d: no reader exists yet, see docs/act2.md "Requests").
// FindTomb tries that tomb first and still checks the orifice, so a wrong value only costs time.
type TombHinter interface {
	StaffTombLevel(ctx context.Context) (level int, ok bool)
}

// TombProber is an optional Act2Env extension for levels too big to judge from the streamed rooms: ProbeFar goes to
// the next unvisited far part of the current area (nav.FarProbe) so more rooms stream in, and reports whether more
// parts remain. Without it FindTomb only judges what the arrival room shows.
type TombProber interface {
	ProbeFar(ctx context.Context) (more bool, err error)
}

// maxTombProbes bounds the probe steps per tomb (a tomb has 8 probe points on its rectangle or one per room).
const maxTombProbes = 14

// FindTomb finds the Tal Rasha's Tomb level of the Canyon (canyon) that holds the orifice object, leaving the player
// inside it. The hinted tomb (TombHinter) goes first. A tomb is judged only after its far rooms were streamed
// (TombProber): V live (ACT2LIVE5) the orifice was never in the arrival room and Explore said "nothing left" there.
// A tomb that cannot be entered (V live: exit 69's gate did not fire) is skipped and retried after the others.
// Wrong tombs are left by GoArea(canyon).
func FindTomb(ctx context.Context, e Act2Env, canyon int) (int, error) {
	order := TombAreas()
	if h, ok := e.(TombHinter); ok {
		if lv, ok := h.StaffTombLevel(ctx); ok && lv >= AreaTalTomb1 && lv <= AreaTalTomb7 {
			order = append([]int{lv}, remove(order, lv)...)
		}
	}
	var failed []int
	var firstErr error
	for pass := 0; pass < 2; pass++ {
		todo := order
		if pass == 1 {
			todo, failed = failed, nil
		}
		for _, t := range todo {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			if err := e.GoArea(ctx, t); err != nil && e.Area() != t {
				if firstErr == nil {
					firstErr = fmt.Errorf("tomb %d: %w", t, err)
				}
				failed = append(failed, t)
				continue
			}
			if tombHasOrifice(ctx, e) {
				return t, nil
			}
			if err := e.GoArea(ctx, canyon); err != nil {
				return 0, fmt.Errorf("back to the canyon from tomb %d: %w", t, err)
			}
		}
		if len(failed) == 0 {
			break
		}
	}
	if firstErr != nil && len(failed) > 0 {
		return 0, fmt.Errorf("%w (tombs not entered: %v): %v", ErrNoTomb, failed, firstErr)
	}
	return 0, ErrNoTomb
}

// tombHasOrifice looks at the current tomb and, while it is not there, probes the far parts.
func tombHasOrifice(ctx context.Context, e Act2Env) bool {
	if e.HasObject(ctx, ObjOrifice) {
		return true
	}
	p, ok := e.(TombProber)
	if !ok {
		return false
	}
	for i := 0; i < maxTombProbes; i++ {
		more, err := p.ProbeFar(ctx)
		if e.HasObject(ctx, ObjOrifice) {
			return true
		}
		if ctx.Err() != nil || err != nil || !more {
			return false
		}
	}
	return false
}

func remove(s []int, v int) []int {
	var out []int
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// RunAll runs the handlers of Act2Order that are not Done, in order, stopping at the first error. Handlers marked
// Skip are left out when optional is false (the progression-only campaign). It returns the ids it ran or skipped.
func (a *Act2Runner) RunAll(ctx context.Context, optional bool) ([]string, error) {
	var ran []string
	for _, id := range Act2Order {
		h, ok := Act2Handler(id)
		if !ok {
			return ran, fmt.Errorf("quest: no Act 2 handler %q", id)
		}
		if h.Skip && !optional {
			continue
		}
		if _, err := a.Run(ctx, h); err != nil {
			return ran, err
		}
		ran = append(ran, id)
	}
	return ran, nil
}
