// mpqcat prints one game table from the user's own MPQs (priority Patch_D2, d2exp, d2data), for table facts in docs.
//
//	mpqcat DIR data\global\excel\DifficultyLevels.txt [COL ...]   (COL: print only these columns, matched case-insensitively)
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"d2lod-bot/internal/data/mpq"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mpqcat DIR PATH [COL ...]")
		os.Exit(2)
	}
	for _, a := range []string{"Patch_D2.mpq", "d2exp.mpq", "d2data.mpq"} {
		ar, err := mpq.Open(filepath.Join(os.Args[1], a))
		if err != nil || !ar.Has(os.Args[2]) {
			continue
		}
		b, err := ar.ReadFile(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("# from", a)
		lines := strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n")
		var idx []int
		for i, l := range lines {
			f := strings.Split(l, "\t")
			if i == 0 && len(os.Args) > 3 {
				for _, c := range os.Args[3:] {
					for j, h := range f {
						if strings.EqualFold(h, c) {
							idx = append(idx, j)
						}
					}
				}
			}
			if len(os.Args) > 3 {
				var o []string
				for _, j := range idx {
					if j < len(f) {
						o = append(o, f[j])
					}
				}
				l = strings.Join(o, "\t")
			}
			fmt.Println(l)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "not found")
	os.Exit(1)
}
