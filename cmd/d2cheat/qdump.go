package main

import (
	"fmt"
	"os"

	"d2lod-bot/internal/cheat/d2s"
)

// qdump prints, read-only and offline, the header bytes, map id, non-zero quest words and
// waypoint masks of a .d2s (live verification of quest/waypoint/difficulty edits).
func qdump(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		fatal(err)
	}
	f, err := d2s.Parse(b)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("%s class %d level %d status 0x%02x progression %d mapid 0x%08x diff %02x %02x %02x\n",
		f.Name(), f.Class(), f.Level(), f.Status(), f.Progression(), f.MapID(), f.Difficulty(0), f.Difficulty(1), f.Difficulty(2))
	for d := 0; d < 3; d++ {
		fmt.Printf("D%d wp=%010x quests:", d, f.WaypointMask(d))
		for i := 0; i < 48; i++ {
			if w, err := f.QuestWord(d, i); err == nil && w != 0 {
				fmt.Printf(" %d=0x%04x", i, w)
			}
		}
		fmt.Println()
	}
}
