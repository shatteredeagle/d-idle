package status

import "fmt"

// areaNames names the areas the panels show. Act 1 is complete: ids 1..39 in Levels.txt order (S: the order matches
// internal/plandata/areas.json and the goals catalog, id 37 = Catacombs Level 4 = Andariel; V live: id 17 was the
// Burial Grounds on the d2d panels run of 2026-11-08, docs/probes/panels-live.md). The names are the in-game ones
// (docs/senses-map.md). Other acts: the ones the bot had seen before; the rest shows as unknown, never as a guess.
var areaNames = map[int]string{
	1: "Rogue Encampment", 2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field",
	5: "Dark Wood", 6: "Black Marsh", 7: "Tamoe Highland", 8: "Den of Evil",
	9: "Cave Level 1", 10: "Underground Passage Level 1", 11: "Hole Level 1", 12: "Pit Level 1",
	13: "Cave Level 2", 14: "Underground Passage Level 2", 15: "Hole Level 2", 16: "Pit Level 2",
	17: "Burial Grounds", 18: "Crypt", 19: "Mausoleum", 20: "Forgotten Tower",
	21: "Tower Cellar Level 1", 22: "Tower Cellar Level 2", 23: "Tower Cellar Level 3", 24: "Tower Cellar Level 4",
	25: "Tower Cellar Level 5", 26: "Monastery Gate", 27: "Outer Cloister", 28: "Barracks",
	29: "Jail Level 1", 30: "Jail Level 2", 31: "Jail Level 3", 32: "Inner Cloister", 33: "Cathedral",
	34: "Catacombs Level 1", 35: "Catacombs Level 2", 36: "Catacombs Level 3", 37: "Catacombs Level 4",
	38: "Tristram", 39: "Moo Moo Farm",
	40: "Lut Gholein", 75: "Kurast Docks", 103: "Pandemonium Fortress", 109: "Harrogath",
}

// AreaName names an area id; id 0 (no area) is empty and an id without a name reads "Unknown area (id N)".
func AreaName(id int) string {
	if n, ok := areaNames[id]; ok {
		return n
	}
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("Unknown area (id %d)", id)
}
