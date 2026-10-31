package data

// MonLevel is one monlvl.txt row: the per-monster-level scaling the game
// applies to monstats' base values. Each field is indexed by Difficulty.
// The file has two column sets: the plain ones (classic) and the "L-" ones
// (Lord of Destruction). LoD data is stored here. UNVERIFIED live: that
// 1.14d uses the L- set (community documentation says so).
type MonLevel struct {
	Level int
	AC    [3]int
	TH    [3]int // to-hit
	HP    [3]int // percent multiplier on monstats min/max HP
	DM    [3]int // percent multiplier on monstats damage
	XP    [3]int // percent multiplier on monstats experience
}

var monLevelCols = func() []string {
	c := []string{"Level"}
	for _, p := range []string{"L-AC", "L-TH", "L-HP", "L-DM", "L-XP", "AC", "TH", "HP", "DM", "XP"} {
		for _, s := range diffSuffix {
			c = append(c, p+s)
		}
	}
	return c
}()

func loadMonLevels(t *Table) []*MonLevel {
	var out []*MonLevel
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		if r.Str("Level") == "" {
			continue
		}
		m := &MonLevel{Level: r.Int("Level")}
		for d, s := range diffSuffix {
			m.AC[d] = r.Int("L-AC"+s, "AC"+s)
			m.TH[d] = r.Int("L-TH"+s, "TH"+s)
			m.HP[d] = r.Int("L-HP"+s, "HP"+s)
			m.DM[d] = r.Int("L-DM"+s, "DM"+s)
			m.XP[d] = r.Int("L-XP"+s, "XP"+s)
		}
		out = append(out, m)
	}
	return out
}

// DifficultyInfo is a difficultylevels.txt row.
type DifficultyInfo struct {
	Name            string
	ResistPenalty   int // added to the player's resistances (0, -40, -100)
	DeathExpPenalty int // percent of level experience lost on death
}

var difficultyCols = []string{"Name", "ResistPenalty", "DeathExpPenalty"}

func loadDifficulties(t *Table) []DifficultyInfo {
	var out []DifficultyInfo
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		out = append(out, DifficultyInfo{r.Str("Name"), r.Int("ResistPenalty"), r.Int("DeathExpPenalty")})
	}
	return out
}
