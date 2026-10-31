package main

import (
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/d2dloot"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/nav"
)

// lootSeam is the d2denv Loot seam of the d2d stack: the stack's own Looter (the one the town judges with and the boss
// runs use, so the left-behind count and recovery state are shared) switched to d2d's ground reads, and wrapped for the
// runner. Pickup is "cmd pickup <unit>" through the controller (docs/d2d-loot.md). Without a d2d reader (tests with a
// foreign env) the looter keeps its own source.
func (s *d2dStack) lootSeam(e *env.Env, _ *nav.Navigator) bot.Looter {
	if rd, ok := e.Read.(*d2denv.Reader); ok {
		d2dloot.Attach(s.lt, rd.Source, e.Data)
	}
	return looterAdapter{s.lt}
}
