package combat

import (
	"slices"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Fighter.Drive (agency.planner 2): nil or empty changes nothing, a named attack the hero can use goes first, one it
// cannot use (or that does not exist) leaves the profile's list as it is.
func TestDriveAttackReordersOnlyUsableSkills(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level = 12
	g.p.Skills = []d2.Skill{{ID: 0, Level: 1}, {ID: 98, Level: 8}, {ID: 107, Level: 1}, {ID: 106, Level: 1}, {ID: 112, Level: 1}}
	base := attackNamesOf(h.f, g.p)
	if len(base) < 2 || base[0] != "Zeal" {
		t.Fatalf("profile order: %v", base)
	}
	h.f.Drive = nil
	if got := attackNamesOf(h.f, g.p); !slices.Equal(got, base) {
		t.Fatalf("nil Drive changed the list: %v vs %v", got, base)
	}
	for _, d := range []func() (string, string){func() (string, string) { return "", "" }, func() (string, string) { return "Nonexistent", "" }} {
		h.f.Drive = d
		if got := attackNamesOf(h.f, g.p); !slices.Equal(got, base) {
			t.Fatalf("empty or unknown drive changed the list: %v vs %v", got, base)
		}
	}
	other := base[1]
	h.f.Drive = func() (string, string) { return other, "" }
	got := attackNamesOf(h.f, g.p)
	if got[0] != other || len(got) != len(base) {
		t.Fatalf("drive %q: %v (profile %v)", other, got, base)
	}
}

// The driven aura goes ahead of the profile's standing auras; one the profile lists keeps its row.
func TestDriveAuraPrepends(t *testing.T) {
	h := &Fighter{}
	ladder := []AuraPref{{Name: "Might"}}
	if got := h.driveAura(ladder); len(got) != 1 {
		t.Fatalf("no Drive: %v", got)
	}
	h.Drive = func() (string, string) { return "", "Concentration" }
	got := h.driveAura(ladder)
	if len(got) != 2 || got[0].Name != "Concentration" || got[1].Name != "Might" {
		t.Fatalf("got %v", got)
	}
	h.Drive = func() (string, string) { return "", "" }
	if got := h.driveAura(ladder); len(got) != 1 {
		t.Fatalf("empty aura: %v", got)
	}
}

// Card fix3 (review 3 finding 4): the driven aura does not bypass the profile's safety rows. Both Paladin profiles: a
// hero driven to its crowd aura drops it against an immune target, under the fight life line and while resting hurt.
func TestDriveAuraKeepsLifeAndImmunityRules(t *testing.T) {
	for _, build := range []Build{"", "hammerdin"} {
		h, g := newHandGame(t)
		h.f.Build = build
		cat := palCatalog()
		cat.Skills[99] = &data.Skill{ID: 99, Key: "Prayer", Name: "Prayer", Class: "pal", Range: "none", Aura: true, ReqLevel: 6}
		h.f.SetCatalog(cat)
		p := g.p
		p.Level = 20
		p.Skills = skillsOf(98, 102, 99, 112)
		h.f.SetHotkeys(map[int]string{98: "f1", 102: "f2", 99: "f8"})
		pr := h.f.profileFor(p)
		crowd := ""
		for _, a := range pr.Aura {
			if a.NeedsCrowd && a.Elem != "" {
				crowd = a.Name
			}
		}
		listed := crowd != ""
		if !listed {
			crowd = "Holy Fire" // not on this profile's ladder: a plain entry, the life rows still come first
		} else if build != "" {
			t.Fatalf("%q now lists a crowd aura: drop the plain-entry arm of this test", build)
		}
		h.f.Drive = func() (string, string) { return "", crowd }
		p.MaxLife = 100
		h.f.auraCrowd = 9
		pick := func(life, potions int, tg *Target, resting bool) string {
			p.Life = life
			h.f.auraPotions, h.f.auraPotKnown = potions, true
			sp, _ := h.f.wantAura(p, pr, tg, resting)
			return sp.Name
		}
		immune := &Target{HasRes: true}
		immune.Res[elemOf("fire")] = 100
		if got := pick(90, 4, &Target{}, false); got != crowd {
			t.Errorf("%q healthy in a pack: %s, want the driven %s", build, got, crowd)
		}
		if got := pick(90, 4, immune, false); listed && (got == crowd || got == "") {
			t.Errorf("%q immune target: %q, the driven aura must give way to the profile's next one", build, got)
		}
		if got := pick(20, 1, &Target{}, false); got != "Prayer" {
			t.Errorf("%q at 20 percent life with one potion: %s, want Prayer", build, got)
		}
		if got := pick(40, 4, nil, true); got != "Prayer" {
			t.Errorf("%q resting at 40 percent life: %s, want Prayer", build, got)
		}
		h.f.auraCrowd = 0
		if got := pick(90, 4, &Target{}, false); listed && got == crowd {
			t.Errorf("%q no pack: the driven crowd aura %s is on", build, got)
		}
	}
}
