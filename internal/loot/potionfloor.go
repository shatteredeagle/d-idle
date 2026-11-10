package loot

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"d2lod-bot/internal/d2"
)

//go:embed potionfloor.json
var potionFloorJSON []byte

// floorRow is one row of potionfloor.json. Class is optional (empty = every class).
type floorRow struct {
	Difficulty string `json:"difficulty"`
	Class      string `json:"class"`
	FromLevel  int    `json:"fromLevel"`
	MinTier    int    `json:"minTier"`
}

var (
	floorOnce sync.Once
	floorRows []floorRow
	floorErr  error
)

func loadFloor() ([]floorRow, error) {
	floorOnce.Do(func() {
		var f struct {
			Rows []floorRow `json:"rows"`
		}
		if floorErr = json.Unmarshal(potionFloorJSON, &f); floorErr != nil {
			floorErr = fmt.Errorf("loot: potionfloor.json: %w", floorErr)
			return
		}
		for i, r := range f.Rows {
			if !knownDifficulty(r.Difficulty) || r.FromLevel < 1 || r.MinTier < 1 || r.MinTier > 5 {
				floorErr = fmt.Errorf("loot: potionfloor.json row %d invalid (difficulty %q, fromLevel %d, minTier %d)", i, r.Difficulty, r.FromLevel, r.MinTier)
				return
			}
		}
		floorRows = f.Rows
	})
	return floorRows, floorErr
}

func knownDifficulty(s string) bool {
	for _, d := range []d2.Difficulty{d2.Normal, d2.Nightmare, d2.Hell} {
		if strings.EqualFold(s, d.String()) {
			return true
		}
	}
	return false
}

// PotionTierFloor is the lowest healing/mana potion tier worth taking for a character (1 when no row applies).
func PotionTierFloor(class d2.Class, clvl int, diff d2.Difficulty) (int, error) {
	rows, err := loadFloor()
	if err != nil {
		return 1, err
	}
	best, bestKey := 1, -1
	for _, r := range rows {
		if !strings.EqualFold(r.Difficulty, diff.String()) || r.FromLevel > clvl {
			continue
		}
		if r.Class != "" && !strings.EqualFold(r.Class, class.String()) {
			continue
		}
		key := r.FromLevel * 2
		if r.Class != "" {
			key++
		}
		if key > bestKey {
			best, bestKey = r.MinTier, key
		}
	}
	return best, nil
}
