package leveler

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestGearBuildsLoadAndDefault(t *testing.T) {
	for _, c := range []struct{ class, def string }{{"amazon", "amazon_bow_normal30"}, {"necromancer", "necromancer_summoner_normal30"}, {"paladin", "paladin-normal30"}, {"druid", "druid_fire_normal30"}, {"assassin", "assassin_fireblast_normal30"}, {"barbarian", "barbarian_melee_normal30"}} {
		if got := DefaultBuildName(c.class); got != c.def {
			t.Errorf("%s default %q want %q", c.class, got, c.def)
		}
		if b := mustBuild(t, c.def); b.Class != c.class || len(b.Rows) == 0 {
			t.Errorf("%s: %+v", c.def, b)
		}
	}
	for _, n := range []string{"necromancer_leveling_normal30", "amazon_javelin_normal30"} {
		mustBuild(t, n)
	}
}

func TestChooseGearKeepsQuiverAndBow(t *testing.T) {
	quiver := Cand{Unit: d2.Unit{ID: 1}, Ranged: true, Usable: true, Score: 1, Slots: nil}
	shield := cand(2, "shield", 50, true, d2.BodyLHand)
	sword := cand(3, "sword", 50, true, d2.BodyRHand)
	if sw := ChooseGear(map[int]Cand{d2.BodyLHand: quiver}, []Cand{shield}); len(sw) != 0 {
		t.Fatalf("shield over quiver: %v", sw)
	}
	bow := quiver
	if sw := ChooseGear(map[int]Cand{d2.BodyRHand: bow}, []Cand{sword}); len(sw) != 0 {
		t.Fatalf("sword over bow: %v", sw)
	}
	if sw := ChooseGear(map[int]Cand{}, []Cand{shield, sword}); len(sw) != 2 {
		t.Fatalf("plain fill: %v", sw)
	}
}

// ENDBUILD: every endgame table loads for its class, the default selection is untouched, and the alias maps per class.
func TestEndgameTables(t *testing.T) {
	for class, name := range endgameBuild {
		if got := ResolveBuild(EndgameAlias, class); got != name {
			t.Errorf("%s alias %q", class, got)
		}
		b := mustBuild(t, name)
		if b.Class != class || len(b.Rows) != 89 || b.Rows[0].Clvl != 2 || b.Rows[88].Clvl != 90 {
			t.Errorf("%s: class %q rows %d", name, b.Class, len(b.Rows))
		}
		for _, r := range b.Rows {
			if r.Attr[Str]+r.Attr[Dex]+r.Attr[Vit]+r.Attr[Ene] != 5 {
				t.Errorf("%s clvl %d: stat points != 5", name, r.Clvl)
			}
		}
		if DefaultBuildName(class) == name || ResolveBuild("", class) != "" {
			t.Errorf("%s: endgame must stay opt-in", class)
		}
	}
	mustBuild(t, "paladin_hammerdin_zeal_endgame90")
	if ResolveBuild("fallback", "sorceress") != "fallback" {
		t.Error("explicit names pass through")
	}
}

// Rows without skills are banked points; the unlock row spends the bank (Paladin row 18 spends 11 in the hammer table).
func TestEndgamePaladinBank(t *testing.T) {
	b := mustBuild(t, "paladin_hammerdin_endgame90")
	var at18 int
	for _, r := range b.Rows {
		if r.Clvl == 18 {
			for _, s := range r.Skills {
				if s.Skill == "Blessed Hammer" {
					at18 = s.Points
				}
			}
		}
		if r.Clvl > 3 && r.Clvl < 6 && len(r.Skills) != 0 {
			t.Errorf("clvl %d should bank", r.Clvl)
		}
	}
	if at18 != 11 {
		t.Errorf("row 18 spends %d, want 11", at18)
	}
}

// W4: how many Smite points exist at clvl 2-17 in the plain table versus the Zeal variant (documented in docs/gear.md).
func TestPaladinEarlyPoints(t *testing.T) {
	count := func(name, skill string) int {
		n := 0
		for _, r := range mustBuild(t, name).Rows {
			if r.Clvl > 17 {
				break
			}
			for _, s := range r.Skills {
				if s.Skill == skill {
					n += s.Points
				}
			}
		}
		return n
	}
	if got := count("paladin_hammerdin_endgame90", "Smite"); got != 1 {
		t.Errorf("plain table Smite points by 17 = %d, want 1", got)
	}
	v := "paladin_hammerdin_zeal_endgame90"
	if count(v, "Smite") < 8 || count(v, "Zeal") != 5 || count(v, "Sacrifice") != 1 {
		t.Errorf("variant smite %d zeal %d sacrifice %d", count(v, "Smite"), count(v, "Zeal"), count(v, "Sacrifice"))
	}
}

// The banked points of clvl 4-17 do not stall the planner: at clvl 5 nothing is planned, at 18 the bank goes into the hammer.
func TestEndgamePaladinPlanSpendsBank(t *testing.T) {
	b := mustBuild(t, "paladin_hammerdin_endgame90")
	have := map[string]int{}
	owned := 0
	for _, r := range b.Rows {
		if r.Clvl > 17 {
			break
		}
		for _, s := range r.Skills {
			have[s.Skill] += s.Points
			owned += s.Points
		}
	}
	h := Have{Attr: [4]int{86, 40, 100, 15}, Start: [4]int{25, 20, 25, 15}, Skills: have, Level: 5, StatPts: 0, SkillPts: 0}
	if got := b.Plan(h, nil, nil); len(got) != 0 {
		t.Fatalf("clvl 5 nothing to spend, got %v", got)
	}
	h.Level, h.SkillPts = 18, 11
	got := b.Plan(h, nil, nil)
	n := 0
	for _, a := range got {
		if a.Kind == KindSkill && a.Skill == "Blessed Hammer" {
			n++
		}
	}
	if n != 11 {
		t.Fatalf("clvl 18 spends %d hammer points of 11: %v", n, got)
	}
}
