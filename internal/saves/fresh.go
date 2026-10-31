package saves

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/cheat/d2s"
)

var classNames = [...]string{"amazon", "sorceress", "necromancer", "paladin", "barbarian", "druid", "assassin"}

// CheckFresh verifies that f is a brand new softcore, non-ladder expansion character of the given name and
// class: level 1 and no experience (a new character has no exp stat entry, V per docs/cheats.md). The status
// byte layout is V (docs/saves.md).
func CheckFresh(f *d2s.File, name, class string) error {
	if !strings.EqualFold(f.Name(), name) {
		return fmt.Errorf("name %q, want %q", f.Name(), name)
	}
	c := f.Class()
	if c < 0 || c >= len(classNames) || !strings.EqualFold(classNames[c], class) {
		return fmt.Errorf("class byte %d, want %s", c, class)
	}
	if f.Level() != 1 {
		return fmt.Errorf("level %d, want 1", f.Level())
	}
	if exp, _ := f.Stat(d2s.StatExp); exp != 0 {
		return fmt.Errorf("experience %d, want 0", exp)
	}
	st := f.Status()
	switch {
	case st&d2s.FlagHardcore != 0:
		return fmt.Errorf("hardcore flag set")
	case st&d2s.FlagLadder != 0:
		return fmt.Errorf("ladder flag set")
	case st&d2s.FlagExpansion == 0:
		return fmt.Errorf("not an expansion character")
	}
	return nil
}
