package boss

import "context"

// Act45Runner is Runner for Acts 1 to 5: the plain scripts go through Runner.KillBoss (Scripts defaults to AllActs),
// and the three composite fights are dispatched by name: Diablo to RunDiablo (seals first), "The Ancients" to
// RunAncients (altar, no town portal) and Baal to RunBaal (waves, chamber portal). Ops is the object capability of
// the adapter; without it the composite fights fail with ErrAborted before any walk.
type Act45Runner struct {
	Runner
	Ops   ObjectOps
	Seals SealOpts
	Waves WaveOpts
	Order []int // Ancients order, DefaultAncientsOrder when empty
}

// KillBoss runs the named boss and records the result in Last.
func (r *Act45Runner) KillBoss(ctx context.Context, name string) error {
	var (
		res Result
		err error
	)
	switch name {
	case NameDiablo:
		res, err = RunDiablo(ctx, r.W, r.A, r.Ops, r.Seals, r.Opts)
	case NameAncients:
		res, err = RunAncients(ctx, r.W, r.A, r.Ops, r.Order, r.Opts)
	case NameBaal:
		res, err = RunBaal(ctx, r.W, r.A, r.Ops, r.Waves, r.Opts)
	default:
		if r.Scripts == nil {
			r.Scripts = AllActs()
		}
		return r.Runner.KillBoss(ctx, name)
	}
	r.Last = res
	return err
}
