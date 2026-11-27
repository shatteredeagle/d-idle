package nav

// teleKey is the hotkey Teleport is cast with: the fixed TeleportKey (rigs, tests) or what TeleportKeyFn reports now.
// Empty means the character has no usable Teleport and every caller walks.
func (n *Navigator) teleKey() string {
	if n.TeleportKey != "" {
		return n.TeleportKey
	}
	if n.TeleportHopFn != nil {
		if h := n.TeleportHopFn(); h.Refused == "" {
			return h.Key
		}
		return ""
	}
	if n.TeleportKeyFn != nil {
		return n.TeleportKeyFn()
	}
	return ""
}

// HasTeleport reports whether a Teleport key is bound right now (task, boss and farm read this as CanTeleport).
func (n *Navigator) HasTeleport() bool { return n.hasTeleport() }

// teleAllowed applies the level rule (towns, Duriel's Lair: data.CanTeleportIn) when the bot installed one.
func (n *Navigator) teleAllowed(level int) bool { return n.TeleportOK == nil || n.TeleportOK(level) }

// TeleportHop is the move driver's description of one hop (combat.HopInfo has the same fields).
type TeleportHop struct {
	Key     string
	Range   float64
	MinHop  float64
	Cost    int
	Refused string // why there is no hop now; empty with a Key means there is one
}

// hopRow is the driver's answer when one is wired and does not refuse.
func (n *Navigator) hopRow() (TeleportHop, bool) {
	if n.TeleportHopFn == nil {
		return TeleportHop{}, false
	}
	h := n.TeleportHopFn()
	return h, h.Refused == "" && h.Key != ""
}

// teleMinHop is the nearest landing worth a cast: the driver's row or the default.
func (n *Navigator) teleMinHop() float64 {
	if h, ok := n.hopRow(); ok && h.MinHop > 0 {
		return h.MinHop
	}
	return teleportMinHop
}
