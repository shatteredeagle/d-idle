package main

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"d2lod-bot/internal/saves"
)

// savedExpReader returns bot.Deps.SavedExp: the exp stored in the character's .d2s, re-read only when the file's
// mtime or size changes (the game rewrites it on Save and Exit). ok is false while the name or file is unknown or
// unreadable; a half-written file keeps the last good value.
func savedExpReader(dir func() string, name func() string, read func(string) (int, uint32, error)) func() (uint32, bool) {
	var (
		mu    sync.Mutex
		mtime time.Time
		size  int64
		path  string
		exp   uint32
		have  bool
	)
	return func() (uint32, bool) {
		d, n := dir(), name()
		if d == "" || n == "" {
			return 0, false
		}
		p := filepath.Join(d, n+".d2s")
		st, err := os.Stat(p)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			return 0, false
		}
		if have && p == path && st.ModTime().Equal(mtime) && st.Size() == size {
			return exp, true
		}
		_, e, err := read(p)
		if err != nil {
			return exp, have && p == path
		}
		path, mtime, size, exp, have = p, st.ModTime(), st.Size(), e, true
		return exp, true
	}
}

func savedExpDep(dir string, name func() string) func() (uint32, bool) {
	return savedExpReader(func() string { return dir }, name, saves.ExpFile)
}
