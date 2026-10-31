// Package plandata embeds the snapshot of docs/plan/data that the bot and the leveler need, so the binary works
// anywhere. docs/plan/data stays the source; TestMatchesDocs keeps this copy equal (findings F15: one embedded copy,
// not one per consumer). Consumers let a plan directory override single files and fall back to FS.
package plandata

import "embed"

//go:embed *.json
var FS embed.FS
