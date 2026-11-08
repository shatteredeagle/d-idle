// Package unlockdata embeds the unlock ladder (data/unlocks/ladder.json). The engine is in internal/unlock.
package unlockdata

import _ "embed"

// JSON is ladder.json.
//
//go:embed ladder.json
var JSON []byte
