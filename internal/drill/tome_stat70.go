package drill

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Tome stat-70 recheck (docs/tomefix.md "Live-check recipe", docs/livecheckdrills.md).
const (
	tomeMinCharges = 13  // the recipe: a tome of 13+ charges
	tomeJunk       = 10  // ground items beside the player so the dump carries Extras.Ground
	tomeLooks      = 60  // reads of the tome through the bot's reader
	tomeGap        = 750 // ticks between looks: 60 * 750 = 45000 ticks, about 30 virtual minutes at 25 ticks a second (U)
)

// TomeLab is the optional part of the lab for the tome drills.
type TomeLab interface {
	PView(ctx context.Context) (PortalView, error)
	Tick(ctx context.Context, n int) error
	ClearPack(ctx context.Context, r *Run) error
	SetGold(ctx context.Context, r *Run, purse int) error
	GiveGround(ctx context.Context, r *Run, code string) error
	Goto(ctx context.Context, r *Run, level int) error
	Chores(ctx context.Context, plan string) ChoreInfo
	JunkCode() string // a pickit-filtered item code to drop on the ground (from the tables)
	LogMark() int
	LogSince(mark int) []string
}

// tomeLogKeys are the log markers of the three open producer paths (docs/tomefix.md).
var tomeLogKeys = []struct{ Key, Path string }{
	{"dump items failed", "path 10 (failed dump items)"},
	{"items verb", "path 9 (stock items verb, no qty)"},
	{"no70", "no stat 70 for the tome"},
	{"tp-debounce-flip", "debounce flip"},
}

// DTome returns the tome recheck. TM.a covers what existing verbs express (ground ids beside the tome, 30 virtual
// minutes of looks, qty dropout seen as a zero read with no failed-dump warning); TM.b and TM.c need new d2d verbs.
func DTome(l GameLab, lim time.Duration) []Drill {
	tl, ok := l.(TomeLab)
	if !ok {
		return []Drill{{ID: "TM", Primitive: "P10", Title: "tome stat 70 recheck", Needs: []string{"the lab has no TomeLab (JunkCode, LogMark, LogSince)"}}}
	}
	return []Drill{
		{ID: "TM.a", Primitive: "P12", Title: "tome of 13+ charges in the field with ground items: 60 looks over 30 virtual minutes, never empty", Limit: lim,
			Do: func(ctx context.Context, r *Run) Outcome { return tomeA(ctx, r, tl) }},
		{ID: "TM.b", Primitive: "P12", Title: "path 9: the stock items verb never sets qty", Needs: []string{"a d2d verb or fork option that makes the producer skip `dump items` and use the stock `items` verb for one look"}},
		{ID: "TM.c", Primitive: "P12", Title: "path 10: `dump items` fails for one look", Needs: []string{"a d2d fault-injection verb that makes `dump items` return an error for one look"}},
	}
}

func tomeA(ctx context.Context, r *Run, l TomeLab) Outcome {
	if err := l.ClearPack(ctx, r); err != nil {
		return setupFail(err)
	}
	if err := l.SetGold(ctx, r, 1000); err != nil {
		return setupFail(err)
	}
	ci := l.Chores(ctx, "scrolls") // buys and fills a tome (D3.tp, V live)
	if ci.Err != "" {
		return setupFail(fmt.Errorf("scrolls chore: %s", ci.Err))
	}
	if err := l.Goto(ctx, r, field); err != nil {
		return setupFail(err)
	}
	for i := 0; i < tomeJunk; i++ {
		if err := l.GiveGround(ctx, r, l.JunkCode()); err != nil {
			return setupFail(err)
		}
	}
	v0, err := l.PView(ctx)
	if err != nil {
		return setupFail(err)
	}
	if v0.Tome < tomeMinCharges {
		return Blocks(fmt.Sprintf("the tome holds %d charges, the recipe needs %d or more", v0.Tome, tomeMinCharges))
	}
	mark := l.LogMark()
	empty, minQ := 0, v0.Tome
	for i := 0; i < tomeLooks; i++ {
		if err := l.Tick(ctx, tomeGap); err != nil {
			return setupFail(err)
		}
		v, err := l.PView(ctx)
		if err != nil {
			return setupFail(err)
		}
		if v.Tome <= 0 {
			empty++
		}
		if v.Tome < minQ {
			minQ = v.Tome
		}
	}
	hits := map[string]int{}
	for _, ln := range l.LogSince(mark) {
		for _, k := range tomeLogKeys {
			if strings.Contains(ln, k.Key) {
				hits[k.Path]++
			}
		}
	}
	o := Outcome{Intended: fmt.Sprintf("0 of %d looks read the tome empty, no failed-dump, stock-verb, no70 or flip line", tomeLooks),
		Observed: fmt.Sprintf("charges at start %d, lowest %d, empty looks %d of %d, log hits %s", v0.Tome, minQ, empty, tomeLooks, countString(hits)),
		Sig:      fmt.Sprintf("empty=%d hits=%d", empty, len(hits))}
	switch {
	case empty > 0 && hits["path 10 (failed dump items)"] > 0:
		o.Verdict, o.Reason = Fail, "empty read next to a failed dump: path 10 occurs in the field"
	case empty > 0 && hits["path 9 (stock items verb, no qty)"] > 0:
		o.Verdict, o.Reason = Fail, "empty read next to the stock items verb: path 9 occurs in the field"
	case empty > 0:
		o.Verdict, o.Reason = Fail, "empty read with neither warning: fork qty dropout (capture the raw dump items JSON)"
	case len(hits) > 0:
		o.Verdict, o.Reason = Fail, "a producer-path log line appeared although the reads were never empty"
	default:
		o.Verdict, o.Reason = Pass, "tome read non-empty at every look"
	}
	return o
}
