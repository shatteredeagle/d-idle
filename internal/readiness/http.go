package readiness

import (
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
)

// Route is GET /api/readiness[?area=Name or id][&difficulty=normal|nightmare|hell]. Read-only: it answers from the hero
// snapshot and the tables and orders nothing (docs/ui-design-system.md section 0: a new endpoint, no existing one changes).
const Route = "/api/readiness"

// Source supplies the hero snapshot and the area the hero stands in ("" area when unknown). ok is false until the first
// in-game read.
type Source func() (h Hero, here string, ok bool)

// View is the response body.
type View struct {
	Schema string  `json:"schema"`
	Report *Report `json:"report,omitempty"`
	Error  string  `json:"error,omitempty"`
}

// Handler serves Route. The asked area and difficulty are matched against the tables, never echoed unchecked.
func Handler(ev *Evaluator, src Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		v := View{Schema: ReportSchema}
		h, here, ok := src()
		switch {
		case !ok:
			v.Error = "the hero's numbers are not read yet"
		default:
			area := clean(r.URL.Query().Get("area"), 40)
			if area == "" {
				area = here
			}
			if d := clean(r.URL.Query().Get("difficulty"), 12); d != "" {
				h.Difficulty = d
			}
			rep, err := ev.Evaluate(h, area)
			if err != nil {
				v.Error = err.Error()
			} else {
				v.Report = &rep
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(v)
	})
}

func clean(s string, n int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '\'' {
			return r
		}
		return -1
	}, s)
	if len(s) > n {
		s = s[:n]
	}
	return strings.TrimSpace(s)
}
