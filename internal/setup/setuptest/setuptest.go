// Package setuptest holds fake host effects for the setup wizard, so the whole first-run flow (game-dir
// mode) runs offline in tests and in tools/container-checklist.sh. Nothing here touches Wine or a game.
package setuptest

import (
	"context"
	"os"
	"path/filepath"

	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/setup"
)

// Deps is the game-dir plan with every host effect faked: version and patch hash pass, the prefix is a
// directory with a system.reg, extraction is a flag, character creation writes <name>.d2s.
func Deps(data string) *setup.Deps {
	env := setup.NewEnv()
	env.GameVersion = func(string) (string, error) { return "1.14.3.71", nil }
	env.SHA256 = func(string) (string, error) { return launch.PatchSHA256, nil }
	d := setup.NewDeps(env)
	d.Config = func(s *setup.State) (launch.Config, error) {
		return launch.Config{Root: s.DataDir, PrefixDir: setup.BasePrefix(s.DataDir), GameDir: setup.GameDirIn(setup.BasePrefix(s.DataDir))}, nil
	}
	d.Prefix = func(ctx context.Context, c launch.Config) error {
		p := filepath.Join(c.PrefixDir, "pfx")
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(p, "system.reg"), []byte("r"), 0o644)
	}
	d.TablesDir = func(dd, g string) (string, error) { return filepath.Join(dd, "cache", "t"), nil }
	d.CacheCurrent = func(string, string) bool {
		_, err := os.Stat(filepath.Join(data, "cache", "t", "extracted"))
		return err == nil
	}
	d.Extract = func(string, string) error {
		p := filepath.Join(data, "cache", "t")
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(p, "extracted"), []byte("1"), 0o644)
	}
	d.CreateCharacter = func(ctx context.Context, c launch.Config, n, cl string) error {
		if err := os.MkdirAll(c.SaveDir(), 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(c.SaveDir(), n+".d2s"), []byte(cl), 0o644)
	}
	d.RunWindowed = func(context.Context, launch.Config, string, string, func() error) error { return nil }
	// the helper binary the doctor looks for
	helper := filepath.Join(data, "helper", "d2input.exe")
	if os.MkdirAll(filepath.Dir(helper), 0o755) == nil {
		_ = os.WriteFile(helper, []byte("h"), 0o644)
	}
	return d
}

// InstallTree writes a minimal fake Diablo II folder (empty-ish files, right names) for game-dir mode.
func InstallTree(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range setup.GameFiles() {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644); err != nil {
			return err
		}
	}
	return nil
}
