package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
)

// GET /api/log?tab=now|journal|story|alerts: the log drawer of the cockpit (web/panels/log). One tab is read per request,
// through the viewer's own mux (the existing routes, not a second copy of their code), and normalised to rows the
// drawer only has to draw. Times are "seconds ago" so the page can show them in the viewer's time zone. An open question
// is not a log line: it is a Needs you row, so the Now tab shows the feed (what was said and done), never the questions.
const (
	LogRoute  = "/api/log"
	logSchema = "watch.log/1"
	logRows   = 40
)

// LogRoutes are the viewer GET routes (add them to the viewer's mux, serve.go).
var LogRoutes = []string{LogRoute}

// logSource names the existing route each tab reads.
var logSource = map[string]string{
	"items":   DropsRoute,
	"now":     "/api/thoughts",
	"journal": "/api/journal?limit=80",
	"story":   "/api/chronicle",
	"alerts":  "/api/alerts",
}

// LogTabs lists the tabs in the order the drawer shows them.
var LogTabs = []string{"items", "now", "journal", "story", "alerts"}

// LogRow is one line of a tab.
type LogRow struct {
	ID    string  `json:"id"`
	Ago   float64 `json:"ago"` // seconds before the reply
	Kind  string  `json:"kind"`
	Text  string  `json:"text"`
	Level string  `json:"level,omitempty"`   // alerts: the severity
	Q     string  `json:"quality,omitempty"` // items: the quality of the pickup
}

// LogView is the reply. Rows are newest first. Note says why a tab has no rows (not unlocked, not built into this run).
type LogView struct {
	Schema string   `json:"schema"`
	Tab    string   `json:"tab"`
	Rows   []LogRow `json:"rows"`
	Note   string   `json:"note,omitempty"`
}

// LogHandler serves LogRoute over m, the mux that holds the routes of logSource.
func (h *Hub) LogHandler(m http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "read-only", http.StatusMethodNotAllowed)
			return
		}
		tab := r.URL.Query().Get("tab")
		if tab == "" {
			tab = "now"
		}
		src, ok := logSource[tab]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"schema": logSchema, "error": "unknown tab", "tabs": LogTabs})
			return
		}
		u, _ := url.Parse(src)
		req := r.Clone(r.Context())
		req.URL.Path, req.URL.RawQuery = u.Path, u.RawQuery
		req.RequestURI = ""
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, req)
		v := LogView{Schema: logSchema, Tab: tab, Rows: []LogRow{}}
		if rec.Code != http.StatusOK {
			v.Note = "This is not available in this run."
			writeJSON(w, http.StatusOK, v)
			return
		}
		now := h.rel(h.cfg.Clock.Now())
		logFill(&v, tab, rec.Body.Bytes(), now)
		sort.SliceStable(v.Rows, func(i, j int) bool { return v.Rows[i].Ago < v.Rows[j].Ago })
		if len(v.Rows) > logRows {
			v.Rows = v.Rows[:logRows]
		}
		writeJSON(w, http.StatusOK, v)
	})
}

func logFill(v *LogView, tab string, b []byte, now float64) {
	ago := func(t float64) float64 {
		if a := now - t; a > 0 {
			return a
		}
		return 0
	}
	switch tab {
	case "items":
		var d DropsView
		if json.Unmarshal(b, &d) == nil {
			for _, x := range d.Rows {
				t := "picked up " + x.Item
				if x.Area != "" {
					t += " (" + x.Area + ")"
				}
				v.Rows = append(v.Rows, LogRow{ID: x.ID, Ago: x.Ago, Kind: "drop", Text: t, Q: x.Quality})
			}
			if len(d.Rows) == 0 {
				v.Note = "No items picked up yet."
			}
		}
	case "now":
		var t ThoughtsView
		if json.Unmarshal(b, &t) == nil {
			for _, f := range t.Feed {
				v.Rows = append(v.Rows, LogRow{ID: f.ID, Ago: ago(f.T), Kind: f.Kind, Text: f.Text})
			}
		}
	case "journal":
		var j JournalView
		if json.Unmarshal(b, &j) == nil {
			for _, l := range j.Lines {
				v.Rows = append(v.Rows, LogRow{ID: l.ID, Ago: ago(l.T), Kind: l.Kind, Text: l.Text})
			}
		}
	case "story":
		var c ChronicleView
		if json.Unmarshal(b, &c) == nil {
			if !c.Open {
				v.Note = logFirst(c.Teaser, c.Title, "The story opens later.")
			}
			for _, l := range c.Lines {
				v.Rows = append(v.Rows, LogRow{ID: "c" + strconv.Itoa(l.N), Ago: ago(l.T), Kind: l.Kind, Text: l.Text})
			}
		}
	case "alerts":
		var a AlertsView
		if json.Unmarshal(b, &a) == nil {
			if !a.Open {
				v.Note = logFirst(a.Teaser, a.Title, "Alerts open later.")
			}
			for _, x := range a.Alerts {
				v.Rows = append(v.Rows, LogRow{ID: "a" + strconv.FormatUint(x.Seq, 10), Ago: ago(x.T), Kind: x.Kind, Text: x.Text, Level: x.Severity})
			}
		}
	}
}

func logFirst(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}
