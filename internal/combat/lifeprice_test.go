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

// palprofile B1: a clvl 5 Paladin with Sacrifice attacks with it only while life, potions and the knob allow.
func TestSacrificeNeverUnderTheLifeFloor(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level = 5
	g.p.Skills = skillsOf(107)
	g.p.LeftSkill = 107
	p := g.p
	if got := attackNamesOf(h.f, p); slices.Contains(got, "Sacrifice") {
		t.Fatalf("nothing noted yet: Sacrifice must wait: %v", got)
	}
	noteLife(t, h, g, 100, 2)
	if got := attackNamesOf(h.f, p); !slices.Contains(got, "Sacrifice") {
		t.Fatalf("healthy: %v", got)
	}
	floor := h.f.tune().SacrificeMinLifePct
	noteLife(t, h, g, floor-1, 2)
	if got := attackNamesOf(h.f, p); slices.Contains(got, "Sacrifice") {
		t.Fatalf("under the floor: %v", got)
	}
	noteLife(t, h, g, floor+5, 2) // back above the floor but inside the hysteresis band
	if got := attackNamesOf(h.f, p); slices.Contains(got, "Sacrifice") {
		t.Fatalf("inside hysteresis: %v", got)
	}
	noteLife(t, h, g, floor+lifePriceHysteresis, 2)
	if got := attackNamesOf(h.f, p); !slices.Contains(got, "Sacrifice") {
		t.Fatalf("recovered: %v", got)
	}
	// every life value below the floor, from any previous state, never offers it
	for lp := 1; lp < floor; lp++ {
		noteLife(t, h, g, 100, 2)
		noteLife(t, h, g, lp, 2)
		if h.f.lifePriceOK() {
			t.Fatalf("life %d%% offers Sacrifice", lp)
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

// No change for other classes: no other profile has a life-price skill, and the Amazon ladder is untouched.
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
	if got[0] != "Multiple Shot" || slices.Contains(got, "Sacrifice") {
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
