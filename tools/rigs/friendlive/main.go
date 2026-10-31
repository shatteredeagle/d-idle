//go:build rigs

// friendlive: read-only dump of every monster near the player with owner words, child states and the
// combat.IsFriendly verdict. Used to live-verify docs/friendly.md. Usage: friendlive [radius]
package main

import (
	"fmt"
	"os"
	"strconv"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/mem"
)

func main() {
	src, err := mem.OpenGame()
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer src.Close()
	home, _ := os.UserHomeDir()
	db, derr := data.Open(home+"/Games/d2lod-inst/"+os.Getenv("D2BOT_INSTANCE")+"/proton/pfx/drive_c/Program Files (x86)/Diablo II", home+"/d2lod-live/combat/data")
	if derr != nil {
		fmt.Println("data:", derr)
	}
	cat := combat.NewCatalog(db)
	snap, err := d2.New(src).Snapshot()
	if err != nil || snap.Player == nil {
		fmt.Println("snapshot:", err)
		os.Exit(1)
	}
	rad := 40
	if len(os.Args) > 1 {
		rad, _ = strconv.Atoi(os.Args[1])
	}
	p := snap.Player
	fmt.Printf("player %s id %#x class %v area %d pos %d,%d\n", p.Name, p.ID, p.Class, p.Area, p.X, p.Y)
	for _, u := range snap.Monsters {
		dx, dy := u.X-p.X, u.Y-p.Y
		if dx*dx+dy*dy > rad*rad {
			continue
		}
		name, ai := "?", "?"
		if m := cat.Monster(u.Class); m != nil {
			name, ai = cat.MonsterName(m), m.AI
		}
		var st []uint8
		if u.Monster != nil {
			st = u.Monster.States
		}
		w, fr := combat.FriendlyWhyOf(snap, u, cat)
		fmt.Printf("class %3d %-18s ai=%-12s id %#x mode %d d=(%d,%d) owner type %d id %#x states %v friendly=%v (%v)\n",
			u.Class, name, ai, u.ID, u.Mode, dx, dy, u.OwnerType, u.OwnerID, st, fr, w)
	}
}
