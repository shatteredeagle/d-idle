package watch

import (
	"io/fs"
	"net/http"
	"strconv"

	"d2lod-bot/internal/senses"
)

// SensesRoutes are the GET routes of the side-panel minimap (docs/senses-map.md). Mount them on the viewer behind the
// same gate as /api/return (serve.go does when ServeOptions.Senses is set).
var SensesRoutes = []string{"/api/senses", "/watch/senses.js"}

// Unlocker says which senses tiers the watcher has earned. The ladder (TASK-149) plugs in here; this card ships
// FixedTier only. Tier 1 is the map and the hero marker.
type Unlocker interface{ Allowed(tier int) bool }

// FixedTier unlocks every tier up to and including N (knob senses.tier, default 1; 0 turns the panel off).
type FixedTier int

func (f FixedTier) Allowed(tier int) bool { return tier >= 1 && tier <= int(f) }

// SensesSource is the bot's published view of the area (senses.Publisher).
type SensesSource interface {
	Snapshot() (senses.Snapshot, bool)
}

// Senses serves GET /api/senses. It only reads the source; nothing it does reaches the bot.
type Senses struct {
	Src  SensesSource
	Lock Unlocker
}

// SensesReply is /api/senses. A locked tier's key is absent, not empty. The map is left out when the caller already
// holds it (?v=<version>).
type SensesReply struct {
	Schema  string         `json:"schema"`
	Tiers   []int          `json:"tiers_unlocked"`
	Version uint64         `json:"version,omitempty"`
	Map     *senses.Map    `json:"map,omitempty"`
	Hero    *senses.Point  `json:"hero,omitempty"`
	Units   []senses.Point `json:"units,omitempty"`
	Waiting bool           `json:"waiting,omitempty"`
}

const maxTier = 6 // the ladder of info-unlocks.md has six rungs

// Reply builds the answer for a caller that holds map version have.
func (s *Senses) Reply(have uint64) SensesReply {
	r := SensesReply{Schema: senses.Schema, Tiers: []int{}}
	for t := 1; t <= maxTier; t++ {
		if s.Lock != nil && s.Lock.Allowed(t) {
			r.Tiers = append(r.Tiers, t)
		}
	}
	if s.Lock == nil || !s.Lock.Allowed(1) {
		return r
	}
	snap, ok := s.Src.Snapshot()
	if !ok {
		r.Waiting = true
		return r
	}
	r.Version = snap.Map.Version
	if snap.Map.Version != have {
		m := snap.Map
		r.Map = &m
	}
	h := snap.Hero
	r.Hero = &h
	return r
}

// Handler serves /api/senses and the panel script. GET and HEAD only.
func (s *Senses) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/senses", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		have, _ := strconv.ParseUint(r.URL.Query().Get("v"), 10, 64)
		writeJSON(w, 200, s.Reply(have))
	})
	m.HandleFunc("/watch/senses.js", func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(webFS, "web/senses.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", pageCSP)
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
	return m
}
