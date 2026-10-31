package paths

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Legacy locations (dev-era builds, findings F16). Every owner reads these as a fallback and moves the
// file into the new layout on first start with MoveIfAbsent; nothing here is removed, and a path that
// exists in the new place always wins.

// HomeDir is the directory legacy locations hang off (Layout.Home overrides os.UserHomeDir).
func (l Layout) HomeDir() string {
	if l.Home != "" {
		return l.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

// LegacyStateDir is where the cheat ledger and the sentinel log used to live (~/.local/state/d2lod).
func (l Layout) LegacyStateDir() string {
	return filepath.Join(l.HomeDir(), ".local", "state", "d2lod")
}

// LegacyLedger is the old cheat ledger path.
func (l Layout) LegacyLedger() string { return filepath.Join(l.LegacyStateDir(), "cheats.jsonl") }

// SentinelLog is the sentinel's JSONL log; LegacySentinelLog its old place.
func (l Layout) SentinelLog() string { return filepath.Join(l.Logs(), "sentinel.jsonl") }
func (l Layout) LegacySentinelLog() string {
	return filepath.Join(l.LegacyStateDir(), "sentinel.jsonl")
}

// CheatBackups is the default folder for save-file backups made by internal/cheat. LegacyCheatBackups is the
// old cold-data folder ($D2_COLD_DIR/d2lod/backups); old ledger entries still point into it, so it is never moved.
func (l Layout) CheatBackups() string { return filepath.Join(l.Root, "ledger", "backups") }
func (l Layout) LegacyCheatBackups() string {
	cold := os.Getenv("D2_COLD_DIR")
	if cold == "" {
		cold = "/mnt/cold"
	}
	return filepath.Join(cold, "d2lod", "backups")
}

// Token is the access-token file of a service (d2watch, ...): <Root>/tokens/<name>. LegacyToken is the
// old ~/.config/<name>/token.
func (l Layout) Token(name string) string { return filepath.Join(l.Root, "tokens", name) }
func (l Layout) LegacyToken(name string) string {
	return filepath.Join(l.HomeDir(), ".config", name, "token")
}

// LegacyGameLock is the old /tmp lock of an instance (i0 had no suffix).
func LegacyGameLock(id string) string {
	if id == "i0" || id == "" {
		return "/tmp/d2lod-game.lock"
	}
	return "/tmp/d2lod-game-" + id + ".lock"
}

// GameLocks lists every lock file a run that drives instance id must hold: the current one first, then the
// legacy /tmp one. Holding both keeps mutual exclusion with tools and services that still use the old path
// (a sentinel that was not redeployed yet); drop the legacy entry once none is left.
func (l Layout) GameLocks(id string) []string {
	out := []string{}
	if i, err := l.Instance(id); err == nil {
		out = append(out, i.GameLock())
	}
	return append(out, LegacyGameLock(id))
}

// GameRoot is the Wine-side root of instance id (Proton prefix under proton/, helper/, logs/). The new
// layout is <Root>/prefixes/<id>. A machine that still has the dev-era ~/Games/d2lod (i0) keeps using it,
// and its extra instances stay under ~/Games/d2lod-inst/<id>: a Wine prefix holds absolute paths and is
// hardlink-cloned, so it is never moved under a running game; to migrate, stop the fleet and move the
// directories by hand. The decision is made on i0 so all instances of a machine use one scheme.
func (l Layout) GameRoot(id string) string {
	legacy := filepath.Join(l.HomeDir(), "Games", "d2lod")
	if dirExists(l.PrefixesDir()+"/i0") || !dirExists(legacy) {
		return filepath.Join(l.PrefixesDir(), id)
	}
	return SiblingRoot(legacy, id)
}

// SiblingRoot is the root of instance id given the root of i0: ~/Games/d2lod -> ~/Games/d2lod-inst/<id>
// (legacy naming), anything else -> a sibling directory named <id>.
func SiblingRoot(i0Root, id string) string {
	if id == "i0" || id == "" {
		return i0Root
	}
	if filepath.Base(i0Root) == "d2lod" {
		return filepath.Join(filepath.Dir(i0Root), "d2lod-inst", id)
	}
	return filepath.Join(filepath.Dir(i0Root), id)
}

func dirExists(p string) bool { st, err := os.Stat(p); return err == nil && st.IsDir() }

// MoveIfAbsent moves file from to to when from exists and to does not (to and its parent are created). It
// reports whether a move happened. Across file systems it copies and then removes the source.
func MoveIfAbsent(from, to string) (bool, error) {
	if st, err := os.Stat(from); err != nil || st.IsDir() {
		return false, nil
	}
	if _, err := os.Stat(to); err == nil {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(from, to); err == nil {
		return true, nil
	}
	if err := copyFile(from, to); err != nil {
		return false, err
	}
	return true, os.Remove(from)
}

// CopyTree copies the regular files of directory from into to (created), never overwriting.
func CopyTree(from, to string) error {
	if err := os.MkdirAll(to, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return copyFile(p, dst)
	})
}

func copyFile(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	mode := os.FileMode(0o644)
	if st, err := in.Stat(); err == nil {
		mode = st.Mode().Perm()
	}
	tmp := to + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, to)
}
