package input

import "errors"

// ErrUnsupported is returned by a backend that cannot do what was asked (never a silent no-op).
var ErrUnsupported = errors.New("input: not supported by this backend")

// SemanticInput is an optional extension of Controller for backends that act on game objects
// instead of pixels (the d2d control channel: "cmd move|skill|interact|pickup|belt"). Positions are
// world subtiles like everywhere else in the bot. The bot may type-assert a Controller to it and
// prefer these calls when present (not wired yet); AsSemantic is the one place to do that.
type SemanticInput interface {
	// Move starts walking to w and returns at once (like ClickWorld).
	Move(w Point) error
	// Cast casts skill id (the game's skill id) at w, at unit when unit != 0.
	Cast(skill int, w Point, unit uint32) error
	// InteractNPC walks to the NPC (index as the backend lists it) and opens its menu.
	InteractNPC(index int) error
	// Pickup walks to a ground item and picks it up.
	Pickup(unit uint32) error
	// ChooseMenu clicks the NPC menu line whose text matches (case-insensitive).
	ChooseMenu(text string) error
	// UsePortal walks to town portal number row and enters it.
	UsePortal(row int) error
	// Belt drinks the potion of belt column slot; merc gives it to the mercenary (the shift-click).
	Belt(slot int, merc bool) error
}

// AsSemantic returns c's semantic side when it has one.
func AsSemantic(c Controller) (SemanticInput, bool) {
	s, ok := c.(SemanticInput)
	return s, ok
}
