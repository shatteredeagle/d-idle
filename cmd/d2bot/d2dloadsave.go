package main

import (
	"log/slog"
	"os"
	"path/filepath"
)

// classForStart is the --d2d-class a managed d2d is started with. A class >= 0 makes d2d create a NEW hero in memory
// with --start-name and write it over the saved one at the next save or quit (VERIFIED LIVE on a lab slot, 2026-11-27:
// a save with level 2 exp 1313 came back level 1 exp 0). The supervisor restarts the child with the arguments of the
// first start, which were a create when the container began without a save, so a restart after a watchdog exit
// destroyed the progress. A save that exists is always loaded (-1).
func classForStart(class int, home, name string, log *slog.Logger) int {
	if class < 0 || home == "" || name == "" {
		return class
	}
	p := filepath.Join(home, "config", "d2d", "save", name+".d2s")
	if st, err := os.Stat(p); err != nil || st.Size() == 0 {
		return class
	}
	log.Warn("d2d class given but a save exists: loading it, not creating over it", "char", name, "class", class)
	return -1
}
