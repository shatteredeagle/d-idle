// Command d2tables extracts the gear tables (item types, weapon/shield/ammo bases, skills.txt prerequisites,
// charstats, armor.txt) from the user's own game MPQs into the tables cache. Nothing it writes belongs in the repo.
//
//	d2tables [--data MPQ_DIR] [--out CACHE_DIR] [--force]
//
// --data defaults to $D2_DATA_DIR, --out to $D2_TABLES_DIR or <user cache dir>/d2lod/tables.
package main

import (
	"flag"
	"fmt"
	"os"

	"d2lod-bot/internal/gear"
)

func main() {
	data := flag.String("data", os.Getenv("D2_DATA_DIR"), "directory with d2data.mpq, d2exp.mpq and Patch_D2.mpq (env D2_DATA_DIR)")
	out := flag.String("out", gear.TablesDir(), "cache directory to write (env D2_TABLES_DIR)")
	force := flag.Bool("force", false, "extract even when the cache is current")
	flag.Parse()
	if *data == "" {
		fmt.Fprintln(os.Stderr, "d2tables: need --data <MPQ dir> or D2_DATA_DIR")
		os.Exit(2)
	}
	if !*force && gear.TablesCurrent(*data, *out) {
		fmt.Println("d2tables: cache is current:", *out)
		return
	}
	if err := gear.ExtractTables(*data, *out); err != nil {
		fmt.Fprintln(os.Stderr, "d2tables:", err)
		os.Exit(1)
	}
	fmt.Println("d2tables: wrote", *out)
}
