package loot

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// ArrivedFor reports the arrival check for a ground unit of the given class with held0 same-class items before the click.
func ArrivedFor(l *Looter, s *d2.Snapshot, id uint32, class, held0 int) bool {
	ok, _ := l.arrived(s, cand{u: d2.Unit{ID: id, Class: class}, held0: held0})
	return ok
}

// HeldOf exposes the pack-and-belt class count.
func HeldOf(l *Looter, inv *d2.Inventory, class int) int { return l.heldOf(inv, class) }

// Env exposes the looter's environment to the external tests.
func (l *Looter) Env() *env.Env { return l.e }
