// Package bestiarydata embeds the bestiary flavour (data/bestiary/notes.json). The code is internal/watch/unlockbestiary.go.
package bestiarydata

import _ "embed"

// JSON is notes.json.
//
//go:embed notes.json
var JSON []byte
