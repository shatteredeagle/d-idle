package env

// InvdeskTuning holds the inventory-desk knobs (docs/design/item-verbs.md). Owner: the itemverbs card.
type InvdeskTuning struct {
	Verbs int // 1: do the player's equip through the backend's item verbs when it advertises them (click path otherwise)
}

func DefaultInvdeskTuning() InvdeskTuning { return InvdeskTuning{Verbs: 0} }

func invdeskKnobs() []Knob {
	return []Knob{
		{Key: "invdesk.verbs", Label: "Item verbs", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: a player's equip command goes through the private fork's item verbs (the server's own item handlers) when the connected d2d advertises them in caps; the click path stays the fallback and the oracle. 0 (default): always the click path. Off until the user has seen it work live (docs/design/item-verbs.md)."},
	}
}
