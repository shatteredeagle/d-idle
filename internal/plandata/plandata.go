// Package plandata embeds the snapshot of docs/plan/data that the bot and the leveler need, so the binary works
// anywhere. This directory is the single source: docs/plan/data is a mirror that sharedgen regenerates at merge
// (tools/plandatasync); never edit the mirror (findings F15: one embedded copy, not one per consumer). Consumers let a plan directory override single files and fall back to FS.
package plandata

import "embed"

//go:embed *.json
var FS embed.FS
