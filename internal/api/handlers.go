package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/goals"
)

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(Prefix+"/", s.apiRoute)
	if s.cfg.Assets != nil {
		files := http.FileServer(http.FS(s.cfg.Assets))
		mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache")
			files.ServeHTTP(w, r)
		}))
	}
	return s.guard(mux)
}

func method(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, m := range allowed {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeJSON(w, http.StatusMethodNotAllowed, errBody{"method not allowed", "method_not_allowed"})
	return false
}

func (s *Server) apiRoute(w http.ResponseWriter, r *http.Request) {
	m := s.cfg.Manager
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, s.base), "/"), "/")
	switch {
	case len(parts) == 1 && parts[0] == "status":
		if method(w, r, "GET") {
			writeJSON(w, 200, s.status())
		}
	case len(parts) == 1 && parts[0] == "catalog":
		if method(w, r, "GET") {
			writeJSON(w, 200, goals.Catalog())
		}
	case len(parts) == 1 && parts[0] == "settings":
		s.settings(w, r)
	case len(parts) == 2 && parts[0] == "control":
		if !method(w, r, "POST") {
			return
		}
		var err error
		switch parts[1] {
		case "start":
			err = m.Start()
		case "pause":
			err = m.Pause()
		case "stop":
			err = m.Stop()
		default:
			writeJSON(w, 404, errBody{"unknown control action", "not_found"})
			return
		}
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, s.status())
	case parts[0] == "goals":
		s.goalsRoute(w, r, parts[1:])
	case len(parts) == 1 && parts[0] == "events":
		if method(w, r, "GET") {
			since, _ := strconv.ParseUint(r.URL.Query().Get("since"), 10, 64)
			limit := atoiDefault(r.URL.Query().Get("limit"), 200)
			evs := m.Events(since, min(limit, 1000))
			if evs == nil {
				evs = []goals.Event{}
			}
			writeJSON(w, 200, evs)
		}
	case len(parts) == 2 && parts[0] == "events" && parts[1] == "stream":
		if method(w, r, "GET") {
			s.stream(w, r)
		}
	case len(parts) == 1 && parts[0] == "stats":
		if method(w, r, "GET") {
			writeJSON(w, 200, m.StatsSummary())
		}
	case len(parts) == 2 && parts[0] == "stats" && parts[1] == "series":
		if !method(w, r, "GET") {
			return
		}
		q := r.URL.Query()
		win, err1 := parseDur(q.Get("window"), 6*time.Hour)
		step, err2 := parseDur(q.Get("step"), 0)
		if err1 != nil || err2 != nil {
			badRequest(w, "window/step must look like 30m, 6h or 7d")
			return
		}
		if step == 0 {
			step = autoStep(win)
		}
		if win/step > 1000 || win < time.Minute {
			badRequest(w, "window/step gives more than 1000 points or less than a minute")
			return
		}
		pts := m.Series(win, step)
		writeJSON(w, 200, map[string]any{"window": int64(win / time.Second), "step": int64(step / time.Second), "points": pts})
	default:
		writeJSON(w, 404, errBody{"no such endpoint", "not_found"})
	}
}

func atoiDefault(s string, d int) int {
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return d
}

func parseDur(s string, def time.Duration) (time.Duration, error) {
	if s == "" {
		return def, nil
	}
	if n, ok := strings.CutSuffix(s, "d"); ok {
		v, err := strconv.Atoi(n)
		if err != nil || v < 0 {
			return 0, fmt.Errorf("bad duration")
		}
		return time.Duration(v) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("bad duration")
	}
	return d, nil
}

func autoStep(win time.Duration) time.Duration {
	switch {
	case win <= 30*time.Minute:
		return time.Minute
	case win <= 3*time.Hour:
		return 5 * time.Minute
	case win <= 12*time.Hour:
		return 15 * time.Minute
	case win <= 48*time.Hour:
		return time.Hour
	}
	return 6 * time.Hour
}

type statusBody struct {
	goals.Snapshot
	Version string      `json:"version,omitempty"`
	Active  *goals.Goal `json:"activeGoal,omitempty"`
}

func (s *Server) status() statusBody {
	snap := s.cfg.Manager.Snapshot()
	b := statusBody{Snapshot: snap, Version: s.cfg.Version}
	if snap.ActiveGoalID != "" {
		if g, err := s.cfg.Manager.Goal(snap.ActiveGoalID); err == nil {
			b.Active = &g
		}
	}
	return b
}

func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	if !method(w, r, "GET", "PUT") {
		return
	}
	if r.Method == "PUT" {
		var in goals.Settings
		if !decode(w, r, &in) {
			return
		}
		if err := s.cfg.Manager.SetSettings(in); err != nil {
			writeErr(w, err)
			return
		}
	}
	writeJSON(w, 200, s.cfg.Manager.Settings())
}

// goalIn is the writable part of a goal.
type goalIn struct {
	Type     goals.Type     `json:"type"`
	Title    string         `json:"title"`
	Params   goals.Params   `json:"params"`
	Priority int            `json:"priority"`
	Schedule []goals.Window `json:"schedule"`
	Position *int           `json:"position"`
}

type goalPatch struct {
	Title    *string         `json:"title"`
	Params   *goals.Params   `json:"params"`
	Priority *int            `json:"priority"`
	Schedule *[]goals.Window `json:"schedule"`
}

func (s *Server) goalsRoute(w http.ResponseWriter, r *http.Request, p []string) {
	m := s.cfg.Manager
	switch {
	case len(p) == 0:
		if !method(w, r, "GET", "POST") {
			return
		}
		if r.Method == "GET" {
			writeJSON(w, 200, m.Goals())
			return
		}
		var in goalIn
		if !decode(w, r, &in) {
			return
		}
		pos := -1
		if in.Position != nil {
			pos = *in.Position
		}
		g, err := m.Add(goals.Goal{Type: in.Type, Title: in.Title, Params: in.Params, Priority: in.Priority, Schedule: in.Schedule}, pos)
		if err != nil {
			writeErr(w, err)
			return
		}
		w.Header().Set("Location", s.base+"/goals/"+g.ID)
		writeJSON(w, 201, g)
	case len(p) == 1 && p[0] == "reorder":
		if !method(w, r, "POST") {
			return
		}
		var in struct {
			IDs []string `json:"ids"`
		}
		if !decode(w, r, &in) {
			return
		}
		if err := m.Reorder(in.IDs); err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, m.Goals())
	case len(p) == 1:
		s.goalOne(w, r, p[0])
	case len(p) == 2:
		if !method(w, r, "POST") {
			return
		}
		id := p[0]
		var err error
		switch p[1] {
		case "pause":
			err = m.PauseGoal(id)
		case "resume":
			err = m.ResumeGoal(id)
		case "skip":
			err = m.Skip(id)
		case "reset":
			err = m.Reset(id)
		case "move":
			var in struct {
				Position *int `json:"position"`
			}
			if !decode(w, r, &in) {
				return
			}
			if in.Position == nil {
				badRequest(w, "position is required")
				return
			}
			err = m.Move(id, *in.Position)
		default:
			writeJSON(w, 404, errBody{"unknown goal action", "not_found"})
			return
		}
		if err != nil {
			writeErr(w, err)
			return
		}
		g, err := m.Goal(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, g)
	default:
		writeJSON(w, 404, errBody{"no such endpoint", "not_found"})
	}
}

func (s *Server) goalOne(w http.ResponseWriter, r *http.Request, id string) {
	m := s.cfg.Manager
	if !method(w, r, "GET", "PATCH", "DELETE") {
		return
	}
	switch r.Method {
	case "GET":
		g, err := m.Goal(id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, g)
	case "DELETE":
		if err := m.Remove(id); err != nil {
			writeErr(w, err)
			return
		}
		w.WriteHeader(204)
	case "PATCH":
		var in goalPatch
		if !decode(w, r, &in) {
			return
		}
		g, err := m.Update(id, func(g *goals.Goal) {
			if in.Title != nil {
				g.Title = *in.Title
			}
			if in.Params != nil {
				g.Params = *in.Params
			}
			if in.Priority != nil {
				g.Priority = *in.Priority
			}
			if in.Schedule != nil {
				g.Schedule = *in.Schedule
			}
		})
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, 200, g)
	}
}

// Status is the /status body of this instance (also pushed on the SSE stream).
func (s *Server) Status() any { return s.status() }
