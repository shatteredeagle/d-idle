package status

import "fmt"

// areaNames is a tiny fallback so the display is readable before the DATA agent
// provides real tables. Ids from the classic levels.txt order. UNVERIFIED beyond
// the ones the bot has seen live (1 = Rogue Encampment).
var areaNames = map[int]string{
	1: "Rogue Encampment", 2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field",
	5: "Dark Wood", 6: "Black Marsh", 7: "Tamoe Highland", 8: "Den of Evil",
	40: "Lut Gholein", 75: "Kurast Docks", 103: "Pandemonium Fortress", 109: "Harrogath",
}

// AreaName names an area id; unknown ids render as "Area N".
func AreaName(id int) string {
	if n, ok := areaNames[id]; ok {
		return n
	}
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("Area %d", id)
}
