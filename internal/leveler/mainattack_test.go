package leveler

import (
	"testing"

	"d2lod-bot/internal/gear"
)

// necroteeth: a hero prepared at clvl 12 with its points banked (the lab prep, featuresoak pass 1) gets the plan's main
// attack as soon as the bot spends those points: the default table learns it by clvl 12 and binds it to a hotkey. The
// lab rows that killed only with plain Attack had the points banked (resource.pointsDecisions 0), not a data gap.
func TestDefaultPlanLearnsAndBindsMainAttackByClvl12(t *testing.T) {
	const clvl = 12
	for _, class := range []string{"necromancer", "druid"} {
		name := gear.DefaultBuild(class)
		if name == "" {
			t.Fatalf("%s: no default build", class)
		}
		ma, err := gear.MainAttackAt(name, clvl)
		if err != nil {
			t.Fatal(err)
		}
		if ma.Skill == "" || ma.Skill == "Attack" {
			t.Fatalf("%s: main attack at clvl %d is %q", name, clvl, ma.Skill)
		}
		b, err := LoadBuild(name, "")
		if err != nil {
			t.Fatal(err)
		}
		pts := 0
		for _, r := range b.Rows {
			if r.Clvl > clvl {
				break
			}
			for _, s := range r.Skills {
				if s.Skill == ma.Skill {
					pts += s.Points
				}
			}
		}
		if pts == 0 {
			t.Errorf("%s: main attack %s has no point by clvl %d", name, ma.Skill, clvl)
		}
		bound := false
		for _, h := range b.Hotkeys {
			if h.Skill == ma.Skill && h.FromClvl <= clvl && (h.UntilClvl == 0 || clvl <= h.UntilClvl) {
				bound = true
			}
		}
		if !bound {
			t.Errorf("%s: main attack %s has no hotkey at clvl %d", name, ma.Skill, clvl)
		}
	}
}
