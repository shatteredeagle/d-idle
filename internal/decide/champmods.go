package decide

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

// Champion and unique modifier names for the senses elite markers (TASK-152). The ids and names are data
// (champmods.json); no mod name is Go.

//go:embed champmods.json
var shippedMods []byte

// ChampMod is one row of champmods.json.
type ChampMod struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag"` // S secondary, U unverified
}

var (
	modsOnce  sync.Once
	modsByID  map[int]ChampMod
	modsError error
)

func loadMods() {
	var f struct {
		Mods []ChampMod `json:"mods"`
	}
	if err := json.Unmarshal(shippedMods, &f); err != nil {
		modsError = fmt.Errorf("champmods.json: %w", err)
		return
	}
	modsByID = map[int]ChampMod{}
	for _, m := range f.Mods {
		if m.Name == "" {
			modsError = fmt.Errorf("champmods.json: id %d has no name", m.ID)
			return
		}
		if _, dup := modsByID[m.ID]; dup {
			modsError = fmt.Errorf("champmods.json: duplicate id %d", m.ID)
			return
		}
		modsByID[m.ID] = m
	}
}

// ModName is the display name of a monster modifier id. An id not in the table (or a broken table) reads
// "unknown mod #N", so a marker never shows a blank or a wrong name.
func ModName(id int) string {
	modsOnce.Do(loadMods)
	if m, ok := modsByID[id]; ok && modsError == nil {
		return m.Name
	}
	return fmt.Sprintf("unknown mod #%d", id)
}

// ChampMods returns the table rows (a copy) or the error that broke the table.
func ChampMods() ([]ChampMod, error) {
	modsOnce.Do(loadMods)
	if modsError != nil {
		return nil, modsError
	}
	out := make([]ChampMod, 0, len(modsByID))
	for _, m := range modsByID {
		out = append(out, m)
	}
	return out, nil
}
