package merc_test

import (
	"errors"
	"testing"

	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/task"
)

type stub struct {
	s   merc.MercState
	err error
}

func (f stub) State() (merc.MercState, error)  { return f.s, f.err }
func (stub) Offers() ([]merc.HireOffer, error) { return nil, nil }

// AliveHook has the signature of task.MercAlive and the semantics the Andariel gate expects.
func TestAliveHookMatchesTaskHook(t *testing.T) {
	old := task.MercAlive
	defer func() { task.MercAlive = old }()
	for _, c := range []struct {
		name         string
		f            stub
		alive, known bool
	}{
		{"alive", stub{s: merc.MercState{Known: true, Owned: true, Alive: true}}, true, true},
		{"dead", stub{s: merc.MercState{Known: true, Owned: true}}, false, true},
		{"none", stub{s: merc.MercState{Known: true}}, false, false},
		{"unread", stub{}, false, false},
		{"error", stub{err: errors.New("x")}, false, false},
	} {
		task.MercAlive = merc.AliveHook(c.f)
		if a, k := task.MercAlive(); a != c.alive || k != c.known {
			t.Errorf("%s: %v %v", c.name, a, k)
		}
	}
}
