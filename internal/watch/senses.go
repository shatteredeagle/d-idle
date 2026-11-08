package watch

import (
	"io/fs"
	"net/http"
	"strconv"

	"d2lod-bot/internal/senses"
)

// SensesRoutes are the GET routes of the side-panel minimap (docs/senses-map.md). Mount them on the viewer behind the
// same gate as /api/return (serve.go does when ServeOptions.Senses is set).
var SensesRoutes = []string{"/api/senses", "/watch/senses.js", "/watch/senses_units.js"}

// Unlocker says which senses tiers the watcher has earned. FixedTier is the plain knob; LadderUnlocker (sensesunlock.go)
// reads the unlock ladder. Tier 1 is the map and the hero marker, tier 2 the unit dots.
type Unlocker interface{ Allowed(tier int) bool }

// Teaser is an optional Unlocker part: the locked tiers, for the silhouettes of the panel.
type Teaser interface{ Locked() []SenseLock }

// SenseLock is a locked tier as the panel may show it: the title and the hint, never what it reveals.
type SenseLock struct {
	ID     string `json:"id"`
	Tier   int    `json:"tier"`
	Title  string `json:"title"`
	Teaser string `json:"teaser,omitempty"`
	Price  int64  `json:"price,omitempty"` // tally, when the tier can be bought (POST /api/unlocks/buy)
}

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
	Schema  string        `json:"schema"`
	Tiers   []int         `json:"tiers_unlocked"`
	Version uint64        `json:"version,omitempty"`
	Map     *senses.Map   `json:"map,omitempty"`
	Hero    *senses.Point `json:"hero,omitempty"`
	Units   []PublicUnit  `json:"units,omitempty"`
	Waiting bool          `json:"waiting,omitempty"`
	// Locked lists the tiers not open yet (titles and hints only) and Next is the lowest of them; both are present only
	// when the Unlocker is a Teaser. A locked tier's data keys stay absent.
	Locked []SenseLock `json:"locked,omitempty"`
	Next   *SenseLock  `json:"next,omitempty"`
}

const maxTier = 7 // the ladder of info-unlocks.md has seven rungs (tier 0 is the video alone)

// Reply builds the answer for a caller that holds map version have.
func (s *Senses) Reply(have uint64) SensesReply {
	r := SensesReply{Schema: senses.Schema, Tiers: []int{}}
	allowed := map[int]bool{}
	for t := 1; t <= maxTier; t++ {
		if s.Lock != nil && s.Lock.Allowed(t) {
			allowed[t] = true
			r.Tiers = append(r.Tiers, t)
		}
	}
	if tz, ok := s.Lock.(Teaser); ok {
		r.Locked = tz.Locked()
		if len(r.Locked) > 0 {
			n := r.Locked[0]
			r.Next = &n
		}
	}
	if !allowed[1] && !allowed[2] {
		return r
	}
	snap, ok := s.Src.Snapshot()
	if !ok {
		r.Waiting = true
		return r
	}
	if allowed[1] {
		r.Version = snap.Map.Version
		if snap.Map.Version != have {
			m := snap.Map
			r.Map = &m
		}
		h := snap.Hero
		r.Hero = &h
	}
	if allowed[2] {
		r.Units = publicUnits(snap.Unit, s.Lock, allowed[3], allowed[4])
	}
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
	for route, file := range map[string]string{"/watch/senses.js": "web/senses.js", "/watch/senses_units.js": "web/senses_units.js"} {
		m.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			b, err := fs.ReadFile(webFS, file)
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
	}
	return m
}
