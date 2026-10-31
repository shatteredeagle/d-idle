package quest

import (
	"context"
	"fmt"
	"sync"
)

// FakeDialog is a Dialog driven by the Menus table. It tracks the open NPC and level, validates that the row
// index and line count match the table for the current quest state, and calls Effect for leaf clicks.
type FakeDialog struct {
	mu     sync.Mutex
	Status func(string) Status // quest state used to filter rows; nil = all rows
	// Effect runs after a click on a leaf row; it may change the fake world (consume an item, move the area).
	Effect func(npc int, level, label string)
	Calls  []string
	Err    error // returned by the next Talk
	// OnTalk runs when a dialog opens (a Talk with a quest item in the pack consumes it, S).
	OnTalk func(npc int)

	NPC   int
	Level string
	Open  bool
}

func (f *FakeDialog) rec(format string, a ...any) {
	f.Calls = append(f.Calls, fmt.Sprintf(format, a...))
}

func (f *FakeDialog) Talk(_ context.Context, npc int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec("talk %d", npc)
	if f.Err != nil {
		err := f.Err
		f.Err = nil
		return err
	}
	if _, ok := FindMenu(npc, LevelMain); !ok {
		return fmt.Errorf("%w: npc %d", ErrNoMenu, npc)
	}
	f.NPC, f.Level, f.Open = npc, LevelMain, true
	if f.OnTalk != nil {
		f.OnTalk(npc)
	}
	return nil
}

func (f *FakeDialog) Choose(_ context.Context, row, of int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.Open {
		return fmt.Errorf("quest: choose with no dialog open")
	}
	m, ok := FindMenu(f.NPC, f.Level)
	if !ok {
		return fmt.Errorf("%w: npc %d level %q", ErrNoMenu, f.NPC, f.Level)
	}
	rows := m.Visible(f.Status)
	if of != len(rows) || row < 0 || row >= len(rows) {
		return fmt.Errorf("quest: choose %d of %d, table has %d rows", row, of, len(rows))
	}
	r := rows[row]
	f.rec("choose %d/%d %s", row, of, r.Label)
	switch {
	case r.Label == "Cancel":
		f.Open = false
	case HasSubmenu(f.NPC, f.Level, r, f.Status):
		f.Level = r.Next
	default:
		if f.Effect != nil {
			f.Effect(f.NPC, f.Level, r.Label)
		}
	}
	return nil
}

func (f *FakeDialog) Close(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rec("close")
	f.Open = false
	return nil
}
