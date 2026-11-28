package scenario

import (
	"slices"
	"strings"
	"testing"
)

// pgfix4: the ring of loot.rare-ring-classified was dropped in the town because "debug give" ran before "debug goto":
// a ground item is placed beside the hero, so it must come after the position verb. Both classes get the same order.
func TestGroundItemAfterPosition(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := Scenario{ID: "x.ring", Kind: KindBehaviour, Pre: Pre{Level: 10, Start: StartField, Area: 2,
			Items: []Item{{Code: "rin", Where: "ground", Quality: 6, Unid: true}}}}
		su, err := BuildSetup(s, class, 3)
		if err != nil {
			t.Fatal(err)
		}
		goto_, give := -1, -1
		for i, v := range su.Verbs {
			switch {
			case strings.HasPrefix(v.Line, "debug goto"):
				goto_ = i
			case strings.HasPrefix(v.Line, "debug give rin"):
				give = i
			}
		}
		if goto_ < 0 || give < 0 || give < goto_ {
			t.Errorf("%s: goto at %d, give at %d: the ring must be dropped after the hero is in the field", class, goto_, give)
		}
	}
}

// pre.stats writes the raw stat after the level verbs (a level write would otherwise reset it) and marks the run CHEATED.
func TestRawStatsFollowTheLevel(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := Scenario{ID: "x.stats", Kind: KindBehaviour, Pre: Pre{Level: 20, Stats: []StatSet{{Stat: 7, Value: 307200}}}}
		su, err := BuildSetup(s, class, 3)
		if err != nil {
			t.Fatal(err)
		}
		lvl, raw := -1, -1
		for i, v := range su.Verbs {
			switch v.Line {
			case "debug stat 12 20":
				lvl = i
			case "debug stat 7 307200":
				raw = i
			}
		}
		if lvl < 0 || raw < lvl {
			t.Errorf("%s: level verb at %d, raw stat at %d", class, lvl, raw)
		}
		if !slices.Contains(su.Tags, "CHEATED") {
			t.Errorf("%s: a raw stat write must mark the run CHEATED", class)
		}
		if _, err := BuildSetup(Scenario{ID: "x.bad", Kind: KindBehaviour, Pre: Pre{Stats: []StatSet{{Stat: 9999}}}}, class, 3); err == nil {
			t.Errorf("%s: stat id 9999 accepted", class)
		}
	}
}

// pre.waypoints -1 means every waypoint (`debug wp all`); a bit stays `debug wp <bit>`.
func TestWaypointAllAndBit(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := Scenario{ID: "x.wp", Kind: KindBehaviour, Pre: Pre{Waypoints: []int{-1, 3}}}
		su, err := BuildSetup(s, class, 3)
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, v := range su.Verbs {
			lines = append(lines, v.Line)
		}
		if !slices.Contains(lines, "debug wp all") || !slices.Contains(lines, "debug wp 3") {
			t.Errorf("%s: verbs %v", class, lines)
		}
	}
}

// pre.merc_level raises the lab merc to a realistic level and heals it after the hire (devp23 labverbs).
func TestMercLevelFollowsTheHire(t *testing.T) {
	zero := 0
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := Scenario{ID: "x.merc", Kind: KindBehaviour, Pre: Pre{Merc: &zero, MercLevel: 17}}
		su, err := BuildSetup(s, class, 3)
		if err != nil {
			t.Fatal(err)
		}
		hire, lvl, heal := -1, -1, -1
		for i, v := range su.Verbs {
			switch v.Line {
			case "debug merc hire 0":
				hire = i
			case "debug merc level 17":
				lvl = i
			case "debug merc heal":
				heal = i
			}
		}
		if hire < 0 || lvl < hire || heal < lvl {
			t.Errorf("%s: hire %d level %d heal %d", class, hire, lvl, heal)
		}
	}
}

// pre.rooms_up sends `debug rooms up` after the goto, before any warp (devp28); without start field it is an error.
func TestRoomsUpFollowsTheGoto(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := Scenario{ID: "x.rooms", Kind: KindBehaviour, Pre: Pre{Start: StartField, Area: 75, RoomsUp: true}}
		su, err := BuildSetup(s, class, 3)
		if err != nil {
			t.Fatal(err)
		}
		g, r := -1, -1
		for i, v := range su.Verbs {
			switch v.Line {
			case "debug goto 75":
				g = i
			case "debug rooms up":
				r = i
			}
		}
		if g < 0 || r <= g {
			t.Fatalf("%s: goto %d rooms up %d", class, g, r)
		}
		if _, err := BuildSetup(Scenario{ID: "x.rooms2", Kind: KindBehaviour, Pre: Pre{RoomsUp: true}}, class, 3); err == nil {
			t.Fatal("rooms_up without start field was accepted")
		}
	}
}

// pre.skills puts class skill points after the level verbs (the level write would not undo them) and marks the run CHEATED.
func TestSkillPointsAfterTheLevel(t *testing.T) {
	for _, class := range []string{"Barbarian", "Sorceress"} {
		s := Scenario{ID: "x.skill", Kind: KindBehaviour, Pre: Pre{Level: 30, Skills: []SkillSet{{Skill: 151, Points: 1}}}}
		su, err := BuildSetup(s, class, 3)
		if err != nil {
			t.Fatal(err)
		}
		lvl, pts := -1, -1
		for i, v := range su.Verbs {
			switch {
			case v.Line == "debug stat 12 30":
				lvl = i
			case v.Line == "debug points 151 1":
				pts = i
			}
		}
		if lvl < 0 || pts < lvl {
			t.Errorf("%s: level at %d, points at %d", class, lvl, pts)
		}
		if !slices.Contains(su.Tags, "CHEATED") {
			t.Errorf("%s: tags %v lack CHEATED", class, su.Tags)
		}
		if _, err := BuildSetup(Scenario{ID: "x.bad", Pre: Pre{Skills: []SkillSet{{Skill: 151}}}}, class, 3); err == nil {
			t.Errorf("%s: zero points accepted", class)
		}
	}
}

// pre.bind and pre.summon reach the setup with the CHEATED tag, on two classes, and every malformed row is refused: an
// id for a name, a bad hand, two skills on one hand, a banked hero, a town start for a summon, a bad count.
func TestHandSkillsSetupAndValidation(t *testing.T) {
	for _, class := range []string{"Necromancer", "Druid"} {
		good := Pre{HeroProfile: HeroPlayed, Level: 24, Start: StartField, Area: 2,
			Bind: []HandBind{{Skill: "Corpse Explosion"}, {Skill: "Teeth", Hand: "left"}}, Summon: []Summon{{Skill: "Clay Golem", N: 2, Min: 1}}}
		su, err := BuildSetup(Scenario{ID: "x.hands", Pre: good}, class, 3)
		if err != nil {
			t.Fatalf("%s: %v", class, err)
		}
		if len(su.Bind) != 2 || len(su.Summon) != 1 || su.Bind[0].HandName() != "right" || su.Summon[0].Casts() != 2 {
			t.Errorf("%s: setup bind %v summon %v", class, su.Bind, su.Summon)
		}
		for _, tag := range []string{"CHEATED", "hand-skill", "summon"} {
			if !slices.Contains(su.Tags, tag) {
				t.Errorf("%s: tags %v lack %s", class, su.Tags, tag)
			}
		}
		for _, v := range su.Verbs { // the names are resolved by the runner with the game tables, never here
			if strings.HasPrefix(v.Line, "cmd ") {
				t.Errorf("%s: setup verb %q", class, v.Line)
			}
		}
		for name, edit := range map[string]func(*Pre){
			"an id for a name":        func(p *Pre) { p.Bind[0].Skill = "74" },
			"an empty name":           func(p *Pre) { p.Summon[0].Skill = "" },
			"a bad hand":              func(p *Pre) { p.Bind[0].Hand = "both" },
			"two on one hand":         func(p *Pre) { p.Bind[1].Hand = "right" },
			"a banked hero":           func(p *Pre) { p.HeroProfile = HeroBanked },
			"a level 1 hero":          func(p *Pre) { p.Level = 1 },
			"a summon in town":        func(p *Pre) { p.Start, p.Area = StartTown, 0 },
			"more standing than cast": func(p *Pre) { p.Summon[0].Min = 3 },
			"too many casts":          func(p *Pre) { p.Summon[0].N = 21 },
		} {
			p := good
			p.Bind, p.Summon = slices.Clone(good.Bind), slices.Clone(good.Summon)
			edit(&p)
			if _, err := BuildSetup(Scenario{ID: "x.bad", Pre: p}, class, 3); err == nil {
				t.Errorf("%s: %s accepted", class, name)
			}
		}
	}
}
