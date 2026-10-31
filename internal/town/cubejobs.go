package town

import (
	"context"
	"errors"
	"fmt"

	"d2lod-bot/internal/cubejobs"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town/hand"
)

// The cube chore is U live: stash to pack moves with the stash open, then Transmute with it closed. Only the
// simulator has run it; Transmute itself is V live.
func (t *Town) cubeOpts(plan ChorePlan) cubejobs.Opts {
	b := &t.e.Tune.Bot
	return cubejobs.Opts{MaxJobs: b.CubeMaxJobs, RuneCap: b.CubeRuneCap, GemCap: b.CubeGemCap, Keep: plan.CubeKeep}
}

// cubeCounts counts gems and runes in the pack and stash and whether the Cube is in the pack.
func (t *Town) cubeCounts(s *d2.Snapshot) (counts map[string]int, haveCube bool) {
	handled := map[string]bool{}
	for _, c := range cubejobs.GemCodes() {
		handled[c] = true
	}
	for _, c := range cubejobs.RuneCodes() {
		handled[c] = true
	}
	counts = map[string]int{}
	for _, c := range []d2.Container{d2.ContInventory, d2.ContStash} {
		for _, u := range s.Inventory.Container(c) {
			b := t.h.Base(u)
			if b == nil {
				continue
			}
			if b.Code == "box" && c == d2.ContInventory {
				haveCube = true
			}
			if handled[b.Code] {
				counts[b.Code]++
			}
		}
	}
	return counts, haveCube
}

// needCube is the cheap check: the knob is on, the Cube is in the pack and the pack and stash hold a job.
func (t *Town) needCube(s *d2.Snapshot, plan ChorePlan) bool {
	if s == nil || s.Inventory == nil || !t.e.Tune.Bot.CubeJobs {
		return false
	}
	counts, ok := t.cubeCounts(s)
	if ok && t.cubeBlocked != "" && t.cubeBlocked == t.cubeSig(s, counts) {
		return false // the last run found no pack room and nothing has changed since
	}
	return ok && len(cubejobs.Plan(counts, t.cubeOpts(plan))) > 0
}

// cubeJobs runs the planned upgrades: at most CubeMaxJobs transmutes, each verified by the code that comes out,
// and the first failure ends the chore (items are then put back in the stash). Gold is never touched.
func (t *Town) cubeJobs(ctx context.Context, a *Act, plan ChorePlan, rep *ChoreReport) (err error) {
	if err := t.openStash(ctx, a); err != nil {
		return err
	}
	s, err := t.current()
	if err != nil {
		return err
	}
	counts, ok := t.cubeCounts(s)
	if !ok {
		return fmt.Errorf("%w: box", ErrCubeMissing)
	}
	jobs := cubejobs.Plan(counts, t.cubeOpts(plan))
	defer func() {
		// everything the chore left in the pack goes back where it was found
		if perr := t.cubeParkBack(ctx, a); perr != nil && err == nil {
			err = perr
		}
	}()
	for _, j := range jobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := t.cubePull(ctx, a, j.Inputs); err != nil {
			if errors.Is(err, ErrCubeNoRoom) {
				if cur, cerr := t.current(); cerr == nil {
					cc, _ := t.cubeCounts(cur)
					t.cubeBlocked = t.cubeSig(cur, cc)
				}
				t.e.Log.Info("town: cube job skipped, pack full", "job", j.String())
				return nil
			}
			return fmt.Errorf("cube job %s: %w", j, err)
		}
		_ = t.h.CloseAll(ctx)
		out, err := t.Transmute(ctx, j.Inputs)
		if err != nil {
			return fmt.Errorf("cube job %s: %w", j, err)
		}
		if len(out) != 1 || out[0] != j.Out {
			return fmt.Errorf("cube job %s: got %v", j, out)
		}
		rep.Cubed++
	}
	return nil
}

// cubePull makes sure the pack holds every input, taking missing ones from the stash (opened here when needed).
func (t *Town) cubePull(ctx context.Context, a *Act, inputs []string) error {
	need := map[string]int{}
	for _, c := range inputs {
		need[c]++
	}
	s, err := t.current()
	if err != nil {
		return err
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if b := t.h.Base(u); b != nil && need[b.Code] > 0 {
			need[b.Code]--
		}
	}
	var pull []d2.Unit
	for _, u := range s.Inventory.Container(d2.ContStash) {
		if b := t.h.Base(u); b != nil && need[b.Code] > 0 {
			need[b.Code]--
			pull = append(pull, u)
		}
	}
	for c, n := range need {
		if n > 0 {
			return fmt.Errorf("%w: %s", ErrCubeMissing, c)
		}
	}
	if len(pull) == 0 {
		return nil
	}
	if s.UI[d2.UIStash] == 0 {
		if err := t.openStash(ctx, a); err != nil {
			return err
		}
	}
	if s, err = t.current(); err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	for _, u := range pull {
		if err := t.h.Transfer(ctx, u, lay.Stash, lay.Inventory, d2.ContInventory); err != nil {
			if errors.Is(err, hand.ErrNoSpace) {
				return fmt.Errorf("%w: %v", ErrCubeNoRoom, err)
			}
			return err
		}
	}
	return nil
}

// cubeParkBack moves the pack's gems and runes into the stash and closes the panels.
func (t *Town) cubeParkBack(ctx context.Context, a *Act) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	counts, _ := t.cubeCounts(s)
	var move []d2.Unit
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if b := t.h.Base(u); b != nil && counts[b.Code] > 0 && isCubeItem(b.Code) {
			move = append(move, u)
		}
	}
	if len(move) == 0 {
		return t.h.CloseAll(ctx)
	}
	if s.UI[d2.UIStash] == 0 {
		if err := t.openStash(ctx, a); err != nil {
			return err
		}
	}
	if s, err = t.current(); err != nil {
		return err
	}
	lay, err := t.layout(s)
	if err != nil {
		return err
	}
	var first error
	for _, u := range move {
		if err := t.h.Transfer(ctx, u, lay.Inventory, lay.Stash, d2.ContStash); err != nil {
			if errors.Is(err, hand.ErrNoSpace) {
				t.e.Log.Info("town: cube items stay in the pack, stash full", "item", t.h.Base(u).Code)
				continue
			}
			if first == nil {
				first = err
			}
		}
	}
	if err := t.h.CloseAll(ctx); err != nil && first == nil {
		first = err
	}
	return first
}

func isCubeItem(code string) bool {
	for _, c := range cubejobs.GemCodes() {
		if c == code {
			return true
		}
	}
	for _, c := range cubejobs.RuneCodes() {
		if c == code {
			return true
		}
	}
	return false
}
