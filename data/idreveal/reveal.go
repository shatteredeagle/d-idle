// Package idrevealdata embeds the identify reveal wording (data/idreveal/reveal.json). The code is
// internal/watch/unlockidreveal.go.
package idrevealdata

import _ "embed"

// JSON is reveal.json.
//
//go:embed reveal.json
var JSON []byte
