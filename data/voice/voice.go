// Package voicedata embeds the hero's voice lines (data/voice/lines.json). The code is internal/watch/unlockvoice.go.
package voicedata

import _ "embed"

// JSON is lines.json.
//
//go:embed lines.json
var JSON []byte
