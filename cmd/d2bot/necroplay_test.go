package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	capdata "d2lod-bot/data/capabilities"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/plandata"
)

// Card necroepic, strengthened by fable review 8 (C3, C4): every plan table of every class resolves to a combat
// profile, and each skill the table uses as a main attack has a key in the profile's layout AND is an attack of one of
// its bands (a key alone is not enough: the Assassin's sentries had keys and were in no band, so only the Go trap driver
// ever reached them). A table without main_attack is not skipped: one of the skills it buys must be a keyed band attack
// of its profile. mainAttackDebt is the ratchet of today's known gaps: a listed gap that is fixed must be removed.
var mainAttackDebt = map[string]string{
	"necromancer_summoner_endgame90/Corpse Explosion": "cast by the corpse driver (corpseAims), not a band attack",
	"necromancer_summoner_normal30/Corpse Explosion":  "cast by the corpse driver (corpseAims), not a band attack",
	"druid_tornado_endgame90/Raven":                   "a summon (minions block), the table lists it as the clvl 2-5 attack",
	"sorceress_blizzard_endgame90/Ice Bolt":           "in the bands, but the blizzard layout has no key for it (all eight keys taken)",
	"necromancer_summoner_sme/":                       "buys no direct attack: the army kills, the band falls through to Attack",
}

func TestMainAttackSkillsHaveAKey(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "internal", "plandata", "build-*.json"))
	perClass := map[string]int{}
	debtSeen := map[string]bool{}
	gap := func(build, skill, format string, a ...any) {
		k := build + "/" + skill
		if _, ok := mainAttackDebt[k]; ok {
			debtSeen[k] = true
			return
		}
		t.Errorf("%s: "+format, append([]any{build}, a...)...)
	}
	for _, f := range files {
		if filepath.Base(f) == plandata.CatalogFile {
			continue
		}
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			Class      string `json:"class"`
			Build      string `json:"build"`
			MainAttack []struct {
				Skill string `json:"skill"`
			} `json:"main_attack"`
			Rows []struct {
				Skills []struct {
					Skill string `json:"skill"`
				} `json:"skills"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(raw, &b); err != nil || b.Build == "" {
			t.Errorf("%s: not a build table (%v)", f, err)
			continue
		}
		c, ok := classByName(b.Class)
		if !ok {
			t.Errorf("%s: unknown class %q", b.Build, b.Class)
			continue
		}
		sel := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "build-"), ".json")
		if _, err := leveler.LoadBuild(sel, ""); err != nil {
			sel = b.Build // tables registered under another selector are still reachable by their inner name or not at all
		}
		pr, err := combat.LookupProfile(c, combat.Build(leveler.TableCombat(sel)))
		if lb, lerr := leveler.LoadBuild(sel, ""); lerr == nil {
			cp := planForClass(c, sel, 30, nil)
			if cp.Err != nil {
				t.Errorf("%s: %v", b.Build, cp.Err)
				continue
			}
			pr, err = combat.LookupProfile(c, combat.Build(lb.Combat))
		} else {
			var keys struct{ Combat string }
			_ = json.Unmarshal(raw, &keys)
			if keys.Combat == "" {
				t.Errorf("%s: no combat key", b.Build)
				continue
			}
			pr, err = combat.LookupProfile(c, combat.Build(keys.Combat))
		}
		if err != nil {
			t.Errorf("%s: %v", b.Build, err)
			continue
		}
		perClass[strings.ToLower(b.Class)]++
		if len(b.MainAttack) == 0 {
			found := false
			for _, r := range b.Rows {
				for _, s := range r.Skills {
					found = found || keyedBandAttack(pr, s.Skill)
				}
			}
			if !found {
				gap(b.Build, "", "no main_attack and none of the skills it buys is a keyed band attack of profile %s", pr.ProfileID())
			}
			continue
		}
		for _, m := range b.MainAttack {
			if m.Skill == "Attack" || m.Skill == "Throw" || m.Skill == "" { // the generic attacks (capabilities.json "generic") are the weapon's basic attack, no key
				continue
			}
			if !keyedBandAttack(pr, m.Skill) {
				_, key := pr.Layout[m.Skill]
				gap(b.Build, m.Skill, "main attack %q needs a key (has one: %v) and a band in profile %s", m.Skill, key, pr.ProfileID())
			}
		}
	}
	for k, why := range mainAttackDebt {
		if !debtSeen[k] {
			t.Errorf("mainAttackDebt entry %q (%s) no longer occurs: remove it", k, why)
		}
	}
	for _, c := range []string{"amazon", "assassin", "barbarian", "druid", "necromancer", "paladin", "sorceress"} {
		if perClass[c] < 1 {
			t.Errorf("no table of class %s was checked (%v)", c, perClass)
		}
	}
	if perClass["necromancer"] < 3 {
		t.Fatalf("tables seen %v", perClass)
	}
}

// keyedBandAttack reports whether skill is an attack of a band of p and can be selected: a right-hand skill needs a
// key in the layout, a left-hand skill (capabilities.json delivery.use.hand) is picked in the left skill list and
// needs none (hotkeys only ever select the right skill, classdata.go Hand).
func keyedBandAttack(p *combat.Profile, skill string) bool {
	if _, ok := p.Layout[skill]; !ok && !leftHandSkills()[skill] {
		return false
	}
	for _, b := range p.Bands {
		for _, a := range b.Attack {
			if a == skill {
				return true
			}
		}
	}
	return false
}

var leftHandSkills = sync.OnceValue(func() map[string]bool {
	var caps struct {
		Skills map[string]struct {
			Delivery struct {
				Use struct {
					Hand string `json:"hand"`
				} `json:"use"`
			} `json:"delivery"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(capdata.JSON, &caps); err != nil {
		panic(err)
	}
	out := map[string]bool{}
	for n, r := range caps.Skills {
		out[n] = r.Delivery.Use.Hand == "left"
	}
	return out
})
