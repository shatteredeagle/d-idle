// Package chronicledata embeds the Chronicle wording (data/chronicle/wording.json): the lines of the kill and loot feed
// in Diablo II words. The code is internal/watch/unlockchronicle.go.
package chronicledata

import _ "embed"

// JSON is wording.json.
//
//go:embed wording.json
var JSON []byte
