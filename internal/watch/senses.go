package watch

import (
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/senses"
)

// SensesRoutes are the GET routes of the side-panel minimap (docs/senses-map.md). Mount them on the viewer behind the
// same gate as /api/return (serve.go does when ServeOptions.Senses is set).
var SensesRoutes = []string{"/api/senses", "/watch/senses_units.js"}

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

// TrailSource is an optional part of a SensesSource (senses.Publisher): the hero's recent positions in the current area.
type TrailSource interface {
	TrailOf() (pts []senses.TrailPoint, area int, now time.Time, ok bool)
}

// CoverageSource is an optional part of a SensesSource: the navigator's roam coverage of the current area.
type CoverageSource interface {
	CoverageOf() (senses.Coverage, bool)
}

// SenseCoverage is the additive coverage key of /api/senses: Cells are [x, y, state] (0 unseen, 1 seen, 2 failed),
// State is "roaming" or "explored", Total counts the points before the cap.
type SenseCoverage struct {
	Area   int      `json:"area"`
	Cell   int      `json:"cell"`
	Cover  int      `json:"cover"`
	Cells  [][3]int `json:"cells"`
	Total  int      `json:"total"`
	Target *[2]int  `json:"target,omitempty"`
	State  string   `json:"state"`
	Legs   int      `json:"legs"`
	Held   int      `json:"held,omitempty"`
}

// SenseTrailPoint is one trail sample: AgoMS is its age at the reply's Ts, E marks the entry into the area.
type SenseTrailPoint struct {
	AgoMS int64 `json:"ago"`
	X     int   `json:"x"`
	Y     int   `json:"y"`
	E     bool  `json:"e,omitempty"`
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
	// Neighbours are visited areas stitched beside Map (internal/senses adjacency.go), tier 1 like the map itself.
	Neighbours []SenseNeighbour `json:"neighbours,omitempty"`
	// Trail is the hero's path in the current area, oldest first (tier 1, additive: older clients ignore it). TrailArea
	// is its area id and Ts the server clock in unix ms the ages count from.
	Trail     []SenseTrailPoint `json:"trail,omitempty"`
	TrailArea int               `json:"trail_area,omitempty"`
	Ts        int64             `json:"ts,omitempty"`
	// Coverage is the roam coverage grid of the current area (tier 1, additive), absent when the bot is not roaming.
	Coverage *SenseCoverage `json:"coverage,omitempty"`
	Waiting  bool           `json:"waiting,omitempty"`
	// Locked lists the tiers not open yet (titles and hints only) and Next is the lowest of them; both are present only
	// when the Unlocker is a Teaser. A locked tier's data keys stay absent.
	Locked []SenseLock `json:"locked,omitempty"`
	Next   *SenseLock  `json:"next,omitempty"`
}

// SenseNeighbour is a neighbour area in the current area's coordinate space: add DX, DY to its subtile coordinates.
// Map is left out when the caller already holds that version (?nb=<areaId>:<version>,...).
type SenseNeighbour struct {
	AreaID  int         `json:"areaId"`
	Version uint64      `json:"version"`
	DX      int         `json:"dx"`
	DY      int         `json:"dy"`
	Map     *senses.Map `json:"map,omitempty"`
}

// parseHave reads ?nb=2:5,1:3 into areaId -> version; malformed items are ignored (the map is then simply sent).
func parseHave(s string) map[int]uint64 {
	out := map[int]uint64{}
	for _, it := range strings.Split(s, ",") {
		a, v, ok := strings.Cut(it, ":")
		if !ok {
			continue
		}
		ai, e1 := strconv.Atoi(a)
		vi, e2 := strconv.ParseUint(v, 10, 64)
		if e1 == nil && e2 == nil {
			out[ai] = vi
		}
	}
	return out
}

const maxTier = 7 // the ladder of info-unlocks.md has seven rungs (tier 0 is the video alone)

// Reply builds the answer for a caller that holds map version have.
func (s *Senses) Reply(have uint64) SensesReply { return s.ReplyNb(have, nil) }

// ReplyNb is Reply for a caller that also holds the given neighbour map versions (areaId -> version).
func (s *Senses) ReplyNb(have uint64, haveNb map[int]uint64) SensesReply {
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
		if cs, ok := s.Src.(CoverageSource); ok {
			if c, ok := cs.CoverageOf(); ok {
				sc := &SenseCoverage{Area: c.Area, Cell: c.Cell, Cover: c.Cover, Cells: c.Cells, Total: c.Total, State: "roaming", Legs: c.Legs, Held: c.Held}
				if c.Explored {
					sc.State = "explored"
				}
				if c.Target != nil {
					sc.Target = &[2]int{c.Target.X, c.Target.Y}
				}
				r.Coverage = sc
			}
		}
		if ts, ok := s.Src.(TrailSource); ok {
			if pts, area, now, ok := ts.TrailOf(); ok && len(pts) > 0 {
				r.TrailArea, r.Ts = area, now.UnixMilli()
				for _, p := range pts {
					r.Trail = append(r.Trail, SenseTrailPoint{AgoMS: now.Sub(p.At).Milliseconds(), X: p.X, Y: p.Y, E: p.Entry})
				}
			}
		}
		for _, n := range snap.Neighbours {
			sn := SenseNeighbour{AreaID: n.AreaID, Version: n.Version, DX: n.DX, DY: n.DY}
			if v, ok := haveNb[n.AreaID]; !ok || v != n.Version {
				m := n.Map
				sn.Map = &m
			}
			r.Neighbours = append(r.Neighbours, sn)
		}
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
		writeJSON(w, 200, s.ReplyNb(have, parseHave(r.URL.Query().Get("nb"))))
	})
	for route, file := range map[string]string{"/watch/senses_units.js": "web/senses_units.js"} {
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
