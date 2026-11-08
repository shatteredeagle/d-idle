// Package questdata embeds the Act 1 quest log text (data/quests/act1.json). The code is internal/watch/unlockquestlog.go.
package questdata

import _ "embed"

// JSON is act1.json.
//
//go:embed act1.json
var JSON []byte
