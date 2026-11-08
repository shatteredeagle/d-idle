package plandata_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/leveler"
)

var unlockClasses = []string{"sorceress", "amazon", "assassin", "barbarian", "druid", "necromancer", "paladin"}

// The normal30 tables each class plays (and the Sorceress v3 table).
var classTables = map[string][]string{
	"sorceress":   {leveler.FallbackV3, "sorceress_normal30"},
	"amazon":      {"amazon_bow_normal30", "amazon_javelin_normal30"},
	"assassin":    {"assassin_fireblast_normal30"},
	"barbarian":   {"barbarian_melee_normal30"},
	"druid":       {"druid_fire_normal30"},
	"necromancer": {"necromancer_leveling_normal30", "necromancer_summoner_normal30"},
	"paladin":     {leveler.PaladinV2, "paladin-normal30"},
}

// Expected clvl at which each event fires (S: veteran estimates, docs/skill-unlocks.md); a table row at or after it is
// never refused by an unlock.
var eventClvl = map[string]int{
	"quest:den_of_evil": 5, "boss:bishibosh": 8, "boss:blood_raven": 9, "quest:rescue_cain": 12,
	"boss:countess": 15, "waypoint:cold_plains": 8, "waypoint:catacombs_2": 17, "boss:andariel": 18,
}

// What each class must keep free: main attack, one defensive buff, a skill against immunities (class-early-game.md
// section 2 "Must stay free"). Names are Skills.txt keys.
var mustStayFree = map[string][]string{
	"sorceress":   {"Charged Bolt", "Frozen Armor", "Fire Bolt"},
	"amazon":      {"Jab", "Magic Arrow", "Fire Arrow", "Poison Javelin", "Multiple Shot", "Exploding Arrow", "Dodge"},
	"assassin":    {"Fire Trauma", "Wake of Fire Sentry", "Cloak of Shadows", "Tiger Strike"},
	"barbarian":   {"Bash", "Double Swing", "Shout", "Battle Orders", "Iron Skin"},
	"druid":       {"Firestorm", "Eruption", "Oak Sage", "Molten Boulder"},
	"necromancer": {"Raise Skeleton", "Teeth", "Bone Armor", "Clay Golem"},
	"paladin":     {"Might", "Holy Fire", "Zeal", "Prayer", "Resist Fire"},
}

func TestEveryClassHasAnUnlockFile(t *testing.T) {
	for _, c := range unlockClasses {
		u, err := leveler.LoadUnlocks(c, "")
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if u.Class != c || len(u.Unlocks) == 0 || len(u.MustFree) == 0 {
			t.Errorf("%s: class %q, %d unlocks, %d mustFree", c, u.Class, len(u.Unlocks), len(u.MustFree))
		}
		for _, e := range u.Unlocks {
			if e.Tag == "" {
				t.Errorf("%s/%s: no evidence tag", c, e.Skill)
			}
			for _, alt := range strings.Split(e.Event, "|") {
				if _, ok := eventClvl[alt]; !ok {
					t.Errorf("%s/%s: event %q has no expected clvl in this test", c, e.Skill, alt)
				}
			}
		}
	}
}

func TestUnknownClassIsAnErrorNamingIt(t *testing.T) {
	if _, err := leveler.LoadUnlocks("warlock", ""); err == nil || !strings.Contains(err.Error(), "warlock") {
		t.Fatalf("want an error naming warlock, got %v", err)
	}
}

func TestEachClassKeepsItsEssentialsFree(t *testing.T) {
	for _, c := range unlockClasses {
		u, err := leveler.LoadUnlocks(c, "")
		if err != nil {
			t.Fatal(err)
		}
		nothing := leveler.NewProgress()
		for _, s := range mustStayFree[c] {
			if !u.Unlocked(s, nothing, 1) {
				t.Errorf("%s: %s is locked, it must stay free", c, s)
			}
		}
	}
}

// With the events at the expected clvl no row of the class's build tables is refused by an unlock, so the unlock
// table cannot change a build (the same property as TestBuildV3WithUnlocks for the Sorceress).
func TestNoBuildRowIsRefusedByAnUnlock(t *testing.T) {
	for _, c := range unlockClasses {
		u, err := leveler.LoadUnlocks(c, "")
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range classTables[c] {
			b, err := leveler.LoadBuild(name, "")
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			for _, r := range b.Rows {
				p := leveler.NewProgress()
				for ev, at := range eventClvl {
					if r.Clvl >= at {
						p.Fire(ev)
					}
				}
				for _, sp := range r.Skills {
					if !u.Unlocked(sp.Skill, p, r.Clvl) {
						t.Errorf("%s: row clvl %d spends %s, still locked at the expected event levels", name, r.Clvl, sp.Skill)
					}
				}
			}
		}
	}
}

// Two classes through the engine: a locked row flows on and the point is not banked.
func TestTwoClassesReorderWithoutBanking(t *testing.T) {
	for c, skill := range map[string]string{"sorceress": "Frost Nova", "paladin": "Cleansing"} {
		u, err := leveler.LoadUnlocks(c, "")
		if err != nil {
			t.Fatal(err)
		}
		p := leveler.NewProgress()
		if u.Unlocked(skill, p, 30) {
			t.Fatalf("%s: %s should be locked with no event", c, skill)
		}
		if !u.Unlocked(skill, leveler.AllMetProgress(), 30) {
			t.Fatalf("%s: the all-met table must open %s", c, skill)
		}
	}
}

// Skill names against the real tables (skipped without the install): every name exists and belongs to the class.
func TestUnlockSkillsExistInTheTables(t *testing.T) {
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Skip("no real install tables: " + err.Error())
	}
	for _, c := range unlockClasses {
		u, err := leveler.LoadUnlocks(c, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := u.Validate(db); err != nil {
			t.Errorf("%v", err)
		}
		for _, s := range mustStayFree[c] {
			if db.SkillByName(s) == nil {
				t.Errorf("%s: test names unknown skill %q", c, s)
			}
		}
	}
}
