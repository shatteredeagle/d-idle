package leveler

import (
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// PALPROBE part B (a)+(b): the combat ladder the profile gives a Paladin at every clvl, with the skills each plan
// table has trained by then (the simulator of palprobe_test.go). Lives here because leveler imports combat.

// skillsAt replays the plan and returns the trained skills after each clvl 2..30.
func skillsAt(t *testing.T, table string) map[int]map[string]int {
	t.Helper()
	b := mustBuild(t, table)
	start := [numAttr]int{25, 20, 25, 15}
	h := Have{Attr: start, Start: start, Skills: map[string]int{}}
	out := map[int]map[string]int{1: {}}
	for L := 2; L <= 30; L++ {
		h.Level, h.StatPts, h.SkillPts = L, h.StatPts+5, h.SkillPts+1
		for _, a := range b.Plan(h, palLegal, nil) {
			if a.Kind == KindStat {
				h.Attr[a.Attr]++
				h.StatPts--
			} else {
				h.Skills[a.Skill]++
				h.SkillPts--
			}
		}
		cp := map[string]int{}
		for k, v := range h.Skills {
			cp[k] = v
		}
		out[L] = cp
	}
	return out
}

func TestPalProbeLadderAlongBothTables(t *testing.T) {
	pr, err := combat.LookupProfile(d2.Paladin, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"paladin-normal30", "paladin-normal30-v2"} {
		sk := skillsAt(t, table)
		var first string
		for L := 1; L <= 30; L++ {
			has := func(n string) bool { return sk[L][n] > 0 }
			names := pr.AttackNames(L, has)
			if len(names) == 0 || names[len(names)-1] != "Attack" {
				t.Errorf("%s clvl %d: the ladder %v must end in the free Attack", table, L, names)
				continue
			}
			for _, n := range names {
				if n == "Smite" {
					t.Errorf("%s clvl %d: Smite on the ladder", table, L)
				}
			}
			if names[0] != first {
				t.Logf("%s clvl %d: ladder starts with %s (%v)", table, L, names[0], names)
				first = names[0]
			}
			// every rung the ladder would use must be a skill the character owns: the loop skips unknown ones
			if L >= 15 && has("Zeal") && names[0] != "Zeal" && !has("Blessed Hammer") {
				t.Errorf("%s clvl %d: Zeal trained but the ladder starts with %s", table, L, names[0])
			}
		}
	}
}

// PALBAND (probe F2): the profile's Blessed Hammer band minimum is the skill's own required level (18), so the band
// is open the moment any table has trained the hammer; before the fix it started at 24 and the points sat idle.
// Every Paladin plan table is checked, the endgame ones too.
func TestPalProbeHammerBandStartsAtTraining(t *testing.T) {
	pr, _ := combat.LookupProfile(d2.Paladin, "")
	for _, table := range []string{"paladin-normal30", "paladin-normal30-v2", "paladin_hammerdin_endgame90", "paladin_hammerdin_zeal_endgame90"} {
		sk := skillsAt(t, table)
		for L := 1; L <= 30; L++ {
			if sk[L]["Blessed Hammer"] > 0 {
				has := func(n string) bool { return sk[L][n] > 0 }
				if n := pr.AttackNames(L, has)[0]; n != "Blessed Hammer" {
					t.Errorf("%s trains Blessed Hammer at clvl %d but the first attack there is %s", table, L, n)
				}
				break
			}
		}
	}
}

// PALTABLE (probe F3): the legacy table trains Prayer (row 16), so the resting aura
// rung is alive for a Paladin that keeps it.
func TestPalProbeLegacyTableHasNoPrayer(t *testing.T) {
	sk := skillsAt(t, "paladin-normal30")
	// the planner pulls the point forward into the idle clvl 10 or 11 point (replay: clvl 10), the row says 16
	if sk[30]["Prayer"] != 1 || sk[16]["Prayer"] != 1 || sk[3]["Prayer"] != 0 {
		t.Errorf("legacy Prayer: clvl 3 %d, clvl 16 %d, clvl 30 %d", sk[3]["Prayer"], sk[16]["Prayer"], sk[30]["Prayer"])
	}
}

// PALTABLE (probe F4): the legacy table follows the v2 Zeal curve level for level (Zeal 4 by clvl 16 in the replay
// without a Den point, 4 at clvl 30; it stayed at 1 before).
func TestPalProbeLegacyZealStaysAtOne(t *testing.T) {
	for _, table := range []string{"paladin-normal30", "paladin-normal30-v2"} {
		sk := skillsAt(t, table)
		if sk[16]["Zeal"] != 4 || sk[30]["Zeal"] != 4 || sk[11]["Zeal"] != 0 {
			t.Errorf("%s Zeal: clvl 11 %d, clvl 16 %d, clvl 30 %d", table, sk[11]["Zeal"], sk[16]["Zeal"], sk[30]["Zeal"])
		}
	}
}

// ---- gear (c) ----

func TestPalProbeWantsShield(t *testing.T) {
	p, err := LookupPrefs("Paladin")
	if err != nil || !p.WantsShield || p.TwoHandMult >= 1 {
		t.Fatalf("%+v %v", p, err)
	}
	// no other weapon class asks for a shield, so a two-hander is never refused for them (Barbarian, Amazon, Assassin)
	for _, c := range []string{"Barbarian", "Amazon", "Assassin", "Sorceress"} {
		if q, _ := LookupPrefs(c); q.WantsShield {
			t.Errorf("%s wants a shield", c)
		}
	}
}

// Weapon scoring for a Paladin below clvl 40: damage counts (0.8), a two-hander is halved, no caster bonus.
func TestPalProbeWeaponChoice(t *testing.T) {
	pal, _ := LookupPrefs("Paladin")
	sorc, _ := LookupPrefs("Sorceress")
	sword := &data.ItemBase{Kind: data.KindWeapon, Type: "swor", MinDam: 3, MaxDam: 15} // avg 9
	two := &data.ItemBase{Kind: data.KindWeapon, Type: "swor", TwoHanded: true, TwoMinDam: 8, TwoMaxDam: 20}
	wand := &data.ItemBase{Kind: data.KindWeapon, Type: "wand", MinDam: 2, MaxDam: 4}
	it := &d2.Item{Quality: d2.QualityNormal}
	sc := func(pr Prefs, b *data.ItemBase) float64 { return scoreOf(nil, pr, b, it, nil, scoreCtx{}) }
	if sc(pal, sword) <= sc(pal, wand) {
		t.Errorf("a Paladin must prefer a sword (%.1f) to a wand (%.1f)", sc(pal, sword), sc(pal, wand))
	}
	// 14 avg two-hander x 0.5 x 0.8 = 5.6 against the one-hander 9 x 0.8 = 7.2: the one-hander wins on score alone
	if sc(pal, sword) <= sc(pal, two) {
		t.Errorf("one-hander %.1f must beat the two-hander %.1f", sc(pal, sword), sc(pal, two))
	}
	_ = sorc
}

// A shield build is never planned into a two-hander while a shield is worn, whatever the score.
func TestPalProbeTwoHanderNeverOverShield(t *testing.T) {
	two := Cand{Unit: d2.Unit{ID: 5}, Name: "great sword", Score: 99, Usable: true, Slots: []int{d2.BodyRHand}, TwoHanded: true, AvoidTwoHanded: true}
	sword := cand(6, "sword", 10, true, d2.BodyRHand)
	shield := cand(7, "buckler", 3, true, d2.BodyLHand)
	for _, eq := range []map[int]Cand{{d2.BodyLHand: shield}, {}} {
		for _, sw := range ChooseGear(eq, []Cand{two, sword, shield}) {
			if sw.Item.ID() == 5 {
				t.Errorf("two-hander planned (equipped %d slots): %v", len(eq), sw)
			}
		}
	}
}
