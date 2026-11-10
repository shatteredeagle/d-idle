package leveler

import (
	"strings"
	"testing"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/decide"
)

func v3Rule(name string, level int, lv map[string]int) bool {
	s, ok := v3Req[name]
	if !ok || s.lvl > level || lv[name] >= 20 {
		return false
	}
	for _, r := range s.need {
		if lv[r] == 0 {
			return false
		}
	}
	return true
}

func mustUnlocks(t *testing.T, class string) *Unlocks {
	t.Helper()
	u, err := LoadUnlocks(class, "")
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestLoadUnlocksUnknownClassNamesIt(t *testing.T) {
	if _, err := LoadUnlocks("bard", ""); err == nil || !strings.Contains(err.Error(), `"bard"`) {
		t.Fatalf("want an error naming bard, got %v", err)
	}
	if _, err := ParseUnlocks([]byte(`{"class":"x","unlocks":[{"skill":"A","event":"nope:1"}]}`)); err == nil {
		t.Fatal("bad event kind accepted")
	}
	if _, err := ParseUnlocks([]byte(`{"class":"x","mustFree":["A"],"unlocks":[{"skill":"A","event":"clvl:3"}]}`)); err == nil {
		t.Fatal("a mustFree skill that is locked was accepted")
	}
}

func TestSorceressUnlocksShape(t *testing.T) {
	u := mustUnlocks(t, "sorceress")
	f, err := decide.ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, x := range f.Firsts {
		ids[x.ID] = true
	}
	for _, e := range u.Unlocks {
		if e.Codex != "" && !ids[e.Codex] {
			t.Errorf("%s: codex first %q does not exist in firsts_act1.json", e.Skill, e.Codex)
		}
	}
	// Teleport is not tied to Andariel (docs/research/skill-req-check.md).
	for _, e := range u.Unlocks {
		if e.Skill == "Teleport" && e.Event != "waypoint:catacombs_2" {
			t.Errorf("Teleport event %q", e.Event)
		}
	}
}

// Against the real tables when present: every skill exists and belongs to the class.
func TestUnlocksValidateAgainstTables(t *testing.T) {
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Skip("no real install tables: " + err.Error())
	}
	if err := mustUnlocks(t, "sorceress").Validate(db); err != nil {
		t.Fatal(err)
	}
	bad, _ := ParseUnlocks([]byte(`{"class":"sorceress","unlocks":[{"skill":"Bash","event":"clvl:3"}]}`))
	if err := bad.Validate(db); err == nil || !strings.Contains(err.Error(), "Bash") {
		t.Fatalf("a Barbarian skill in a Sorceress table: %v", err)
	}
	bad, _ = ParseUnlocks([]byte(`{"class":"sorceress","unlocks":[{"skill":"Nonesuch","event":"clvl:3"}]}`))
	if err := bad.Validate(db); err == nil || !strings.Contains(err.Error(), "Nonesuch") {
		t.Fatalf("unknown skill: %v", err)
	}
}

// replayUnlocks is replaySkills with the unlock table: events fire at the given clvl (never when absent).
func replayUnlocks(t *testing.T, build string, u *Unlocks, p func(lv int) *Progress, to int, quests map[int]int) (spent []string, refused []string, free int) {
	t.Helper()
	b, err := LoadBuild(build, "")
	if err != nil {
		t.Fatal(err)
	}
	cur := map[string]int{}
	for lv := 2; lv <= to; lv++ {
		free += 1 + quests[lv]
		prog := p(lv)
		rule := func(name string, level int, lv2 map[string]int) bool {
			ok := u.WrapRule(v3Rule, prog)(name, level, lv2)
			if !ok && v3Rule(name, level, lv2) {
				refused = append(refused, name)
			}
			return ok
		}
		for _, a := range u.Plan(b, Have{Skills: cur, Level: lv, SkillPts: free}, rule, nil, prog) {
			if a.Kind == KindSkill {
				cur[a.Skill]++
				free--
				spent = append(spent, a.Skill)
			}
		}
	}
	return spent, refused, free
}

func firedAt(at map[string]int) func(lv int) *Progress {
	return func(lv int) *Progress {
		p := NewProgress()
		for ev, from := range at {
			if lv >= from {
				p.Fire(ev)
			}
		}
		return p
	}
}

// The buildv3 replay with the Sorceress table: with every event met nothing changes against the plain replay, and
// with the events at their expected clvl no unlock refuses a row (AC 2).
func TestBuildV3WithUnlocks(t *testing.T) {
	u := mustUnlocks(t, "sorceress")
	quests := map[int]int{7: 1, 20: 1, 33: 2}
	plain, _, _ := replaySkills(t, FallbackV3, 60, quests)
	spent, refused, free := replayUnlocks(t, FallbackV3, u, func(int) *Progress { return AllMetProgress() }, 60, quests)
	if strings.Join(spent, ",") != strings.Join(plain, ",") || len(refused) != 0 {
		t.Fatalf("all-met table changed the replay: refused %v", refused)
	}
	_ = free
	// events at the expected clvl (S estimates, docs/skill-unlocks.md)
	timed := firedAt(map[string]int{"quest:den_of_evil": 5, "boss:bishibosh": 8, "boss:blood_raven": 9, "waypoint:catacombs_2": 17, "boss:countess": 22, "boss:andariel": 24})
	spent, refused, _ = replayUnlocks(t, FallbackV3, u, timed, 60, quests)
	if len(refused) != 0 {
		t.Fatalf("unlock refused v3 rows: %v", refused)
	}
	if strings.Join(spent, ",") != strings.Join(plain, ",") {
		t.Fatal("timed unlocks changed the v3 order")
	}
}

// Points are never banked: a Catacombs waypoint that is lit late moves Teleport behind Chain Lightning, every point is
// still spent.
func TestLateTeleportUnlockReordersWithoutBanking(t *testing.T) {
	u := mustUnlocks(t, "sorceress")
	quests := map[int]int{7: 1, 20: 1, 33: 2}
	late := firedAt(map[string]int{"quest:den_of_evil": 5, "boss:bishibosh": 8, "boss:blood_raven": 9, "waypoint:catacombs_2": 21})
	spent, _, free := replayUnlocks(t, FallbackV3, u, late, 24, quests)
	if free != 0 {
		t.Fatalf("%d points left unspent: points were banked", free)
	}
	ti, ci := -1, -1
	for i, s := range spent {
		if s == "Teleport" && ti < 0 {
			ti = i
		}
		if s == "Chain Lightning" && ci < 0 {
			ci = i
		}
	}
	if ci < 0 || (ti >= 0 && ti < ci) {
		t.Fatalf("expected Chain Lightning before Teleport, got teleport at %d, chain at %d", ti, ci)
	}
}

// Two classes: the unlock refusal is class-agnostic data. A Paladin table that locks Prayer (build row clvl 4) until
// Bishibosh flows the point to the next row and gives it back to Prayer afterwards.
func TestUnlockRuleTwoClasses(t *testing.T) {
	pal, err := ParseUnlocks([]byte(`{"class":"paladin","unlocks":[{"skill":"Prayer","event":"boss:bishibosh"},{"skill":"Vengeance","event":"boss:blood_raven|clvl:30"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	sor := mustUnlocks(t, "sorceress")
	cases := []struct {
		name   string
		u      *Unlocks
		skill  string
		fire   []string
		level  int
		unlock bool
	}{
		{"sor Frost Nova locked", sor, "Frost Nova", nil, 6, false},
		{"sor Frost Nova after the Den", sor, "Frost Nova", []string{"quest:den_of_evil"}, 6, true},
		{"sor Static Field by waypoint alt", sor, "Static Field", []string{"waypoint:cold_plains"}, 8, true},
		{"sor Charged Bolt never locked", sor, "Charged Bolt", nil, 1, true},
		{"pal Prayer locked", pal, "Prayer", nil, 4, false},
		{"pal Prayer opened", pal, "Prayer", []string{"boss:bishibosh"}, 4, true},
		{"pal Vengeance by clvl alt", pal, "Vengeance", nil, 30, true},
		{"pal Vengeance below clvl alt", pal, "Vengeance", nil, 29, false},
		{"pal Might not listed", pal, "Might", nil, 2, true},
	}
	for _, c := range cases {
		p := NewProgress()
		for _, e := range c.fire {
			if err := p.Fire(e); err != nil {
				t.Fatal(err)
			}
		}
		if got := c.u.Unlocked(c.skill, p, c.level); got != c.unlock {
			t.Errorf("%s: unlocked %v, want %v", c.name, got, c.unlock)
		}
	}
	b, err := LoadBuild(PaladinV2, "")
	if err != nil {
		t.Fatal(err)
	}
	h := Have{Level: 4, SkillPts: 3, Skills: map[string]int{"Might": 1, "Sacrifice": 1}}
	skills := func(p *Progress) string {
		var s []string
		for _, a := range pal.Plan(b, h, nil, nil, p) {
			if a.Kind == KindSkill {
				s = append(s, a.Skill)
			}
		}
		return strings.Join(s, ",")
	}
	locked := skills(NewProgress())
	if strings.Contains(locked, "Prayer") || len(strings.Split(locked, ",")) != 3 {
		t.Errorf("locked Prayer must flow on and bank nothing: %s", locked)
	}
	p := NewProgress()
	p.Fire("boss:bishibosh")
	if got := skills(p); !strings.HasPrefix(got, "Prayer") {
		t.Errorf("opened Prayer first: %s", got)
	}
	if got := skills(AllMetProgress()); !strings.HasPrefix(got, "Prayer") {
		t.Errorf("all-met table: %s", got)
	}
}

func TestFreeChoicePoint(t *testing.T) {
	u := mustUnlocks(t, "sorceress")
	b, err := LoadBuild(FallbackV3, "")
	if err != nil {
		t.Fatal(err)
	}
	h := Have{Level: 9, SkillPts: 1, Skills: map[string]int{"Charged Bolt": 6, "Frozen Armor": 1, "Frost Nova": 1}}
	p := NewProgress()
	p.Fire("quest:den_of_evil")
	if got := u.FreeChoices(b, h, v3Rule, p); len(got) != 0 {
		t.Fatalf("nothing but table skills unlocked yet: %v", got)
	}
	p.Fire("boss:bishibosh")
	p.Fire("boss:blood_raven")
	got := u.FreeChoices(b, h, v3Rule, p)
	names := map[string]FreeChoice{}
	for _, c := range got {
		names[c.Skill] = c
	}
	if _, ok := names["Static Field"]; !ok {
		t.Fatalf("Static Field missing: %v", got)
	}
	if _, ok := names["Warmth"]; ok {
		t.Fatal("Warmth is free since sorcready2 (the normal30 table spends clvl 2 on it), not a free choice")
	}
	if _, ok := names["Frost Nova"]; ok {
		t.Fatal("Frost Nova is in the build table, not a free choice")
	}
	if c := names["Static Field"]; c.Displaces != "Charged Bolt" || !strings.Contains(c.Consequence(), "Charged Bolt") {
		t.Fatalf("consequence: %+v %q", c, c.Consequence())
	}
	if err := u.ChooseFree(p, b, h, v3Rule, "Teleport"); err == nil {
		t.Fatal("a locked skill was accepted")
	}
	if err := u.ChooseFree(p, b, h, v3Rule, "Static Field"); err != nil {
		t.Fatal(err)
	}
	if err := u.ChooseFree(p, b, h, v3Rule, "Warmth"); err == nil {
		t.Fatal("a second free-choice point was accepted")
	}
	acts := u.Plan(b, h, v3Rule, nil, p)
	if len(acts) != 1 || acts[0].Skill != "Static Field" {
		t.Fatalf("plan with the choice: %v", acts)
	}
	// once spent the table resumes and the choice is not spent again
	h.Skills["Static Field"] = 1
	acts = u.Plan(b, h, v3Rule, nil, p)
	if len(acts) != 1 || acts[0].Skill != "Charged Bolt" {
		t.Fatalf("plan after the choice: %v", acts)
	}
}
