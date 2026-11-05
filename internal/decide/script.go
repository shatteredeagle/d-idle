package decide

import (
	"fmt"
	"time"
)

// Step is one line of an answer script: wait, reach a decision point, then reply the way a player profile would.
// Reply "" lets the thought fade; "notsure" or an answer id answers it. Confirm is "yes", "no" or "" (fade) for the
// confirm thought a risky point may open. Op "" asks; other ops are the player's list actions.
type Step struct {
	Wait    time.Duration
	Op      string // "", "revoke" (Key), "revokeall", "sacrifice" (Point), "rebirth" (Key = hero)
	Key     string
	Point   string
	Obs     Obs
	Reply   string
	Confirm string
}

// RunScript drives the store with a script on the virtual clock. It returns the decisions in order.
func RunScript(s *Store, clk *ManualClock, stage int, urg UrgencyProvider, steps []Step) ([]Decision, error) {
	var out []Decision
	for i, st := range steps {
		clk.Advance(st.Wait)
		if _, err := s.Tick(); err != nil {
			return out, fmt.Errorf("step %d: %w", i, err)
		}
		var err error
		switch st.Op {
		case "":
			var d Decision
			if d, err = s.step(st, stage, urg); err == nil {
				out = append(out, d)
			}
		case "revoke":
			err = s.Revoke(st.Key)
		case "revokeall":
			err = s.RevokeAll()
		case "sacrifice":
			err = s.Sacrifice(st.Point)
		case "rebirth":
			err = s.Rebirth(st.Key)
		default:
			err = fmt.Errorf("unknown script op %q", st.Op)
		}
		if err != nil {
			return out, fmt.Errorf("step %d: %w", i, err)
		}
	}
	return out, nil
}

func (s *Store) step(st Step, stage int, urg UrgencyProvider) (Decision, error) {
	ctx, err := s.cat.Build(st.Point, st.Obs, stage, urg)
	if err != nil {
		return Decision{}, err
	}
	d, err := s.Ask(ctx)
	if err != nil {
		return d, err
	}
	if d.Thought == nil || st.Reply == "" {
		return d, nil
	}
	if err := s.Answer(d.Thought.ID, st.Reply); err != nil {
		return d, err
	}
	s.mu.Lock()
	id, ok := s.openByKey[ctx.Key]
	confirm := ok && s.thoughts[id].Confirm
	s.mu.Unlock()
	if confirm && st.Confirm != "" {
		return d, s.Confirm(id, st.Confirm == "yes")
	}
	return d, nil
}
