package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed assets/d2idle-watch.svg
var iconSVG []byte

// installDesktop writes a freedesktop entry (menu item with jump-list actions) and the icon for the current user.
// Windows: see docs/desktop-app.md (a shortcut to the exe with --page arguments; no installer yet).
func installDesktop(cfgDir string) error {
	if goos != "linux" {
		return errors.New("--install writes a Linux desktop entry only; on Windows make a shortcut (docs/desktop-app.md)")
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	icon := filepath.Join(home, ".local/share/icons/hicolor/scalable/apps/d2idle-watch.svg")
	entry := filepath.Join(home, ".local/share/applications/d2idle-watch.desktop")
	for _, d := range []string{filepath.Dir(icon), filepath.Dir(entry)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	if err := os.WriteFile(icon, iconSVG, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(entry, []byte(DesktopEntry(exe)), 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", entry)
	return nil
}

// DesktopEntry renders the .desktop file for exe. Exec values are quoted so a path with spaces works.
func DesktopEntry(exe string) string {
	q := `"` + exe + `"`
	return fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=d.idle watch
Comment=Watch the d.idle bot
Exec=%[1]s
Icon=d2idle-watch
Terminal=false
Categories=Game;
StartupWMClass=d2idle-watch
Actions=inventory;policy;return;address;

[Desktop Action inventory]
Name=Inventory
Exec=%[1]s --page inventory

[Desktop Action policy]
Name=Policy
Exec=%[1]s --page policy

[Desktop Action return]
Name=Return screen
Exec=%[1]s --page return

[Desktop Action address]
Name=Change address
Exec=%[1]s --setup
`, q)
}
