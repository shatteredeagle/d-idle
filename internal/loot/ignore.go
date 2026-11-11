package loot

import "d2lod-bot/internal/d2"

// SetIgnore installs a test for ground items the pickup must leave alone: the items the player trashed from the watch
// page (invdesk.Desk.Ignored, docs/inventory-actions.md). nil removes it. Safe to call while the looter runs.
func (l *Looter) SetIgnore(f func(u d2.Unit) bool) {
	if f == nil {
		l.ignore.Store(nil)
		return
	}
	l.ignore.Store(&f)
}

func (l *Looter) ignored(u d2.Unit) bool {
	f := l.ignore.Load()
	return f != nil && (*f)(u)
}
