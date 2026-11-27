package combat

import "testing"

// Card classapply (SME review 2026-11-27, confidence medium-high): a skill a profile band lists as an attack must not
// be a KindBuff skill, because the cast loop (castfirst.go, rules.go PickAttack) skips buffs, so the band entry is
// never cast. War Cry was one.
func TestBandAttackSkillsAreNotBuffKind(t *testing.T) {
	db := loadClassData()
	for _, p := range Profiles() {
		for _, b := range p.Bands {
			for _, n := range b.Attack {
				if m, ok := db.skills[n]; ok && m.Kind == KindBuff {
					t.Errorf("profile %s band attack %q is kind buff and is never cast", p.ProfileID(), n)
				}
			}
		}
	}
}

func TestWarCryIsAnAttackKind(t *testing.T) {
	m, ok := loadClassData().skills["War Cry"]
	if !ok || m.Kind != KindMelee || m.Range <= 0 {
		t.Fatalf("War Cry meta %+v ok=%v, want melee with a range", m, ok)
	}
}
