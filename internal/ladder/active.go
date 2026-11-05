package ladder

import "sync/atomic"

var active atomic.Pointer[Ladder]

// Active is the installed ladder, nil when ladder.enabled is off: every wired call site tests it first, so with no
// ladder installed the legacy code path runs unchanged.
func Active() *Ladder { return active.Load() }

// Install makes l the active ladder; Install(nil) removes it.
func Install(l *Ladder) { active.Store(l) }
