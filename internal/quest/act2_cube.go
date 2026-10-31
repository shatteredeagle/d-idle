package quest

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// The Horadric Cube primitive. This file is only the contract and the pure layout maths; the UI work belongs to
// the town package (proposal: internal/town/cube.go, method (*Town).Transmute, built on town/hand Take/Put and the
// panel reader, next to Stash). Nothing here clicks.
//
// Item layout (V, items.txt): the Cube is a 2x2 item in the pack, its panel holds a 3 columns x 4 rows grid of 29 px
// cells (inventory.txt "Transmogrify Box Page 1": grid 87 x 114 px). Inputs of the Staff recipe: Shaft msf 1x3, Amulet
// vip 1x1 -> Horadric Staff hst 1x4 (cubemain.txt: msf + vip -> hst, V). The recipe is not ladder-only and has no
// difficulty floor.
//
// Panel UI steps the implementation must perform (steps 1-3 and 6 V live 2026-10-29 on i1; the Transmute click is U, see docs/act2.md C1):
//  1. Be in town (or anywhere safe) with the Cube in the pack, not in the stash; no other panel open.
//  2. Open the inventory (key i, UI flag 0x01), right-click the Cube item: the cube panel opens (flag 0x1A = d2.UICube, V live; the inventory stays open on the right).
//  3. For each input in order: pick it up from the pack (hand.Take), put it on the next free cube cell (hand.Put at the
//     cell centre from PlanCubeLayout). Verify the cursor is empty after each put.
//  4. Click the Transmute button (position V by screenshot: about (240,328) client pixels, an icon under the grid; the click itself is U), wait for the result item to appear in the
//     cube grid (poll the cube container's item list, up to 3 s).
//  5. Take the result out of the cube (hand.Take) and put it into a free pack cell; verify the code.
//  6. Close the panels (Esc, one press per open panel) and verify by the pack: inputs gone, output present.
//
// A failed Transmute (wrong inputs, no room) leaves the inputs in the cube: the implementation must take them back to
// the pack before returning an error, or the next run starts with a pack that lacks the quest items.
type Cube interface {
	// Transmute puts the items with the given codes into the cube, presses Transmute and returns the codes of the
	// items that came out (already moved to the pack).
	Transmute(ctx context.Context, inputs []string) (outputs []string, err error)
}

var (
	ErrCubeMissing  = errors.New("quest: cube input not in the pack")
	ErrCubeNoFit    = errors.New("quest: cube inputs do not fit the 3x4 grid")
	ErrCubeNoResult = errors.New("quest: transmute produced no expected item")
)

// Cube grid geometry. V: 3 x 4 cells of 29 px (inventory.txt). The screen origin of the grid is V live: panel.Layout.Cube = left 198, top 199
// (hand.Take/Put into the cube verified, container d2.ContCube, screenshot matches); it comes from panel.Layout, not from here.
const (
	CubeCols = 3
	CubeRows = 4
	CubeCell = 29
)

// Size is an item's footprint in grid cells (items.txt invwidth x invheight).
type Size struct{ W, H int }

// Cell is the top-left grid cell of one placed item.
type Cell struct{ Col, Row int }

// PlanCubeLayout places items first-fit into the 3x4 grid, tallest first, and returns the cells in INPUT order.
// ErrCubeNoFit when they do not fit. The cube is empty at the start (the implementation clears it first).
func PlanCubeLayout(sizes []Size) ([]Cell, error) {
	var occ [CubeRows][CubeCols]bool
	order := make([]int, len(sizes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return sizes[order[a]].H*sizes[order[a]].W > sizes[order[b]].H*sizes[order[b]].W })
	out := make([]Cell, len(sizes))
	for _, i := range order {
		s := sizes[i]
		if s.W < 1 || s.H < 1 {
			return nil, fmt.Errorf("%w: item %d has size %dx%d", ErrCubeNoFit, i, s.W, s.H)
		}
		placed := false
	scan:
		for r := 0; r+s.H <= CubeRows; r++ {
			for c := 0; c+s.W <= CubeCols; c++ {
				if fits(&occ, c, r, s) {
					for y := r; y < r+s.H; y++ {
						for x := c; x < c+s.W; x++ {
							occ[y][x] = true
						}
					}
					out[i], placed = Cell{c, r}, true
					break scan
				}
			}
		}
		if !placed {
			return nil, fmt.Errorf("%w: item %d (%dx%d)", ErrCubeNoFit, i, s.W, s.H)
		}
	}
	return out, nil
}

func fits(occ *[CubeRows][CubeCols]bool, c, r int, s Size) bool {
	for y := r; y < r+s.H; y++ {
		for x := c; x < c+s.W; x++ {
			if occ[y][x] {
				return false
			}
		}
	}
	return true
}

// CellCentre is the offset of a cell's centre from the grid origin in pixels (the origin is U, see CubeCell).
func CellCentre(c Cell, s Size) (dx, dy int) {
	return c.Col*CubeCell + s.W*CubeCell/2, c.Row*CubeCell + s.H*CubeCell/2
}
