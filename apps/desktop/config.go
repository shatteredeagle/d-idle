package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const appDir = "d2idle-watch"

// Config is the whole stored state: the address and nothing else. The login passphrase is typed on the watch page
// and the session cookie lives in the browser profile (ProfileDir); neither passes through this file.
type Config struct {
	Address string `json:"address"`
}

// ConfigDir is <user config dir>/d2idle-watch (XDG_CONFIG_HOME or ~/.config on Linux, %AppData% on Windows).
func ConfigDir() (string, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, appDir), nil
}

func ProfileDir(cfgDir string) string { return filepath.Join(cfgDir, "profile") }

// LoadConfig returns an empty Config (no error) when the file does not exist yet: that is the first run.
func LoadConfig(dir string) (Config, error) {
	var c Config
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func SaveConfig(dir string, c Config) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	tmp := filepath.Join(dir, "config.json.tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, "config.json"))
}
