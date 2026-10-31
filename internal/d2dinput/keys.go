package d2dinput

import (
	"fmt"
	"strings"

	"d2lod-bot/internal/input"
)

// keyNames maps the key names the bot uses (the d2input helper's, lower case) to d2d's "key <name>" argument.
//
// Tags: V = accepted by a real d2d (docs/d2d-spike.md: "key Escape", "key Return"); S = SDL key names as the
// spike section 7 / d2d notes say ("key i opens panels by SDL key name"), not run by us; U = a guess.
// "key shift" is "err unknown key" (V), so modifiers are not keys here.
var keyNames = map[string]string{
	"esc": "Escape", "escape": "Escape", // V
	"enter": "Return", "return": "Return", // V
	"backspace": "Backspace", // U (SDL spelling; d2d may want BackSpace)
	"tab":       "Tab",       // S
	"space":     "Space",     // S
}

func init() {
	for c := 'a'; c <= 'z'; c++ { // S: panel hotkeys i c s t w ...; lower case is what the spike sent for "i"
		keyNames[string(c)] = string(c)
	}
	for c := '0'; c <= '9'; c++ { // U: belt potions 1-4
		keyNames[string(c)] = string(c)
	}
	for n := 1; n <= 12; n++ { // U: skill hotkeys F1-F8
		keyNames[fmt.Sprintf("f%d", n)] = fmt.Sprintf("F%d", n)
	}
}

// KeyName translates a bot key name to d2d's. Modifier keys are ErrUnsupported, other unknown names an error.
func KeyName(name string) (string, error) {
	l := strings.ToLower(name)
	switch l {
	case "shift", "ctrl", "control", "alt", "lshift", "rshift":
		return "", fmt.Errorf("%w: key %q (d2d has no modifier keys on the channel)", input.ErrUnsupported, name)
	}
	if d, ok := keyNames[l]; ok {
		return d, nil
	}
	return "", fmt.Errorf("d2dinput: unknown key %q (no d2d name known)", name)
}
