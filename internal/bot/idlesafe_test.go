package bot

import "testing"

func TestIdleSafeByState(t *testing.T) {
	for st, want := range map[State]bool{
		StateIdle: true, StateInTown: true, StatePaused: true, StateError: true,
		StateTraveling: false, StateClearing: false, StateResupplying: false, StateRecovering: false,
	} {
		r := &Runner{}
		r.st = st
		if got := r.IdleSafe(); got != want {
			t.Errorf("%s: IdleSafe %v, want %v", st, got, want)
		}
	}
}
