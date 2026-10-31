package main

import (
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// Backend is the label every stream-facing view shows: this viewer only ever looks at the reimplementation.
const Backend = "d2d (reimplementation)"

// Engine names the game engine in the status row; badge is the header text (flag -badge, env D2DVIEW_BADGE).
const Engine = "d2d"

var badge = Backend

// Pub is the stream-safe state: the combined page (/stream) reads only this, never the full Snap (which carries
// socket errors, the raw bot status and monster lists). Every free-text field goes through safeText.
type Pub struct {
	Backend string  `json:"backend"`
	Engine  string  `json:"engine"`
	Live    bool    `json:"live"` // a d2d answered the last poll
	Speed   float64 `json:"speed"`
	Area    string  `json:"area"`
	Name    string  `json:"name"`
	Class   string  `json:"class"`
	CLvl    int     `json:"clvl"`
	XP      int     `json:"xp"`
	Kills   int64   `json:"kills"`
	Deaths  int64   `json:"deaths"`
	Goal    string  `json:"goal"`
	MapVer  uint64  `json:"mapVer"`
	Level   int     `json:"level"`
	// the map pane draws these (cells, not names)
	X, Y    float64
	MapRect [4]int
	Mons    []PubMon `json:"mons"`
}

type PubMon struct {
	X, Y       float64
	Boss, Dead bool
	Friend     bool
}

var (
	unsafeChars = regexp.MustCompile(`[^A-Za-z0-9 ,.'()+:%-]`)
	ipLike      = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
	secretWords = regexp.MustCompile(`(?i)token|secret|password|passwd|key|\.sock|\.lan\b|localhost`)
)

// hostWord reports whether s names the worker or NAS host (first label of D2_WORKER_HOST / D2_NAS_HOST), so a
// lab hostname never reaches the public feed. Unset or the generic .example defaults match nothing.
func hostWord(s string) bool {
	ls := strings.ToLower(s)
	for _, k := range []string{"D2_WORKER_HOST", "D2_NAS_HOST"} {
		h := strings.ToLower(os.Getenv(k))
		if strings.HasSuffix(h, ".example") {
			continue
		}
		if l, _, _ := strings.Cut(h, "."); len(l) >= 3 && strings.Contains(ls, l) {
			return true
		}
	}
	return false
}

// safeText keeps a plain-text label: no paths, URLs, hosts, addresses or anything that looks like a credential.
// A suspicious string is dropped whole rather than scrubbed.
func safeText(s string, max int) string {
	if strings.ContainsAny(s, "/\\~@") || ipLike.MatchString(s) || secretWords.MatchString(s) || hostWord(s) {
		return ""
	}
	s = strings.Join(strings.Fields(unsafeChars.ReplaceAllString(s, " ")), " ")
	if len(s) > max {
		s = strings.TrimSpace(s[:max])
	}
	return s
}

func pubOf(s Snap) Pub {
	p := Pub{Backend: badge, Engine: Engine, Live: s.OK, Speed: s.Speed, Area: safeText(s.Area, 40), Name: safeText(s.Name, 20),
		Class: safeText(s.Class, 20), CLvl: s.CLvl, XP: s.XP, Kills: int64(s.Dead), MapVer: s.MapVer, Level: s.Level,
		X: s.X, Y: s.Y, MapRect: s.MapRect}
	if len(s.Bot) > 0 {
		var b struct {
			Totals struct{ Kills, Deaths int64 }
			Runner struct{ Activity string }
			Active *struct{ Title string } `json:"activeGoal"`
		}
		if json.Unmarshal(s.Bot, &b) == nil {
			p.Kills, p.Deaths = max(p.Kills, b.Totals.Kills), b.Totals.Deaths
			p.Goal = safeText(b.Runner.Activity, 50)
			if b.Active != nil {
				if t := safeText(b.Active.Title, 50); t != "" {
					p.Goal = t + map[bool]string{true: ": " + p.Goal, false: ""}[p.Goal != ""]
				}
			}
		}
	}
	for _, m := range s.Mons {
		p.Mons = append(p.Mons, PubMon{X: m.X, Y: m.Y, Boss: m.Boss, Dead: m.Dead, Friend: m.Friend})
	}
	return p
}

func (s *server) pub(w http.ResponseWriter, r *http.Request) {
	s.feed.Touch()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(pubOf(s.feed.Snap()))
}

func (s *server) streamPage(w http.ResponseWriter, r *http.Request) {
	b, _ := webFS.ReadFile("web/stream.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(b)
}
