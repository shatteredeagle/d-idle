package input

// CapUseItem is the capability of a fork build that lists "cmd use <item>" as ordinary input: the client's own
// use-item message (C->S 0x20, the right-click on a pack item; V fork source read, docs/design/input-i3.md). Its first
// user is the town portal cast from a scroll or tome; the bot adopts it only behind the knob input.planner and only
// when the connected build lists the name (U: no build advertises it yet; the differential comes first).
const CapUseItem = "input.use-item"

// ItemUser is the verb side of using a carried item (the right-click on it in the pack). The answer is only a request;
// callers verify the effect from game state.
type ItemUser interface {
	UseItem(id uint32) error
}

// AsItemUser returns c's item verb: c itself, or the world verbs a wrapper holds when they also use items.
func AsItemUser(c Controller) (ItemUser, bool) {
	if u, ok := c.(ItemUser); ok {
		return u, true
	}
	if h, ok := c.(worldVerbsHolder); ok {
		if v, ok := h.WorldVerbs(); ok {
			if u, ok := v.(ItemUser); ok {
				return u, true
			}
		}
	}
	return nil, false
}
