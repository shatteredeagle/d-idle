package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/resource"
)

// SIMCLASS: the Paladin stock scenario names its attacks after the class profile's ladder at clvl 15 with Zeal
// trained, so the profile stays the one source.
func TestPaladinScenarioFollowsProfileLadder(t *testing.T) {
	pr, err := LookupProfile(d2.Paladin, "")
	if err != nil {
		t.Fatal(err)
	}
	names := pr.AttackNames(15, func(n string) bool { return n == "Zeal" || n == "Sacrifice" })
	got := resource.BloodMoorPaladin().WithLadder(names)
	if got.MainAttack != "Zeal" || got.WeaponSkill != "Attack" {
		t.Errorf("profile ladder %v gives %s / %s", names, got.MainAttack, got.WeaponSkill)
	}
	if want := resource.BloodMoorPaladin(); want.MainAttack != got.MainAttack || want.WeaponSkill != got.WeaponSkill {
		t.Errorf("stock scenario names %s / %s differ from the profile", want.MainAttack, want.WeaponSkill)
	}
}
