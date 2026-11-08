// Package pactdata embeds the Pact catalogue (data/pacts/pacts.json). The loader and the rules are in internal/pact.
package pactdata

import _ "embed"

// JSON is pacts.json.
//
//go:embed pacts.json
var JSON []byte
