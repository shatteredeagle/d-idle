package admit

import "testing"

const (
	bloodMoor  = 2
	stonyField = 4
	darkWood   = 5
	den        = 8
	caveL1     = 9
	rogueCamp  = 1
)

// hero is a Normal clvl 12 hero standing in the camp with a full kit; tests change one thing at a time.
func hero(arch string) View {
	return View{
		Archetype: arch, Normal: true, Level: 12, Life: 176, MaxLife: 176, Healing: 6, HasTP: true, Gold: 500,
		InTown: true, Area: rogueCamp, FirstField: bloodMoor, PotionGold: 50, TPGold: 100,
		IsTown: func(a int) bool { return a == rogueCamp },
		MLvl:   func(a int) int { return map[int]int{bloodMoor: 2, stonyField: 6, darkWood: 7, den: 3, caveL1: 3}[a] },
		Knobs:  map[string]int{"campaign.denMinLevelMelee": 3, "campaign.denMinLevelCaster": 5},
	}
}

func mustRows(t *testing.T) Rows {
	t.Helper()
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMayEnterTable(t *testing.T) {
	r := mustRows(t)
	cases := []struct {
		name   string
		arch   string
		tweak  func(*View)
		area   int
		p      Purpose
		refuse bool
		need   string
		row    string
	}{
		{"full kit melee field", "melee", nil, stonyField, Level, false, "", ""},
		{"full kit caster field", "caster", nil, stonyField, Level, false, "", ""},
		// Questy 2026-11-26: lean hero (no scroll, potions 0 of 4) entering Stony Field
		{"questy lean no potions melee", "melee", func(v *View) { v.HasTP, v.Healing, v.InTown, v.Area = false, 0, false, bloodMoor }, stonyField, Level, true, NeedPotions, "kit-floor"},
		{"questy lean no potions caster", "caster", func(v *View) { v.HasTP, v.Healing, v.InTown, v.Area = false, 0, false, bloodMoor }, stonyField, Level, true, NeedPotions, "kit-floor"},
		{"poverty monster level", "melee", func(v *View) { v.Healing, v.Level, v.FirstField = 0, 8, stonyField }, stonyField, Level, true, NeedPotions, "poverty"},
		// potions 0/4 with a scroll: the kit floor holds the first field only
		{"potions 0 with scroll melee", "melee", func(v *View) { v.Healing, v.Level = 0, 8 }, stonyField, Farm, true, NeedPotions, "poverty"},
		{"potions 3 level 11 kit floor not poverty", "melee", func(v *View) { v.Healing, v.Level = 3, 11 }, stonyField, Farm, true, NeedPotions, "kit-floor"},
		{"potions 2 with scroll caster first field ok", "caster", func(v *View) { v.Healing, v.Level = 2, 9 }, bloodMoor, Farm, false, "", ""},
		{"potions 2 with scroll caster stony", "caster", func(v *View) { v.Healing, v.Level = 2, 9 }, stonyField, Farm, true, NeedPotions, "kit-floor"},
		// 90/176 life, no portal, 3 potions entering a field from a field
		{"life 90 of 176 lean melee", "melee", func(v *View) { v.HasTP, v.Healing, v.Life, v.InTown, v.Area = false, 3, 90, false, stonyField }, bloodMoor, Level, true, NeedNotReady, "lean"},
		{"life 90 of 176 with portal", "melee", func(v *View) { v.Life, v.InTown, v.Area = 90, false, bloodMoor }, stonyField, Level, false, "", ""},
		{"life 120 of 176 lean ok", "caster", func(v *View) { v.HasTP, v.Healing, v.Life, v.InTown, v.Area = false, 3, 120, false, stonyField }, bloodMoor, Level, false, "", ""},
		// Dark Wood life floor 180 (melee), 100 (caster)
		{"dark wood melee life 176", "melee", nil, darkWood, QuestWalk, true, NeedTooWeak, "underground_passage"},
		{"dark wood melee life 185", "melee", func(v *View) { v.MaxLife = 185 }, darkWood, QuestWalk, false, "", ""},
		{"dark wood caster life 176", "caster", nil, darkWood, QuestWalk, false, "", ""},
		{"dark wood caster life 90", "caster", func(v *View) { v.MaxLife = 90 }, darkWood, QuestWalk, true, NeedTooWeak, "underground_passage"},
		{"dark wood melee level 6", "melee", func(v *View) { v.Level, v.MaxLife = 6, 300 }, darkWood, QuestWalk, true, NeedTooWeak, "underground_passage"},
		{"dark wood caster level 7", "caster", func(v *View) { v.Level = 7 }, 10, QuestWalk, true, NeedTooWeak, "underground_passage"},
		{"dark wood one potion", "caster", func(v *View) { v.Healing = 1 }, darkWood, QuestWalk, true, NeedPotions, "underground_passage"},
		{"dark wood no scroll", "caster", func(v *View) { v.HasTP = false }, darkWood, QuestWalk, true, NeedPotions, "underground_passage"},
		// corpse run, owner try anyway, other difficulty, trip home: exempt
		{"corpse run exempt", "melee", func(v *View) { v.Healing, v.HasTP = 0, false }, darkWood, CorpseRun, false, "", ""},
		{"try anyway", "caster", func(v *View) { v.Healing, v.HasTP, v.Try = 0, false, true }, darkWood, QuestWalk, false, "", ""},
		{"nightmare unruled", "melee", func(v *View) { v.Normal, v.Healing = false, 0 }, darkWood, Level, false, "", ""},
		{"trip home", "melee", func(v *View) { v.Healing = 0 }, rogueCamp, Trip, false, "", ""},
		{"town target", "melee", func(v *View) { v.Healing = 0 }, rogueCamp, Level, false, "", ""},
		// knob floor: the Den
		{"den melee 3", "melee", func(v *View) { v.Level = 2 }, den, QuestWalk, true, NeedTooWeak, "den_of_evil"},
		{"den caster clvl 4", "caster", func(v *View) { v.Level = 4 }, den, QuestWalk, true, NeedTooWeak, "den_of_evil"},
		{"den caster clvl 5", "caster", func(v *View) { v.Level = 5 }, den, QuestWalk, false, "", ""},
		{"den without knob value", "caster", func(v *View) { v.Level, v.Knobs = 1, nil }, den, QuestWalk, false, "", ""},
		// legacy provider
		{"legacy judge refuses", "melee", func(v *View) { v.Legacy = func(a int) string { return "level 20 needed for the Barracks" } }, 28, Level, true, NeedRoute, "legacy"},
		{"legacy judge open", "melee", func(v *View) { v.Legacy = func(int) string { return "" } }, 28, Level, false, "", ""},
		{"standing in the area not asked", "melee", func(v *View) { v.Healing, v.Area, v.InTown = 0, stonyField, false; v.Level = 7 }, stonyField, Farm, false, "", ""},
	}
	for _, c := range cases {
		v := hero(c.arch)
		if c.tweak != nil {
			c.tweak(&v)
		}
		got := r.MayEnter(v, c.area, c.p)
		if got.Refused != c.refuse || got.Need != c.need || got.Row != c.row {
			t.Errorf("%s: got %+v, want refused=%v need=%q row=%q", c.name, got, c.refuse, c.need, c.row)
		}
		if got.Refused && (got.Wake == "" || got.Why == "") {
			t.Errorf("%s: a refusal names its wake and why: %+v", c.name, got)
		}
	}
}

func TestMayLeaveTownTable(t *testing.T) {
	r := mustRows(t)
	cases := []struct {
		name   string
		arch   string
		tweak  func(*View)
		p      Purpose
		refuse bool
		need   string
		kit    bool
	}{
		{"full kit quest walk", "melee", nil, QuestWalk, false, "", false},
		{"zero potions rich purse", "caster", func(v *View) { v.Healing = 0 }, QuestWalk, true, NeedPotions, true},
		{"zero potions poor purse", "melee", func(v *View) { v.Healing, v.Gold = 0, 7 }, QuestWalk, true, NeedGoldFarm, false},
		{"no scroll rich", "caster", func(v *View) { v.HasTP = false }, QuestWalk, true, NeedPotions, true},
		{"no scroll poor", "melee", func(v *View) { v.HasTP, v.Gold = false, 10 }, QuestWalk, true, NeedGoldFarm, false},
		{"above the quest walk band", "melee", func(v *View) { v.Level, v.Healing = 15, 0 }, QuestWalk, false, "", false},
		{"below the quest walk band", "caster", func(v *View) { v.Level, v.Healing = 4, 0 }, QuestWalk, false, "", false},
		{"farm needs no quest kit", "melee", func(v *View) { v.Healing = 0 }, Farm, false, "", false},
		{"lean hurt leaves", "melee", func(v *View) { v.HasTP, v.Healing, v.Life = false, 3, 90 }, Level, true, NeedNotReady, false},
		{"lean healthy leaves", "caster", func(v *View) { v.HasTP, v.Healing, v.Life = false, 3, 150 }, Level, false, "", false},
		{"hurt with portal leaves", "melee", func(v *View) { v.Life = 90 }, Level, false, "", false},
		{"corpse run hurt lean", "melee", func(v *View) { v.HasTP, v.Healing, v.Life = false, 0, 20 }, CorpseRun, false, "", false},
		{"try anyway", "caster", func(v *View) { v.Healing, v.Try = 0, true }, QuestWalk, false, "", false},
	}
	for _, c := range cases {
		v := hero(c.arch)
		if c.tweak != nil {
			c.tweak(&v)
		}
		got := r.MayLeaveTown(v, c.p)
		if got.Refused != c.refuse || got.Need != c.need || got.KitOnly != c.kit {
			t.Errorf("%s: got %+v, want refused=%v need=%q kitOnly=%v", c.name, got, c.refuse, c.need, c.kit)
		}
	}
}

// The package-level functions read the embedded rows and agree with them.
func TestEmbeddedWrappers(t *testing.T) {
	v := hero("melee")
	v.Healing = 0
	if got, err := MayLeaveTown(v, QuestWalk); err != nil || !got.Refused {
		t.Fatalf("MayLeaveTown: %+v %v", got, err)
	}
	if got, err := MayEnter(hero("caster"), stonyField, Level); err != nil || got.Refused {
		t.Fatalf("MayEnter: %+v %v", got, err)
	}
}

// Every need the package emits has a wake, and the rows are sane (a row without an area never matches).
func TestRowsSane(t *testing.T) {
	r := mustRows(t)
	for _, n := range []string{NeedPotions, NeedGoldFarm, NeedTooWeak, NeedRoute, NeedNotReady} {
		if Wake[n] == "" {
			t.Errorf("need %s has no wake", n)
		}
	}
	for _, rd := range r.Roads {
		if len(rd.Areas) == 0 || rd.ID == "" {
			t.Errorf("road %+v needs id and areas", rd)
		}
		for _, arch := range []string{"melee", "caster"} {
			if rd.MinLevel[arch] == 0 {
				t.Errorf("road %s has no level floor for %s", rd.ID, arch)
			}
		}
	}
}

// The kit floor tiers (Questy 2026-11-26 poverty deadlock): two potions open the first two fields and areas far below the hero;
// below two potions the first field only; the first field is always open.
func TestKitFloorTiers(t *testing.T) {
	rs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	k := rs.KitFloor
	fields := []int{2, 3, 4}
	ml := func(a int) int { return map[int]int{2: 1, 3: 3, 4: 5, 9: 3, 5: 8}[a] }
	for _, c := range []struct {
		healing, level, area int
		want                 bool
	}{
		{0, 13, 2, true}, {1, 13, 3, false}, {2, 13, 3, true}, {2, 13, 4, true}, {2, 13, 9, true}, {2, 13, 5, false},
		{2, 9, 9, false}, // below farFromLevel the far rule is off
	} {
		if got := k.Open(c.healing, c.level, c.area, fields, ml); got != c.want {
			t.Errorf("potions %d clvl %d area %d: open %v, want %v", c.healing, c.level, c.area, got, c.want)
		}
	}
}

// Necromancer SME rows (card necrofloors, S): a caster needs 130 max life for the sewers and 220 for Duriel's lair. The rows
// are archetype data: the caster values hold for any caster, and the melee archetype has no sewers life rule (its numbers are unchanged).
func TestCasterLifeFloorsSewersAndDuriel(t *testing.T) {
	r := mustRows(t)
	const (
		sewers = 47
		duriel = 73
	)
	deep := func(arch string, level, maxLife int) View {
		v := hero(arch)
		v.Level, v.MaxLife, v.Life, v.InTown, v.Area = level, maxLife, maxLife, false, 40
		return v
	}
	for _, c := range []struct {
		name    string
		arch    string
		level   int
		maxLife int
		area    int
		refuse  bool
		row     string
	}{
		{"caster sewers 129 life", "caster", 16, 129, sewers, true, "act2_sewers"},
		{"caster sewers 130 life", "caster", 16, 130, sewers, false, ""},
		{"melee sewers 129 life has no life rule", "melee", 14, 129, sewers, false, ""},
		{"caster duriel 219 life", "caster", 26, 219, duriel, true, "act2_duriel_lair"},
		{"caster duriel 220 life", "caster", 26, 220, duriel, false, ""},
		{"caster tomb 150 life stays open", "caster", 26, 150, 66, false, ""},
		{"melee duriel 219 life", "melee", 24, 219, duriel, true, "act2_tombs"},
		{"melee duriel 220 life", "melee", 24, 220, duriel, false, ""},
	} {
		got := r.MayEnter(deep(c.arch, c.level, c.maxLife), c.area, Level)
		if got.Refused != c.refuse || got.Row != c.row {
			t.Errorf("%s: refused=%v row=%q, want refused=%v row=%q (%s)", c.name, got.Refused, got.Row, c.refuse, c.row, got.Why)
		}
	}
}

// The Act 2 quest dungeons the walks enter all have a road row with a level and a life floor, and the arrival row lists them
// (card act2temple). A hero below the floor is refused with a level need, not walked into the temple.
func TestAct2QuestDungeonRoadsAndArrival(t *testing.T) {
	rs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []int{47, 56, 58, 61, 62, 52} {
		for _, arch := range []string{"melee", "caster"} {
			v := View{Archetype: arch, Normal: true, Level: 5, Healing: 4, HasTP: true, Life: 500, MaxLife: 500}
			if got := rs.MayEnter(v, a, QuestWalk); !got.Refused || got.Need != NeedTooWeak {
				t.Errorf("area %d %s clvl 5: %+v, want too_weak", a, arch, got)
			}
			v.Level = 30
			v.MaxLife = 20
			if got := rs.MayEnter(v, a, QuestWalk); !got.Refused || got.Need != NeedTooWeak {
				t.Errorf("area %d %s life 20: %+v, want too_weak", a, arch, got)
			}
			v.MaxLife = 600
			if got := rs.MayEnter(v, a, QuestWalk); got.Refused {
				t.Errorf("area %d %s strong hero refused: %+v", a, arch, got)
			}
		}
	}
	for _, a := range []int{56, 57, 58, 60, 61, 62, 63, 64} {
		if _, ok := rs.ArrivalFor(a); !ok {
			t.Errorf("area %d has no arrival row", a)
		}
	}
	if _, ok := rs.ArrivalFor(41); ok {
		t.Error("an open field has an arrival row")
	}
}

// Review 14 H3: the on-road kit skip does not reach the boss or chest level of the road (Radament 49, Halls of the
// Dead 3, Fangskin 61, Coldworm 64). The levels come from the quest handler rows, not from a list here.
func TestOnRoadKitSkipStopsAtTheObjectiveLevel(t *testing.T) {
	rs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	rs.KitFloor.PotionsBelow = 0 // the road rows alone (the general kit floor has its own tests)
	for _, c := range []struct{ from, boss int }{{48, 49}, {57, 60}, {58, 61}, {63, 64}} {
		for _, arch := range []string{"melee", "caster"} {
			v := View{Archetype: arch, Normal: true, Level: 40, Life: 900, MaxLife: 900, Area: c.from, Healing: 0, HasTP: true}
			if got := rs.MayEnter(v, c.boss, Boss); !got.Refused || got.Need != NeedPotions || !got.KitOnly {
				t.Errorf("%s %d->%d potionless with a portal: %+v, want a kit-only refusal", arch, c.from, c.boss, got)
			}
			v.HasTP = false
			if got := rs.MayEnter(v, c.boss, Boss); !got.Refused || got.KitOnly {
				t.Errorf("%s %d->%d potionless without a portal: %+v, want a hold (no trip)", arch, c.from, c.boss, got)
			}
			v.Healing = 2 // the lab case of 2026-11-28 stays open: potions in hand, no portal
			if got := rs.MayEnter(v, c.boss, Boss); got.Refused {
				t.Errorf("%s %d->%d with potions, no portal: %+v, want allow", arch, c.from, c.boss, got)
			}
			v.Healing, v.Area = 0, c.boss // already inside: the trip rules decide
			if got := rs.MayEnter(v, c.boss, Boss); got.Refused {
				t.Errorf("%s standing in %d: %+v, want allow", arch, c.boss, got)
			}
		}
	}
	// an approach level of the same road is still entered with a thinned kit
	v := View{Archetype: "melee", Normal: true, Level: 40, Life: 900, MaxLife: 900, Area: 62, Healing: 0}
	if got := rs.MayEnter(v, 63, Boss); got.Refused {
		t.Errorf("62->63 on the road: %+v, want allow", got)
	}
}
