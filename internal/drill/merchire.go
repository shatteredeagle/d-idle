package drill

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// HireGold is the purse SETUP gives for the paid-hire drill: well above any act-1 quote (S docs/merclive.md).
const HireGold = 5000

// HireInfo is one visit to the act's hirer: the price read from the hire window before paying, the row that was hired
// and the code the lab expects the dump to show for it (taken from the hirer's class tables by the lab, not here).
type HireInfo struct {
	Quote    int
	Row      string
	WantCode string
}

// ErrHireCantAfford is what a HireLab returns when it read the quote and refused because the purse was below it,
// without clicking Hire (the lab maps d2dmerc.ErrCantAfford onto it).
var ErrHireCantAfford = errors.New("drill: the purse is below the quoted hire price")

// HireLab is the optional part of the lab for the paid hire (D8.h): walk to the act hirer, open the hire window, read
// the quoted price, hire the first row (or stop when the purse is below the quote).
type HireLab interface {
	PaidHire(ctx context.Context, r *Run) (HireInfo, error)
}

// D8h returns the paid-hire drills (P18, docs/merchire.md). D8.h2 sets gold below the quote and wants the refusal.
// Both need a character without a merc.
func D8h(l GameLab, lim time.Duration) []Drill {
	hl, ok := l.(HireLab)
	if !ok {
		return []Drill{{ID: "D8.h", Primitive: "P18", Title: "paid hire at the hirer", Needs: []string{"the lab has no hire step"}}}
	}
	return []Drill{
		{ID: "D8.h", Primitive: "P18", Title: "paid hire at the hirer: gold set by the logged cheat, the bot reads the quote and hires; merc in the dump, gold down by the quote", Limit: lim,
			Do: func(ctx context.Context, r *Run) Outcome { return d8hire(ctx, r, l, hl, HireGold) }},
		{ID: "D8.h2", Primitive: "P18", Title: "paid hire with gold below the quote: ErrCantAfford, no merc, gold unchanged", Limit: lim,
			Do: func(ctx context.Context, r *Run) Outcome { return d8hire(ctx, r, l, hl, 1) }},
	}
}

func d8hire(ctx context.Context, r *Run, l GameLab, hl HireLab, gold int) Outcome {
	poor := gold < HireGold
	m0, _ := l.MercState(ctx)
	if m0.Present {
		return Outcome{Verdict: Blocked, Reason: "the character already owns a merc (" + m0.String() + "): the drill needs a character without one", Intended: "no merc before the hire"}
	}
	if err := l.SetGold(ctx, r, gold); err != nil {
		return setupFail(err)
	}
	if err := l.Tick(ctx, 5); err != nil {
		return setupFail(err)
	}
	g0, _ := l.TView(ctx)
	info, herr := hl.PaidHire(ctx, r)
	_ = l.Tick(ctx, 60)
	m1, _ := l.MercState(ctx)
	g1, _ := l.TView(ctx)
	paid := g0.Gold - g1.Gold
	obs := fmt.Sprintf("quote %d row %q; gold %d -> %d; merc %s -> %s; err %v", info.Quote, info.Row, g0.Gold, g1.Gold, m0, m1, herr)
	o := Outcome{Observed: obs, Sig: fmt.Sprintf("poor=%v err=%v present=%v", poor, herr != nil, m1.Present)}
	if poor {
		o.Intended = "gold below the quote: the hire is refused with ErrCantAfford, no merc, gold unchanged"
		switch {
		case herr == nil || !errors.Is(herr, ErrHireCantAfford):
			o.Verdict, o.Reason = Fail, fmt.Sprintf("wanted ErrCantAfford, got %v", herr)
		case info.Quote <= g0.Gold:
			o.Verdict, o.Reason = Fail, fmt.Sprintf("refused although the quote %d is within the purse %d", info.Quote, g0.Gold)
		case m1.Present || paid != 0:
			o.Verdict, o.Reason = Fail, fmt.Sprintf("refused but a merc appeared or gold moved (paid %d, merc %s)", paid, m1)
		default:
			o.Verdict, o.Reason = Pass, fmt.Sprintf("refused: quote %d above the purse %d, nothing spent", info.Quote, g0.Gold)
		}
		return o
	}
	o.Intended = "merc present with the class code of the hired row, gold down by the quote read from the window"
	switch {
	case herr != nil:
		o.Verdict, o.Reason = Fail, "hire: "+herr.Error()
	case info.Quote <= 0:
		o.Verdict, o.Reason = Fail, "no quote was read from the hire window"
	case !m1.Present || m1.Dead || m1.Code == "":
		o.Verdict, o.Reason = Fail, "no living merc with a code in the dump after the hire"
	case info.WantCode != "" && !strings.EqualFold(m1.Code, info.WantCode):
		o.Verdict, o.Reason = Fail, fmt.Sprintf("merc code %q, expected %q", m1.Code, info.WantCode)
	case paid != info.Quote:
		o.Verdict, o.Reason = Fail, fmt.Sprintf("gold fell by %d, the quote was %d", paid, info.Quote)
	default:
		o.Verdict, o.Reason = Pass, fmt.Sprintf("hired %s (%s) for the quoted %d gold", info.Row, m1.Code, paid)
	}
	return o
}
