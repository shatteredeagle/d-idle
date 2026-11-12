// Command uishots is the Go half of the watch UI screenshot suite (docs/ui-regression-plan.md section 4; the shell
// half is tools/uishots.sh, which runs headless Chrome under a private Xvfb on the worker host).
//
//	uishots serve -addr 127.0.0.1:18941 [-class sorceress]   the frozen fixture viewer the pictures are taken of
//	uishots diff -golden DIR -got DIR [-max 0.005]          compare PNGs; fail when an image changed more than max
//	uishots check DIR                                       check the DOM dumps (*.html) against the panel contract
//
// Stdlib only.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ExitOnError)
		addr := fs.String("addr", "127.0.0.1:18941", "listen address (loopback)")
		class := fs.String("class", "sorceress", "simulated build: sorceress or paladin")
		fs.StringVar(&fogStage, "fog", "", "serve the real senses publisher over a cave instead of the fixture: start or walked (docs/mapfog.md)")
		fs.Parse(os.Args[2:])
		if err := serve(*addr, *class); err != nil {
			fmt.Fprintln(os.Stderr, "uishots:", err)
			os.Exit(1)
		}
	case "diff":
		fs := flag.NewFlagSet("diff", flag.ExitOnError)
		golden := fs.String("golden", "", "directory of golden PNGs")
		got := fs.String("got", "", "directory of new PNGs")
		max := fs.Float64("max", 0.005, "largest allowed share of changed pixels per image")
		fs.Parse(os.Args[2:])
		res, err := diffDirs(*golden, *got, *max)
		for _, r := range res {
			fmt.Println(r)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "uishots:", err)
			os.Exit(1)
		}
	case "check":
		if len(os.Args) < 3 {
			usage()
		}
		bad, err := checkDir(os.Args[2])
		for _, b := range bad {
			fmt.Println(b)
		}
		if err != nil || len(bad) > 0 {
			if err != nil {
				fmt.Fprintln(os.Stderr, "uishots:", err)
			}
			os.Exit(1)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: uishots serve|diff|check (see the package comment)")
	os.Exit(2)
}
