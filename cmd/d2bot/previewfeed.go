package main

import (
	"log/slog"
	"path/filepath"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/watch"
)

// useFightFeed feeds the policy preview (card previewfeed, docs/policy-preview.md): every finished fight becomes one
// "close" event of the away log (lowest life percent, potions, seconds, how it ended), and the away log's close and
// drop events are spooled under stateDir so a restart keeps them. Passive: it reads the fight's own telemetry and
// changes nothing the fighter does. No rig, no Fun or no fighter: nothing.
func useFightFeed(rig *ladderRig, f *combat.Fighter, stateDir string, log *slog.Logger) {
	if rig == nil || rig.Fun == nil || rig.Fun.Away == nil || f == nil {
		return
	}
	if stateDir != "" {
		if err := rig.Fun.Away.AttachSpool(filepath.Join(stateDir, "away-spool.jsonl")); err != nil && log != nil {
			log.Warn("away spool unavailable: the policy preview forgets its events on restart", "err", err)
		}
	}
	f.OnFight = fightFeed(rig.Fun.Away)
}

func fightFeed(away *watch.AwayLog) func(combat.FightEnd) {
	return func(e combat.FightEnd) {
		away.Add(watch.AwayEvent{Kind: watch.AwayClose, LifePct: e.MinLifePct, Area: status.AreaName(e.Area), AreaID: e.Area,
			Potions: e.Potions, Sec: e.Sec, Damage: e.Damage, Left: e.Left})
	}
}
