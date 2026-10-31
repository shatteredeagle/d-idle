// Command d2dmap prints the d2d world source (internal/d2dworld) for the current level as ASCII, plus the exits and
// presets, for manual validation against the game. Read-only.
//
//	d2dmap -sock ~/d2d-lab/fork/d2d.sock [-origin none|tiles|subtiles] [-w 160 -h 60] [-max CELLS]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dworld"
)

func main() {
	sock := flag.String("sock", os.Getenv("D2D_SOCK"), "d2d control socket (default $D2D_SOCK)")
	origin := flag.String("origin", "none", "cell origin: none (level-relative), tiles (+5*wx,5*wy) or subtiles (+wx,wy)")
	mw := flag.Int("w", 160, "max output columns")
	mh := flag.Int("h", 60, "max output rows")
	maxCells := flag.Int("max", 0, "grid cell cap (0: 4096*4096)")
	flag.Parse()
	if *sock == "" {
		fmt.Fprintln(os.Stderr, "d2dmap: need -sock or $D2D_SOCK")
		os.Exit(2)
	}
	var mode d2dworld.OriginMode
	switch *origin {
	case "none":
	case "tiles":
		mode = d2dworld.OriginTiles
	case "subtiles":
		mode = d2dworld.OriginSubtiles
	default:
		fmt.Fprintln(os.Stderr, "d2dmap: -origin must be none, tiles or subtiles")
		os.Exit(2)
	}
	c := d2dctl.New(*sock, d2dctl.Options{Timeout: 15 * time.Second})
	defer c.Close()
	w := d2dworld.New(d2dworld.Client{Client: c}, d2dworld.Options{Origin: mode, MaxCells: *maxCells, Timeout: 30 * time.Second})
	st, err := w.Update()
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2dmap:", err)
		os.Exit(1)
	}
	lm := w.Current()
	fmt.Printf("level %d act %d player %v rect %+v known %d/%d note %q\n", st.LevelNo, st.Act, st.Pos, lm.Rect, lm.Grid.KnownCount(), lm.Rect.W*lm.Rect.H, w.LastGridNote)
	fmt.Print(d2dworld.ASCII(lm, *mw, *mh))
	fmt.Printf("borders: %s\n", w.LastBorderNote)
	for _, a := range lm.Adjacent {
		fmt.Printf("adjacent level %d rect %+v contact %+v\n", a.LevelNo, a.Rect, a.Contact)
	}
	for _, e := range lm.Exits {
		fmt.Printf("exit to level %d at %v\n", e.Dest, e.Pos)
	}
	for i, p := range lm.Presets {
		fmt.Printf("preset id %d at %v\n", p.ID, p.Pos)
		if i > 40 {
			fmt.Println("...")
			break
		}
	}
	if npcs, err := w.NPCs(context.Background()); err == nil {
		for _, n := range npcs {
			fmt.Printf("npc %q index %d at %v\n", n.Name, n.Index, n.Pos)
		}
	}
	fmt.Printf("walkable at player: %v\n", lm.Grid.Walkable(st.Pos.X, st.Pos.Y))
}
