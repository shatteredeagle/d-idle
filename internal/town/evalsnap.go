package town

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
)

// snapEvaluator is a judge that can bind its context to one snapshot (loot.Looter.EvaluatorFor).
type snapEvaluator interface {
	EvaluatorFor(s *d2.Snapshot) func(d2.Unit) loot.Verdict
}

// evalFor is the judge bound to s: a pass over the pack judges every item against s instead of reading the game once
// per item (sellsnap: on d2d each EvaluateUnit took a full world snapshot, and the sell count is asked several times a
// tick). A judge without EvaluatorFor keeps EvaluateUnit.
func (t *Town) evalFor(s *d2.Snapshot) func(d2.Unit) loot.Verdict {
	if j, ok := t.judge.(snapEvaluator); ok {
		return j.EvaluatorFor(s)
	}
	return t.judge.EvaluateUnit
}
