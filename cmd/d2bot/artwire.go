package main

import (
	"log/slog"
	"path/filepath"

	"d2lod-bot/internal/art"
	"d2lod-bot/internal/watch"
)

// artSource opens the game art of the install for the watch pages (internal/art). nil without an install or when
// its archives cannot be read: the pages then keep their drawn cells. The disk cache is <data>/cache/art; nothing
// is written next to the code or into an image layer.
func artSource(gameDir, dataDir string) watch.ArtSource {
	if gameDir == "" {
		return nil
	}
	s, err := art.Open(gameDir, filepath.Join(dataDir, "cache", "art"))
	if err != nil {
		slog.Info("game art unavailable, the inventory page draws plain cells", "err", err)
		return nil
	}
	return s
}

// artAPI is what newFun mounts: always present, so /api/art can say available:false.
func artAPI(src watch.ArtSource) *watch.ArtAPI {
	a := watch.NewArtAPI()
	a.Bind(src)
	return a
}
