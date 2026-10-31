package cheat

import "path/filepath"

// DefaultSaveDir is the folder with the instance's .d2s files (one Wine prefix per instance, docs/multi.md).
func DefaultSaveDir(inst string) string {
	return filepath.Join(gameRoot(inst), "proton", "pfx", "drive_c", "users", "steamuser", "Saved Games", "Diablo II")
}

// DefaultInstallDir is the instance's own copy of the game folder (a hardlink clone, so changing files
// there never touches another instance).
func DefaultInstallDir(inst string) string {
	return filepath.Join(gameRoot(inst), "proton", "pfx", "drive_c", "Program Files (x86)", "Diablo II")
}

// DefaultDataDir is where the game looks for loose data files when started with -direct -txt.
func DefaultDataDir(inst string) string { return filepath.Join(DefaultInstallDir(inst), "Data") }
