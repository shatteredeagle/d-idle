package hand

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// worldProbe are world-space offsets (subtiles) from a unit's feet to click when hover detection never confirms
// it: the sprite is drawn above the feet, which is up-left in the isometric view (-1,-1).
var worldProbe = [][2]int{{0, 0}, {-1, -1}, {1, 1}}

// interactBudget bounds one InteractNPC call (hover probes plus plain-click fallbacks).
const interactBudget = 3 * time.Second

// InteractNPC clicks a town NPC until its dialog menu opens. Input.Interact first (hover-verified click); when
// hover never reports the unit (U: an NPC standing on stairs or off-centre, where a plain click still works) it
// falls back to plain world clicks at the unit and accepts the click when the NPC menu opens. The menu is the
// proof, so a wrong click costs only a few hundred milliseconds. Errors other than "not hovered" pass through.
func (h *Hand) InteractNPC(ctx context.Context, t input.Target) error {
	start := h.E.Clock.Now()
	err := h.E.In.Interact(t)
	if !errors.Is(err, input.ErrNotHovered) && !errors.Is(err, input.ErrNoHover) {
		return err
	}
	for _, off := range worldProbe {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Fail fast: when hover is dead (first-contact greeting playing, V live i5: three tries of ~4 s each burned for
		// nothing) the caller re-reads the position and re-approaches instead of probing on.
		if h.E.Clock.Now().Sub(start) > interactBudget {
			break
		}
		if err := h.E.In.ClickWorld(input.Point{X: t.Pos.X + off[0], Y: t.Pos.Y + off[1]}); err != nil {
			return err
		}
		_, ok, err := h.Wait(ctx, h.settle()*4, func(s *d2.Snapshot) bool { return s.UI.Open(d2.UINPCMenu) })
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("%w (plain clicks opened no dialog either)", input.ErrNotHovered)
}
