package leveler

import (
	"fmt"
	"sort"
	"testing"
)

// PALPROBE part B (a): both Paladin Normal tables run through the plan simulator, clvl 1 to 30, against the
// game's own skill rules. The rule table is the SME's read of skills.txt (docs/sme/class-paladin.md section 2,
// tag D = computed from the install, not re-read here); TestPalProbeRulesMatchInstall repeats the check against
// the real tables when /tmp/d2mpq exists (the worker host), so the fixture cannot drift unseen.
type palRule struct {
	clvl int
	req  []string
}

var palSkills = map[string]palRule{
	"Sacrifice": {1, nil}, "Smite": {1, nil}, "Might": {1, nil}, "Prayer": {1, nil}, "Resist Fire": {1, nil},
	"Holy Bolt": {6, nil}, "Holy Fire": {6, []string{"Might"}}, "Thorns": {6, nil}, "Defiance": {6, nil}, "Resist Cold": {6, nil},
	"Zeal": {12, []string{"Sacrifice"}}, "Charge": {12, []string{"Smite"}}, "Blessed Aim": {12, []string{"Might"}},
	"Cleansing": {12, []string{"Prayer"}}, "Resist Lightning": {12, nil},
	"Vengeance": {18, []string{"Zeal"}}, "Blessed Hammer": {18, []string{"Holy Bolt"}}, "Concentration": {18, []string{"Blessed Aim"}},
	"Holy Freeze": {18, []string{"Holy Fire"}}, "Vigor": {18, []string{"Cleansing", "Defiance"}},
	"Conversion": {24, []string{"Vengeance"}}, "Holy Shield": {24, []string{"Charge", "Blessed Hammer"}},
	"Holy Shock": {24, []string{"Holy Freeze"}}, "Sanctuary": {24, []string{"Thorns", "Holy Freeze"}},
	"Meditation": {24, []string{"Cleansing"}}, "Fist of the Heavens": {30, []string{"Blessed Hammer", "Conversion"}},
	"Fanaticism": {30, []string{"Concentration"}}, "Conviction": {30, []string{"Sanctuary"}},
	"Redemption": {30, []string{"Vigor"}}, "Salvation": {30, nil},
}

func palLegal(name string, level int, lv map[string]int) bool {
	r, ok := palSkills[name]
	if !ok || r.clvl > level {
		return false
	}
	for _, q := range r.req {
		if lv[q] == 0 {
			return false
		}
	}
	return true
}

type palRun struct {
	skills  map[string]int
	firstAt map[string]int // clvl the skill was first trained
	attr    [numAttr]int   // points spent per attribute
	spent   int            // skill points spent
	illegal []string       // legal-at-the-time violations
	stuck   []string       // clvl rows where points stayed unspent
}

// simPaladin plays clvl 2 to 30 the way the game hands out points: 5 stat points and 1 skill point per level, plus
// denAt (0 = never) one extra skill point from the Den of Evil.
func simPaladin(t *testing.T, table string, denAt int) palRun {
	t.Helper()
	b := mustBuild(t, table)
	start := [numAttr]int{25, 20, 25, 15} // charstats.txt Paladin (docs/sme/class-paladin.md s1, D)
	h := Have{Attr: start, Start: start, Skills: map[string]int{}}
	r := palRun{skills: h.Skills, firstAt: map[string]int{}}
	for L := 2; L <= 30; L++ {
		h.Level, h.StatPts, h.SkillPts = L, h.StatPts+5, h.SkillPts+1
		if L == denAt {
			h.SkillPts++
			h.QuestPts++
		}
		for _, a := range b.Plan(h, palLegal, nil) {
			if a.Kind == KindStat {
				h.Attr[a.Attr]++
				h.StatPts--
				r.attr[a.Attr]++
				continue
			}
			if !palLegal(a.Skill, L, h.Skills) {
				r.illegal = append(r.illegal, fmt.Sprintf("clvl %d %s", L, a.Skill))
			}
			h.Skills[a.Skill]++
			h.SkillPts--
			r.spent++
			if _, ok := r.firstAt[a.Skill]; !ok {
				r.firstAt[a.Skill] = L
			}
		}
		if h.SkillPts > 0 || h.StatPts > 0 {
			r.stuck = append(r.stuck, fmt.Sprintf("clvl %d: %d skill, %d stat unspent", L, h.SkillPts, h.StatPts))
		}
	}
	return r
}

func TestPalProbeTablesSimulate(t *testing.T) {
	for _, table := range []string{"paladin-normal30", "paladin-normal30-v2"} {
		r := simPaladin(t, table, 5)
		if len(r.illegal) > 0 {
			t.Errorf("%s: skill trained before the game allows it: %v", table, r.illegal)
		}
		var names []string
		for n := range r.skills {
			names = append(names, n)
		}
		sort.Strings(names)
		var sb string
		for _, n := range names {
			sb += fmt.Sprintf(" %s=%d@%d", n, r.skills[n], r.firstAt[n])
		}
		t.Logf("%s: %d skill points spent, stats Str+%d Dex+%d Vit+%d Ene+%d;%s", table, r.spent, r.attr[Str], r.attr[Dex], r.attr[Vit], r.attr[Ene], sb)
		t.Logf("%s: unspent after a level: %v", table, r.stuck)
	}
}

// Zeal reaches its cap (level 4 is the hit-count cap, D docs/sme/class-paladin.md section 2) by the clvl the recon
// expects: the v2 table (the fresh-character default) by clvl 15.
func TestPalProbeZealCap(t *testing.T) {
	v2 := simPaladin(t, "paladin-normal30-v2", 5)
	if v2.skills["Zeal"] < 4 {
		t.Fatalf("v2 Zeal %d at clvl 30", v2.skills["Zeal"])
	}
	// the level the 4th Zeal point lands: replay and stop at the first level with Zeal 4
	b := mustBuild(t, "paladin-normal30-v2")
	zeal, at := 0, 0
	for _, row := range b.Rows {
		for _, s := range row.Skills {
			if s.Skill == "Zeal" {
				zeal += s.Points
				if zeal >= 4 && at == 0 {
					at = row.Clvl
				}
			}
		}
	}
	if at == 0 || at > 15 {
		t.Fatalf("v2 Zeal 4 at clvl %d (the recon expects by 15)", at)
	}
	t.Logf("v2 table: Zeal 4 at clvl %d (table row), first Zeal point at clvl %d", at, v2.firstAt["Zeal"])
	legacy := simPaladin(t, "paladin-normal30", 5)
	t.Logf("legacy table: Zeal %d at clvl 30, first at clvl %d", legacy.skills["Zeal"], legacy.firstAt["Zeal"])
}

func TestPalProbeSacrificePrecedesZeal(t *testing.T) {
	for _, table := range []string{"paladin-normal30", "paladin-normal30-v2"} {
		r := simPaladin(t, table, 5)
		if r.firstAt["Zeal"] != 0 && (r.firstAt["Sacrifice"] == 0 || r.firstAt["Sacrifice"] > r.firstAt["Zeal"]) {
			t.Errorf("%s: Sacrifice at %d after Zeal at %d", table, r.firstAt["Sacrifice"], r.firstAt["Zeal"])
		}
		if r.firstAt["Zeal"] < 12 && r.firstAt["Zeal"] != 0 {
			t.Errorf("%s: Zeal at clvl %d, the game needs 12", table, r.firstAt["Zeal"])
		}
	}
}

// Stat points follow the plan: every point the game grants is spent, Str gets enough for gear, Vit is not everything.
func TestPalProbeStatPointsFollowPlan(t *testing.T) {
	for _, table := range []string{"paladin-normal30", "paladin-normal30-v2"} {
		b := mustBuild(t, table)
		var want [numAttr]int
		for _, r := range b.Rows {
			for a := range want {
				want[a] += r.Attr[a]
			}
		}
		got := simPaladin(t, table, 5)
		if got.attr != want {
			t.Errorf("%s: spent %v, table says %v", table, got.attr, want)
		}
		if tot := got.attr[Str] + got.attr[Dex] + got.attr[Vit] + got.attr[Ene]; tot != 29*5 {
			t.Errorf("%s: %d stat points spent of %d", table, tot, 29*5)
		}
	}
}

// The fixture above against the real skills.txt: skipped where there is no install.
func TestPalProbeRulesMatchInstall(t *testing.T) {
	db := realDB(t)
	for name, r := range palSkills {
		s := db.SkillByName(name)
		if s == nil {
			t.Errorf("%s: not in skills.txt", name)
			continue
		}
		if s.ReqLevel != r.clvl {
			t.Errorf("%s: reqlevel %d in skills.txt, fixture %d", name, s.ReqLevel, r.clvl)
		}
		var got []string
		for _, q := range s.ReqSkills {
			if q != "" {
				got = append(got, q)
			}
		}
		sort.Strings(got)
		want := append([]string(nil), r.req...)
		sort.Strings(want)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s: prerequisites %v in skills.txt, fixture %v", name, got, want)
		}
	}
}
