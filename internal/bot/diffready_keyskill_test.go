package bot

import "testing"

// Review 8 C9: the readiness key skill is the table's main attack at the hero's level (data), for two builds; a
// table without main_attack, an unknown table and a weapon attack stay unread (-1) and never block.
func TestKeySkillLevelOf(t *testing.T) {
	skill := func(n string) bool { return n != "Attack" && n != "Throw" }
	for _, tc := range []struct {
		build string
		clvl  int
		owned map[string]int
		want  int
	}{
		{"paladin_hammerdin_endgame90", 40, map[string]int{"Blessed Hammer": 14, "Smite": 1}, 14},
		{"paladin_hammerdin_endgame90", 10, map[string]int{"Smite": 3}, 3},
		{"necromancer_leveling_normal30", 25, map[string]int{"Teeth": 5, "Bone Spear": 7}, 7},
		{"necromancer_leveling_normal30", 25, map[string]int{"Teeth": 5}, 0}, // the key skill is not learned: level 0 blocks
		{"amazon_bow_normal30", 2, nil, -1},                                  // Throw is a weapon attack, not a skill
		{"-", 30, nil, -1},
	} {
		if got := keySkillLevelOf(tc.build, tc.clvl, tc.owned, skill); got != tc.want {
			t.Errorf("%s clvl %d: %d, want %d", tc.build, tc.clvl, got, tc.want)
		}
	}
}
