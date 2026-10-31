// d2snap: runs the d2 readers against the live game and records fixtures; also hosts a few
// read-only exploration subcommands used to resolve open questions (docs/mem.md).
package main

import (
	"d2lod-bot/internal/buildinfo"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/mem"
)

func main() {
	buildinfo.Exit("d2snap")
	if len(os.Args) < 2 {
		fmt.Println("usage: d2snap hex RVA LEN | ...")
		os.Exit(2)
	}
	p, err := mem.OpenGame()
	if err != nil {
		fmt.Println("open game:", err)
		os.Exit(1)
	}
	defer p.Close()
	for _, a := range os.Args[1:] {
		switch {
		case a == "stats":
			exploreStats(p)
		case a == "inv":
			exploreInv(p)
		case a == "itemtxt":
			exploreItemTxt(p, 63, 587, 529, 530, 0, 1, 2, 3, 100)
		case strings.HasPrefix(a, "record="):
			f := strings.SplitN(a[7:], ",", 2)
			key := ""
			if len(f) == 2 {
				key = f[1]
			}
			record(p, f[0], key)
		case strings.HasPrefix(a, "cursor="):
			if h, err := dialHelper(); err == nil {
				fmt.Println(h.send("move " + strings.Replace(a[7:], ",", " ", 1)))
				h.c.Close()
				time.Sleep(300 * time.Millisecond)
				fmt.Println("ui nonzero:", nonzero(uiArray(p)))
			}
		case a == "sweep":
			liveSweep(p)
		case a == "hover":
			liveHover(p)
		case strings.HasPrefix(a, "ui="):
			liveUI(p, strings.Split(a[3:], ","))
		case a == "player":
			explorePlayer(p)
		case a == "mon":
			exploreUnits(p, 1, 4)
		case a == "obj":
			exploreUnits(p, 2, 3)
		case a == "skills":
			exploreSkills(p)
		case strings.Contains(a, ":"): // RVA:LEN
			f := strings.SplitN(a, ":", 2)
			rva, _ := strconv.ParseUint(f[0], 0, 32)
			n, _ := strconv.Atoi(f[1])
			b := make([]byte, n)
			if err := p.ReadAt(b, mem.Base+uint32(rva)); err != nil {
				fmt.Println(a, err)
				continue
			}
			fmt.Println(a, hex.EncodeToString(b))
		default:
			fmt.Println("unknown command", a)
		}
	}
}
