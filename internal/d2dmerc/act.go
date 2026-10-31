package d2dmerc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dtown"
)

// Hireling NPCs by act (names as d2d lists them). V-d2d: Kashya (docs/d2d-probes.md section 1); the others come from
// D2D-src kMercNpcs (S, docs/d2d-probes.md section 4).
var hirelings = []string{"Kashya", "Greiz", "Asheara", "Tyrael", "Qual-Kehk"}

var (
	ErrNoHireling   = errors.New("d2dmerc: no hireling NPC in the npcs list")
	ErrNoReviveLine = errors.New("d2dmerc: the NPC menu has no resurrect line")
	ErrNotRevived   = errors.New("d2dmerc: the resurrect click changed neither gold nor merc state")
)

// Reviver revives a dead merc through the hireling NPC's menu. Every step after opening the menu is U: the
// "resurrect" line (string 0x1507) is known to appear while the merc is dead (docs/d2d-probes.md section 4), but no
// merc ever died in the d2d runs, so the click and its gold effect were never seen. Revive proves itself: it returns
// the gold actually taken and fails when neither gold nor the merc code in "state" changed.
type Reviver struct {
	C       *d2dctl.Client
	Timeout time.Duration // menu open budget, 0 = 20 s
}

func (r *Reviver) Revive(ctx context.Context) (int, error) {
	st, _, err := r.C.State(ctx)
	if err != nil {
		return 0, err
	}
	npcs, err := r.C.NPCs(ctx)
	if err != nil {
		return 0, err
	}
	idx := -1
	for _, name := range hirelings {
		for _, n := range npcs.Items {
			if strings.EqualFold(n.Name, name) {
				idx = n.Index
			}
		}
		if idx >= 0 {
			break
		}
	}
	if idx < 0 {
		return 0, ErrNoHireling
	}
	to := r.Timeout
	if to == 0 {
		to = 20 * time.Second
	}
	lines, err := r.C.OpenNPCMenu(ctx, idx, to)
	if err != nil {
		return 0, err
	}
	if _, ok := d2dctl.FindLine(lines, "resurrect"); !ok {
		_ = r.C.PickMenuLine(ctx, "cancel")
		return 0, ErrNoReviveLine
	}
	if err := r.C.PickMenuLine(ctx, "resurrect"); err != nil {
		return 0, err
	}
	for i := 0; i < 10; i++ {
		if err := d2dctl.Pause(ctx, 300*time.Millisecond); err != nil {
			return 0, err
		}
		now, _, err := r.C.State(ctx)
		if err != nil {
			return 0, err
		}
		if paid := st.Gold - now.Gold; paid > 0 || now.M["merc"] != st.M["merc"] {
			_ = r.C.PickMenuLine(ctx, "cancel") // close the menu if it stayed open; a miss is harmless
			return max(paid, 0), nil
		}
	}
	_ = r.C.PickMenuLine(ctx, "cancel")
	return 0, ErrNotRevived
}

// Claimer takes the free Rogue from Kashya (d2dtown.ClaimRogue, V-d2d). It implements the claim capability of the
// bot's wiring (cmd/d2bot/mercwire.go Claimer).
type Claimer struct{ C *d2dctl.Client }

// ClaimFreeRogue reports granted=true when a Rogue appeared. A merc that was already owned is not an error: the
// reward is consumed and there is nothing more to do (granted=false, nil).
func (c Claimer) ClaimFreeRogue(ctx context.Context) (bool, error) {
	res, err := d2dtown.ClaimRogue(ctx, c.C)
	if err != nil {
		return false, err
	}
	switch {
	case res.Granted:
		return true, nil
	case res.HadMerc:
		return false, nil
	}
	return false, fmt.Errorf("d2dmerc: Kashya granted no merc (burial state %d -> %d)", res.BurialBefore, res.BurialAfter)
}
