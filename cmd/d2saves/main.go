// d2saves: live driver for internal/saves (snapshot, list, restore, verify) on this instance's save folder.
// Run through tools/live.sh; refuses nothing itself beyond what saves does, so only use it with the game at a menu.
//
//	d2saves snapshot NAME | list NAME | restore NAME [ID] | verify NAME
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"d2lod-bot/internal/cheat"
	"d2lod-bot/internal/saves"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: d2saves snapshot|list|restore|verify NAME [ID]")
		os.Exit(2)
	}
	inst := os.Getenv("D2BOT_INSTANCE")
	dir := cheat.DefaultSaveDir(inst)
	home, _ := os.UserHomeDir()
	bak := filepath.Join(home, "d2lod-live", "saves-backup", inst)
	name := os.Args[3-1]
	o := saves.Options{AtMenu: true}
	var err error
	switch os.Args[1] {
	case "snapshot":
		var r saves.Result
		r, err = saves.Snapshot(dir, bak, name, o, time.Now())
		fmt.Printf("snapshot %s unchanged=%v pruned=%v\n", r.ID, r.Unchanged, r.Pruned)
	case "list":
		var l []saves.Info
		l, err = saves.List(bak, name)
		for _, i := range l {
			fmt.Println(i.ID, i.Files)
		}
	case "restore":
		id := ""
		if len(os.Args) > 3 {
			id = os.Args[3]
		}
		var i saves.Info
		i, err = saves.Restore(dir, bak, name, id, o, time.Now())
		fmt.Println("restored", i.ID)
	case "verify":
		var p saves.Progress
		p, err = saves.ProgressFile(filepath.Join(dir, name+".d2s"))
		if err == nil {
			_, err = saves.VerifyFile(filepath.Join(dir, name+".d2s"))
		}
		fmt.Printf("%s level %d map 0x%08x\n", p.Name, p.Level, p.MapID)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "d2saves:", err)
		os.Exit(1)
	}
}
