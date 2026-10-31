package cheat

import (
	"os"
	"sync"

	"d2lod-bot/internal/paths"
)

// layout resolves the data-dir layout from the environment (paths.FromEnv). Without a home directory it
// falls back to a layout under the temp dir so the tool still starts.
func layout() paths.Layout {
	l, err := paths.FromEnv()
	if err != nil {
		l = paths.Layout{Root: os.TempDir() + "/d2bot"}
	}
	return l
}

// gameRoot is the Wine-side root of an instance (paths.Layout.GameRoot: the new prefixes/<id> layout, or the
// dev-era ~/Games/d2lod[-inst/<id>] on a machine that still has it). Empty inst means i0.
func gameRoot(inst string) string {
	if inst == "" {
		inst = "i0"
	}
	return layout().GameRoot(inst)
}

// DefaultBackupDir is $D2CHEAT_BACKUPS or <data>/ledger/backups. Backups made before the data-dir layout
// are in $D2_COLD_DIR/d2lod/backups; ledger entries keep their absolute path, so they stay valid there.
func DefaultBackupDir() string {
	if p := os.Getenv("D2CHEAT_BACKUPS"); p != "" {
		return p
	}
	return layout().CheatBackups()
}

var legacyOnce sync.Once

// MigrateLegacy moves the cheat ledger written by older builds (~/.local/state/d2lod/cheats.jsonl) to
// <data>/ledger/cheats.jsonl when the new file does not exist yet. A ledger already at the new place wins
// and the old file stays untouched.
func MigrateLegacy(l paths.Layout) error {
	_, err := paths.MoveIfAbsent(l.LegacyLedger(), l.CheatLedger())
	return err
}
