package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

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
	Profile      string       // combat profile name, e.g. "paladin"
	CombatBuild  combat.Build // combat profile id inside the class ("" = class default)
	LevelerBuild string       // leveler table, "" when the class has none
	Err          error        // no combat profile for this class and table: the caller must log it and not fight on a guess
}

// planForClass resolves the pair. cfgBuild is the operator's explicit leveler table (config levelerBuild), which wins.
// level and owned (skills by name) pick the BUILDV2 table for a fresh character (leveler.SelectBuild).
func planForClass(c d2.Class, cfgBuild string, level int, owned map[string]int) classPlan {
	lb := leveler.ResolveBuild(cfgBuild, strings.ToLower(c.String()))
	if lb == "" {
		lb = leveler.SelectBuild(strings.ToLower(c.String()), level, owned)
	}
	cp := classPlan{Class: c, LevelerBuild: lb}
	if lb == "" { // a class without a leveler table fights with its own class default, named by the class
		cp.CombatBuild = combat.Build(strings.ToLower(c.String()))
	} else {
		// A table written for another class is refused here too (the leveler refuses it at spend time, after the fighter
		// has already taken that class's combat line). The loader refuses a table without a class or combat key.
		bd, err := leveler.LoadBuild(lb, "")
		if err != nil {
			cp.Err = fmt.Errorf("class %s, table %q: %w", c, lb, err)
			return cp
		}
		if !strings.EqualFold(bd.Class, c.String()) {
			cp.Err = fmt.Errorf("class %s was given table %q which belongs to class %s", c, lb, bd.Class)
			return cp
		}
		cp.CombatBuild = combat.Build(bd.Combat) // the table names its combat profile
	}
	pr, err := combat.LookupProfile(c, cp.CombatBuild)
	if err != nil {
		cp.Err = err
		return cp
	}
	cp.Profile = pr.ProfileID()
	return cp
}

// planBuild is the leveler build id chosen at the class-aware start; the play session record reads it (empty until then).
var planBuild atomic.Pointer[string]

// applyClass reads the character's class from memory and sets the fighter's build. Returns the plan for the log.
func applyClass(ctx context.Context, snap func() (*d2.Snapshot, error), cb *combat.Fighter, tw *town.Town, lt *loot.Looter, cfgBuild string, db *data.DB, log *slog.Logger) {
	s, err := snap()
	if err != nil || s.Player == nil {
		return
	}
	cp := planForClass(s.Player.Class, cfgBuild, s.Player.Level, leveler.OwnedSkills(db, s.Player))
	// No profile, no fight: the fighter refuses Clear and Kill while the error is set (it never fights on another
	// class's or build's profile) and the invariant monitor counts no-combat-profile every tick until it is fixed.
	cb.SetProfileErr(cp.Err)
	if cp.Err != nil {
		log.Error("class-aware start: NO COMBAT PROFILE, the hero will not fight (fix the table's class and combat keys)", "class", s.Player.Class.String(), "levelerBuild", cp.LevelerBuild, "err", cp.Err)
	} else if cb.Build != cp.CombatBuild {
		cb.Build = cp.CombatBuild
	}
	cb.GearBuild = cp.LevelerBuild
	planBuild.Store(&cp.LevelerBuild)
	if tw != nil {
		tw.SetAmmoPath(ammoPath(cp.LevelerBuild))
		tw.SetBuild(cp.LevelerBuild)
	}
	if lt != nil {
		lt.SetBuild(cp.LevelerBuild)
	}
	log.Info("class-aware start", "class", s.Player.Class.String(), "level", s.Player.Level, "combatProfile", cp.Profile,
		"combatBuild", string(cp.CombatBuild), "levelerBuild", cp.LevelerBuild)
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
