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

// PaidHire (drill.HireLab): walk to Kashya, open her menu, get to the hire line or window row that carries the
// price (through "talk" when the first menu has no hire line, docs/hirefixlive.md) and click it when the purse
// covers it. U: whether the price is parseable there; d2d has no hire window reader (d2dmerc.ErrNoOffers). A missing
// quote ends in an error, never a guessed price (docs/merchire.md).
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
	return hireFlow(ctx, l.be.Client, lines, st.Gold, 400*time.Millisecond)
}

// menuClient is the part of d2dctl.Client the hire flow needs (a scripted fake stands in for it in the tests).
type menuClient interface {
	Menu(ctx context.Context) (d2dctl.List[d2dctl.MenuLine], error)
	PickMenuLine(ctx context.Context, text string) error
}

// hireLine finds the line that carries the quote: a "hire" line or a window row with "cost" in it.
func hireLine(lines []d2dctl.MenuLine) (d2dctl.MenuLine, bool) {
	if l, ok := d2dctl.FindLine(lines, "hire"); ok {
		return l, true
	}
	return d2dctl.FindLine(lines, "cost")
}

// hireFlow follows the two-step menu: Kashya's dialog is header, "talk", "cancel" (V live, docs/livedrills2.md). A
// hire line in the first menu is taken at once; otherwise "talk" is clicked and the menu polled for a hire line or a
// priced row (U: that the hire window sits behind "talk"). The row is clicked by its own text. WantCode stays empty:
// the class code comes from the dump after the hire, not from a literal here.
func hireFlow(ctx context.Context, c menuClient, lines []d2dctl.MenuLine, gold int, gap time.Duration) (drill.HireInfo, error) {
	var info drill.HireInfo
	line, ok := hireLine(lines)
	if !ok {
		if _, has := d2dctl.FindLine(lines, "talk"); !has {
			return info, fmt.Errorf("no hire or talk line in Kashya's menu: %v", lines)
		}
		if err := c.PickMenuLine(ctx, "talk"); err != nil {
			return info, err
		}
		seen := lines
		for i := 0; i < 8 && !ok; i++ {
			if err := d2dctl.Pause(ctx, gap); err != nil {
				return info, err
			}
			m, err := c.Menu(ctx)
			if err != nil {
				return info, err
			}
			if len(m.Items) > 0 {
				seen = m.Items
			}
			line, ok = hireLine(m.Items)
		}
		if !ok {
			return info, fmt.Errorf("no hire line after talk; last menu: %v", seen)
		}
	}
	q, ok := d2dmerc.ParseQuote(line.Text)
	if !ok {
		_ = c.PickMenuLine(ctx, "cancel")
		return info, fmt.Errorf("no price in the hire line %q (U: the quote may only be in the window)", line.Text)
	}
	info.Quote, info.Row = q, line.Text
	if gold < q {
		_ = c.PickMenuLine(ctx, "cancel")
		return info, fmt.Errorf("%w: %d gold, quote %d", drill.ErrHireCantAfford, gold, q)
	}
	return info, c.PickMenuLine(ctx, line.Text)
}
