package combat

import (
	"slices"
	"testing"

	"d2lod-bot/internal/d2"
)

func attackNamesOf(f *Fighter, p *d2.Player) []string {
	var out []string
	for _, sp := range f.attackSkills(p) {
		out = append(out, sp.Name)
	}
	return out
}

// noteLife feeds noteLifePrice a snapshot with the given life percent and healing potions in the belt.
func noteLife(t *testing.T, h *harness, g *handGame, lifePct, potions int) {
	t.Helper()
	h.s.stock(itHP1, 0, potions)
	s, _ := g.snap()
	s.Player.MaxLife, s.Player.Life = 100, lifePct
	h.f.noteLifePrice(s)
}

// The life-price gate (floor, hysteresis) still guards any profile that lists a life-price skill, but the Paladin
// profiles list none (Sacrifice is a prerequisite only, SME paladin-auras 2026-11-26): healthy or not, no Sacrifice attack.
func TestSacrificeNeverUnderTheLifeFloor(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level = 5
	g.p.Skills = skillsOf(107)
	g.p.LeftSkill = 107
	p := g.p
	floor := h.f.tune().SacrificeMinLifePct
	for _, lp := range []int{100, floor + 5, floor - 1, 1} {
		noteLife(t, h, g, lp, 2)
		if got := attackNamesOf(h.f, p); slices.Contains(got, "Sacrifice") {
			t.Fatalf("life %d%%: Sacrifice is an attack: %v", lp, got)
		}
	}
	// the gate itself: under the floor it stays off until the hysteresis is cleared, every life below the floor never offers it
	noteLife(t, h, g, floor-1, 2)
	if h.f.lifePriceOK() {
		t.Fatal("under the floor")
	}
	noteLife(t, h, g, floor+5, 2)
	if h.f.lifePriceOK() {
		t.Fatal("inside hysteresis")
	}
	noteLife(t, h, g, floor+lifePriceHysteresis, 2)
	if !h.f.lifePriceOK() {
		t.Fatal("recovered")
	}
	for lp := 1; lp < floor; lp++ {
		noteLife(t, h, g, 100, 2)
		noteLife(t, h, g, lp, 2)
		if h.f.lifePriceOK() {
			t.Fatalf("life %d%% offers a life-price skill", lp)
		}
	}
}

func TestSacrificeNeedsPotionsAndLiveMerc(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level = 5
	g.p.Skills = skillsOf(107)
	noteLife(t, h, g, 100, 0)
	if h.f.lifePriceOK() {
		t.Fatal("no potion in the belt")
	}
	noteLife(t, h, g, 100, 1)
	if !h.f.lifePriceOK() {
		t.Fatal("one potion is enough by default")
	}
	h.f.lifePrice.mercBad = true
	if h.f.lifePriceOK() {
		t.Fatal("dead merc")
	}
	h.f.lifePrice.mercBad = false
	h.f.e.Tune.Combat.Sacrifice = 0
	if h.f.lifePriceOK() {
		t.Fatal("knob off")
	}
}

// No change for other classes: no other profile has a life-price skill (the Amazon ladder moved Exploding Arrow first in card classapply).
func TestLifePriceOnlyPaladin(t *testing.T) {
	for _, pr := range Profiles() {
		if pr.Class != "paladin" && len(pr.LifePrice) != 0 {
			t.Errorf("%s has life-price skills %v", pr.ProfileID(), pr.LifePrice)
		}
		for _, b := range pr.Bands {
			if pr.Class != "paladin" && slices.Contains(b.Attack, "Sacrifice") {
				t.Errorf("%s ranks Sacrifice", pr.ProfileID())
			}
		}
		for _, a := range pr.Aura {
			if a.NeedsCrowd && pr.Class != "paladin" {
				t.Errorf("%s: needsCrowd on %s", pr.ProfileID(), a.Name)
			}
		}
	}
	known := func(n string) bool { return n == "Multiple Shot" || n == "Exploding Arrow" }
	got := ProfileFor(d2.Amazon, "").AttackNames(20, known)
	if got[0] != "Exploding Arrow" || slices.Contains(got, "Sacrifice") {
		t.Errorf("amazon: %v", got)
	}
}

// B2: Holy Fire needs combat.holyFireMinMonsters hostiles close; Might otherwise; once on it stays one monster longer.
func TestHolyFireCrowdCrossover(t *testing.T) {
	h, g := newHandGame(t)
	p := g.p
	p.Skills = skillsOf(98, 102)
	pr := h.f.profileFor(p)
	min := h.f.tune().HolyFireMinMonsters
	pick := func(n int, tg *Target) string {
		h.f.auraCrowd = n
		sp, _ := h.f.wantAura(p, pr, tg, false)
		return sp.Name
	}
	for n := 0; n < min; n++ {
		if got := pick(n, &Target{}); got != "Might" {
			t.Errorf("%d monsters: %s", n, got)
		}
	}
	if got := pick(min, &Target{}); got != "Holy Fire" {
		t.Errorf("%d monsters: %s", min, got)
	}
	if got := pick(min+5, nil); got != "Might" {
		t.Errorf("no target: %s", got)
	}
	g.aura, p.RightSkill = 102, 102 // Holy Fire is up: it stays at min-1
	if got := pick(min-1, &Target{}); got != "Holy Fire" {
		t.Errorf("hysteresis: %s", got)
	}
	if got := pick(min-2, &Target{}); got != "Might" {
		t.Errorf("pack gone: %s", got)
	}
	h.f.e.Tune.Combat.HolyFireMinMonsters = 1
	g.aura, p.RightSkill = 98, 98
	if got := pick(1, &Target{}); got != "Holy Fire" {
		t.Errorf("knob 1: %s", got)
	}
}

// Two Paladin profiles, every level 1 to 40, every skill learned: Sacrifice is never an attack and no life is paid.
func TestPaladinProfilesNeverAttackWithSacrifice(t *testing.T) {
	all := func(string) bool { return true }
	for _, build := range []Build{"", "hammerdin"} {
		pr, err := LookupProfile(d2.Paladin, build)
		if err != nil {
			t.Fatal(err)
		}
		if len(pr.LifePrice) != 0 {
			t.Errorf("%s: life price %v", pr.ProfileID(), pr.LifePrice)
		}
		for lv := 1; lv <= 40; lv++ {
			if slices.Contains(pr.AttackNames(lv, all), "Sacrifice") {
				t.Errorf("%s clvl %d: Sacrifice attacks", pr.ProfileID(), lv)
			}
		}
	}
}

// Profiles that avoid a skill list it in Avoid, and the Paladin ones avoid Sacrifice (the planner reads the same field).
func TestPaladinProfilesAvoidSacrifice(t *testing.T) {
	for _, build := range []Build{"", "hammerdin"} {
		pr, err := LookupProfile(d2.Paladin, build)
		if err != nil || !slices.Contains(pr.Avoid, "Sacrifice") {
			t.Errorf("%q: Sacrifice not avoided (err %v)", build, err)
		}
	}
}
