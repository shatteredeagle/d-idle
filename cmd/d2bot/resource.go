package main

import (
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

// internal/resource imports env, so its knobs are registered from here: the dashboard Tuning view (env.Knobs) then
// lists the 23 resource.* knobs and the Tuner reads and writes them in env.Tuning.Resource.
func init() { env.RegisterKnobs(resource.Knobs) }
