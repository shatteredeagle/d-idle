package main

import (
	"context"
	"log/slog"
	"strings"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town"
)

// classPlan is what the bot runs for the logged-in character's class: the combat profile and build and the leveler's
// stat/skill table. COMBATCLASS made the fighter pick its profile per snapshot and LEVELERCLASS made the leveler default
// its table by class; this joins the two so they agree (a Sorceress on the leveler's "fallback" table fights with the
// Lightning profile) and puts the decision in the log at the start of every game.
type classPlan struct {
	Class        d2.Class
	Profile      string // combat profile name, e.g. "paladin"
	CombatBuild  combat.Build
	LevelerBuild string // leveler table, "" when the class has none
}

// planForClass resolves the pair. cfgBuild is the operator's explicit leveler table (config levelerBuild), which wins.
// level and owned (skills by name) pick the BUILDV2 table for a fresh character (leveler.SelectBuild).
func planForClass(c d2.Class, cfgBuild string, level int, owned map[string]int) classPlan {
	lb := leveler.ResolveBuild(cfgBuild, strings.ToLower(c.String()))
	if lb == "" {
		lb = leveler.SelectBuild(strings.ToLower(c.String()), level, owned)
	}
	cp := classPlan{Class: c, LevelerBuild: lb, CombatBuild: combat.BuildBlizzard}
	if c == d2.Sorceress && (lb == "fallback" || lb == leveler.FallbackV2) {
		cp.CombatBuild = combat.BuildLightning
	}
	pr := combat.ProfileFor(c, cp.CombatBuild)
	cp.Profile = pr.Class
	if pr.Build != "" {
		cp.Profile += "/" + pr.Build
	}
	return cp
}

// applyClass reads the character's class from memory and sets the fighter's build. Returns the plan for the log.
func applyClass(ctx context.Context, snap func() (*d2.Snapshot, error), cb *combat.Fighter, tw *town.Town, lt *loot.Looter, cfgBuild string, db *data.DB, log *slog.Logger) {
	s, err := snap()
	if err != nil || s.Player == nil {
		return
	}
	cp := planForClass(s.Player.Class, cfgBuild, s.Player.Level, leveler.OwnedSkills(db, s.Player))
	if cb.Build != cp.CombatBuild {
		cb.Build = cp.CombatBuild
	}
	cb.GearBuild = cp.LevelerBuild
	if tw != nil {
		tw.SetAmmoPath(ammoPath(cp.LevelerBuild))
		tw.SetBuild(cp.LevelerBuild)
	}
	if lt != nil {
		lt.SetBuild(cp.LevelerBuild)
	}
	log.Info("class-aware start", "class", s.Player.Class.String(), "level", s.Player.Level, "combatProfile", cp.Profile,
		"combatBuild", int(cp.CombatBuild), "levelerBuild", cp.LevelerBuild)
}

// ammoPath maps a gear build name to the town's ammo path ("" for builds without a launcher or thrown stack).
func ammoPath(build string) string {
	switch build {
	case "amazon_bow_normal30", "amazon_multishot_endgame90":
		return "bow"
	case "amazon_javelin_normal30":
		return "javelin"
	}
	return ""
}
