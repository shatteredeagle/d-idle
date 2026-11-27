package leveler

import "testing"

// walkTable replays a table row by row against the prerequisite and level rule of prereq_test.go (S: skills.txt) and
// returns the points per skill at clvl 30. A row spends one level point plus any quest-tagged points.
func walkTable(t *testing.T, name string) map[string]int {
	t.Helper()
	lb, err := LoadBuild(name, "")
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]int{}
	for _, r := range lb.Rows {
		if r.Clvl > 30 {
			break
		}
		level := 0
		for _, s := range r.Skills {
			if !fakeRule(s.Skill, r.Clvl, have) {
				t.Errorf("%s clvl %d: %s is not legal here (have %v)", name, r.Clvl, s.Skill, have)
			}
			have[s.Skill] += s.Points
			if s.Quest == "" {
				level += s.Points
			}
		}
		if level > 1 {
			t.Errorf("%s clvl %d: %d level points in one row", name, r.Clvl, level)
		}
	}
	return have
}

// SME review 2026-11-27, confidence high: the bonespear table has no respec, so Iron Maiden and Bone Wall points are
// dead; rows 12, 15 and 16 go to Teeth. Second build: the summoner default must stay legal and keep its own points.
func TestBonespearTableNoDeadPoints(t *testing.T) {
	got := walkTable(t, "necromancer_leveling_normal30")
	if got["Iron Maiden"] != 0 || got["Bone Wall"] != 0 {
		t.Errorf("dead points: Iron Maiden %d Bone Wall %d", got["Iron Maiden"], got["Bone Wall"])
	}
	if got["Teeth"] != 14 {
		t.Errorf("Teeth %d points, want 14", got["Teeth"])
	}
	sum := 0
	for _, n := range got {
		sum += n
	}
	if sum != 32 { // 29 level points (clvl 2..30) + Den, Radament, Izual
		t.Errorf("total points %d, want 32", sum)
	}
	sum = 0
	for _, n := range walkTable(t, "necromancer_summoner_normal30") {
		sum += n
	}
	if sum != 32 {
		t.Errorf("summoner total points %d, want 32", sum)
	}
}
