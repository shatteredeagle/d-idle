package d2dtown

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
)

// ClaimResult says what ClaimRogue saw.
type ClaimResult struct {
	Granted      bool   // a merc appeared that was not there before
	Merc         string // the "state merc=" value afterwards ("x,y:RG"), "-" none
	HadMerc      bool   // a merc was present before: the reward is forfeited (V-d2d, docs/d2d-probes.md section 1)
	BurialBefore int    // Act 1 quest record 2 (Blood Raven) before: 3 entered, 4 killed, 5 reward paid
	BurialAfter  int
}

var ErrKashyaMissing = errors.New("d2dtown: Kashya is not in the npcs list (not in the Act 1 town)")

// ClaimRogue opens Kashya's menu and reports whether the Blood Raven reward (the free Rogue) was granted. It is the
// verb-level step the bot's BRKILL stub can call later; it is NOT wired into the bot.
//
// V-d2d (docs/d2d-probes.md section 1, re-checked by cmd/d2dshop): the grant happens when the menu opens, no click is
// needed; with a merc already owned nothing is granted and the reward is consumed. The caller must therefore claim
// BEFORE hiring anyone. The menu is closed by clicking "cancel" (Escape would toggle the game menu on d2d, V).
// Not verified on Game.exe (U): there the reward may need the dialogue line.
func ClaimRogue(ctx context.Context, c *d2dctl.Client) (ClaimResult, error) {
	var res ClaimResult
	st, _, err := c.State(ctx)
	if err != nil {
		return res, err
	}
	merc := func(st d2dctl.State) string { return st.M["merc"] }
	res.HadMerc = merc(st) != "" && merc(st) != "-"
	q, err := c.QuestShow(ctx, 2)
	if err == nil && len(q.States) > 1 {
		res.BurialBefore = q.States[1]
	}
	npcs, err := c.NPCs(ctx)
	if err != nil {
		return res, err
	}
	idx := -1
	for _, n := range npcs.Items {
		if strings.EqualFold(n.Name, "Kashya") {
			idx = n.Index
		}
	}
	if idx < 0 {
		return res, ErrKashyaMissing
	}
	if _, err := c.OpenNPCMenu(ctx, idx, 20*time.Second); err != nil {
		return res, err
	}
	// the grant shows in "state" within a few ticks of the menu opening
	for i := 0; i < 10; i++ {
		if st, _, err = c.State(ctx); err != nil {
			return res, err
		}
		if m := merc(st); !res.HadMerc && m != "" && m != "-" {
			break
		}
		if err := d2dctl.Pause(ctx, 300*time.Millisecond); err != nil {
			return res, err
		}
	}
	res.Merc = merc(st)
	res.Granted = !res.HadMerc && res.Merc != "" && res.Merc != "-"
	if q, err := c.QuestShow(ctx, 2); err == nil && len(q.States) > 1 {
		res.BurialAfter = q.States[1]
	}
	if err := c.PickMenuLine(ctx, "cancel"); err != nil {
		return res, fmt.Errorf("claim done but closing the menu failed: %w", err)
	}
	return res, nil
}
