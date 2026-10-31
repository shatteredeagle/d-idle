package data

// Level is one levels.txt row: an area. The Id column is the area id the game
// reports for the player and for level-change messages.
type Level struct {
	ID        int
	Key       string // Name column, e.g. "Act 1 - Wilderness 1"
	Name      string // LevelName column: display name, e.g. "Blood Moor"
	WarpName  string // LevelWarp: text on the exit sign
	Act       int    // 0-based act index (0..4)
	Layer     int
	Depend    int  // parent area id
	Teleport  int  // 0 no, 1 yes, 2 forbidden... raw column value
	Outdoor   bool // !IsInside
	DrlgType  int  // 1 maze, 2 preset, 3 outdoor (raw)
	LevelType int
	// Size per difficulty in subtiles/tiles as the game stores it.
	SizeX, SizeY     [3]int
	OffsetX, OffsetY int
	// MonLvl is the monster level per difficulty for LoD (MonLvl*Ex columns
	// on 1.10+; classic MonLvl* otherwise).
	MonLvl           [3]int
	MonDen           [3]int
	MonUMin, MonUMax [3]int
	NumMon           int
	// Vis are neighbouring area ids reachable from this one (0 = none); Warp
	// are the matching warp indices into lvlwarp.
	Vis  [8]int
	Warp [8]int
	// Monsters lists monster keys that spawn here: normal, nightmare/hell,
	// and unique/champion pools (mon1-10, nmon1-10, umon1-10).
	Mon, NMon, UMon  []string
	Waypoint         int // waypoint number (255 = none)
	Quest            int
	Portal, Position int
}

var levelCols = []string{
	"Name", "Id", "Act", "Layer", "SizeX", "SizeY", "SizeX(N)", "SizeY(N)", "SizeX(H)", "SizeY(H)", "OffsetX", "OffsetY",
	"Depend", "Teleport", "IsInside", "DrlgType", "LevelType", "Vis0", "Vis1", "Vis2", "Vis3", "Vis4", "Vis5", "Vis6", "Vis7",
	"Warp0", "Warp1", "Warp2", "Warp3", "Warp4", "Warp5", "Warp6", "Warp7", "Portal", "Position", "Quest",
	"MonLvl1", "MonLvl2", "MonLvl3", "MonLvl1Ex", "MonLvl2Ex", "MonLvl3Ex",
	"MonDen", "MonDen(N)", "MonDen(H)", "MonUMin", "MonUMax", "MonUMin(N)", "MonUMax(N)", "MonUMin(H)", "MonUMax(H)", "NumMon",
	"mon1", "mon2", "mon3", "mon4", "mon5", "mon6", "mon7", "mon8", "mon9", "mon10",
	"nmon1", "nmon2", "nmon3", "nmon4", "nmon5", "nmon6", "nmon7", "nmon8", "nmon9", "nmon10",
	"umon1", "umon2", "umon3", "umon4", "umon5", "umon6", "umon7", "umon8", "umon9", "umon10",
	"Waypoint", "LevelName", "LevelWarp",
}

func loadLevels(t *Table) []*Level {
	var out []*Level
	for i := 0; i < t.Len(); i++ {
		r := t.Row(i)
		key := r.Str("Name")
		if key == "" || key == "Expansion" {
			continue
		}
		l := &Level{
			ID: r.Int("Id"), Key: key, Name: r.Str("LevelName"), WarpName: r.Str("LevelWarp"),
			Act: r.Int("Act"), Layer: r.Int("Layer"), Depend: r.Int("Depend"), Teleport: r.Int("Teleport"),
			Outdoor: !r.Bool("IsInside"), DrlgType: r.Int("DrlgType"), LevelType: r.Int("LevelType"),
			OffsetX: r.Int("OffsetX"), OffsetY: r.Int("OffsetY"), NumMon: r.Int("NumMon"),
			Waypoint: r.Int("Waypoint"), Quest: r.Int("Quest"), Portal: r.Int("Portal"), Position: r.Int("Position"),
		}
		for d := 0; d < 3; d++ {
			s := diffSuffix[d]
			l.SizeX[d], l.SizeY[d] = r.Int("SizeX"+s), r.Int("SizeY"+s)
			l.MonDen[d] = r.Int("MonDen" + s)
			l.MonUMin[d], l.MonUMax[d] = r.Int("MonUMin"+s), r.Int("MonUMax"+s)
			n := string(rune('1' + d))
			l.MonLvl[d] = r.Int("MonLvl"+n+"Ex", "MonLvl"+n)
		}
		for k := 0; k < 8; k++ {
			l.Vis[k] = r.Int("Vis" + string(rune('0'+k)))
			l.Warp[k] = r.Int("Warp" + string(rune('0'+k)))
		}
		for k := 1; k <= 10; k++ {
			n := itoa(k)
			if s := r.Str("mon" + n); s != "" {
				l.Mon = append(l.Mon, s)
			}
			if s := r.Str("nmon" + n); s != "" {
				l.NMon = append(l.NMon, s)
			}
			if s := r.Str("umon" + n); s != "" {
				l.UMon = append(l.UMon, s)
			}
		}
		out = append(out, l)
	}
	return out
}
