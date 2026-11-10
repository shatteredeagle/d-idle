// Package questdata embeds the Act 1 quest log text (data/quests/act1.json). The code is internal/watch/unlockquestlog.go.
package questdata

import _ "embed"

// JSON is act1.json.
//
//go:embed act1.json
var JSON []byte

// HintsJSON is act1-hints.json, the town hint table (TASK-197; the code is internal/watch/questhints.go).
//
//go:embed act1-hints.json
var HintsJSON []byte

// GiversJSON is act1-givers.json, the quest givers the campaign visits in town (docs/quest-givers.md).
//
//go:embed act1-givers.json
var GiversJSON []byte
