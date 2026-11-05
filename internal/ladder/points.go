package ladder

import (
	"fmt"

	"d2lod-bot/internal/decide"
)

// Wired points (docs/t0-slice.md): only those the golden parity tests prove equal to the legacy code with the shipped
// rule file. Every other point stays on its legacy path.
const (
	PointRepair = "t1.repair"
	PointTPLost = "t1.tp.lost"
)

// RepairWanted answers t1.repair: is the lowest worn durability (percent) low enough to repair now? The legacy guards
// (latch, gold for the estimate) stay with the caller.
func (l *Ladder) RepairWanted(h Hero, durabilityPct int) (bool, error) {
	h.Durability = max(durabilityPct, 0)
	a, err := l.Decide(PointRepair, h, decide.Obs{})
	if err != nil {
		return false, err
	}
	return l.is(PointRepair, a, "repair")
}

// TPBuyWanted answers t1.tp.lost: buy a town portal scroll or walk. The caller still needs the gold for the price.
func (l *Ladder) TPBuyWanted(h Hero) (bool, error) {
	a, err := l.Decide(PointTPLost, h, decide.Obs{})
	if err != nil {
		return false, err
	}
	return l.is(PointTPLost, a, "buy")
}

func (l *Ladder) is(point, got, want string) (bool, error) {
	p, err := l.cfg.Cat.Point(point)
	if err != nil {
		return false, err
	}
	for _, a := range p.Answers {
		if a.ID == got {
			return got == want, nil
		}
	}
	return false, fmt.Errorf("ladder: %s: %q is not an answer", point, got)
}
