package leveler

import (
	"d2lod-bot/internal/gear/geartest"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear"
)

// EARLYGAME audit: a fresh character of every class replayed through the default build table for clvl 1..12, with the
// skills.txt rules read from the extracted gear tables (gear.json in d2tables cache, V: from Patch_D2.mpq).
// The golden rows below document what a fresh bot spends; docs/earlygame.md has the table and the SME comparison.

type skillRow struct {
	Class    string   `json:"class"`
	ReqLevel int      `json:"req_level"`
	Req      []string `json:"req"`
	Passive  bool     `json:"passive"`
}

func gearSkills(t *testing.T) map[string]skillRow {
	t.Helper()
	geartest.Need(t)
	b, err := os.ReadFile(filepath.Join(gear.TablesDir(), "gear.json"))
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Skills map[string]skillRow `json:"skills"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	return d.Skills
}

var earlyClasses = []struct {
	name string
	cls  d2.Class
}{
	{"sorceress", d2.Sorceress}, {"necromancer", d2.Necromancer}, {"paladin", d2.Paladin}, {"amazon", d2.Amazon},
	{"barbarian", d2.Barbarian}, {"druid", d2.Druid}, {"assassin", d2.Assassin},
}

// replay spends the points of clvl 2..to for a class and returns the owned base attributes and skills after each level.
// The Den of Evil point is granted at clvl denAt (the SME tables assume 5).
type snapshotAt struct {
	attr   [numAttr]int
	skills map[string]int
}

func replay(t *testing.T, class string, to, denAt int) (map[int]snapshotAt, []string) {
	t.Helper()
	// the table a fresh character gets (BUILDV2: the v2 tables for Sorceress and Paladin)
	return replayBuild(t, class, SelectBuild(class, 1, nil), to, denAt)
}

func replayBuild(t *testing.T, class, build string, to, denAt int) (map[int]snapshotAt, []string) {
	t.Helper()
	sk := gearSkills(t)
	ci, _ := gear.ClassStart(class)
	start := [numAttr]int{ci.Str, ci.Dex, ci.Vit, ci.Ene}
	b, err := LoadBuild(build, "")
	if err != nil {
		t.Fatal(class, err)
	}
	code := gear.ClassCode(class)
	rule := func(name string, level int, lv map[string]int) bool {
		s, ok := sk[name]
		if !ok || s.Class != code || s.ReqLevel > level {
			return false
		}
		for _, r := range s.Req {
			if lv[r] == 0 {
				return false
			}
		}
		return true
	}
	l := &Leveler{}
	floor := l.floor(&d2.Player{Class: classByName(class)})
	cur := snapshotAt{attr: start, skills: map[string]int{}}
	out := map[int]snapshotAt{1: {attr: start, skills: map[string]int{}}}
	var problems []string
	unspentSkill := 0
	for lv := 2; lv <= to; lv++ {
		unspentSkill++
		if lv == denAt {
			unspentSkill++
		}
		h := Have{Attr: cur.attr, Start: start, Skills: cur.skills, Level: lv, StatPts: 5, SkillPts: unspentSkill, Floor: floor}
		for _, a := range b.Plan(h, rule, nil) {
			if a.Kind == KindStat {
				cur.attr[a.Attr]++
			} else {
				cur.skills[a.Skill]++
				unspentSkill--
			}
		}
		snap := snapshotAt{attr: cur.attr, skills: map[string]int{}}
		for k, v := range cur.skills {
			snap.skills[k] = v
		}
		out[lv] = snap
		spent := cur.attr[Str] + cur.attr[Dex] + cur.attr[Vit] + cur.attr[Ene] - (start[Str] + start[Dex] + start[Vit] + start[Ene])
		if spent != 5*(lv-1) {
			problems = append(problems, fmt.Sprintf("clvl %d: %d stat points spent, want %d", lv, spent, 5*(lv-1)))
		}
	}
	if unspentSkill != 0 {
		problems = append(problems, fmt.Sprintf("%d skill points unspent at clvl %d", unspentSkill, to))
	}
	return out, problems
}

func classByName(n string) d2.Class {
	for _, c := range earlyClasses {
		if c.name == n {
			return c.cls
		}
	}
	return d2.Sorceress
}

func TestEarlyReplaySpendsEveryPoint(t *testing.T) {
	geartest.Need(t)
	for _, c := range earlyClasses {
		_, problems := replay(t, c.name, 12, 5)
		for _, p := range problems {
			t.Errorf("%s: %s", c.name, p)
		}
	}
}

// Every skill the default table buys at clvl 2..12 must be legal when bought (level, prerequisite) and must belong to
// the class; the plan rule already refuses the rest, so a mismatch shows up as unspent points above.
func TestEarlyTableNamesExistInSkillData(t *testing.T) {
	geartest.Need(t)
	sk := gearSkills(t)
	for _, c := range earlyClasses {
		b, err := LoadBuild(SelectBuild(c.name, 1, nil), "")
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range b.Rows {
			for _, s := range r.Skills {
				row, ok := sk[s.Skill]
				if !ok {
					t.Errorf("%s: skill %q (clvl %d) is not in skills.txt data", c.name, s.Skill, r.Clvl)
					continue
				}
				if row.Class != gear.ClassCode(c.name) {
					t.Errorf("%s: skill %q belongs to class %q", c.name, s.Skill, row.Class)
				}
			}
		}
	}
}

func TestEarlyStrengthFloorSorceress(t *testing.T) {
	geartest.Need(t)
	snaps, _ := replay(t, "sorceress", 12, 5)
	if snaps[2].attr[Str] < 12 {
		t.Fatalf("sorceress Str at clvl 2 = %d, want the 12 floor for the first body armor", snaps[2].attr[Str])
	}
}

// goldenEarly pins what a fresh character owns after clvl 2, 6, 9 and 12 (Den point at clvl 5): sorted "Skill=level".
var goldenEarly = map[string]map[int]string{
	"sorceress": {2: "Charged Bolt=1", 6: "Charged Bolt=4 Frost Nova=1 Frozen Armor=1", 9: "Charged Bolt=7 Frost Nova=1 Frozen Armor=1"},
}

func skillString(m map[string]int) string {
	var s []string
	for k, v := range m {
		if v > 0 {
			s = append(s, fmt.Sprintf("%s=%d", k, v))
		}
	}
	sort.Strings(s)
	return strings.Join(s, " ")
}

// TestEarlyPrintTable logs the per-class table (go test -v -run EarlyPrintTable ./internal/leveler).
func TestEarlyPrintTable(t *testing.T) {
	geartest.Need(t)
	for _, c := range earlyClasses {
		snaps, _ := replay(t, c.name, 12, 5)
		for _, lv := range []int{2, 3, 6, 9, 12} {
			s := snaps[lv]
			t.Logf("%-11s clvl %2d str %d dex %d vit %d ene %d | %s", c.name, lv, s.attr[Str], s.attr[Dex], s.attr[Vit], s.attr[Ene], skillString(s.skills))
		}
	}
	_ = goldenEarly
}

// usableAttack is what the fighter would pick first at a clvl with the starting kit: the profile list filtered by the
// weapon rules of internal/gear (the same filter as Fighter.gearFilter) and by the skills the table has bought.
func usableAttack(class string, cls d2.Class, clvl int, skills map[string]int, kit gear.Equipped) []string {
	pr := combat.ProfileFor(cls, combat.BuildLightning)
	known := func(n string) bool {
		return skills[n] > 0 || n == "Attack" || (n == "Fire Bolt" && cls == d2.Sorceress)
	}
	var out []string
	for _, n := range pr.AttackNames(clvl, known) {
		if !known(n) {
			continue
		}
		if ok, _ := gear.SkillUsable(n, kit); ok {
			out = append(out, n)
		}
	}
	return out
}

func startKit(class string) gear.Equipped {
	var e gear.Equipped
	for _, it := range gear.StartItems(class) {
		switch it.Loc {
		case "rarm":
			e.Weapon = it.Code
		case "larm":
			e.Offhand = it.Code
		}
	}
	return e
}

func TestEarlyEveryClassHasAnAttackAtEveryLevel(t *testing.T) {
	geartest.Need(t)
	for _, c := range earlyClasses {
		snaps, _ := replay(t, c.name, 12, 5)
		kit := startKit(c.name)
		for lv := 1; lv <= 12; lv++ {
			att := usableAttack(c.name, c.cls, lv, snaps[lv].skills, kit)
			if len(att) == 0 {
				t.Errorf("%s clvl %d: no usable attack skill with the starting kit %+v", c.name, lv, kit)
			}
			if lv == 1 || lv == 3 || lv == 6 || lv == 9 {
				t.Logf("%-11s clvl %2d attack order: %v", c.name, lv, att)
			}
		}
	}
}

// hotkeyOf is the F-key the combat layout gives a skill ("" = none: left-hand skills are chosen through the left popup).
func hotkeyOf(cls d2.Class, skill string) string {
	return combat.ProfileFor(cls, combat.BuildLightning).Layout[skill]
}

// The Sorceress of the fallback table fights with Charged Bolt and falls back to Fire Bolt, her starting skill: both
// need a key (a right-hand skill without one is only usable while it is the selected skill, so Fire Bolt was lost
// the moment Charged Bolt was bound).
func TestEarlySorceressBoltsHaveKeys(t *testing.T) {
	for _, s := range []string{"Charged Bolt", "Fire Bolt", "Static Field"} {
		if hotkeyOf(d2.Sorceress, s) == "" {
			t.Errorf("sorceress lightning layout has no key for %s", s)
		}
	}
	seen := map[string]string{}
	for sk, k := range combat.ProfileFor(d2.Sorceress, combat.BuildLightning).Layout {
		if o, dup := seen[k]; dup {
			t.Errorf("key %s bound to both %s and %s", k, o, sk)
		}
		seen[k] = sk
	}
}

func TestEarlyPrintKeys(t *testing.T) {
	geartest.Need(t)
	for _, c := range earlyClasses {
		snaps, _ := replay(t, c.name, 12, 5)
		var s []string
		for sk, lv := range snaps[12].skills {
			if lv > 0 {
				s = append(s, fmt.Sprintf("%s=%q", sk, hotkeyOf(c.cls, sk)))
			}
		}
		sort.Strings(s)
		t.Logf("%-11s keys at clvl 12: %s", c.name, strings.Join(s, " "))
	}
}

// The Amazon table buys Exploding Arrow from clvl 12 (the gear build names it as the main attack to 30), so the
// combat profile has to be able to cast it, with a key; before this audit it never did.
func TestEarlyAmazonCastsExplodingArrowFromClvl12(t *testing.T) {
	geartest.Need(t)
	snaps, _ := replay(t, "amazon", 14, 5)
	bow := gear.Equipped{Weapon: "sbw", Offhand: "aqv", OffhandQty: 350}
	att := usableAttack("amazon", d2.Amazon, 14, snaps[14].skills, bow)
	if len(att) == 0 || att[0] != "Exploding Arrow" {
		t.Fatalf("amazon clvl 14 with a bow: attack order %v", att)
	}
	if hotkeyOf(d2.Amazon, "Exploding Arrow") == "" {
		t.Fatal("Exploding Arrow has no key")
	}
	for _, lv := range []int{3, 6, 9} {
		att := usableAttack("amazon", d2.Amazon, lv, snaps[lv].skills, bow)
		t.Logf("amazon clvl %d with a bought bow: %v", lv, att)
		if len(att) < 2 || att[0] != "Magic Arrow" && att[0] != "Fire Arrow" {
			t.Errorf("amazon clvl %d with a bow: %v", lv, att)
		}
	}
}

// ---- BUILDV2 (SME decisions for fresh characters) -------------------------------------------------------------

func skillAt(snaps map[int]snapshotAt, lv int, name string) int { return snaps[lv].skills[name] }

// Sorceress: Charged Bolt to 4 at clvl 5, the Den reward point on Frozen Armor, Frost Nova at 6 (not Static Field),
// Charged Bolt 8 at clvl 10, Telekinesis at 11, Charged Bolt 12 at clvl 15; 5 Vitality per level after the Str floor.
func TestBuildV2SorceressSequence(t *testing.T) {
	geartest.Need(t)
	for _, denAt := range []int{3, 5, 8} { // the reward point arrives whenever the Den is done: the order is the table's
		snaps, problems := replay(t, "sorceress", 15, denAt)
		for _, p := range problems {
			t.Errorf("den at %d: %s", denAt, p)
		}
		// the 5th skill point (4 level points + Den) is Frozen Armor, whichever level it arrives at
		want := map[int]string{
			2: "Charged Bolt=1", 3: "Charged Bolt=2",
		}
		for lv, w := range want {
			if denAt > lv {
				if got := skillString(snaps[lv].skills); got != w {
					t.Errorf("den at %d clvl %d: %q, want %q", denAt, lv, got, w)
				}
			}
		}
		if got := skillString(snaps[15].skills); got != "Charged Bolt=12 Frost Nova=1 Frozen Armor=1 Telekinesis=1" {
			t.Errorf("den at %d clvl 15: %q", denAt, got)
		}
		if skillAt(snaps, 15, "Static Field") != 0 {
			t.Errorf("v2 table bought Static Field")
		}
	}
	snaps, _ := replay(t, "sorceress", 15, 5)
	for lv, w := range map[int]string{
		4:  "Charged Bolt=3",
		5:  "Charged Bolt=4 Frozen Armor=1",
		6:  "Charged Bolt=4 Frost Nova=1 Frozen Armor=1",
		10: "Charged Bolt=8 Frost Nova=1 Frozen Armor=1",
		11: "Charged Bolt=8 Frost Nova=1 Frozen Armor=1 Telekinesis=1",
	} {
		if got := skillString(snaps[lv].skills); got != w {
			t.Errorf("clvl %d: %q, want %q", lv, got, w)
		}
	}
	ci, _ := gear.ClassStart("sorceress")
	for lv := 2; lv <= 15; lv++ {
		a := snaps[lv].attr
		if a[Ene] != ci.Ene || a[Dex] != ci.Dex {
			t.Errorf("clvl %d: energy/dexterity moved (%v), v2 spends everything on Vitality after the Str floor", lv, a)
		}
		if a[Str] != 12 {
			t.Errorf("clvl %d: Str %d, want the floor of 12 and no more", lv, a[Str])
		}
	}
}

// Amazon bow line: Magic Arrow at 2, Fire Arrow 3-6 and 8-11, Multiple Shot at 7, Exploding Arrow at 12.
func TestBuildV2AmazonSequence(t *testing.T) {
	geartest.Need(t)
	snaps, problems := replay(t, "amazon", 12, 5)
	for _, p := range problems {
		t.Error(p)
	}
	if skillAt(snaps, 2, "Magic Arrow") != 1 || skillAt(snaps, 12, "Exploding Arrow") < 1 {
		t.Fatalf("magic arrow at 2 / exploding at 12: %v %v", snaps[2].skills, snaps[12].skills)
	}
	// row 7, one level earlier with the Den point (req_level 6)
	if skillAt(snaps, 5, "Multiple Shot") != 0 || skillAt(snaps, 8, "Multiple Shot") != 1 {
		t.Errorf("Multiple Shot must arrive at clvl 6-8 (owned at 5: %d, at 8: %d)", skillAt(snaps, 5, "Multiple Shot"), skillAt(snaps, 8, "Multiple Shot"))
	}
	if skillAt(snaps, 5, "Fire Arrow") < 3 {
		t.Errorf("Fire Arrow at clvl 5: %d", skillAt(snaps, 5, "Fire Arrow"))
	}
	if DefaultBuildName("amazon") != "amazon_bow_normal30" {
		t.Errorf("the bow line must stay the default, got %s", DefaultBuildName("amazon"))
	}
	if _, err := LoadBuild("amazon_javelin_normal30", ""); err != nil {
		t.Errorf("the javelin variant must stay loadable: %v", err)
	}
}

// Paladin: Sacrifice only as the Zeal prerequisite (1 point at clvl 3, never more), Prayer at 4-5, Smite at 10, Zeal 4 points
// at clvl 12-15, plain Attack with the starting kit from clvl 1 to 9 and no Sacrifice in the attack order.
func TestBuildV2PaladinSequence(t *testing.T) {
	geartest.Need(t)
	snaps, problems := replay(t, "paladin", 15, 5)
	for _, p := range problems {
		t.Error(p)
	}
	for lv := 1; lv <= 15; lv++ {
		if n := skillAt(snaps, lv, "Sacrifice"); (lv < 3 && n != 0) || (lv >= 3 && n != 1) {
			t.Errorf("clvl %d: Sacrifice %d", lv, n)
		}
	}
	if skillAt(snaps, 3, "Sacrifice") != 1 || skillAt(snaps, 2, "Might") != 1 {
		t.Errorf("clvl 2 Might / clvl 3 Sacrifice: %v", snaps[3].skills)
	}
	if skillAt(snaps, 5, "Prayer") != 1 {
		t.Errorf("Prayer by clvl 5: %v", snaps[5].skills)
	}
	if skillAt(snaps, 9, "Smite") != 0 || skillAt(snaps, 10, "Smite") != 1 {
		t.Errorf("Smite must arrive at clvl 10")
	}
	if skillAt(snaps, 11, "Zeal") != 0 || skillAt(snaps, 15, "Zeal") != 4 {
		t.Errorf("Zeal 0 at 11 and 4 at 15, got %d / %d", skillAt(snaps, 11, "Zeal"), skillAt(snaps, 15, "Zeal"))
	}
	kit := startKit("paladin")
	for lv := 1; lv <= 9; lv++ {
		att := usableAttack("paladin", d2.Paladin, lv, snaps[lv].skills, kit)
		if len(att) != 1 || att[0] != "Attack" {
			t.Errorf("clvl %d attack order %v, want plain Attack", lv, att)
		}
	}
	att := usableAttack("paladin", d2.Paladin, 13, snaps[13].skills, kit)
	if len(att) < 2 || att[0] != "Zeal" || att[1] != "Smite" {
		t.Errorf("clvl 13 order %v, want Zeal, Smite, Attack", att)
	}
}

// Barbarian opens with Bash (one weapon), not Double Swing.
func TestBuildV2BarbarianOpensWithBash(t *testing.T) {
	geartest.Need(t)
	snaps, _ := replay(t, "barbarian", 12, 5)
	for lv := 3; lv <= 12; lv++ {
		att := usableAttack("barbarian", d2.Barbarian, lv, snaps[lv].skills, startKit("barbarian"))
		if len(att) == 0 || att[0] == "Double Swing" {
			t.Errorf("clvl %d order %v: Double Swing before Bash with one weapon", lv, att)
		}
	}
}

// A character that already spent points with the old table (Soaky: clvl 12 with Static Field) keeps it: no table switch,
// and its next points are the old table's, never Frost Nova or Frozen Armor.
func TestBuildV2ExistingCharacterKeepsOldTable(t *testing.T) {
	geartest.Need(t)
	for _, class := range []string{"sorceress", "paladin"} {
		old := DefaultBuildName(class)
		snaps, problems := replayBuild(t, class, old, 12, 5)
		for _, p := range problems {
			t.Errorf("%s old table: %s", class, p)
		}
		if got := SelectBuild(class, 12, snaps[12].skills); got != old {
			t.Fatalf("%s clvl 12 with the old table: selected %q, want %q", class, got, old)
		}
		if got := SelectBuild(class, 1, nil); got == old {
			t.Errorf("%s clvl 1 fresh: still the old table", class)
		}
	}
	// the v2 characters keep v2 at clvl 12
	for _, class := range []string{"sorceress", "paladin"} {
		v2 := SelectBuild(class, 1, nil)
		snaps, _ := replayBuild(t, class, v2, 12, 5)
		if got := SelectBuild(class, 12, snaps[12].skills); got != v2 {
			t.Errorf("%s clvl 12 on v2: selected %q, want %q", class, got, v2)
		}
	}
	// Soaky: the next point on the old table is a Charged Bolt, whatever the new table would say
	sk, _ := replayBuild(t, "sorceress", "fallback", 12, 5)
	b, _ := LoadBuild("fallback", "")
	have := Have{Attr: sk[12].attr, Start: [numAttr]int{10, 25, 10, 35}, Skills: sk[12].skills, Level: 13, SkillPts: 1}
	acts := b.Plan(have, nil, nil)
	if len(acts) != 1 || acts[0].Skill != "Charged Bolt" {
		t.Errorf("Soaky-like next point: %v", acts)
	}
}
