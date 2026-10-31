package town

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

var (
	ErrCubeMissing  = errors.New("town: cube or cube input not in the pack")
	ErrCubeNoResult = errors.New("town: transmute produced nothing")
)

// CubeTransmuteBase is the Transmute button of the cube panel at the 640x480 base layout; the screen position is
// this plus Layout.Offset. V live 2026-10-29 (tools/rigs/act2live cubemix): (240,328) client px at 800x600, offset (80,60).
var CubeTransmuteBase = panel.Pt{X: 160, Y: 268}

// Transmute puts the items named by inputs (base code, e.g. "msf", or display name) into the Horadric Cube, presses
// Transmute and returns the base codes that came out; the results are moved to the pack. The Cube must be in the
// pack. Inputs that are worn (the game equips quest items on pickup, docs/act2.md P1) are taken off first. On any
// failure the cube is emptied back into the pack before returning. Flow V live (docs/act2.md C3): right-click the
// Cube in the open pack -> flag 0x1A, Transfer in, click the button, wait, Transfer out.
func (t *Town) Transmute(ctx context.Context, inputs []string) ([]string, error) {
	h := t.h
	sn, err := h.OpenInventory(ctx)
	if err != nil {
		return nil, err
	}
	lay, err := h.Layout(sn)
	if err != nil {
		return nil, err
	}
	ids, err := t.cubeInputs(sn, inputs)
	if err != nil {
		return nil, err
	}
	if err := t.unequipIDs(ctx, ids); err != nil {
		return nil, err
	}
	if sn, err = h.Fresh(); err != nil {
		return nil, err
	}
	var cube *d2.Unit
	for _, it := range sn.Inventory.Container(d2.ContInventory) {
		if b := h.Base(it); b != nil && b.Code == "box" {
			c := it
			cube = &c
			break
		}
	}
	if cube == nil {
		return nil, fmt.Errorf("%w: box", ErrCubeMissing)
	}
	if err := h.RightClick(*cube, lay.Inventory); err != nil {
		return nil, err
	}
	if _, err := h.WaitFlag(ctx, d2.UICube, true); err != nil {
		return nil, err
	}
	out, err := t.cubeRun(ctx, ids, lay)
	if err != nil {
		t.cubeEmpty(ctx, lay)
	}
	if cerr := h.CloseAll(ctx); cerr != nil && err == nil {
		err = cerr
	}
	return out, err
}

func (t *Town) cubeRun(ctx context.Context, ids []uint32, lay *panel.Layout) ([]string, error) {
	h := t.h
	t.cubeEmpty(ctx, lay) // leftovers from an earlier failed run would change the recipe
	for _, id := range ids {
		s, err := h.Fresh()
		if err != nil {
			return nil, err
		}
		u, ok := hand.Find(s.Inventory, id)
		if !ok {
			return nil, fmt.Errorf("%w: item %d vanished", ErrCubeMissing, id)
		}
		if err := h.Transfer(ctx, u, lay.Inventory, lay.Cube, d2.ContCube); err != nil {
			return nil, err
		}
	}
	before := map[uint32]bool{}
	for _, id := range ids {
		before[id] = true
	}
	btn := panel.Pt{X: CubeTransmuteBase.X + lay.Offset.X, Y: CubeTransmuteBase.Y + lay.Offset.Y}
	if err := t.e.In.Click(input.Left, btn.X, btn.Y); err != nil {
		return nil, err
	}
	s, ok, err := h.Wait(ctx, 3*time.Second, func(s *d2.Snapshot) bool {
		for _, u := range s.Inventory.Container(d2.ContCube) {
			if !before[u.ID] {
				return true
			}
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrCubeNoResult
	}
	var codes []string
	for _, u := range s.Inventory.Container(d2.ContCube) {
		if before[u.ID] {
			continue
		}
		if err := h.Transfer(ctx, u, lay.Cube, lay.Inventory, d2.ContInventory); err != nil {
			return codes, err
		}
		if b := h.Base(u); b != nil {
			codes = append(codes, b.Code)
		}
	}
	return codes, nil
}

// cubeEmpty moves everything in the open cube back to the pack (best effort).
func (t *Town) cubeEmpty(ctx context.Context, lay *panel.Layout) {
	s, err := t.h.Fresh()
	if err != nil {
		return
	}
	for _, u := range s.Inventory.Container(d2.ContCube) {
		_ = t.h.Transfer(ctx, u, lay.Cube, lay.Inventory, d2.ContInventory)
	}
}

// cubeInputs resolves each wanted code or name to a distinct item in the pack or worn.
func (t *Town) cubeInputs(s *d2.Snapshot, want []string) ([]uint32, error) {
	used := map[uint32]bool{}
	var ids []uint32
	for _, w := range want {
		found := false
		for _, c := range []d2.Container{d2.ContInventory, d2.ContEquipment} {
			for _, it := range s.Inventory.Container(c) {
				b := t.h.Base(it)
				if used[it.ID] || b == nil || !(b.Code == w || strings.EqualFold(b.Display, w)) {
					continue
				}
				used[it.ID] = true
				ids = append(ids, it.ID)
				found = true
				break
			}
			if found {
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: %s", ErrCubeMissing, w)
		}
	}
	return ids, nil
}

// unequipIDs takes worn inputs off the body (hand.Unequip).
func (t *Town) unequipIDs(ctx context.Context, ids []uint32) error {
	for _, id := range ids {
		if err := t.h.Unequip(ctx, id); err != nil {
			return fmt.Errorf("town: unequip: %w", err)
		}
	}
	return nil
}
