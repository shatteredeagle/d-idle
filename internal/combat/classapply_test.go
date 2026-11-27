package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
)

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

func known(names ...string) func(string) bool {
	return func(n string) bool {
		for _, k := range names {
			if k == n {
				return true
			}
		}
		return false
	}
}

// SME review 2026-11-27, confidence high: the bow table's 19-point Exploding Arrow leads from clvl 18, not the
// one-point Multiple Shot. The multishot table (own profile, confidence medium-high) leads with Multiple Shot from
// clvl 6. Two builds of one class.
func TestAmazonBandOrder(t *testing.T) {
	all := known("Multiple Shot", "Exploding Arrow", "Fire Arrow", "Magic Arrow")
	for _, tc := range []struct {
		build string
		clvl  int
		want  string
	}{{"", 18, "Exploding Arrow"}, {"", 30, "Exploding Arrow"}, {"multishot", 6, "Multiple Shot"}, {"multishot", 17, "Multiple Shot"}, {"multishot", 5, "Magic Arrow"}} {
		var p *Profile
		for i := range Profiles() {
			c := &Profiles()[i]
			if c.Class == "amazon" && c.Build == tc.build {
				p = c
			}
		}
		if p == nil {
			t.Fatalf("no amazon profile %q", tc.build)
		}
		if got := p.AttackNames(tc.clvl, all)[0]; got != tc.want {
			t.Errorf("amazon %q clvl %d leads with %s, want %s", tc.build, tc.clvl, got, tc.want)
		}
	}
}

// SME review 2026-11-27, confidence high: planTrap lays only keyed skills, so every trap the sentry table buys needs a
// layout key; Death Sentry had none.
func TestAssassinLayoutKeysTheSentryTable(t *testing.T) {
	for _, n := range []string{"Death Sentry", "Lightning Sentry", "Charged Bolt Sentry", "Wake of Fire Sentry"} {
		if k := ProfileFor(d2.Assassin, "").Layout[n]; k == "" { // d2.Assassin
			t.Errorf("assassin layout has no key for %s", n)
		}
	}
}
