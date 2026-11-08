// Package alertsdata embeds the alert rules (data/alerts/rules.json). The code is internal/watch/unlockalerts.go.
package alertsdata

import _ "embed"

// JSON is rules.json.
//
//go:embed rules.json
var JSON []byte
