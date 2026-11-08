// Package atlasdata embeds the Act 1 waypoint table (data/atlas/waypoints.json). The code is internal/watch/unlockatlas.go.
package atlasdata

import _ "embed"

// JSON is waypoints.json.
//
//go:embed waypoints.json
var JSON []byte
