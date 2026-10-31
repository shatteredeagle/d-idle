package data

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"d2lod-bot/internal/paths"
)

// CacheDir is the extraction directory of an install under the data layout: <data>/cache/data/v<schema>-<stamp>/.
// The stamp is a short hash of InstallStamp (file sizes and the patch hash), so installs and extraction
// schemas never overwrite each other and an old directory is simply unused after a game patch.
func CacheDir(installDir string, l paths.Layout) (string, error) {
	st, err := InstallStamp(installDir)
	if err != nil {
		return "", err
	}
	b, _ := json.Marshal(st)
	sum := sha256.Sum256(b)
	return l.DataTables(fmt.Sprintf("v%d", st.Schema), hex.EncodeToString(sum[:6])), nil
}

// OpenCached is Open with the cache in the data layout. Each legacy dir (an older cache location such as
// ~/.cache/d2lod-cheat-data or <data>/cache/tables) is checked first: when it is current for this install it
// is copied into the new place instead of re-extracting, and left alone.
func OpenCached(installDir string, l paths.Layout, legacy ...string) (*DB, error) {
	dir, err := CacheDir(installDir, l)
	if err != nil {
		return nil, err
	}
	if !CacheCurrent(installDir, dir) {
		migrated := false
		for _, old := range legacy {
			if old == "" || !CacheCurrent(installDir, old) {
				continue
			}
			tmp := dir + ".tmp"
			os.RemoveAll(tmp)
			if err := paths.CopyTree(old, tmp); err != nil {
				os.RemoveAll(tmp)
				continue
			}
			os.RemoveAll(dir)
			if err := os.Rename(tmp, dir); err == nil {
				migrated = true
				break
			}
		}
		if !migrated {
			if err := Extract(installDir, dir); err != nil {
				return nil, err
			}
		}
	}
	return Load(dir)
}
