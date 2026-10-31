// d2savecheck prints mtime, size, level, exp and gold of this instance's .d2s next to the game's memory values.
// Run through tools/live.sh (read-only).
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/saves"
)

func main() {
	name := "Multi"
	if len(os.Args) > 1 && os.Args[1] != "" {
		name = os.Args[1]
	}
	p := filepath.Join(cheat.DefaultSaveDir(os.Getenv("D2BOT_INSTANCE")), name+".d2s")
	st, err := os.Stat(p)
	if err != nil {
		fmt.Println("stat:", err)
		os.Exit(1)
	}
	b, _ := os.ReadFile(p)
	f, err := saves.Verify(b)
	if err != nil {
		fmt.Println("verify:", err)
		os.Exit(1)
	}
	exp, _ := f.Stat(d2s.StatExp)
	gold, _ := f.Stat(d2s.StatGold)
	if len(os.Args) > 2 { // d2savecheck NAME CLASS: assert a fresh softcore expansion character (tools/soak/fresh.sh)
		if err := saves.CheckFresh(f, name, os.Args[2]); err != nil {
			fmt.Println("FRESH FAIL:", err)
			os.Exit(1)
		}
		fmt.Println("FRESH OK", name, os.Args[2])
	}
	fmt.Printf("FILE mtime=%s now=%s size=%d level=%d exp=%d gold=%d\n", st.ModTime().Format("01-02 15:04:05"), time.Now().Format("01-02 15:04:05"), st.Size(), f.Level(), exp, gold)
}
