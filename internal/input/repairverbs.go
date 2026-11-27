package input

// The devp14 verbs repair and menu-choose (docs/design/input-i4.md). repair is the client's repair message (C->S 0x35)
// the Repair and Repair All buttons send; menu-choose picks a line of the open NPC dialog by its text. Both are listed
// in the fork's caps (data/input/caps/d2d-fork-forkcaps.json); the bot adopts them only behind input.planner.
const (
	CapRepair     = "input.repair"
	CapMenuChoose = "input.menu-choose"
)

// RepairAllItems is the item id that asks for Repair All.
const RepairAllItems = -1

// Repairer is the verb side of the smith's repair. The answer is only a request: callers read the durability back.
type Repairer interface {
	// Repair repairs carried or worn item id, or every item when id is RepairAllItems, in the open smith window.
	Repair(id int) error
}

// MenuChooser picks the line of the open NPC dialog that contains text.
type MenuChooser interface {
	MenuChoose(text string) error
}

// AsRepairer returns c's repair verb: c itself, or the one a wrapper's world verbs hold.
func AsRepairer(c Controller) (Repairer, bool) {
	if r, ok := c.(Repairer); ok {
		return r, true
	}
	if h, ok := c.(worldVerbsHolder); ok {
		if v, ok := h.WorldVerbs(); ok {
			r, ok := v.(Repairer)
			return r, ok
		}
	}
	return nil, false
}

// AsMenuChooser returns c's menu verb, the same way.
func AsMenuChooser(c Controller) (MenuChooser, bool) {
	if m, ok := c.(MenuChooser); ok {
		return m, true
	}
	if h, ok := c.(worldVerbsHolder); ok {
		if v, ok := h.WorldVerbs(); ok {
			m, ok := v.(MenuChooser)
			return m, ok
		}
	}
	return nil, false
}
