package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dmerc"
	"d2lod-bot/internal/drill"
)

// act1HireCode is the dump code of the Act 1 hireling class (d2.MercClasses[0], the Rogue Scout; the code RG is V
// live in docs/merclive.md). The d2dsrc table is private, so the pairing is repeated here for the one class.
const act1HireCode = "RG"

// PaidHire (drill.HireLab): walk to Kashya, open her menu, read the price of the "hire" line and click it when the
// purse covers it. U: whether the menu line (or the window it opens) carries a parseable price, and whether the first
// row is clickable by line text; d2d has no hire window reader (d2dmerc.ErrNoOffers). A missing quote ends in an
// error, never a guessed price (docs/merchire.md).
func (l *lab) PaidHire(ctx context.Context, r *drill.Run) (drill.HireInfo, error) {
	var info drill.HireInfo
	ai := l.ApproachKashya(ctx, r, true)
	if ai.Err != "" {
		return info, fmt.Errorf("approach: %s", ai.Err)
	}
	st, _, err := l.be.Client.State(ctx)
	if err != nil {
		return info, err
	}
	npcs, err := l.be.Client.NPCs(ctx)
	if err != nil {
		return info, err
	}
	idx := -1
	for _, n := range npcs.Items {
		if strings.EqualFold(n.Name, "Kashya") {
			idx = n.Index
		}
	}
	if idx < 0 {
		return info, d2dmerc.ErrNoHireling
	}
	lines, err := l.be.Client.OpenNPCMenu(ctx, idx, 20*time.Second)
	if err != nil {
		return info, err
	}
	defer func() { _ = l.CloseAll(ctx) }()
	line, ok := d2dctl.FindLine(lines, "hire")
	if !ok {
		return info, fmt.Errorf("no hire line in Kashya's menu: %v", lines)
	}
	q, ok := d2dmerc.ParseQuote(line.Text)
	if !ok {
		return info, fmt.Errorf("no price in the hire line %q (U: the quote may only be in the window)", line.Text)
	}
	info.Quote, info.Row, info.WantCode = q, line.Text, act1HireCode
	if st.Gold < q {
		_ = l.be.Client.PickMenuLine(ctx, "cancel")
		return info, fmt.Errorf("%w: %d gold, quote %d", drill.ErrHireCantAfford, st.Gold, q)
	}
	return info, l.be.Client.PickMenuLine(ctx, "hire")
}
