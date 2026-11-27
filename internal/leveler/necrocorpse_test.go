package leveler

import "testing"

func pointsTo30(t *testing.T, name string) (map[string]int, int) {
	t.Helper()
	lb, err := LoadBuild(name, "")
	if err != nil {
		t.Fatal(err)
	}
	got, total := map[string]int{}, 0
	for _, r := range lb.Rows {
		if r.Clvl > 30 {
			break
		}
		for _, s := range r.Skills {
			got[s.Skill] += s.Points
			total += s.Points
		}
	}
	return got, total
}

// necrocorpse: the pair for the Corpse Explosion points test. The candidate keeps one point and moves the other four
// to the minions; same points in total, same prerequisites, default untouched (the live scenario pair decides).
func TestCorpseExplosionPointsPair(t *testing.T) {
	base, baseTotal := pointsTo30(t, "necromancer_summoner_normal30")
	cand, candTotal := pointsTo30(t, "necromancer_summoner_ce1")
	if base["Corpse Explosion"] != 5 || cand["Corpse Explosion"] != 1 {
		t.Fatalf("Corpse Explosion points: default %d candidate %d, want 5 and 1", base["Corpse Explosion"], cand["Corpse Explosion"])
	}
	if baseTotal != candTotal {
		t.Errorf("points spent: default %d candidate %d", baseTotal, candTotal)
	}
	if cand["Raise Skeletal Mage"] != base["Raise Skeletal Mage"]+3 || cand["Clay Golem"] != base["Clay Golem"]+1 {
		t.Errorf("freed points: mage %d->%d golem %d->%d", base["Raise Skeletal Mage"], cand["Raise Skeletal Mage"], base["Clay Golem"], cand["Clay Golem"])
	}
	for _, k := range []string{"Teeth", "Raise Skeleton", "Skeleton Mastery", "Amplify Damage", "Bone Armor"} {
		if base[k] != cand[k] {
			t.Errorf("%s changed: %d -> %d", k, base[k], cand[k])
		}
	}
	// Teeth is the Corpse Explosion prerequisite: it must come at or before the first Corpse Explosion point.
	lb, _ := LoadBuild("necromancer_summoner_ce1", "")
	teeth, ce := 0, 0
	for _, r := range lb.Rows {
		for _, s := range r.Skills {
			if s.Skill == "Teeth" && teeth == 0 {
				teeth = r.Clvl
			}
			if s.Skill == "Corpse Explosion" && ce == 0 {
				ce = r.Clvl
			}
		}
	}
	if teeth == 0 || ce == 0 || teeth > ce {
		t.Errorf("Teeth at clvl %d, first Corpse Explosion at %d", teeth, ce)
	}
	if TableCombat("necromancer_summoner_ce1") != "summoner" {
		t.Errorf("candidate combat line %q", TableCombat("necromancer_summoner_ce1"))
	}
	if n := DefaultBuildName("necromancer"); n != "necromancer_summoner_normal30" {
		t.Errorf("default changed: %s", n)
	}
	if n := ResolveBuild("necromancer_summoner_ce1", "necromancer"); n != "necromancer_summoner_ce1" {
		t.Errorf("selector resolves to %q", n)
	}
}
