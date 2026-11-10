package main

import (
	"log/slog"
	"path/filepath"

	"d2lod-bot/internal/art"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/watch"
)

// artSource opens the game art of the install for the watch pages (internal/art). nil without an install or when
// its archives cannot be read: the pages then keep their drawn cells. The disk cache is <data>/cache/art; nothing
// is written next to the code or into an image layer.
//
// It also starts the pre-extraction in the background (every icon the tables name, the slot silhouettes and the
// panel frame), so no page load pays a decode; a cache that is already complete costs one stat per icon, and a
// changed install or table set only fills what is new. It never blocks start-up or the first page paint.
func artSource(gameDir, dataDir string, db *data.DB) watch.ArtSource {
	if gameDir == "" {
		return nil
	}
	s, err := art.Open(gameDir, filepath.Join(dataDir, "cache", "art"))
	if err != nil {
		slog.Info("game art unavailable, the inventory page draws plain cells", "err", err)
		return nil
	}
	s.StartExtract(art.Keys(db.InvFiles()), func(p art.Progress, err error) {
		if err != nil {
			slog.Warn("game art extraction stopped", "done", p.Done, "total", p.Total, "err", err)
			return
		}
		slog.Info("game art ready", "icons", p.Total, "missing", p.Missing)
	})
	return s
}

// artAPI is what newFun mounts: always present, so /api/art can say available:false.
func artAPI(src watch.ArtSource) *watch.ArtAPI {
	a := watch.NewArtAPI()
	a.Bind(src)
	return a
}
